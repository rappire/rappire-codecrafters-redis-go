package list

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

// ====== 작은 uvarint(LEB128) 유틸 ======

// forward: 7비트 + 계속비트(1이면 계속)
func putUvarint(dst []byte, x uint64) int {
	i := 0
	for x >= 0x80 {
		dst[i] = byte(x) | 0x80
		x >>= 7
		i++
	}
	dst[i] = byte(x)
	return i + 1
}

func uvarintLen(x uint64) int {
	n := 1
	for x >= 0x80 {
		x >>= 7
		n++
	}
	return n
}

func readUvarint(buf []byte, off int) (val uint64, n int) {
	shift := 0
	for i := 0; i < 10 && off+i < len(buf); i++ {
		b := buf[off+i]
		val |= uint64(b&0x7F) << shift
		n++
		if (b & 0x80) == 0 {
			return
		}
		shift += 7
	}
	return 0, 0
}

func readUvarintBackward(buf []byte, endIdx int) (val uint64, start int, n int) {
	if endIdx < 0 || endIdx >= len(buf) {
		return 0, 0, 0
	}
	i := endIdx
	n = 1
	for j := endIdx - 1; j >= 0; j-- {
		if buf[j]&0x80 == 0 {
			break
		}
		n++
		i = j
	}
	start = i
	v, nn := readUvarint(buf, start)
	if nn != n {
		return 0, 0, 0
	}
	return v, start, n
}

type Listpack struct {
	buf []byte // [total(4)][count(2)] ... entries ... [0xFF]
}

const (
	lpHeaderBytes = 4 + 2 // total(4) + count(2)
	lpEndMarker   = 0xFF
)

func NewListpack(initialCap int) *Listpack {
	if initialCap < lpHeaderBytes+1 {
		initialCap = lpHeaderBytes + 1
	}
	lp := &Listpack{
		buf: make([]byte, lpHeaderBytes+1, initialCap),
	}
	binary.LittleEndian.PutUint32(lp.buf[0:4], uint32(len(lp.buf)))
	binary.LittleEndian.PutUint16(lp.buf[4:6], 0)
	lp.buf[len(lp.buf)-1] = lpEndMarker
	return lp
}

func (lp *Listpack) total() int {
	return int(binary.LittleEndian.Uint32(lp.buf[0:4]))
}
func (lp *Listpack) count() int {
	return int(binary.LittleEndian.Uint16(lp.buf[4:6]))
}
func (lp *Listpack) setTotal(n int) {
	binary.LittleEndian.PutUint32(lp.buf[0:4], uint32(n))
}
func (lp *Listpack) setCount(n int) {
	binary.LittleEndian.PutUint16(lp.buf[4:6], uint16(n))
}

func (lp *Listpack) endPos() int { return len(lp.buf) - 1 }

// entry: [len(uvarint)][data...][backlen(uvarint)]
func (lp *Listpack) encodeEntry(data []byte) []byte {
	llen := uvarintLen(uint64(len(data)))
	blen := 1
	// blen은 자기 자신을 포함 → 길이가 안정화될 때까지 반복 (최대 3회면 충분)
	for i := 0; i < 3; i++ {
		total := llen + len(data) + blen
		bl2 := uvarintLen(uint64(total))
		if bl2 == blen {
			break
		}
		blen = bl2
	}
	total := llen + len(data) + blen

	out := make([]byte, 0, total)
	tmp := make([]byte, 10)

	// len
	n := putUvarint(tmp, uint64(len(data)))
	out = append(out, tmp[:n]...)
	// data
	out = append(out, data...)
	// backlen
	n = putUvarint(tmp, uint64(total))
	out = append(out, tmp[:n]...)

	return out
}

func (lp *Listpack) insertRawAt(offset int, entry []byte) {
	oldLen := len(lp.buf)
	newLen := oldLen + len(entry)

	lp.buf = append(lp.buf, make([]byte, len(entry))...)
	copy(lp.buf[offset+len(entry):newLen], lp.buf[offset:oldLen])
	copy(lp.buf[offset:offset+len(entry)], entry)

	lp.setCount(lp.count() + 1)
	lp.setTotal(len(lp.buf))
}

func (lp *Listpack) entryOffsetAt(index int) int {
	if index < 0 || index >= lp.count() {
		return -1
	}
	pos := lpHeaderBytes
	cur := 0
	for pos < lp.endPos() {
		if cur == index {
			return pos
		}
		// len
		lv, lN := readUvarint(lp.buf, pos)
		if lN == 0 {
			return -1
		}
		dataEnd := pos + lN + int(lv)
		// backlen
		_, blN := readUvarint(lp.buf, dataEnd)
		if blN == 0 {
			return -1
		}
		total := lN + int(lv) + blN
		pos += total
		cur++
	}
	return -1
}

// 정방향 스캔 유틸
func (lp *Listpack) scanForward(fn func(idx, start, total int, val []byte) bool) {
	pos := lpHeaderBytes
	idx := 0
	for pos < lp.endPos() {
		// len
		lv, lN := readUvarint(lp.buf, pos)
		if lN == 0 {
			return
		}
		dataStart := pos + lN
		dataEnd := dataStart + int(lv)
		// backlen으로 total size 구함
		_, blN := readUvarint(lp.buf, dataEnd)
		if blN == 0 {
			return
		}
		total := lN + int(lv) + blN
		if !fn(idx, pos, total, lp.buf[dataStart:dataEnd]) {
			return
		}
		pos += total
		idx++
	}
}

func (lp *Listpack) AppendBack(data []byte) {
	entry := lp.encodeEntry(data)
	lp.insertRawAt(lp.endPos(), entry) // END 앞
}
func (lp *Listpack) AppendFront(data []byte) {
	entry := lp.encodeEntry(data)
	lp.insertRawAt(lpHeaderBytes, entry) // header 다음
}
func (lp *Listpack) InsertAt(index int, data []byte) {
	if index <= 0 {
		lp.AppendFront(data)
		return
	}
	if index >= lp.count() {
		lp.AppendBack(data)
		return
	}
	insertOff := lp.entryOffsetAt(index)
	if insertOff < 0 {
		lp.AppendBack(data)
		return
	}
	entry := lp.encodeEntry(data)
	lp.insertRawAt(insertOff, entry)
}

func (lp *Listpack) DeleteAt(index int) bool {
	if index < 0 || index >= lp.count() {
		return false
	}
	start := lp.entryOffsetAt(index)
	if start < 0 {
		return false
	}
	// total size 산출
	lv, lN := readUvarint(lp.buf, start)
	if lN == 0 {
		return false
	}
	dataEnd := start + lN + int(lv)
	_, blN := readUvarint(lp.buf, dataEnd)
	if blN == 0 {
		return false
	}
	total := lN + int(lv) + blN

	// 구간 삭제
	copy(lp.buf[start:], lp.buf[start+total:])
	lp.buf = lp.buf[:len(lp.buf)-total]

	// header
	lp.setCount(lp.count() - 1)
	lp.setTotal(len(lp.buf))
	return true
}

// 역방향 순회: END 바로 앞의 backlen 끝바이트로부터 시작.
func (lp *Listpack) scanBackward(fn func(idxFromEnd, start, total int, val []byte) bool) {
	// pos는 "마지막 엔트리의 backlen 마지막 바이트" 위치부터 시작
	if lp.count() == 0 {
		return
	}
	pos := lp.endPos() - 1 // EOF 바로 앞
	idx := 0
	for pos >= lpHeaderBytes {
		// 현재 pos는 어떤 엔트리의 backlen 끝바이트(= 마지막 바이트)여야 한다.
		totalV, blStart, blN := readUvarintBackward(lp.buf, pos)
		if blN == 0 {
			return
		}
		total := int(totalV)
		start := blStart - (total - blN)
		if start < lpHeaderBytes {
			return
		}

		// len 읽고 value 추출
		lv, lN := readUvarint(lp.buf, start)
		if lN == 0 {
			return
		}
		data := lp.buf[start+lN : start+lN+int(lv)]

		if !fn(idx, start, total, data) {
			return
		}
		// 다음은 이번 엔트리 시작 바로 앞 바이트가 다음 pos
		pos = start - 1
		idx++
		// 종료 조건: header 경계를 넘어가면 끝
		if pos < lpHeaderBytes {
			return
		}
	}
}

func (lp *Listpack) Values() [][]byte {
	out := make([][]byte, 0, lp.count())
	lp.scanForward(func(i, start, total int, val []byte) bool {
		cp := make([]byte, len(val))
		copy(cp, val)
		out = append(out, cp)
		return true
	})
	return out
}

func (lp *Listpack) Get(index int) ([]byte, bool) {
	offset := lp.entryOffsetAt(index)
	if offset < 0 {
		return nil, false
	}

	length := int(lp.buf[offset])
	offset += 1
	if offset+length > len(lp.buf) {
		return nil, false
	}

	return lp.buf[offset : offset+length], true
}

func (lp *Listpack) DebugDump(title string) {
	fmt.Printf("== %s == len=%d cap=%d total=%d count=%d\n",
		title, len(lp.buf), cap(lp.buf), lp.total(), lp.count())
	fmt.Println(hex.Dump(lp.buf))
}

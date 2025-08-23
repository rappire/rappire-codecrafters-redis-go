package list

const (
	mergeLimit = 3
	splitLimit = 10
)

type quickNode struct {
	prev *quickNode
	next *quickNode
	lp   *Listpack
}

func (n *quickNode) isSplitNeed() bool {
	return n.lp.count() > splitLimit
}

func (n *quickNode) isMergeNeed() bool {
	return n.lp.count() < mergeLimit
}

func (n *quickNode) count() int {
	return n.lp.count()
}

func newNode(prev *quickNode, next *quickNode) *quickNode {
	return &quickNode{prev: prev, next: next, lp: NewListpack(10)}
}

type QuickList struct {
	head *quickNode
	tail *quickNode
	size int
}

func NewQuickList() *QuickList {
	n := newNode(nil, nil)
	return &QuickList{head: n, tail: n, size: 0}
}

func (q *QuickList) Len() int {
	return q.size
}

func (q *QuickList) RPush(value [][]byte) int {
	q.size += len(value)

	for i := 0; i < len(value); i++ {
		q.tail.lp.AppendBack(value[i])
		if q.tail.isSplitNeed() {
			q.splitNode(q.tail)
		}
	}

	return q.size
}

func (q *QuickList) LPush(value [][]byte) int {
	q.size += len(value)

	for i := 0; i < len(value); i++ {
		q.head.lp.AppendFront(value[i])
		if q.head.isSplitNeed() {
			q.splitNode(q.tail)
		}
	}

	return q.size
}

func (q *QuickList) scanForward(index int) (*quickNode, int) {
	cur := 0
	now := q.head
	for {
		if now.count()+cur >= index {
			return now, index - cur
		}
		cur += now.count()
		if now.next == nil {
			return nil, -1
		}
		now = now.next
	}
}

func (q *QuickList) scanBackward(index int) (*quickNode, int) {
	cur := 0
	now := q.tail
	for {
		if now.count()+cur >= index {
			return now, index - cur
		}
		cur += now.count()
		if now.prev == nil {
			return nil, -1
		}
		now = now.prev
	}
}

func (q *QuickList) LRange(start int, end int) [][]byte {
	if q.size == 0 {
		return [][]byte{}
	}
	// 음수 인덱스 처리
	if start < 0 {
		start = q.size + start
	}
	if end < 0 {
		end = q.size + end
	}
	if start < 0 {
		start = 0
	}
	if end >= q.size {
		end = q.size - 1
	}
	if start > end {
		return [][]byte{}
	}

	out := make([][]byte, 0, end-start+1)

	node, offset := q.scanForward(start)
	idx := start
	for node != nil && idx <= end {
		values := node.lp.Values()
		for offset < len(values) && idx <= end {
			cp := make([]byte, len(values[offset]))
			copy(cp, values[offset])
			out = append(out, cp)
			idx++
			offset++
		}
		node = node.next
		offset = 0
	}
	return out
}

func (q *QuickList) LPop(count int) [][]byte {
	if q.size == 0 || count <= 0 {
		return [][]byte{}
	}

	if count > q.size {
		count = q.size
	}

	out := make([][]byte, 0, count)
	remain := count

	for remain > 0 && q.head != nil {
		values := q.head.lp.Values()
		n := len(values)
		if n == 0 {
			// 비어있으면 다음 노드로 이동
			if q.head.next != nil {
				q.head = q.head.next
				q.head.prev = nil
				continue
			} else {
				break
			}
		}

		toPop := remain
		if toPop > n {
			toPop = n
		}

		// 값 복사
		for i := 0; i < toPop; i++ {
			cp := make([]byte, len(values[i]))
			copy(cp, values[i])
			out = append(out, cp)
		}

		// 실제 삭제 (뒤에서부터 삭제하면 index 안전)
		for i := toPop - 1; i >= 0; i-- {
			q.head.lp.DeleteAt(i)
		}

		q.size -= toPop
		remain -= toPop

		// 노드가 작아졌으면 merge 시도
		if q.head.count() == 0 && q.head.next != nil {
			q.head = q.head.next
			q.head.prev = nil
		} else if q.head.isMergeNeed() && q.head.next != nil {
			q.mergeNode(q.head)
		}
	}

	return out
}

func (q *QuickList) splitNode(n *quickNode) {
	count := n.lp.count()
	if count <= splitLimit {
		return
	}
	mid := count / 2
	values := n.lp.Values()

	newN := newNode(n, n.next)
	if n.next != nil {
		n.next.prev = newN
	}
	n.next = newN
	if q.tail == n {
		q.tail = newN
	}

	for i := mid; i < count; i++ {
		newN.lp.AppendBack(values[i])
	}
	// 원래 노드에서 뒤쪽 절반 삭제
	for i := count - 1; i >= mid; i-- {
		n.lp.DeleteAt(i)
	}
}

func (q *QuickList) mergeNode(n *quickNode) {
	if n.prev != nil && n.count()+n.prev.count() <= splitLimit {
		for _, v := range n.lp.Values() {
			n.prev.lp.AppendBack(v)
		}
		n.prev.next = n.next
		if n.next != nil {
			n.next.prev = n.prev
		} else {
			q.tail = n.prev
		}
		return
	}
	if n.next != nil && n.count()+n.next.count() <= splitLimit {
		for _, v := range n.next.lp.Values() {
			n.lp.AppendBack(v)
		}
		n.next = n.next.next
		if n.next != nil {
			n.next.prev = n
		} else {
			q.tail = n
		}
	}
}

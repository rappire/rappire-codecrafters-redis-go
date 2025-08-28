package server

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/codecrafters-io/redis-starter-go/app/client"
	"github.com/codecrafters-io/redis-starter-go/app/commands"
	"github.com/codecrafters-io/redis-starter-go/app/protocol"
	"github.com/codecrafters-io/redis-starter-go/app/store"
	"github.com/codecrafters-io/redis-starter-go/app/transaction"
	"github.com/codecrafters-io/redis-starter-go/app/types"
)

type Server struct {
	listener      net.Listener
	eventChan     chan types.CommandEvent
	commandManger *commands.CommandManger
	shutdownCh    chan struct{}
	wg            sync.WaitGroup
	client        *client.Client
	info          *types.ServerInfo
}

func NewServer(addr string, replicaOf string, port int) (*Server, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to %s: %v", addr, err)
	}

	newStore := store.NewStore()
	serverInfo := types.NewServerInfo(port, replicaOf)
	fmt.Println("New server info:", serverInfo)
	var newClient *client.Client

	if serverInfo.IsSlave() {
		newClient, err = client.NewClient(serverInfo)
		if err != nil {
			return nil, err
		}
		err = newClient.Init()
		if err != nil {
			return nil, err
		}
	}

	server := &Server{
		listener:      listener,
		eventChan:     make(chan types.CommandEvent, 100), // 버퍼링된 채널
		commandManger: commands.NewCommandManger(newStore, serverInfo),
		shutdownCh:    make(chan struct{}),
		client:        newClient,
		info:          serverInfo,
	}

	return server, nil
}

func (s *Server) Start() {
	if s.info.IsSlave() {
		s.SlaveStart()
		return
	}

	fmt.Printf("Redis server starting on %s\n", s.listener.Addr().String())

	// 이벤트 루프를 별도 고루틴에서 시작
	s.wg.Add(1)
	go s.eventLoop()

	// 클라이언트 연결을 받는 메인 루프
	for {
		select {
		case <-s.shutdownCh:
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				select {
				case <-s.shutdownCh:
					return
				default:
					fmt.Printf("Error accepting connection: %v\n", err)
					continue
				}
			}

			fmt.Printf("New connection: %s\n", conn.RemoteAddr())

			s.wg.Add(1)
			go s.handleConnection(conn)
		}
	}
}

func (s *Server) Stop() {
	fmt.Println("Server shutting down...")

	s.client.CloseClient()

	close(s.shutdownCh)
	s.listener.Close()
	close(s.eventChan)

	s.wg.Wait()
	fmt.Println("Server stopped")
}

func (s *Server) eventLoop() {
	defer s.wg.Done()

	for event := range s.eventChan {
		s.processEvent(event)
	}
}

func (s *Server) processEvent(event types.CommandEvent) {

	handler, exists := s.commandManger.GetHandler(event.Command)
	if !exists {
		event.Ctx.Write(protocol.AppendError(nil, "ERR unknown command '"+event.Command+"'"))
		return
	}

	wrappedHandler := s.wrapHandlerForTransaction(*handler, event.Command)
	wrappedHandler(event)
}

func (s *Server) wrapHandlerForTransaction(handler types.Handler, commandName string) types.Handler {
	return func(e types.CommandEvent) {
		if transaction.IsTransactionCommand(commandName) {
			handler(e)
			return
		}
		if e.Ctx.GetTransaction().IsInTransaction() {
			if e.Ctx.GetTransaction().AddCommand(transaction.Cmd{Name: e.Command, Args: e.Args}) {
				e.Ctx.Write(protocol.AppendString([]byte{}, "QUEUED"))
			} else {
				e.Ctx.Write(protocol.AppendError([]byte{}, "ERR command not allowed in transaction"))
			}
			return
		}
		handler(e)
	}
}

func (s *Server) handleReplicaConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	ctx := types.NewConnContext(conn, transaction.NewTransaction())
	reader := bufio.NewReader(conn)

	for {
		select {
		case <-s.shutdownCh:
			return
		default:
		}

		resp, err := protocol.ReadRESP(reader)
		s.info.AddOffset(len(resp.Raw))
		if err != nil {
			if err.Error() == "EOF" {
				fmt.Printf("Client disconnected: %s\n", conn.RemoteAddr())
				return
			}
			fmt.Printf("Error reading from %s: %v\n", conn.RemoteAddr(), err)
			return
		}

		if resp.Type == protocol.Array && resp.Length > 0 {
			cmd := strings.ToUpper(string(resp.Arr[0].Data))
			args := make([][]byte, 0, resp.Length-1)

			for _, arg := range resp.Arr[1:] {
				args = append(args, arg.Data)
			}

			fmt.Print(cmd)
			for _, arg := range args {
				fmt.Printf(" %s", string(arg))
			}
			fmt.Println()

			select {
			case s.eventChan <- types.CommandEvent{Command: cmd, Args: args, Ctx: ctx}:
			case <-s.shutdownCh:
				return
			}
		}
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	ctx := types.NewConnContext(conn, transaction.NewTransaction())
	reader := bufio.NewReader(conn)

	for {
		select {
		case <-s.shutdownCh:
			return
		default:
		}

		resp, err := protocol.ReadRESP(reader)
		s.info.AddOffset(len(resp.Raw))
		if err != nil {
			if err.Error() == "EOF" {
				fmt.Printf("Client disconnected: %s\n", conn.RemoteAddr())
				return
			}
			fmt.Printf("Error reading from %s: %v\n", conn.RemoteAddr(), err)
			return
		}

		if resp.Type == protocol.Array && resp.Length > 0 {
			cmd := strings.ToUpper(string(resp.Arr[0].Data))
			args := make([][]byte, 0, resp.Length-1)

			for _, arg := range resp.Arr[1:] {
				args = append(args, arg.Data)
			}

			fmt.Print(cmd)
			for _, arg := range args {
				fmt.Printf(" %s", string(arg))
			}
			fmt.Println()

			select {
			case s.eventChan <- types.CommandEvent{Command: cmd, Args: args, Ctx: ctx}:
			case <-s.shutdownCh:
				return
			}
		}
	}
}

func (s *Server) SlaveStart() {
	fmt.Printf("Redis slave starting on %s\n", s.listener.Addr().String())

	// 이벤트 루프를 별도 고루틴에서 시작
	s.wg.Add(2)
	go s.handleConnection(s.client.GetConn())
	go s.eventLoop()

	// 클라이언트 연결을 받는 메인 루프
	for {
		select {
		case <-s.shutdownCh:
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				select {
				case <-s.shutdownCh:
					return
				default:
					fmt.Printf("Error accepting connection: %v\n", err)
					continue
				}
			}

			fmt.Printf("New connection: %s\n", conn.RemoteAddr())

			s.wg.Add(1)
			go s.handleConnection(conn)
		}
	}
}

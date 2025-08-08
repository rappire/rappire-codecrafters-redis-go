package main

import (
	"fmt"
	"net"
	"os"
	"sync"
)

// Ensures gofmt doesn't remove the "net" and "os" imports in stage 1 (feel free to remove this!)
var _ = net.Listen
var _ = os.Exit

type Event struct {
	Task       func()
	Callback   func()
	IsBlocking bool
}

type EventLoop struct {
	mainTasks chan Event
	taskQueue chan Event
	stop      chan bool
}

func Add(eventLoop *EventLoop, event *Event) {
	eventLoop.mainTasks <- *event
}

func AddToTaskQueue(eventLoop *EventLoop, event *Event) {
	eventLoop.taskQueue <- *event
}

func StopEventLoop(eventLoop *EventLoop) {
	eventLoop.stop <- true
}

func initEventLoop(eventLoop *EventLoop) *sync.WaitGroup {
	var wg sync.WaitGroup
	wg.Add(1)
	workerPool := make(chan struct{}, 10)

	func() {
		defer wg.Done()
		for {
			select {
			case task := <-eventLoop.mainTasks:
				if task.IsBlocking {
					workerPool <- struct{}{}
					go func() {
						defer func() {
							<-workerPool
						}()

						task.Task()

						if task.Callback != nil {
							AddToTaskQueue(eventLoop, &Event{
								Task: task.Callback,
							})
						}
					}()
				} else {
					task.Task()
				}
			case task := <-eventLoop.taskQueue:
				task.Task()

			case stop := <-eventLoop.stop:
				if stop {
					return
				}
			}
		}
	}()

	return &wg
}

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}

	eventLoop := EventLoop{
		mainTasks: make(chan Event),
		taskQueue: make(chan Event),
		stop:      make(chan bool),
	}

	wg := initEventLoop(&eventLoop)

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		Add(&eventLoop, &Event{
			Task: func() {
				handleConnection(conn)
			},
			IsBlocking: false,
		})
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	readBuffer := make([]byte, 1024)

	for {
		n, err := conn.Read(readBuffer)
		if err != nil {
			fmt.Println("Error reading: ", err.Error())
			return
		}
		if n > 0 {
			conn.Write([]byte("+PONG\r\n"))
		}
	}
}

// Implementation of a MultiEchoServer. Students should write their code in this file.

package p0

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
)

const OUTGOING_MESSAGE_QUEUE_SIZE = 100
const BROADCAST_MESSAGE_QUEUE_SIZE = 10

type richConn struct {
	connection           net.Conn
	outgoingMessageQueue chan []byte
	closeSignal          chan int
}

type requestMessage struct {
	responseChan chan int
}

type multiEchoServer struct {
	// TODO: implement this!
	listener                     net.Listener
	registerConnections          chan *richConn
	unregisterConnections        chan *richConn
	activeConnections            map[*richConn]bool
	broadcastMessageQueue        chan []byte
	signalRequestConnectionCount chan *requestMessage
	signalCloseMasterRoutine     chan int
	signalCloseAcceptRoutine     chan int
}

type MyError string

func (e MyError) Error() string {
	return string(e)
}

// New creates and returns (but does not start) a new MultiEchoServer.
func New() MultiEchoServer {
	// TODO: implement this!
	return &multiEchoServer{
		registerConnections:          make(chan *richConn),
		unregisterConnections:        make(chan *richConn),
		activeConnections:            make(map[*richConn]bool),
		broadcastMessageQueue:        make(chan []byte, BROADCAST_MESSAGE_QUEUE_SIZE),
		signalRequestConnectionCount: make(chan *requestMessage),
		signalCloseMasterRoutine:     make(chan int, 1),
		signalCloseAcceptRoutine:     make(chan int, 1),
	}
}

func (mes *multiEchoServer) Start(port int) error {
	// TODO: implement this!
	listner, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		myError := MyError("Cannot create listner")
		return myError
	}

	// initialize the listener in multiEchoServer structure
	mes.listener = listner
	// start the master go routine.
	go masterRoutine(mes.registerConnections, mes.unregisterConnections, mes.activeConnections, mes.broadcastMessageQueue, mes.signalCloseMasterRoutine, mes.signalRequestConnectionCount)

	// start a slave go routine which accpets incoming connections.
	go acceptConnections(listner, mes.registerConnections, mes.signalCloseAcceptRoutine)

	//fmt.Println("hello")
	return nil
}

func (mes *multiEchoServer) Close() {
	// send close signals to close master and "acceptConnection" go routines.
	mes.signalCloseMasterRoutine <- -1
	// close the listner to interrupt the Accept function in "acceptConnections" go routine
	mes.listener.Close()
	mes.signalCloseAcceptRoutine <- -1
}

func (mes *multiEchoServer) Count() int {
	// TODO: implement this!
	request := &requestMessage{responseChan: make(chan int)}
	mes.signalRequestConnectionCount <- request
	return <-request.responseChan
}

// TODO: add additional methods/functions below!

/*
A go routine which accepts incoming connections and send the new incoming connection to master go routine for further handling.
*/
func acceptConnections(listner net.Listener, incomingConnections chan *richConn, closeSignal chan int) {
	for {
		select {
		case <-closeSignal:
			return
		default:
			conn, err := listner.Accept()
			if err != nil {
				fmt.Println("Cannot accept: ", err.Error())
				continue
			}

			incomingConnections <- &richConn{
				connection:           conn,
				outgoingMessageQueue: make(chan []byte, OUTGOING_MESSAGE_QUEUE_SIZE),
				closeSignal:          make(chan int, 1),
			}
		}
	}
}

/*
A go routine which reads line oriented data from client and send the message to braodcast message queue, the master routine
will receive message from braodcast message queue and send the message to all active connections' outgoing message queue.
The slow-reading of the client won't cause this go routine to be blocked because it sends data to the outgoing message queue (channel)
rather than writing to the socket directly, which means even though the client doesn't call read for an extended period of time, the server
(this go routine) will still be reponsive (not blocked).
*/
func handleConnection(wrappedConn *richConn, broadcastMessageQueue chan []byte, unregisterConnections chan *richConn) {
	bufReader := bufio.NewReader(wrappedConn.connection)
	for {
		line, err := bufReader.ReadBytes('\n')
		if err != nil {
			select {
			// if receiveing a close signal from master, then just close the outgoing message queue (because in this case the master wants to close all workers,
			// rather than the client closes the connectoin or network error), and since the worker is already unregistered at master, we don't need to send a
			// unregister message to master in this case
			case <-wrappedConn.closeSignal:
				close(wrappedConn.outgoingMessageQueue)
			// in case the client closes the connection or there is network error
			default:
				// close the current connection's outgoing message queue (channel), so the corresponding "writing to connectoin" go routine will also be stopped.
				close(wrappedConn.outgoingMessageQueue)
				// close the connection
				wrappedConn.connection.Close()

				unregisterConnections <- wrappedConn
			}
			return
		}
		broadcastMessageQueue <- line
	}
}

/*
A go routine which writes the messages of the connection's outgoing message queue to the Socket.
This go routine may be blocked if the buffer of TCP connection is full.
*/
func writeOutgoingQueueToSocket(wrappedConn *richConn) {
	for message := range wrappedConn.outgoingMessageQueue {
		wrappedConn.connection.Write(message)
	}
}

/*
A master go routine which has two major functionality:
1. receive incoming connections from "accept connection" go routine, start another two go routines to handle the incoming connection
2. recieve broadcast message request from connections, and broadcast the message to all active connections
It will also update the active connections table when new connection comes
*/
func masterRoutine(registerConnections chan *richConn, unregisteredConnections chan *richConn, activeConnections map[*richConn]bool, broadcastMessageQueue chan []byte, closeSignal chan int, requestCountChan chan *requestMessage) {
	for {
		select {
		// if there is an incoming connection to be handled.
		case wrappedConn := <-registerConnections:
			activeConnections[wrappedConn] = true
			go handleConnection(wrappedConn, broadcastMessageQueue, unregisteredConnections)
			go writeOutgoingQueueToSocket(wrappedConn)
		// if there is a message to be broadcasted to all active connections.
		case broadcaseMessage := <-broadcastMessageQueue:
			for wrappedConn := range activeConnections {
				select {
				case wrappedConn.outgoingMessageQueue <- broadcaseMessage:
				// if the outgoing message queue of the connection is full, just drop the message rather than being blocked.
				default:
				}
			}
		case unregisterConn := <-unregisteredConnections:
			delete(activeConnections, unregisterConn)
		case <-closeSignal:
			// send signals to close all client connections and their corresponding handle go routines.
			for wrappedConn := range activeConnections {
				// close the connection to interrupt the Readbytes() function in "handleConnection"
				wrappedConn.connection.Close()
				wrappedConn.closeSignal <- -1
				delete(activeConnections, wrappedConn)
			}
			return
		case request := <-requestCountChan:
			request.responseChan <- len(activeConnections)
		}
	}
}

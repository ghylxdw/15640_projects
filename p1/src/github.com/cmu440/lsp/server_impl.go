// Contains the implementation of a LSP server.
// @author: Chun Chen

package lsp

import (
	"container/list"
	"errors"
	"github.com/cmu440/lspnet"
	"strconv"
)

type server struct {
	params            *Params
	requestc          chan *request
	closeSignal       chan struct{}
	networkUtility    *networkUtility
	readBuffer        map[int]*buffer
	writeBuffer       map[int]*buffer
	unAckedMsgBuffer  map[int]*buffer
	latestAckBuffer   map[int]*buffer
	deferedRead       *list.List
	deferedClose      *list.List
	hostportConnIdMap map[string]int
	connIdHostportMap map[int]*lspnet.UDPAddr
	activeConn        map[int]bool
	expectedSeqNum    map[int]int
	seqNum            map[int]int
	latestActiveEpoch map[int]int
	currEpoch         int
	connId            int
	serverRunning     bool
	connLostInClosing bool
}

// struct which bundles connId and payLoad, this is used for passing the parameters of Write() function
// to event handler, and get the return values of Read() fuunction from the event handler
type connIdPayloadBundle struct {
	connId  int
	payload []byte
}

// NewServer creates, initiates, and returns a new server. This function should
// NOT block. Instead, it should spawn one or more goroutines (to handle things
// like accepting incoming client connections, triggering epoch events at
// fixed intervals, synchronizing events using a for-select loop like you saw in
// project 0, etc.) and immediately return. It should return a non-nil error if
// there was an error resolving or listening on the specified port number.
func NewServer(port int, params *Params) (Server, error) {
	s := &server{
		params:            params,
		requestc:          make(chan *request, 100),
		closeSignal:       make(chan struct{}),
		readBuffer:        make(map[int]*buffer),
		writeBuffer:       make(map[int]*buffer),
		unAckedMsgBuffer:  make(map[int]*buffer),
		latestAckBuffer:   make(map[int]*buffer),
		deferedRead:       list.New(),
		deferedClose:      list.New(),
		hostportConnIdMap: make(map[string]int),
		connIdHostportMap: make(map[int]*lspnet.UDPAddr),
		activeConn:        make(map[int]bool),
		expectedSeqNum:    make(map[int]int),
		seqNum:            make(map[int]int),
		latestActiveEpoch: make(map[int]int),
		currEpoch:         0,
		connId:            0,
		serverRunning:     true,
		connLostInClosing: false,
	}
	s.networkUtility = NewNetworkUtility(s.requestc)

	err := s.networkUtility.listen(":" + strconv.Itoa(port))
	if err != nil {
		return nil, err
	}

	go s.networkUtility.networkHandler()
	go s.eventHandler()
	go epochTimer(s.requestc, s.closeSignal, params.EpochMillis)

	return s, nil
}

func (s *server) Read() (int, []byte, error) {
	retVal, err := s.doRequest(doread, nil)
	bundle := retVal.(*connIdPayloadBundle)
	return bundle.connId, bundle.payload, err
}

func (s *server) Write(connID int, payload []byte) error {
	bundle := &connIdPayloadBundle{connID, payload}
	_, err := s.doRequest(dowrite, bundle)
	return err
}

func (s *server) CloseConn(connID int) error {
	_, err := s.doRequest(docloseconn, connID)
	return err
}

func (s *server) Close() error {
	_, err := s.doRequest(doclose, nil)
	return err
}

// submit the request to reqeust channel (requestc). waiting for the event handler to handle
func (s *server) doRequest(op int, val interface{}) (interface{}, error) {
	req := &request{op, val, make(chan *retType)}
	s.requestc <- req
	retVal := <-req.replyc
	return retVal.val, retVal.err
}

// do corresponding actions when a new message comes from network handler
func (s *server) handleReceivedMsg(req *request) {
	clientAddr := req.val.(*receivedPacket).raddr
	receivedMsg := req.val.(*receivedPacket).msg
	switch receivedMsg.Type {
	case MsgConnect:
		// if the hostport was never seen, or the connection is lost/closed, and the server is not closed,
		// establish a connection and initiate related resources
		if connId := s.hostportConnIdMap[clientAddr.String()]; s.deferedClose.Len() == 0 && connId == 0 {
			s.connId += 1
			s.hostportConnIdMap[clientAddr.String()] = s.connId
			s.connIdHostportMap[s.connId] = clientAddr
			ackMsg := NewAck(s.connId, 0)
			s.networkUtility.sendMessageToAddr(clientAddr, ackMsg)
			s.readBuffer[s.connId] = NewBuffer()
			s.writeBuffer[s.connId] = NewBuffer()
			s.unAckedMsgBuffer[s.connId] = NewBuffer()
			s.latestAckBuffer[s.connId] = NewBuffer()
			s.expectedSeqNum[s.connId] = 1
			s.seqNum[s.connId] = 0
			s.latestActiveEpoch[s.connId] = s.currEpoch
			s.activeConn[s.connId] = true
		}
	case MsgAck:
		// if the connection with this client is established before, and not lost
		if clientConnId := s.hostportConnIdMap[clientAddr.String()]; clientConnId > 0 {
			// set the latest active epoch of the specified connection
			s.latestActiveEpoch[clientConnId] = s.currEpoch

			unAckedMsgBuffer := s.unAckedMsgBuffer[clientConnId]
			msgExist := unAckedMsgBuffer.Delete(receivedMsg.SeqNum)

			// move messages from write buffer to unAckedMsg buffer and send them out via network
			// if their seq nums are in the sliding window
			writeBuffer := s.writeBuffer[clientConnId]
			if msgExist && writeBuffer.Len() > 0 {
				if unAckedMsgBuffer.Len() == 0 {
					sentMsg := writeBuffer.Remove()
					s.networkUtility.sendMessageToAddr(clientAddr, sentMsg)
					unAckedMsgBuffer.Insert(sentMsg)
				}

				wLeft := unAckedMsgBuffer.Front().SeqNum
				wSize := s.params.WindowSize
				for writeBuffer.Len() != 0 {
					if writeBuffer.Front().SeqNum-wSize >= wLeft {
						break
					} else {
						sentMsg := writeBuffer.Remove()
						unAckedMsgBuffer.Insert(sentMsg)
						s.networkUtility.sendMessageToAddr(clientAddr, sentMsg)
					}
				}
			}

			// if the connection is closed/the server is closed and all pending messages are sent and acked,
			// clean up all resources that relates to the connection
			if msgExist && unAckedMsgBuffer.Len() == 0 && (!s.activeConn[clientConnId] || s.deferedClose.Len() > 0) {
				delete(s.hostportConnIdMap, clientAddr.String())
				delete(s.connIdHostportMap, clientConnId)
				delete(s.readBuffer, clientConnId)
				delete(s.writeBuffer, clientConnId)
				delete(s.unAckedMsgBuffer, clientConnId)
				delete(s.latestAckBuffer, clientConnId)
				delete(s.latestActiveEpoch, clientConnId)
				delete(s.expectedSeqNum, clientConnId)
				delete(s.seqNum, clientConnId)
			}
		}
	case MsgData:
		// don't receive data message if the connection is closed/lost or the server is closed
		if clientConnId := s.hostportConnIdMap[clientAddr.String()]; clientConnId > 0 && s.activeConn[clientConnId] && s.deferedClose.Len() == 0 {
			// set the latest active epoch of the specified connection
			s.latestActiveEpoch[clientConnId] = s.currEpoch

			readBuffer := s.readBuffer[clientConnId]

			// ignore data messages whose seq num is smaller than the expected seq num
			if receivedMsg.SeqNum >= s.expectedSeqNum[clientConnId] {
				readBuffer.Insert(receivedMsg)
			}

			// send ack for the data message and put the ack into latestAck buffer
			latestAckBuffer := s.latestAckBuffer[clientConnId]
			ackMsg := NewAck(clientConnId, receivedMsg.SeqNum)
			s.networkUtility.sendMessageToAddr(clientAddr, ackMsg)

			latestAckBuffer.Insert(ackMsg)
			// adjust the buffer to conform sliding window size
			latestAckBuffer.AdjustUsingWindow(s.params.WindowSize)

			// wake a deferred read reqeusts up if the seq num of incoming message equals the expected seq num of the connection
			if receivedMsg.SeqNum == s.expectedSeqNum[clientConnId] && s.deferedRead.Len() > 0 {
				readReq := s.deferedRead.Front().Value.(*request)
				s.deferedRead.Remove(s.deferedRead.Front())
				s.handleRead(readReq)
			}
		}
	}
}

// do corresponding actions when epoch fires
func (s *server) handleEpoch() {
	s.currEpoch += 1

	// detect connection lost
	for connId, latestActiveEpoch := range s.latestActiveEpoch {
		if s.currEpoch-latestActiveEpoch >= s.params.EpochLimit {
			if s.deferedClose.Len() > 0 {
				s.connLostInClosing = true
			}
			delete(s.activeConn, connId)
			// wake one deferred read up if no message is ready in read buffer of the connection
			// if condition is matched, then clean up all realted resources of the connection including the read buffer and expected seq num
			// otherwise clean up all related resources except read buffer and expected seq num since we allow further Read on a lost connection
			if (s.readBuffer[connId].Len() == 0 || s.readBuffer[connId].Front().SeqNum != s.expectedSeqNum[connId]) &&
				s.deferedRead.Len() > 0 {
				// since the connection has nothing to read and unblocks a deferred read, clean up its readbuffer and expectedSeqNum
				delete(s.readBuffer, connId)
				delete(s.expectedSeqNum, connId)
				readReq := s.deferedRead.Front().Value.(*request)
				s.deferedRead.Remove(s.deferedRead.Front())
				retVal := &retType{&connIdPayloadBundle{connId, nil}, errors.New("some connection gets lost")}
				readReq.replyc <- retVal
			}
			// clean up all remaining related resources of this connection
			clientAddr := s.connIdHostportMap[connId]
			delete(s.hostportConnIdMap, clientAddr.String())
			delete(s.connIdHostportMap, connId)
			delete(s.writeBuffer, connId)
			delete(s.unAckedMsgBuffer, connId)
			delete(s.latestAckBuffer, connId)
			delete(s.latestActiveEpoch, connId)
			delete(s.seqNum, connId)
		}
	}

	// don't resend latest ack if the server is closed
	if s.deferedClose.Len() == 0 {
		for connId, latestAckBuffer := range s.latestAckBuffer {
			// if the connection is not closed
			if s.activeConn[connId] {
				// if the connection hasn't received any data message after connection is established, send a ack with seq num 0
				if latestAckBuffer.Len() == 0 {
					ackMsg := NewAck(connId, 0)
					clientAddr := s.connIdHostportMap[connId]
					s.networkUtility.sendMessageToAddr(clientAddr, ackMsg)
				} else {
					clientAddr := s.connIdHostportMap[connId]
					latestAckMsg := latestAckBuffer.ReturnAll()
					for _, msg := range latestAckMsg {
						s.networkUtility.sendMessageToAddr(clientAddr, msg)
					}
				}
			}
		}
	}

	// resend sent but unacked data messages, even though the connection is closed or the server is closed
	nonEmptyBuffer := 0
	for connId, unAckedMsgBuffer := range s.unAckedMsgBuffer {
		if unAckedMsgBuffer.Len() > 0 || s.writeBuffer[connId].Len() > 0 {
			nonEmptyBuffer += 1
		}

		clientAddr := s.connIdHostportMap[connId]
		unAckMsg := unAckedMsgBuffer.ReturnAll()
		for _, msg := range unAckMsg {
			s.networkUtility.sendMessageToAddr(clientAddr, msg)
		}
	}

	// if there is deferred Close() request and all pending messages are sent and acked, wake the pending Close request up
	if s.deferedClose.Len() > 0 && nonEmptyBuffer == 0 {
		for e := s.deferedClose.Front(); e != nil; e = e.Next() {
			closeReq := e.Value.(*request)
			var retVal *retType
			if s.connLostInClosing {
				retVal = &retType{nil, errors.New("some connections lost in the cleaning up process")}
			} else {
				retVal = &retType{nil, nil}
			}
			closeReq.replyc <- retVal
		}
		// stop the main handler routine
		s.serverRunning = false
		s.shutDown()
	}
}

// handle user read request
func (s *server) handleRead(req *request) {
	// return an error if the server is closed
	if s.deferedClose.Len() > 0 {
		req.replyc <- &retType{nil, errors.New("server is closed")}
		return
	}

	// check if there is any data message ready for read in any read buffer
	for connId, readBuffer := range s.readBuffer {
		if readBuffer.Len() > 0 && readBuffer.Front().SeqNum == s.expectedSeqNum[connId] {
			readMsg := readBuffer.Remove()
			s.expectedSeqNum[connId] += 1
			retVal := &retType{&connIdPayloadBundle{connId, readMsg.Payload}, nil}
			req.replyc <- retVal
			return
		}
	}

	// if nothing can be read from any read buffer, check if there is any lost connection in them
	for connId, _ := range s.readBuffer {
		// if the connection is lost and has no data message for reading
		if !s.activeConn[connId] {
			retVal := &retType{&connIdPayloadBundle{connId, nil}, errors.New("some connection is lost")}
			req.replyc <- retVal
			// clean up the realted resource of the lost connection
			delete(s.readBuffer, connId)
			delete(s.expectedSeqNum, connId)
			return
		}
	}

	// if nothing can be read from any read buffer and no lost connection with no data ready for reading, defer the Read
	s.deferedRead.PushBack(req)
}

// handle user write request
func (s *server) handleWrite(req *request) {
	connId := req.val.(*connIdPayloadBundle).connId
	payload := req.val.(*connIdPayloadBundle).payload
	// return an error if the connection is closed/lost, or the server is closed
	if s.deferedClose.Len() > 0 || !s.activeConn[connId] {
		req.replyc <- &retType{nil, errors.New("server/connection closed or connection lost")}
	} else {
		// if the conn id exists
		if clientAddr := s.connIdHostportMap[connId]; clientAddr != nil {
			s.seqNum[connId] += 1
			seqNum := s.seqNum[connId]
			windowSize := s.params.WindowSize
			sentMsg := NewData(connId, seqNum, payload)
			unAckedMsgBuffer := s.unAckedMsgBuffer[connId]
			writeBuffer := s.writeBuffer[connId]
			if writeBuffer.Len() == 0 &&
				(unAckedMsgBuffer.Len() == 0 || seqNum-windowSize < unAckedMsgBuffer.Front().SeqNum) {
				unAckedMsgBuffer.Insert(sentMsg)
				s.networkUtility.sendMessageToAddr(clientAddr, sentMsg)
			} else {
				writeBuffer.Insert(sentMsg)
			}
			req.replyc <- &retType{nil, nil}
		} else {
			req.replyc <- &retType{nil, errors.New("connection doesn't exist")}
		}
	}
}

// handle user close a specified connection
func (s *server) handleCloseConn(req *request) {
	connId := req.val.(int)
	// if the connection is not closed or lost
	if s.activeConn[connId] {
		// unblock one defered read
		if s.deferedRead.Len() > 0 {
			readReq := s.deferedClose.Front().Value.(*request)
			s.deferedRead.Remove(s.deferedRead.Front())
			retVal := &retType{&connIdPayloadBundle{connId, nil}, errors.New("some connection gets closed explicitly")}
			readReq.replyc <- retVal
		}

		clientAddr := s.connIdHostportMap[connId]
		// if there is no pending messages to be resent and acked, clean up resources that are used for resending unAcked messages
		if s.unAckedMsgBuffer[connId].Len() == 0 && s.writeBuffer[connId].Len() == 0 {
			delete(s.writeBuffer, connId)
			delete(s.unAckedMsgBuffer, connId)
			delete(s.hostportConnIdMap, clientAddr.String())
			delete(s.connIdHostportMap, connId)
		}

		delete(s.readBuffer, connId)
		delete(s.latestAckBuffer, connId)
		delete(s.latestActiveEpoch, connId)
		delete(s.expectedSeqNum, connId)
		delete(s.seqNum, connId)

		delete(s.activeConn, connId)
		req.replyc <- &retType{nil, nil}
	} else {
		req.replyc <- &retType{nil, errors.New("connection ID doesn't exist")}
	}
}

// handle user close the server
func (s *server) handleClose(req *request) {
	// unblcok all defered reads
	for s.deferedRead.Len() != 0 {
		readReq := s.deferedClose.Front().Value.(*request)
		s.deferedRead.Remove(s.deferedRead.Front())
		retVal := &retType{&connIdPayloadBundle{0, nil}, errors.New("the server is closed")}
		readReq.replyc <- retVal
	}

	nonEmptyBuffer := 0
	for connId, unAckedMsgBuffer := range s.unAckedMsgBuffer {
		if unAckedMsgBuffer.Len() > 0 || s.writeBuffer[connId].Len() > 0 {
			nonEmptyBuffer += 1
			break
		}
	}
	// if there is no sent but unacked messages, shut down the whole server immediately
	if nonEmptyBuffer == 0 {
		s.shutDown()
		req.replyc <- &retType{nil, nil}
		s.serverRunning = false
	} else {
		// otherwise defer the Close request
		s.deferedClose.PushBack(req)
	}
}

// shut down the network handler and epoch timer go routines
func (s *server) shutDown() {
	s.networkUtility.close()
	s.closeSignal <- struct{}{}
}

// go routine which handles multiple requests (notification) from reqeust channel (requestc),
// including reqeusts from user and notifications from epoch timer and network handler
func (s *server) eventHandler() {
	for s.serverRunning {
		req := <-s.requestc
		switch req.op {
		case receivemsg:
			s.handleReceivedMsg(req)
		case epochtimer:
			s.handleEpoch()
		case doread:
			s.handleRead(req)
		case dowrite:
			s.handleWrite(req)
		case docloseconn:
			s.handleCloseConn(req)
		case doclose:
			s.handleClose(req)
		}
	}
}

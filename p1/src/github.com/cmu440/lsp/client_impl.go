// Contains the implementation of a LSP client.
// @author: Chun Chen

package lsp

import (
	"container/list"
	"errors"
)

type client struct {
	params                *Params
	networkUtility        *networkUtility
	requestc              chan *request
	closeSignal           chan struct{}
	connLostSignal        chan struct{}
	connEstablishedSignal chan struct{}
	writeBuffer           *buffer
	unAckedMsgBuffer      *buffer
	latestAckBuffer       *buffer
	readBuffer            *buffer
	deferedRead           *list.List
	deferedClose          *list.List
	connId                int
	seqNum                int
	expectedSeqNum        int
	currEpoch             int
	latestActiveEpoch     int
	clientRunning         bool
	connLost              bool
}

// NewClient creates, initiates, and returns a new client. This function
// should return after a connection with the server has been established
// (i.e., the client has received an Ack message from the server in response
// to its connection request), and should return a non-nil error if a
// connection could not be made (i.e., if after K epochs, the client still
// hasn't received an Ack message from the server in response to its K
// connection requests).
//
// hostport is a colon-separated string identifying the server's host address
// and port number (i.e., "localhost:9999").
func NewClient(hostport string, params *Params) (Client, error) {
	c := &client{
		params:                params,
		requestc:              make(chan *request, 100),
		closeSignal:           make(chan struct{}),
		connLostSignal:        make(chan struct{}, 1),
		connEstablishedSignal: make(chan struct{}, 1),
		writeBuffer:           NewBuffer(),
		unAckedMsgBuffer:      NewBuffer(),
		latestAckBuffer:       NewBuffer(),
		readBuffer:            NewBuffer(),
		deferedRead:           list.New(),
		deferedClose:          list.New(),
		connId:                0,
		seqNum:                0,
		expectedSeqNum:        1,
		currEpoch:             0,
		latestActiveEpoch:     0,
		clientRunning:         true,
		connLost:              false,
	}
	c.networkUtility = NewNetworkUtility(c.requestc)
	err := c.networkUtility.dial(hostport)
	if err != nil {
		return nil, err
	}
	go c.eventHandler()
	go c.networkUtility.networkHandler()
	go epochTimer(c.requestc, c.closeSignal, c.params.EpochMillis)

	err = c.connect()
	if err != nil {
		return nil, err
	}

	// waiting for signal whether the connection is established or lost
	select {
	case <-c.connEstablishedSignal:
		return c, nil
	case <-c.connLostSignal:
		return nil, errors.New("connection lost before established")
	}

}

func (c *client) ConnID() int {
	ret, _ := c.doRequest(doconnid, nil)
	return ret.(int)
}

func (c *client) Read() ([]byte, error) {
	ret, err := c.doRequest(doread, nil)
	if err != nil {
		return nil, err
	} else {
		return ret.([]byte), nil
	}
}

func (c *client) Write(payload []byte) error {
	_, err := c.doRequest(dowrite, payload)
	return err
}

func (c *client) Close() error {
	_, err := c.doRequest(doclose, nil)
	return err
}

func (c *client) connect() error {
	_, err := c.doRequest(doconnect, nil)
	return err
}

// submit the request to reqeust channel (requestc). waiting for the event handler to handle
func (c *client) doRequest(op int, val interface{}) (interface{}, error) {
	req := &request{op, val, make(chan *retType)}
	c.requestc <- req
	retV := <-req.replyc
	return retV.val, retV.err
}

// handle user read request
func (c *client) handleRead(req *request) {
	// if client is closed, return an error
	if c.deferedClose.Len() > 0 {
		req.replyc <- &retType{nil, errors.New("client closed")}
		return
	}
	// if connection is lost and there is nothing to read in the read buffer, return an error
	if c.readBuffer.Len() == 0 || c.readBuffer.Front().SeqNum != c.expectedSeqNum {
		req.replyc <- &retType{nil, errors.New("connection lost")}
		return
	}

	// return the expected message in the read buffer
	readMsg := c.readBuffer.Front()
	c.readBuffer.Remove()
	c.expectedSeqNum += 1
	req.replyc <- &retType{readMsg.Payload, nil}
}

// handle user write request
func (c *client) handleWrite(req *request) {
	// if connection is lost, return an error
	if c.connLost {
		req.replyc <- &retType{nil, errors.New("connection lost")}
		return
	}
	// if client is closed, return an error
	if c.deferedClose.Len() > 0 {
		req.replyc <- &retType{nil, errors.New("client closed")}
		return
	}

	// if there is space in unAckedMsgBuffer, insert the message into the buffer and send it out via network
	// otherwise insert it into write buffer
	payload := req.val.([]byte)
	c.seqNum += 1
	sentMsg := NewData(c.connId, c.seqNum, payload)
	windowSize := c.params.WindowSize
	if c.writeBuffer.Len() == 0 &&
		(c.unAckedMsgBuffer.Len() == 0 || sentMsg.SeqNum-windowSize < c.unAckedMsgBuffer.Front().SeqNum) {
		c.unAckedMsgBuffer.Insert(sentMsg)
		c.networkUtility.sendMessage(sentMsg)
	} else {
		c.writeBuffer.Insert(sentMsg)
	}
	req.replyc <- &retType{nil, nil}
}

// do corresponding actions when epoch fires
func (c *client) handleEpoch() {
	c.currEpoch += 1
	// detect if connection is lost
	if c.currEpoch-c.latestActiveEpoch >= c.params.EpochLimit {
		c.connLost = true
		c.connLostSignal <- struct{}{}
		c.shutDown()
		return
	}
	// if the connection is established but no data messages have been received, send a ack message with seq number 0
	if c.connId > 0 && c.latestAckBuffer.Len() == 0 && c.deferedClose.Len() == 0 {
		ackMsg := NewAck(c.connId, 0)
		c.networkUtility.sendMessage(ackMsg)
	}

	// resend unacknowledged messages in the buffer
	unAckMsg := c.unAckedMsgBuffer.ReturnAll()
	for _, msg := range unAckMsg {
		c.networkUtility.sendMessage(msg)
	}

	// resend latest sent acknowledgements
	if c.deferedClose.Len() == 0 {
		latestAck := c.latestAckBuffer.ReturnAll()
		for _, msg := range latestAck {
			c.networkUtility.sendMessage(msg)
		}
	}
}

// do corresponding actions when a new message comes from network handler
func (c *client) handleReceivedMsg(req *request) {
	// update the latest active epoch interval
	c.latestActiveEpoch = c.currEpoch
	receivedMsg := req.val.(*receivedPacket).msg
	switch receivedMsg.Type {
	case MsgAck:
		// remove the message that has received acknowledgement from the buffer
		msgExist := c.unAckedMsgBuffer.Delete(receivedMsg.SeqNum)

		// if some messages in the buffer receive ack, check whether messages in the write buffer
		// can be moved into the unAckedMsgbuffer according to the sliding window size
		if msgExist && c.writeBuffer.Len() > 0 {
			if c.unAckedMsgBuffer.Len() == 0 {
				sentMsg := c.writeBuffer.Remove()
				c.unAckedMsgBuffer.Insert(sentMsg)
				c.networkUtility.sendMessage(sentMsg)
			}
			wLeft := c.unAckedMsgBuffer.Front().SeqNum
			wSize := c.params.WindowSize
			for c.writeBuffer.Len() != 0 {
				if c.writeBuffer.Front().SeqNum-wSize >= wLeft {
					break
				} else {
					sentMsg := c.writeBuffer.Remove()
					c.unAckedMsgBuffer.Insert(sentMsg)
					c.networkUtility.sendMessage(sentMsg)
				}
			}
		}
		// check if the ack message is an ack for the connection message
		if msgExist && receivedMsg.SeqNum == 0 {
			c.connId = receivedMsg.ConnID
			c.connEstablishedSignal <- struct{}{}
		}
	case MsgData:
		// if connection is not established or the client is closed
		if c.connId == 0 || c.deferedClose.Len() > 0 {
			return
		}

		// ignore messages whose seq num is smaller than expected seq num
		// epoch handler will resend the acks that haven't been received on the other side (if their seq num is smaller than expected seq num)
		// and the same size of sliding window on both sending and receiving side guarantee the correctness, otherwise we may have to send ack
		// for every data message we receive no matter whether its seq num is larger or smaller than the expected seq num
		if receivedMsg.SeqNum >= c.expectedSeqNum {
			c.readBuffer.Insert(receivedMsg)
			// send ack for the data message and store the ack to latest sent ack buffer
			ackMsg := NewAck(c.connId, receivedMsg.SeqNum)
			// send ack message out
			c.networkUtility.sendMessage(ackMsg)

			c.latestAckBuffer.Insert(ackMsg)
			c.latestAckBuffer.AdjustUsingWindow(c.params.WindowSize)
		}
	}
}

// handle user get conn id request
func (c *client) handleConnId(req *request) {
	req.replyc <- &retType{c.connId, nil}
}

// handle user close client request
func (c *client) handleClose(req *request) {
	// if connection doesn't get lost when user tries to close the client, shut down the epoch timer and network handler go routines
	if !c.connLost {
		c.shutDown()
	}
	// if there is any pending message that is not sent or acked, this implies the connection get lost when user tries to close the client
	// then return an error
	if c.writeBuffer.Len() > 0 || c.unAckedMsgBuffer.Len() > 0 {
		req.replyc <- &retType{nil, errors.New("connection lost before sending out all pending messages")}
	} else {
		req.replyc <- &retType{nil, nil}
	}

	// unblock all close function
	for e := c.deferedClose.Front(); e != nil; e = e.Next() {
		req = e.Value.(*request)
		if c.writeBuffer.Len() > 0 || c.unAckedMsgBuffer.Len() > 0 {
			req.replyc <- &retType{nil, errors.New("connection lost before sending out all pending messages")}
		} else {
			req.replyc <- &retType{nil, nil}
		}
	}
	// set the running flag as false so the event handler will return from infinite loop
	c.clientRunning = false
}

// handle connect request when creating new client, will send a connect message to server
func (c *client) handleConnect(req *request) {
	msg := NewConnect()
	c.unAckedMsgBuffer.Insert(msg)
	c.networkUtility.sendMessage(msg)
	req.replyc <- &retType{nil, nil}
}

// shut down the network handler and epoch timer go routines
func (c *client) shutDown() {
	c.networkUtility.close()
	c.closeSignal <- struct{}{}
}

// go routine which handles multiple requests (notification) from reqeust channel (requestc),
// including reqeusts from user and notifications from epoch timer and network handler
func (c *client) eventHandler() {
	var req *request
	for c.clientRunning {
		// if the connection lost when trying establishing the connection
		if c.connId == 0 && c.connLost {
			return
		}
		// unblock deferred read if the read buffer is ready or the connection is closed or lost
		if c.deferedRead.Len() > 0 && ((c.readBuffer.Len() > 0 && c.readBuffer.Front().SeqNum == c.expectedSeqNum) || c.connLost || c.deferedClose.Len() > 0) {
			req = c.deferedRead.Front().Value.(*request)
			c.deferedRead.Remove(c.deferedRead.Front())
		} else if c.deferedClose.Len() > 0 && ((c.writeBuffer.Len() == 0 && c.unAckedMsgBuffer.Len() == 0) || c.connLost) {
			// unblock deferred close if all pending messages are sent and acked or connection is lost
			req = c.deferedClose.Front().Value.(*request)
			c.deferedClose.Remove(c.deferedClose.Front())
		} else {
			// if there is no need to unblock any deferred request, get new request from reqeust channel
			req = <-c.requestc
			// defer read or close request if necessary
			if req.op == doread && (c.readBuffer.Len() == 0 || c.readBuffer.Front().SeqNum != c.expectedSeqNum) {
				c.deferedRead.PushBack(req)
				continue
			}
			if req.op == doclose && (c.writeBuffer.Len() > 0 || c.unAckedMsgBuffer.Len() > 0) {
				c.deferedClose.PushBack(req)
				continue
			}
		}

		// handle request according to its type
		switch req.op {
		case doread:
			c.handleRead(req)
		case dowrite:
			c.handleWrite(req)
		case doclose:
			c.handleClose(req)
		case doconnid:
			c.handleConnId(req)
		case doconnect:
			c.handleConnect(req)
		case epochtimer:
			c.handleEpoch()
		case receivemsg:
			c.handleReceivedMsg(req)
		}
	}
}

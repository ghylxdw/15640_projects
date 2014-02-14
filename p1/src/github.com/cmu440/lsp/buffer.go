// Constains implementation of a sequential buffer, which is designed specially for Message struct as its element
// the Message elements in the buffer will be kept in increasing order acoording to their seq num
// @author: Chun Chen

package lsp

import (
	"container/list"
)

type buffer struct {
	l *list.List
}

func NewBuffer() *buffer {
	return &buffer{list.New()}
}

// insert a message into the buffer, the function will guarantee the order of messages in the buffer
func (buf *buffer) Insert(val *Message) {
	l := buf.l
	if l.Len() == 0 || val.SeqNum > l.Back().Value.(*Message).SeqNum {
		l.PushBack(val)
	} else {
		insertSeqNumber := val.SeqNum
		var insertBeforeEle *list.Element
		insertBeforeEle = nil
		for e := l.Front(); e != nil; e = e.Next() {
			eleSeqNum := e.Value.(*Message).SeqNum
			if insertSeqNumber <= eleSeqNum {
				insertBeforeEle = e
				break
			}
		}

		if insertSeqNumber < insertBeforeEle.Value.(*Message).SeqNum {
			l.InsertBefore(val, insertBeforeEle)
		}
	}
}

// remove the first message in the buffer
func (buf *buffer) Remove() *Message {
	l := buf.l
	retVal := l.Front().Value
	l.Remove(l.Front())
	return retVal.(*Message)
}

// return the first message in the buffer
func (buf *buffer) Front() *Message {
	l := buf.l
	return l.Front().Value.(*Message)
}

// delete message with specified seq num in the buffer
// return true if message with specified seq num exists in the buffer
func (buf *buffer) Delete(delSeqNum int) bool {
	l := buf.l
	for e := l.Front(); e != nil; e = e.Next() {
		if e.Value.(*Message).SeqNum == delSeqNum {
			l.Remove(e)
			return true
		}
	}
	return false
}

// adjust the buffer with specified window size
// this function will remove all messages whose seq num is smaller than $seq num of last message in buffer$ - $window size$
// this function is useful for latestAckBuffer
func (buf *buffer) AdjustUsingWindow(windowSize int) {
	l := buf.l
	wRightBoundary := l.Back().Value.(*Message).SeqNum
	wLeftBoundary := wRightBoundary - windowSize + 1
	e := l.Front()
	for e != nil {
		nextE := e.Next()
		if e.Value.(*Message).SeqNum < wLeftBoundary {
			l.Remove(e)
		} else {
			break
		}
		e = nextE
	}
}

// return all messages in the buffer to a slice
func (buf *buffer) ReturnAll() []*Message {
	l := buf.l
	retVal := make([]*Message, l.Len())
	pos := 0
	for e := l.Front(); e != nil; e = e.Next() {
		retVal[pos] = e.Value.(*Message)
		pos += 1
	}
	return retVal
}

// return number of messages in the buffer
func (buf *buffer) Len() int {
	return buf.l.Len()
}

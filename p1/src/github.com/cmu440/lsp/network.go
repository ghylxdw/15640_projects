// Contents contain network handler routine, which is reponsible for receiving data from network,
// unmarshall it and send message to event handler via channel. Also contains some wrapped network
// utilities on top of UDP protocal.
// @author: Chun Chen

package lsp

import (
	"encoding/json"
	"github.com/cmu440/lspnet"
)

type receivedPacket struct {
	msg   *Message
	raddr *lspnet.UDPAddr
}

type networkUtility struct {
	conn        *lspnet.UDPConn
	requestc    chan *request
	closeSignal chan struct{}
}

// create a new network utility
func NewNetworkUtility(requestc chan *request) *networkUtility {
	handler := &networkUtility{
		conn:        nil,
		requestc:    requestc,
		closeSignal: make(chan struct{}),
	}
	return handler
}

// dial the remote server using the hostport given, used on client side
func (h *networkUtility) dial(hostport string) error {
	serverAddr, err := lspnet.ResolveUDPAddr("udp", hostport)
	if err != nil {
		return err
	}
	conn, err := lspnet.DialUDP("udp", nil, serverAddr)
	if err != nil {
		return err
	}
	h.conn = conn
	return nil
}

// listen to incoming UDP data using the given hostport, used on server side
func (h *networkUtility) listen(hostport string) error {
	serverAddr, err := lspnet.ResolveUDPAddr("udp", hostport)
	if err != nil {
		return nil
	}
	conn, err := lspnet.ListenUDP("udp", serverAddr)
	if err != nil {
		return err
	}
	h.conn = conn
	return nil
}

// send message via UDP protocal, used on client side
func (h *networkUtility) sendMessage(msg *Message) error {
	buf, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = h.conn.Write(buf)
	if err != nil {
		return err
	}
	return nil
}

// send message via UDP protocal to the given remote address, used on server side
func (h *networkUtility) sendMessageToAddr(raddr *lspnet.UDPAddr, msg *Message) error {
	buf, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = h.conn.WriteToUDP(buf, raddr)
	if err != nil {
		return err
	}
	return nil
}

// network handler go routine which receives incoming UDP data, unmarshall it and send it to event handler via channel
func (h *networkUtility) networkHandler() {
	buf := make([]byte, 1500)
	for {
		select {
		case <-h.closeSignal:
			return
		default:
			n, addr, err := h.conn.ReadFromUDP(buf[0:])
			if err == nil {
				msg := &Message{}
				err = json.Unmarshal(buf[:n], msg)
				if err == nil {
					packet := &receivedPacket{msg, addr}
					req := &request{receivemsg, packet, make(chan *retType)}
					h.requestc <- req
				}
			}
		}
	}
}

// shut the network handler go routine down
func (h *networkUtility) close() {
	if h.conn != nil {
		h.conn.Close()
		h.closeSignal <- struct{}{}
	}
}

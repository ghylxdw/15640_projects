// Paxos library, to be included in an application.
// Multiple applications will run, each including
// a Paxos peer.
//
// Manages a sequence of agreed-on values.
// Store all status in memory.
// Support dynamic/fixed size Paxos.
// Support fail-recover.
//
// Author: Chun Chen, Bo Ma

package paxos

import (
	"encoding/gob"
	"errors"
	"github.com/cmu440/flight_reservation/rpc/paxosrpc"
	"net"
	"net/http"
	"net/rpc"
	"strings"
	"sync"
	"time"
)

const (
	MinorReplyRetryThreshold = 10
)

const (
	httpType dialType = iota + 1
	unixType
)

type dialType int

type timeSlotValue struct {
	value          interface{}
	isDecided      bool
	seqNumAccepted int64
	seqNumHighest  int64
}

type prepareReply struct {
	success bool
	reply   *paxosrpc.PrepareReply
}

type acceptReply struct {
	success bool
	reply   *paxosrpc.AcceptReply
}

type Paxos struct {
	mu               sync.Mutex
	l                net.Listener
	deaf             bool
	unreliable       bool
	rpcCount         int
	peers            []string
	me               int
	timeSlot         map[int]*timeSlotValue
	timeSLotLock     map[int]*sync.Mutex
	isLocked         bool
	maxDone          map[int]int
	maxDoneLock      sync.Mutex
	maxDecidedSlot   int
	minUndecidedSlot int
	testMode         bool
}

// helper function to make a RPC call to remote server.
// return false if error occurs during Dial or Call.
// the function uses Unix rpc call when testing.
func call(srv string, name string, args interface{}, reply interface{}) bool {
	var c *rpc.Client
	var err error

	c, err = rpc.DialHTTP("tcp", srv)

	if err != nil {
		return false
	}
	defer c.Close()

	err = c.Call(name, args, reply)
	if err == nil {
		return true
	}
	return false
}

func getMax(a int, b int) int {
	if a > b {
		return a
	} else {
		return b
	}
}

func getMin(a int, b int) int {
	if a < b {
		return a
	} else {
		return b
	}
}

func (px *Paxos) getTimeSlotValue(slotNum int) *timeSlotValue {

	var timeSlotVal, found = px.timeSlot[slotNum]
	if !found {
		px.timeSlot[slotNum] = &timeSlotValue{
			value:          nil,
			isDecided:      false,
			seqNumAccepted: 0,
			seqNumHighest:  0,
		}
		timeSlotVal = px.timeSlot[slotNum]
	}

	return timeSlotVal
}

// initialize the lock of a specified time slot if necessary, and return the pointer of the lock
func (px *Paxos) getTimeSlotLock(slotNum int) *sync.Mutex {
	px.mu.Lock()
	defer px.mu.Unlock()

	if px.timeSLotLock[slotNum] == nil {
		px.timeSLotLock[slotNum] = &sync.Mutex{}
	}
	return px.timeSLotLock[slotNum]
}

// the prepare phase of Paxos protocol
func (px *Paxos) Prepare(args *paxosrpc.PrepareArgs, reply *paxosrpc.PrepareReply) error {
	// fmt.Println("prepare")
	lock := px.getTimeSlotLock(args.SlotNum)
	lock.Lock()
	defer lock.Unlock()

	// if px.testMode {
	// 	px.cleanMemory(args)
	// }

	timeslot := px.getTimeSlotValue(args.SlotNum)
	if args.SeqNum > timeslot.seqNumHighest {
		timeslot.seqNumHighest = args.SeqNum
		// px.timeSlot[args.SeqNum] = timeslot
		reply.Status = paxosrpc.OK
		reply.SeqNumAccepted = timeslot.seqNumAccepted
		reply.ValueAccepted = timeslot.value
	} else {
		reply.Status = paxosrpc.Reject
		reply.SeqNumHighest = timeslot.seqNumHighest
	}
	return nil
}

// the Accept phase of Paxos protocol
func (px *Paxos) Accept(args *paxosrpc.AcceptArgs, reply *paxosrpc.AcceptReply) error {
	lock := px.getTimeSlotLock(args.SlotNum)
	lock.Lock()
	defer lock.Unlock()

	timeslot := px.getTimeSlotValue(args.SlotNum)

	if args.SeqNum >= timeslot.seqNumHighest {
		timeslot.seqNumAccepted = args.SeqNum
		timeslot.seqNumHighest = args.SeqNum
		timeslot.value = args.Value

		reply.Status = paxosrpc.OK

	} else {
		reply.Status = paxosrpc.Reject
		reply.SeqNumHighest = timeslot.seqNumHighest
	}

	return nil
}

// the Decide phase of Paxos protocol
func (px *Paxos) Decide(args *paxosrpc.DecideArgs, reply *paxosrpc.DecideReply) error {

	lock := px.getTimeSlotLock(args.SlotNum)
	lock.Lock()
	defer lock.Unlock()

	timeslot := px.getTimeSlotValue(args.SlotNum)
	timeslot.value = args.Value
	timeslot.isDecided = true

	px.mu.Lock()
	px.maxDecidedSlot = getMax(px.maxDecidedSlot, args.SlotNum)
	px.mu.Unlock()

	reply.Status = paxosrpc.OK

	return nil
}

// put the value, SeqNumAccepted and SeqNumHighest into the specified slot directly if the value in the slot is not decided
// PaxosAdmin use only
func (px *Paxos) Put(args *paxosrpc.PutArgs, reply *paxosrpc.PutReply) error {
	slotLock := px.getTimeSlotLock(args.SlotNum)
	slotLock.Lock()
	defer slotLock.Unlock()

	slotVal := px.getTimeSlotValue(args.SlotNum)
	// reject the put request if the value in the slot is already decided
	if slotVal.isDecided {
		reply.Status = paxosrpc.Reject
	} else {
		slotVal.isDecided = true
		slotVal.value = args.Value
		slotVal.seqNumAccepted = args.SeqNumAccepted
		slotVal.seqNumHighest = args.SeqNumHighest
		px.mu.Lock()
		px.maxDecidedSlot = getMax(px.maxDecidedSlot, args.SlotNum)
		px.mu.Unlock()

		reply.Status = paxosrpc.OK
	}
	return nil
}

// get the peer list stored in the paxos instance.
// PaxosAdmin use only
func (px *Paxos) GetPeers(args *paxosrpc.GetPeersArgs, reply *paxosrpc.GetPeersReply) error {
	px.mu.Lock()
	defer px.mu.Unlock()
	return nil

	reply.Peers = px.peers
	reply.Status = paxosrpc.OK
	return nil
}

// update the peer list and "me" of the paxos instance.
// PaxosAdmin use only
func (px *Paxos) UpdatePeers(args *paxosrpc.UpdatePeersArgs, reply *paxosrpc.UpdatePeersReply) error {
	px.mu.Lock()
	defer px.mu.Unlock()

	px.peers = args.Peers
	px.me = args.Me

	reply.Status = paxosrpc.OK
	return nil
}

// set global lock for the paxos instance, so if the server calls Start or Status the Paxos instance will
// return an error indicating the Paxos cluster is under maintainance.
// PaxosAdmin use only
func (px *Paxos) SetLock(args *paxosrpc.SetLockArgs, reply *paxosrpc.SetLockReply) error {
	px.mu.Lock()
	defer px.mu.Unlock()

	px.isLocked = args.SetLock
	reply.Status = paxosrpc.OK

	return nil
}

// sync gapped slot in the paxos instance and return the decided value
// PaxosAdmin use only
func (px *Paxos) SyncPaxosValue(args *paxosrpc.SyncPaxosValueArgs, reply *paxosrpc.SyncPaxosValueReply) error {
	slotLock := px.getTimeSlotLock(args.SlotNum)
	slotLock.Lock()
	slotVal := px.getTimeSlotValue(args.SlotNum)
	// if the value in specified slot is already dicided, return its value, seqNumHighest and seqNumAccepted directly
	if slotVal.isDecided {
		reply.Status = paxosrpc.OK
		reply.SyncedValue = slotVal.value
		reply.SeqNumHighest = slotVal.seqNumHighest
		reply.SeqNumAccepted = slotVal.seqNumAccepted
		slotLock.Unlock()
		return nil
	}
	slotLock.Unlock()

	// if the value in the specified slot doesn't exist or not decided, do another round of paxos to get the decided result
	err := px.startHelper(args.SlotNum, paxosrpc.PaxosValue{Type: paxosrpc.NoOp})
	if err != nil {
		return err
	}

	slotLock.Lock()
	defer slotLock.Unlock()
	slotVal = px.getTimeSlotValue(args.SlotNum)
	if slotVal.isDecided {
		reply.Status = paxosrpc.OK
		reply.SyncedValue = slotVal.value
		reply.SeqNumHighest = slotVal.seqNumHighest
		reply.SeqNumAccepted = slotVal.seqNumAccepted
		return nil
	} else {
		reply.Status = paxosrpc.Reject
		return errors.New("Cannot get the synced value, something went wrong")
	}
}

// checking whether the replica is alive or not
// PaxosAdmin use only
func (px *Paxos) CheckStatus(args *paxosrpc.CheckStatusArgs, reply *paxosrpc.CheckStatusReply) error {
	reply.Peers = px.peers
	// ////fmt.Println("mynumber", px.me, "peers:", px.peers)
	reply.Status = paxosrpc.OK
	return nil
}

// return status of all slots in local Paxos instance.
// PaxosAdmin use only
func (px *Paxos) CheckSlot(args *paxosrpc.CheckSlotArgs, reply *paxosrpc.CheckSlotReply) error {
	px.mu.Lock()
	defer px.mu.Unlock()

	var slotList []paxosrpc.SlotValue
	var slot paxosrpc.SlotValue
	for i := 0; i <= px.maxDecidedSlot; i++ {
		slotLock, exist := px.timeSLotLock[i]
		if !exist {
			slotLock = &sync.Mutex{}
			px.timeSLotLock[i] = slotLock
		}
		slotLock.Lock()

		val, ok := px.timeSlot[i]

		if ok {
			slot = paxosrpc.SlotValue{
				Value:     val.value,
				IsDecided: val.isDecided,
			}
		} else {
			slot = paxosrpc.SlotValue{
				IsDecided: false,
			}
		}
		slotList = append(slotList, slot)
		slotLock.Unlock()
	}

	reply.Status = paxosrpc.OK
	reply.Slot = slotList
	return nil
}

// close the listner of current Paxos instance.
// PaxosAdmin use only
func (px *Paxos) RemoteKill(args *paxosrpc.RemoteKillArgs, reply *paxosrpc.RemoteKillReply) error {
	px.deaf = true
	reply.Status = paxosrpc.OK
	if px.l != nil {
		px.l.Close()
	}
	return nil
}

// return the maximum decided slot num in the paxos instance
// PaxosAdmin use only
func (px *Paxos) MaxDecidedSlot(args *paxosrpc.MaxDecidedSlotArgs, reply *paxosrpc.MaxDecidedSlotReply) error {
	px.mu.Lock()
	defer px.mu.Unlock()

	reply.SlotNum = px.maxDecidedSlot
	reply.Status = paxosrpc.OK

	return nil
}

// return the minimal undecided slot num in the paxos instance
// PaxosAdmin use only
func (px *Paxos) MinUndecidedSlot(args *paxosrpc.MinUndecidedSlotArgs, reply *paxosrpc.MinUndecidedSlotReply) error {
	px.mu.Lock()
	slotNum := px.minUndecidedSlot

	for {
		slotLock, exist := px.timeSLotLock[slotNum]
		if !exist {
			slotLock = &sync.Mutex{}
			px.timeSLotLock[slotNum] = slotLock
		}
		slotLock.Lock()
		slotVal := px.getTimeSlotValue(slotNum)
		if slotVal.isDecided {
			slotNum++
		} else {
			slotLock.Unlock()
			break
		}
		slotLock.Unlock()
	}

	// change min undecided slot
	px.minUndecidedSlot = slotNum
	px.mu.Unlock()

	reply.SlotNum = slotNum
	reply.Status = paxosrpc.OK

	return nil
}

// start a new round of Paxos on slotNum with value v. The function will return an error if Paxos admin is adding/removing
// Paxos node from the cluster. The function will not return until it finishes a round of Paxos successfully.
func (px *Paxos) Start(slotNum int, v interface{}) error {
	px.mu.Lock()
	if px.isLocked {
		defer px.mu.Unlock()
		return errors.New("The paxos instances are trying to add a new paxos replica, thus unable to do any paxos operation at this moment")
	}
	px.mu.Unlock()

	px.startHelper(slotNum, v)
	return nil
}

// a helper function for Start, so Start will return error when adding new replica but PaxosAdmin can still call SyncPaxosValue
//which will invoke startHelper to do Paxos internally
func (px *Paxos) startHelper(slotNum int, v interface{}) error {
	timeSlotLock := px.getTimeSlotLock(slotNum)
	// get seqNumHighest from current time slot
	timeSlotLock.Lock()

	currTimeSlot := px.getTimeSlotValue(slotNum)
	maxSeqNumHighest := currTimeSlot.seqNumHighest
	timeSlotLock.Unlock()

	px.maxDoneLock.Lock()
	myMaxDone := px.maxDone[px.me]
	px.maxDoneLock.Unlock()

	// the code below does't use any variable in current time slot, so there is no need to lock the time slot. Another reason is calling Prepare,
	// Accept and Decide on the paxos peer itself will lock the time slot, thus may cause deadlock if we lock the time slot here
	// consecutiveMinorReply := 0
	for {
		now := time.Now()
		// create a globally unique seq number based on time and PAXOS peer number
		seqNum := now.UnixNano()*int64(len(px.peers)) + int64(px.me)
		// in case seqNum is smaller than the Nh of the peer itself or the highest Nh among rejected Prepare or Accept, add seqNum to a number which
		// is higher than Nh but still obey the seqNum generating rule of the current paxos peer
		if seqNum <= maxSeqNumHighest {
			seqNum += (maxSeqNumHighest - seqNum) - (maxSeqNumHighest-seqNum)%int64(len(px.peers)) + int64(len(px.peers))
		}

		// prepare phase
		numOk := 0
		proposeValue := v
		maxSeqNumAccepted := int64(0)
		maxSeqNumHighest = 0
		var success bool
		var err error
		prepareReplyCh := make(chan prepareReply, len(px.peers))

		// send prepare to every paxos peer including itself
		for i := 0; i < len(px.peers); i++ {
			args := &paxosrpc.PrepareArgs{SlotNum: slotNum, SeqNum: seqNum, PeerNum: px.me, MaxDone: myMaxDone}
			reply := &paxosrpc.PrepareReply{}
			if i != px.me {
				go prepareHelper(prepareReplyCh, px.peers[i], args, reply)
			} else {
				err = px.Prepare(args, reply)
				if err == nil {
					success = true
				} else {
					success = false
				}
				prepareReplyCh <- prepareReply{success: success, reply: reply}
			}
		}

		for i := 0; i < len(px.peers); i++ {
			prepareRep := <-prepareReplyCh
			success = prepareRep.success
			reply := prepareRep.reply
			if success {
				if reply.Status == paxosrpc.OK {
					numOk++
					if reply.SeqNumAccepted > maxSeqNumAccepted {
						maxSeqNumAccepted = reply.SeqNumAccepted
						proposeValue = reply.ValueAccepted
					}
				} else {
					if reply.SeqNumHighest > maxSeqNumHighest {
						maxSeqNumHighest = reply.SeqNumHighest
					}
				}
			}

			if numOk > len(px.peers)/2 {
				break
			}
		}
		// if minority of paxos peers reply ok, restart paxos
		if numOk <= len(px.peers)/2 {
			continue
		}

		// accept phase
		numOk = 0
		maxSeqNumHighest = 0
		acceptReplyCh := make(chan acceptReply, len(px.peers))

		for i := 0; i < len(px.peers); i++ {
			args := &paxosrpc.AcceptArgs{SlotNum: slotNum, SeqNum: seqNum, Value: proposeValue}
			reply := &paxosrpc.AcceptReply{}
			if i != px.me {
				go acceptHelper(acceptReplyCh, px.peers[i], args, reply)
			} else {
				err = px.Accept(args, reply)
				if err == nil {
					success = true
				} else {
					success = false
				}
				acceptReplyCh <- acceptReply{success: success, reply: reply}
			}
		}

		for i := 0; i < len(px.peers); i++ {
			acceptRep := <-acceptReplyCh
			success = acceptRep.success
			reply := acceptRep.reply
			if success {
				if reply.Status == paxosrpc.OK {
					numOk++
				} else {
					if reply.SeqNumHighest > maxSeqNumHighest {
						maxSeqNumHighest = reply.SeqNumHighest
					}
				}
			}
			if numOk > len(px.peers)/2 {
				break
			}
		}

		// if monority of the paxos peers reply ok, restart paxos
		if numOk <= len(px.peers)/2 {
			continue
		}

		// decide phase
		for i := 0; i < len(px.peers); i++ {
			args := &paxosrpc.DecideArgs{SlotNum: slotNum, Value: proposeValue}
			reply := &paxosrpc.DecideReply{}
			// the reply of the Decide function does not matter
			if i != px.me {
				go decideHelper(px.peers[i], args, reply)
			} else {
				px.Decide(args, reply)
			}

		}
		break
	}

	return nil
}

func prepareHelper(ch chan prepareReply, srv string, args *paxosrpc.PrepareArgs, reply *paxosrpc.PrepareReply) {
	success := call(srv, "RemotePaxosPeer.Prepare", args, reply)
	ch <- prepareReply{success: success, reply: reply}
}

func acceptHelper(ch chan acceptReply, srv string, args *paxosrpc.AcceptArgs, reply *paxosrpc.AcceptReply) {
	success := call(srv, "RemotePaxosPeer.Accept", args, reply)
	ch <- acceptReply{success: success, reply: reply}
}

func decideHelper(srv string, args *paxosrpc.DecideArgs, reply *paxosrpc.DecideReply) {
	call(srv, "RemotePaxosPeer.Decide", args, reply)
}

// return the local status of the slot given
func (px *Paxos) Status(slotNum int) (bool, interface{}, error) {
	px.mu.Lock()
	if px.isLocked {
		defer px.mu.Unlock()
		return false, nil, errors.New("The paxos instances are trying to add a new paxos replica, thus unable to do any paxos operation at this moment")
	}
	px.mu.Unlock()

	lock := px.getTimeSlotLock(slotNum)
	lock.Lock()
	defer lock.Unlock()

	// if slotNum < px.Min() {
	// 	return false, nil, nil
	// }

	timeslot := px.getTimeSlotValue(slotNum)
	return timeslot.isDecided, timeslot.value, nil
}

func (px *Paxos) Deaf() {
	px.deaf = true
	if px.l != nil {
		px.l.Close()
	}
}

func (px *Paxos) Wake() {
	var err error
	if px.deaf {
		port := strings.Split(px.peers[px.me], ":")[1]
		px.l, err = net.Listen("tcp", ":"+port)
		for err != nil {
			px.l, err = net.Listen("tcp", ":"+port)
		}
		go http.Serve(px.l, nil)
		px.deaf = false
	}
}

// Make a Paxos instance, given hostports of all peers in the Paxos cluster.
// This server's hostport is peers[me]
func Make(peers []string, me int, rpcs *rpc.Server) (*Paxos, error) {
	// check the hostport format of all peers
	var err error

	for _, peer := range peers {
		if len(strings.Split(peer, ":")) != 2 {
			return nil, errors.New("peers format invalid")
		}
	}

	// register the concrete value for interface{} for encoding and decoding in RPC call
	gob.Register(paxosrpc.PaxosValue{})

	px := &Paxos{}
	px.peers = peers
	px.me = me

	px.timeSlot = make(map[int]*timeSlotValue)
	px.timeSLotLock = make(map[int]*sync.Mutex)
	px.isLocked = false
	px.testMode = false
	// initialize maxDone
	px.maxDone = make(map[int]int)
	for i := 0; i < len(px.peers); i++ {
		px.maxDone[i] = -1
	}

	if rpcs != nil {
		// caller will create socket &c
		rpcs.RegisterName("RemotePaxosPeer", paxosrpc.Wrap(px))
	} else {
		port := strings.Split(peers[me], ":")[1]
		// open listener until success
		px.l, err = net.Listen("tcp", ":"+port)
		for err != nil {
			px.l, err = net.Listen("tcp", ":"+port)
		}
		// register RPC until success
		err = rpc.RegisterName("RemotePaxosPeer", paxosrpc.Wrap(px))
		for err != nil {
			err = rpc.RegisterName("RemotePaxosPeer", paxosrpc.Wrap(px))
		}
		// bind listener with the RPC service
		rpc.HandleHTTP()
		go http.Serve(px.l, nil)
	}

	return px, nil
}

// func MakeTest(peers []string, me int, rpcs *rpc.Server) *Paxos {

// 	px := &Paxos{}
// 	px.peers = peers
// 	px.me = me

// 	px.timeSlot = make(map[int]*timeSlotValue)
// 	px.timeSLotLock = make(map[int]*sync.Mutex)
// 	px.isLocked = false
// 	px.testMode = true
// 	// initialize maxDone
// 	px.maxDone = make(map[int]int)
// 	for i := 0; i < len(px.peers); i++ {
// 		px.maxDone[i] = -1
// 	}

// 	if rpcs != nil {
// 		// caller will create socket &c
// 		rpcs.Register(px)
// 	} else {
// 		rpcs = rpc.NewServer()
// 		rpcs.RegisterName("RemotePaxosPeer", paxosrpc.Wrap(px))

// 		// prepare to receive connections from clients.
// 		// change "unix" to "tcp" to use over a network.
// 		os.Remove(peers[me]) // only needed for "unix"
// 		l, e := net.Listen("unix", peers[me])
// 		if e != nil {
// 			log.Fatal("listen error: ", e)
// 		}
// 		px.l = l

// 		// please do not change any of the following code,
// 		// or do anything to subvert it.

// 		// create a thread to accept RPC connections
// 		go func() {
// 			for px.deaf == false {
// 				conn, err := px.l.Accept()
// 				if err == nil && px.deaf == false {
// 					if px.unreliable && (rand.Int63()%1000) < 100 {
// 						// discard the request.
// 						conn.Close()
// 					} else if px.unreliable && (rand.Int63()%1000) < 200 {
// 						// process the request but force discard of reply.
// 						c1 := conn.(*net.UnixConn)
// 						f, _ := c1.File()
// 						err := syscall.Shutdown(int(f.Fd()), syscall.SHUT_WR)
// 						if err != nil {
// 							// ////fmt.Printf("shutdown: %v\n", err)
// 						}
// 						px.rpcCount++
// 						go rpcs.ServeConn(conn)
// 					} else {
// 						px.rpcCount++
// 						go rpcs.ServeConn(conn)
// 					}
// 				} else if err == nil {
// 					conn.Close()
// 				}
// 				if err != nil && px.deaf == false {
// 					// ////fmt.Printf("Paxos(%v) accept: %v\n", me, err.Error())
// 				}
// 			}
// 		}()
// 	}

// 	return px
// }

// func (px *Paxos) Done(slotNum int) {
// 	px.maxDoneLock.Lock()
// 	defer px.maxDoneLock.Unlock()
// 	px.maxDone[px.me] = getMax(px.maxDone[px.me], slotNum)
// 	// Your code here.
// }

//
// the application wants to know the
// highest instance sequence known to
// this peer.
//
// func (px *Paxos) Max() int {
// 	// Your code here.
// 	px.mu.Lock()
// 	px.mu.Unlock()
// 	max := 0
// 	for key, _ := range px.timeSlot {
// 		max = getMax(max, key)
// 	}
// 	return max
// }

// func (px *Paxos) Min() int {
// 	// You code here.
// 	px.maxDoneLock.Lock()
// 	defer px.maxDoneLock.Unlock()

// 	min := px.maxDone[px.me]
// 	for _, value := range px.maxDone {
// 		min = getMin(value, min)
// 	}
// 	return min + 1
// }

// func (px *Paxos) cleanMemory(args *paxosrpc.PrepareArgs) {

// 	px.maxDoneLock.Lock()
// 	px.maxDone[args.PeerNum] = getMax(px.maxDone[args.PeerNum], args.MaxDone)
// 	min := math.MaxInt32
// 	for _, maxDone := range px.maxDone {
// 		min = getMin(min, maxDone)
// 	}
// 	px.maxDoneLock.Unlock()

// 	min++
// 	for key := range px.timeSlot {
// 		if key < min {
// 			delete(px.timeSlot, key)
// 		}
// 	}
// }

//
// tell the peer to shut itself down.
// for testing.
// please do not change this function.
//
// func (px *Paxos) Kill() {
// 	px.deaf = true
// 	if px.l != nil {
// 		px.l.Close()
// 	}
// }

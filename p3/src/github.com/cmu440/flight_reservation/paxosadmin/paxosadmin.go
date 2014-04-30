package paxosadmin

import (
	"encoding/gob"
	"errors"
	"fmt"
	// "fmt"
	"github.com/cmu440/flight_reservation/rpc/paxosrpc"
	"math"
	"net/rpc"
	"strconv"
	"time"
)

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

// helper function to make a RPC call to remote server.
// return false if error occurs during Dial or Call.
// the function uses Unix rpc call when testing.
func paxosAdminCall(srv string, name string, args interface{}, reply interface{}) bool {
	// c, err := rpc.Dial("unix", srv)
	var c *rpc.Client
	var err error

	c, err = rpc.DialHTTP("tcp", srv)

	if err != nil {
		// fmt.Println(err)
		return false
	}
	defer c.Close()

	err = c.Call(name, args, reply)
	if err == nil {
		return true
	}
	// fmt.Println(err)
	return false
}

// This function is help to see the slot value on each server
func ShowSlot(peers []string) (map[string][]paxosrpc.SlotValue, error) { //first is good, second is bad
	allSlotValue := make(map[string][]paxosrpc.SlotValue)
	gob.Register(paxosrpc.PaxosValue{})

	for i := 0; i < len(peers); i++ {
		args := &paxosrpc.CheckSlotArgs{}
		reply := &paxosrpc.CheckSlotReply{}
		success := paxosAdminCall(peers[i], "RemotePaxosPeer.CheckSlot", args, reply)
		if success {
			if reply.Status == paxosrpc.OK {
				allSlotValue[peers[i]] = reply.Slot
			} else {
				return nil, errors.New("Showallslot error")
			}
		}
	}
	return allSlotValue, nil
}

// this function is to show all the servers hostport
func ShowPeers(peers []string) (map[string][]string, []string, error) { //first is good, second is bad

	goodPeers := make(map[string][]string)
	var badPeers []string

	for i := 0; i < len(peers); i++ {

		args := &paxosrpc.CheckStatusArgs{}
		reply := &paxosrpc.CheckStatusReply{}
		success := paxosAdminCall(peers[i], "RemotePaxosPeer.CheckStatus", args, reply)
		if success {

			goodPeers[peers[i]] = reply.Peers
		} else {
			badPeers = append(badPeers, peers[i])
		}

	}
	return goodPeers, badPeers, nil
}

// this function is to lock all the server when you add or remove the server
func lockPeers(peers []string, value bool) error {
	errMsg := ""

	for i := 0; i < len(peers); i++ {
		SetLockArgs := &paxosrpc.SetLockArgs{
			SetLock: value,
		}
		SetLockReply := &paxosrpc.SetLockReply{}
		success := paxosAdminCall(peers[i], "RemotePaxosPeer.SetLock", SetLockArgs, SetLockReply)
		if !success {
			errMsg += "cannot connect to peer " + strconv.Itoa(i)
		}
	}
	if len(errMsg) == 0 {
		return nil
	} else {
		return errors.New(errMsg)
	}
}

// this function function is to sync all the slot
func syncPeers(peers []string) (int, int, error) {
	minUndecidedSlot := math.MaxInt32
	maxDecidedSlot := math.MinInt32
	errMsg := ""

	// get the minimal undecided slot and maximal decided slot from all peers to determine the start and end slot
	// for following sync works
	for i := 0; i < len(peers); i++ {
		// fmt.Println("starting peer " + peers[i])
		MinUndecidedSlotArgs := &paxosrpc.MinUndecidedSlotArgs{}
		MinUndecidedSlotReply := &paxosrpc.MinUndecidedSlotReply{}
		success := paxosAdminCall(peers[i], "RemotePaxosPeer.MinUndecidedSlot", MinUndecidedSlotArgs, MinUndecidedSlotReply)

		if !success || MinUndecidedSlotReply.Status == paxosrpc.Reject {
			// fmt.Println("error occurs when trying to get minimal undecided slot from peer " + peers[i])
			errMsg += "error occurs when trying to get minimal undecided slot from peer " + peers[i]
		} else {
			minUndecidedSlot = getMin(minUndecidedSlot, MinUndecidedSlotReply.SlotNum)
		}

		MaxDecidedSlotArgs := &paxosrpc.MaxDecidedSlotArgs{}
		MaxDecidedSlotReply := &paxosrpc.MaxDecidedSlotReply{}
		success = paxosAdminCall(peers[i], "RemotePaxosPeer.MaxDecidedSlot", MaxDecidedSlotArgs, MaxDecidedSlotReply)

		if !success || MaxDecidedSlotReply.Status == paxosrpc.Reject {
			// fmt.Println("error occurs when trying to get maximal decided slot from peer " + peers[i])
			errMsg += "error occurs when trying to get maximal decided slot from peer " + peers[i]
		} else {
			maxDecidedSlot = getMax(maxDecidedSlot, MaxDecidedSlotReply.SlotNum)
		}

	}

	// sync all gapped slots from minimal undecided slot to maximal decided slot on all paxos peers
	for i := minUndecidedSlot; i <= maxDecidedSlot; i++ {
		for j := 0; j < len(peers); j++ {
			SyncPaxosValueArgs := &paxosrpc.SyncPaxosValueArgs{
				SlotNum: i,
			}
			SyncPaxosValueReply := &paxosrpc.SyncPaxosValueReply{}
			success := paxosAdminCall(peers[j], "RemotePaxosPeer.SyncPaxosValue", SyncPaxosValueArgs, SyncPaxosValueReply)
			if !success || SyncPaxosValueReply.Status == paxosrpc.Reject {
				errMsg += "error occurs when trying to sync gapped slot " + strconv.Itoa(i) + " on peer " + peers[i]
			}
		}
	}

	if len(errMsg) == 0 {
		return minUndecidedSlot, maxDecidedSlot, nil
	} else {
		return 0, 0, errors.New(errMsg)
	}

}

func sliceEqual(slice1 []string, slice2 []string) bool {
	if len(slice1) != len(slice2) {
		return false
	}

	for i := 0; i < len(slice1); i++ {
		for j := i + 1; j < len(slice1); j++ {
			if slice1[i] > slice1[j] {
				slice1[i], slice1[j] = slice1[j], slice1[i]
			}
		}
	}

	for i := 0; i < len(slice2); i++ {
		for j := i + 1; j < len(slice2); j++ {
			if slice2[i] > slice2[j] {
				slice2[i], slice2[j] = slice2[j], slice2[i]
			}
		}
	}

	for i := 0; i < len(slice1); i++ {
		if slice1[i] != slice2[i] {
			return false
		}
	}

	return true
}

// add the the server for the paxos
func AddPeer(peers []string, addPeer string, sleepSecs int) error {
	var newPeers []string
	var err error
	lockPeers(peers, false) // first unlock all
	gob.Register(paxosrpc.PaxosValue{})

	if len(peers) == 0 {
		return errors.New("number of existing paxos peers cannot be zero")
	}

	goodPeersmap, _, _ := ShowPeers(peers)
	// check if all Paxos peers in the peers are working, which is required when adding new Paxos node
	if len(goodPeersmap) < len(peers) {
		return errors.New("Some Paxos peers in the peers list given are not working, which is not permitted when adding new Paxos node, you must first remove these un-working Paxos nodes")
	}
	// check if the peer list in all paxos peers is consistent with the peer list given
	for _, peersList := range goodPeersmap {
		if !sliceEqual(peersList, peers) {
			return errors.New("The peers list in some Paxos peer is not consistent with the peers list given, please re-check the peers list on all Paxos peers and update the unconsistent one")
		}
	}

	newPeers = append(peers, addPeer)

	// quiesce all peers
	err = lockPeers(newPeers, true)
	defer lockPeers(newPeers, false)
	if sleepSecs > 0 {
		time.Sleep(time.Duration(sleepSecs) * time.Second)
	}

	if err != nil {
		return err
	}

	// fmt.Println("lock finish")

	_, maxDecidedSlot, err := syncPeers(peers)
	if err != nil {
		return err
	}

	// fmt.Println("sync finish")
	// update peers list on every existing peers
	for i := 0; i < len(peers); i++ {
		updatePeersArgs := &paxosrpc.UpdatePeersArgs{
			Peers: newPeers,
			Me:    i,
		}
		updatePeersReply := &paxosrpc.UpdatePeersReply{}
		success := paxosAdminCall(peers[i], "RemotePaxosPeer.UpdatePeers", updatePeersArgs, updatePeersReply)

		if !success || updatePeersReply.Status == paxosrpc.Reject {
			return errors.New("error occurs when updating peers in AddPeer")
		}
	}

	// update peer list on the new added peer
	updatePeersArgs := &paxosrpc.UpdatePeersArgs{
		Peers: newPeers,
		Me:    len(newPeers) - 1,
	}
	updatePeersReply := &paxosrpc.UpdatePeersReply{}

	success := paxosAdminCall(addPeer, "RemotePaxosPeer.UpdatePeers", updatePeersArgs, updatePeersReply)
	if !success || updatePeersReply.Status == paxosrpc.Reject {
		errors.New("error occurs when updating peers in AddPeer")
	}

	// fmt.Println("update peers finish")
	// sync the new peer that we add
	if len(peers) > 0 {
		for i := 0; i <= maxDecidedSlot; i++ {
			syncPaxosValueArgs := &paxosrpc.SyncPaxosValueArgs{
				SlotNum: i,
			}
			syncPaxosValueReply := &paxosrpc.SyncPaxosValueReply{}
			success = paxosAdminCall(peers[0], "RemotePaxosPeer.SyncPaxosValue", syncPaxosValueArgs, syncPaxosValueReply)
			if !success || syncPaxosValueReply.Status == paxosrpc.Reject {
				return errors.New("error occurs when trying to sync all slots from one paxos peer to the new added peer")
			} else {
				putArgs := paxosrpc.PutArgs{
					SlotNum:        i,
					Value:          syncPaxosValueReply.SyncedValue,
					SeqNumAccepted: syncPaxosValueReply.SeqNumAccepted,
					SeqNumHighest:  syncPaxosValueReply.SeqNumHighest,
				}
				// fmt.Println("put", putArgs)
				// fmt.Println(addPeer)
				putReply := &paxosrpc.PutReply{}
				success = paxosAdminCall(addPeer, "RemotePaxosPeer.Put", putArgs, putReply)
				if !success || putReply.Status == paxosrpc.Reject {
					return errors.New("slot:" + strconv.Itoa(i) + "error occurs when trying to put all slots from one paxos peer to the new added peer")
				}
			}
		}
	}

	// fmt.Println("copy slots finish")
	// un-quiesce all peers
	// lockPeers(newPeers, false)

	// fmt.Println("unlock finish")
	return nil
}

// remove the the server for the paxos
func RemovePeer(peers []string, removePeer string, sleepSecs int) error {
	var newPeers []string
	lockPeers(peers, false) // first unlock all
	RemoteKillArgs := &paxosrpc.RemoteKillArgs{}
	RemoteKillReply := &paxosrpc.RemoteKillReply{}
	_ = paxosAdminCall(removePeer, "RemotePaxosPeer.RemoteKill", RemoteKillArgs, RemoteKillReply)

	goodPeersmap, _, _ := ShowPeers(peers)
	// check if the peer list in all paxos peers is consistent with the peer list given
	for _, peersList := range goodPeersmap {
		if !sliceEqual(peersList, peers) {
			return errors.New("The peers list in some Paxos peer is not consistent with the peers list given, please re-check the peers list on all Paxos peers and update the unconsistent one")
		}
	}

	exist := false
	for _, peer := range peers {
		if peer == removePeer {
			exist = true
			break
		}
	}
	if !exist {
		return errors.New("the hostport of removed peer is not in the list of peers given")
	}

	goodPeersSet := make(map[string]bool)
	var goodPeers []string
	for peer, _ := range goodPeersmap {
		goodPeers = append(goodPeers, peer)
	}

	for _, peer := range goodPeers {
		goodPeersSet[peer] = true
	}

	lockPeers(goodPeers, true)
	defer lockPeers(goodPeers, false)

	if sleepSecs > 0 {
		time.Sleep(time.Duration(sleepSecs) * time.Second)
	}
	// fmt.Println("here1")
	// we only need to sync the good peers
	_, _, err := syncPeers(goodPeers)
	if err != nil {
		return err
	}

	for i := 0; i < len(peers); i++ {
		if peers[i] != removePeer {
			newPeers = append(newPeers, peers[i])
		}
	}
	// fmt.Println("here2")
	for i := 0; i < len(newPeers); i++ {

		updatePeersArgs := &paxosrpc.UpdatePeersArgs{
			Peers: newPeers,
			Me:    i,
		}
		updatePeersReply := &paxosrpc.UpdatePeersReply{}
		success := paxosAdminCall(newPeers[i], "RemotePaxosPeer.UpdatePeers", updatePeersArgs, updatePeersReply)
		if (!success || updatePeersReply.Status == paxosrpc.Reject) && goodPeersSet[newPeers[i]] {
			return errors.New("error occurs when updating peers on paxos peer " + newPeers[i] + " in RemovePeer")
		}
	}
	return nil
}

func ChangePeer(peers []string, peerme string) error {
	// fmt.Println("peers:", peers)
	// fmt.Println("peerme", peerme)
	lockPeers(peers, false) // first unlock all
	gob.Register(paxosrpc.PaxosValue{})

	if len(peers) == 0 {
		return errors.New("number of existing paxos peers cannot be zero")
	}

	// quiesce all peers
	lockPeers(peers, true)
	defer lockPeers(peers, false)

	me := 0
	for i := 0; i < len(peers); i++ {
		if peers[i] == peerme {
			me = i
		}
	}
	fmt.Println(peers, peerme, me)
	for i := 0; i < len(peers); i++ {
		updatePeersArgs := &paxosrpc.UpdatePeersArgs{
			Peers: peers,
			Me:    me,
		}
		updatePeersReply := &paxosrpc.UpdatePeersReply{}
		_ = paxosAdminCall(peers[i], "RemotePaxosPeer.UpdatePeers", updatePeersArgs, updatePeersReply)

	}

	return nil
}

// this file contains constants and arguments used to perform RPCs between different PAXOS peers
// Author: Chun Chen

package paxosrpc

import (
	"time"
)

// Status represents the status of a RPC's reply
type Status int
type OpType int

const (
	OK Status = iota + 1
	Reject
)

const (
	MyReservation OpType = iota + 1
	CheckTickets
	ReserveTicket
	AddFlight
	CancelFlight
	EditFlight
	NoOp
)

type PaxosValue struct {
	Type      OpType
	Timestamp time.Time
	UserName  string
	FlightId  string
	StartDate time.Time
	EndDate   time.Time
	TicketNum int
}

type SlotValue struct {
	Value     interface{}
	IsDecided bool
}

type PrepareArgs struct {
	SlotNum int
	SeqNum  int64
	PeerNum int
	MaxDone int
}

type PrepareReply struct {
	Status         Status
	SeqNumAccepted int64
	ValueAccepted  interface{}
	SeqNumHighest  int64
}

type AcceptArgs struct {
	SlotNum int
	SeqNum  int64
	Value   interface{}
}

type AcceptReply struct {
	Status        Status
	SeqNumHighest int64
}

type DecideArgs struct {
	SlotNum int
	Value   interface{}
}

type DecideReply struct {
	Status Status
}

type PutArgs struct {
	SlotNum        int
	Value          interface{}
	SeqNumAccepted int64
	SeqNumHighest  int64
}

type PutReply struct {
	Status Status
}

type GetPeersArgs struct {
}

type GetPeersReply struct {
	Status Status
	Peers  []string
}

type UpdatePeersArgs struct {
	Peers []string
	Me    int
}

type UpdatePeersReply struct {
	Status Status
}

type SetLockArgs struct {
	SetLock bool
}

type SetLockReply struct {
	Status Status
}

type SyncPaxosValueArgs struct {
	SlotNum int
}

type SyncPaxosValueReply struct {
	Status         Status
	SyncedValue    interface{}
	SeqNumAccepted int64
	SeqNumHighest  int64
}

type CheckStatusArgs struct {
}

type CheckStatusReply struct {
	Peers  []string
	Status Status
}

type MaxDecidedSlotArgs struct {
}

type MaxDecidedSlotReply struct {
	SlotNum int
	Status  Status
}

type MinUndecidedSlotArgs struct {
}

type MinUndecidedSlotReply struct {
	SlotNum int
	Status  Status
}

type RemoteKillArgs struct {
}

type RemoteKillReply struct {
	Status Status
}

type CheckSlotArgs struct {
}

type CheckSlotReply struct {
	Status Status
	Slot   []SlotValue
}

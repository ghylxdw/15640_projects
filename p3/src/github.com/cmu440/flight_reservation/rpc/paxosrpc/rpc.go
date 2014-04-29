// this file provides a type-safe wrapper that should be used to register the
// paxos peer to receive RPCs from other RPC peers or PaxosAdmin

package paxosrpc

type RemotePaxosPeer interface {
	Prepare(*PrepareArgs, *PrepareReply) error
	Accept(*AcceptArgs, *AcceptReply) error
	Decide(*DecideArgs, *DecideReply) error
	RemoteKill(*RemoteKillArgs, *RemoteKillReply) error

	Put(*PutArgs, *PutReply) error
	GetPeers(*GetPeersArgs, *GetPeersReply) error
	UpdatePeers(*UpdatePeersArgs, *UpdatePeersReply) error
	SetLock(*SetLockArgs, *SetLockReply) error
	SyncPaxosValue(*SyncPaxosValueArgs, *SyncPaxosValueReply) error
	CheckStatus(*CheckStatusArgs, *CheckStatusReply) error
	MaxDecidedSlot(*MaxDecidedSlotArgs, *MaxDecidedSlotReply) error
	MinUndecidedSlot(*MinUndecidedSlotArgs, *MinUndecidedSlotReply) error
	CheckSlot(*CheckSlotArgs, *CheckSlotReply) error
}

type PaxosPeer struct {
	RemotePaxosPeer
}

func Wrap(p RemotePaxosPeer) RemotePaxosPeer {
	return &PaxosPeer{p}
}

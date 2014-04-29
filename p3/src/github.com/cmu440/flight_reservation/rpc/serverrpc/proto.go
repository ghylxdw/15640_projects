// this file contains constants and arguments used to perform RPCs
// between client/airline admin and the airline server
// Author: Chun Chen

package serverrpc

import (
	"time"
)

// Status represents the status of a RPC's reply
type Status int

const (
	Ok Status = iota + 1
	Reject
)

type FlightInfo struct {
	FlightId  string
	Date      time.Time
	TicketNum int
}

type MyReservationArgs struct {
	UserName string
}

type MyReservationReply struct {
	Flights []FlightInfo
}

type CheckTicketsArgs struct {
	FlightId  string
	StartDate time.Time
	EndDate   time.Time
}

type CheckTicketsReply struct {
	Flights []FlightInfo
}

type ReserveTicketArgs struct {
	UserName string
	FlightId string
	Date     time.Time
}

type ReserveTicketReply struct {
	Status Status
}

type AddFlightArgs struct {
	FlightId  string
	Date      time.Time
	TicketNum int
}

type AddFlightReply struct {
	Status Status
}

type CancelFlightArgs struct {
	FlightId string
	Date     time.Time
}

type CancelFlightReply struct {
	Status Status
}

type EditFlightArgs struct {
	FlightId  string
	Date      time.Time
	TicketNum int
}

type EditFlightReply struct {
	Status Status
}

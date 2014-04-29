package server

import (
	"github.com/cmu440/flight_reservation/rpc/serverrpc"
)

type Server interface {
	MyReservation(*serverrpc.MyReservationArgs, *serverrpc.MyReservationReply) error
	CheckTickets(*serverrpc.CheckTicketsArgs, *serverrpc.CheckTicketsReply) error
	ReserveTicket(*serverrpc.ReserveTicketArgs, *serverrpc.ReserveTicketReply) error

	AddFlight(*serverrpc.AddFlightArgs, *serverrpc.AddFlightReply) error
	CancelFlight(*serverrpc.CancelFlightArgs, *serverrpc.CancelFlightReply) error
	EditFlight(*serverrpc.EditFlightArgs, *serverrpc.EditFlightReply) error

	Deaf()
	Wake()
}

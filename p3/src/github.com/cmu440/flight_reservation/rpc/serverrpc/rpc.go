// this file provides a type-safe wrapper that should be used to register the
// airline server to receive RPCS from clients or AirlineAdmin

package serverrpc

type RemoteServer interface {
	MyReservation(*MyReservationArgs, *MyReservationReply) error
	CheckTickets(*CheckTicketsArgs, *CheckTicketsReply) error
	ReserveTicket(*ReserveTicketArgs, *ReserveTicketReply) error

	AddFlight(*AddFlightArgs, *AddFlightReply) error
	CancelFlight(*CancelFlightArgs, *CancelFlightReply) error
	EditFlight(*EditFlightArgs, *EditFlightReply) error
}

type Server struct {
	RemoteServer
}

func Wrap(p RemoteServer) RemoteServer {
	return &Server{p}
}

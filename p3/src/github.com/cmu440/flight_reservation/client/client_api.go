package client

import "github.com/cmu440/flight_reservation/rpc/serverrpc"

const shortForm = "2006.1.2"

type Client interface {
	MyReservation(username string) ([]serverrpc.FlightInfo, error)
	CheckTickets(flightId string, startDate string, endDate string) ([]serverrpc.FlightInfo, error)
	ReserveTicket(username string, flightId string, date string) error

	Close() error
}

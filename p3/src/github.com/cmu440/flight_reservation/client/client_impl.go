package client

import (
	"errors"
	"github.com/cmu440/flight_reservation/rpc/serverrpc"
	"net/rpc"
	"time"
)

type client struct {
	client *rpc.Client
}

type channel struct {
	client *rpc.Client
	err    error
}

func NewClient(serverHostPort string) (Client, error) {
	c := make(chan channel, 1)
	go func() {
		client, err := rpc.DialHTTP("tcp", serverHostPort)
		c <- channel{
			client: client,
			err:    err,
		}
	}()

	select {
	case now := <-c:
		if now.err != nil {
			return nil, now.err
		}
		return &client{client: now.client}, nil
	case <-time.After(2 * time.Second):
		return nil, errors.New("Client Dial tcp timeout")
	}
}

func (cli *client) MyReservation(username string) ([]serverrpc.FlightInfo, error) {
	args := &serverrpc.MyReservationArgs{UserName: username}
	reply := &serverrpc.MyReservationReply{}
	if err := cli.client.Call("Server.MyReservation", args, reply); err != nil {
		return nil, err
	}
	return reply.Flights, nil
}

func (cli *client) CheckTickets(flightId string, startDate string, endDate string) ([]serverrpc.FlightInfo, error) {
	// fmt.Println("input stateDate:", startDate, ";enddDate:", endDate)
	starttime, starttimeerr := time.Parse(shortForm, startDate) //"2014-05-01"
	endtime, endtimeerr := time.Parse(shortForm, endDate)       //"2014-05-01"
	// fmt.Println("parse stateDate:", starttime, ";enddDate:", endtime)
	if starttimeerr != nil {
		return nil, starttimeerr
	}
	if endtimeerr != nil {
		return nil, endtimeerr
	}
	// fmt.Println("starttime", starttime, "endtime", endtime)
	args := &serverrpc.CheckTicketsArgs{
		FlightId:  flightId,
		StartDate: starttime,
		EndDate:   endtime,
	}
	reply := &serverrpc.CheckTicketsReply{}
	if err := cli.client.Call("Server.CheckTickets", args, reply); err != nil {
		return nil, err
	}
	return reply.Flights, nil
}

func (cli *client) ReserveTicket(username string, flightId string, date string) error {
	datetime, err := time.Parse(shortForm, date) //"2014-05-01"
	if err != nil {
		return err
	}
	args := &serverrpc.ReserveTicketArgs{
		UserName: username,
		FlightId: flightId,
		Date:     datetime,
	}
	reply := &serverrpc.ReserveTicketReply{}
	if err := cli.client.Call("Server.ReserveTicket", args, reply); err != nil {
		return err
	}
	if reply.Status == serverrpc.Ok {
		return nil
	} else {
		return errors.New("Sorry, there is not tickets available for flight " + flightId + " on " + date)
	}

}

// func (cli *client) AddFlight(flightId string, date string, number int) error {
// 	datetime, _ := time.Parse(shortForm, date) //"2014-05-01"

// 	args := &serverrpc.AddFlightArgs{
// 		FlightId: flightId,
// 		Date:     datetime,
// 		Number:   number,
// 	}
// 	reply := &serverrpc.AddFlightReply{}
// 	if err := cli.client.Call("Server.AddFlight", args, &reply); err != nil {
// 		return err
// 	}
// 	if reply.Status == serverrpc.Ok {
// 		return nil
// 	} else {
// 		return errors.New("add flight error")
// 	}
// }

// func (cli *client) CancelFlight(flightId string, date string) error {
// 	datetime, _ := time.Parse(shortForm, date) //"2014-05-01"
// 	args := &serverrpc.CancelFlightArgs{
// 		FlightId: flightId,
// 		Date:     datetime,
// 	}
// 	reply := &serverrpc.CancelFlightReply{}
// 	if err := cli.client.Call("Server.CancelFlight", args, &reply); err != nil {
// 		return err
// 	}
// 	if reply.Status == serverrpc.Ok {
// 		return nil
// 	} else {
// 		return errors.New("cancel flight error")
// 	}
// }

// func (cli *client) EditFlight(flightId string, date string, number int) error {
// 	datetime, _ := time.Parse(shortForm, date) //"2014-05-01"
// 	args := &serverrpc.EditFlightArgs{
// 		FlightId: flightId,
// 		Date:     datetime,
// 		Number:   number,
// 	}
// 	reply := &serverrpc.EditFlightReply{}
// 	if err := cli.client.Call("Server.EditFlight", args, &reply); err != nil {
// 		return err
// 	}
// 	if reply.Status == serverrpc.Ok {
// 		return nil
// 	} else {
// 		return errors.New("edit flight  error")
// 	}
// }

func (cli *client) Close() error {
	return cli.client.Close()
}

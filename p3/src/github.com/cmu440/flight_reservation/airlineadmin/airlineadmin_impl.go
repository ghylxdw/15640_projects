package airlineadmin

import (
	"errors"
	"fmt"
	"github.com/cmu440/flight_reservation/rpc/serverrpc"
	"net/rpc"
	"os"
	"strconv"
	"time"
)

type airlineadmin struct {
	client *rpc.Client
}

type channel struct {
	client *rpc.Client
	err    error
}

/*start and newairline admin and use the channel to dial the server*/
func NewAirlineadmin(HostPort string) (Airlineadmin, error) {
	c := make(chan channel, 1)
	go func() {
		client, err := rpc.DialHTTP("tcp", HostPort)
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
		return &airlineadmin{client: now.client}, nil
	case <-time.After(2 * time.Second):
		return nil, errors.New("Client Dial tcp timeout")
	}

}

func (air *airlineadmin) AddFlight(flightId string, date string, number string) error {
	// fmt.Println("addflight")
	// fmt.Println("input date", date)
	// parse the input time and add the ticket for the server
	datetime, err := time.Parse(shortForm, date) //"2014-05-01"
	if err != nil {
		return err

	}
	// fmt.Println("parse date", datetifme)
	num, err := strconv.Atoi(number)
	if err != nil {
		// handle error
		fmt.Println(err)
		os.Exit(2)
	}

	args := &serverrpc.AddFlightArgs{
		FlightId:  flightId,
		Date:      datetime,
		TicketNum: num,
	}
	// fmt.Println("addflight")
	reply := &serverrpc.AddFlightReply{}
	if err := air.client.Call("Server.AddFlight", args, reply); err != nil {
		return err
	}
	// fmt.Println("addflight")
	if reply.Status == serverrpc.Ok {
		return nil
	} else {
		return errors.New("Sorry, add flight error, you add a flight that you already added")
	}
}

//Cancel the flight for the ticket
func (air *airlineadmin) CancelFlight(flightId string, date string) error {
	// fmt.Println("cancelflight")
	// fmt.Println("input date", date)
	datetime, err := time.Parse(shortForm, date) //"2014-05-01"
	if err != nil {
		return err

	}
	// fmt.Println("parse date", datetime)
	args := &serverrpc.CancelFlightArgs{
		FlightId: flightId,
		Date:     datetime,
	}
	reply := &serverrpc.CancelFlightReply{}
	if err := air.client.Call("Server.CancelFlight", args, reply); err != nil {
		return err
	}
	if reply.Status == serverrpc.Ok {
		return nil
	} else {
		return errors.New("Sorry, cancel flight error, you cancel a flight that does not exist")
	}
}

func (air *airlineadmin) EditFlight(flightId string, date string, number string) error {
	// fmt.Println("editflight")
	// fmt.Println("input date", date)
	datetime, err := time.Parse(shortForm, date) //"2014-05-01"
	if err != nil {
		return err

	}
	// fmt.Println("parse date", datetime)
	num, err := strconv.Atoi(number)

	if err != nil {
		// handle error
		fmt.Println(err)
		os.Exit(2)
	}
	args := &serverrpc.EditFlightArgs{
		FlightId:  flightId,
		Date:      datetime,
		TicketNum: num,
	}
	// fmt.Println("editflight")
	reply := &serverrpc.EditFlightReply{}
	if err := air.client.Call("Server.EditFlight", args, reply); err != nil {
		return err
	}
	if reply.Status == serverrpc.Ok {
		return nil
	} else {
		return errors.New("Sorry, edit flight error, you edit a flight that does not exist")
	}
}

func (admin *airlineadmin) Close() error {
	return admin.client.Close()
}

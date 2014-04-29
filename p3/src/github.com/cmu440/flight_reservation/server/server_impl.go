// Implementation of Airline Server
// Each Airline Server binds with a Paxos instance
// Use Paxos to guarantee consistency between different Airline servers
// Author: Chun Chen

package server

import (
	"github.com/cmu440/flight_reservation/paxos"
	"github.com/cmu440/flight_reservation/rpc/paxosrpc"
	"github.com/cmu440/flight_reservation/rpc/serverrpc"
	"net/rpc"
	"sync"
	"time"
)

type flight struct {
	flightId string
	date     time.Time
}

type server struct {
	mutex           sync.Mutex
	currSlotNum     int
	paxosInstance   *paxos.Paxos
	reservedFlights map[string][]serverrpc.FlightInfo
	flightStatus    map[flight]int
}

func NewServer(paxosPeers []string, paxosMe int) (Server, error) {
	var err error
	srv := &server{}

	// start and initialize the paxos instance
	srv.paxosInstance, err = paxos.Make(paxosPeers, paxosMe, nil)
	if err != nil {
		return nil, err
	}

	srv.reservedFlights = make(map[string][]serverrpc.FlightInfo)
	srv.flightStatus = make(map[flight]int)

	// since server and the paxos instance run on the same process, we don't need to create
	// a brand new Http handler. And since the paxos instance is instantiated, we can just
	// reuse its Http handler to register the server
	err = rpc.RegisterName("Server", serverrpc.Wrap(srv))
	for err != nil {
		err = rpc.RegisterName("Server", serverrpc.Wrap(srv))
	}

	return srv, nil
}

func (srv *server) MyReservation(args *serverrpc.MyReservationArgs, reply *serverrpc.MyReservationReply) error {
	paxosOp := paxosrpc.PaxosValue{
		Type:      paxosrpc.MyReservation,
		Timestamp: time.Now(),
		UserName:  args.UserName,
	}
	retVal, err := srv.addPaxosLogAndUpdateState(paxosOp)
	if err != nil {
		return err
	}
	reply.Flights = retVal.([]serverrpc.FlightInfo)
	return nil
}

func (srv *server) CheckTickets(args *serverrpc.CheckTicketsArgs, reply *serverrpc.CheckTicketsReply) error {
	paxosOp := paxosrpc.PaxosValue{
		Type:      paxosrpc.CheckTickets,
		Timestamp: time.Now(),
		FlightId:  args.FlightId,
		StartDate: args.StartDate,
		EndDate:   args.EndDate,
	}
	retVal, err := srv.addPaxosLogAndUpdateState(paxosOp)
	if err != nil {
		return err
	}
	reply.Flights = retVal.([]serverrpc.FlightInfo)
	return nil
}

func (srv *server) ReserveTicket(args *serverrpc.ReserveTicketArgs, reply *serverrpc.ReserveTicketReply) error {
	paxosOp := paxosrpc.PaxosValue{
		Type:      paxosrpc.ReserveTicket,
		Timestamp: time.Now(),
		UserName:  args.UserName,
		FlightId:  args.FlightId,
		StartDate: args.Date,
		EndDate:   args.Date,
	}
	retVal, err := srv.addPaxosLogAndUpdateState(paxosOp)
	if err != nil {
		return err
	}
	success := retVal.(bool)
	if success {
		reply.Status = serverrpc.Ok
	} else {
		reply.Status = serverrpc.Reject
	}
	return nil
}

func (srv *server) AddFlight(args *serverrpc.AddFlightArgs, reply *serverrpc.AddFlightReply) error {
	paxosOp := paxosrpc.PaxosValue{
		Type:      paxosrpc.AddFlight,
		Timestamp: time.Now(),
		FlightId:  args.FlightId,
		StartDate: args.Date,
		EndDate:   args.Date,
		TicketNum: args.TicketNum,
	}
	retVal, err := srv.addPaxosLogAndUpdateState(paxosOp)
	if err != nil {
		return err
	}

	success := retVal.(bool)
	if success {
		reply.Status = serverrpc.Ok
	} else {
		reply.Status = serverrpc.Reject
	}
	return nil
}

func (srv *server) CancelFlight(args *serverrpc.CancelFlightArgs, reply *serverrpc.CancelFlightReply) error {
	paxosOp := paxosrpc.PaxosValue{
		Type:      paxosrpc.CancelFlight,
		Timestamp: time.Now(),
		FlightId:  args.FlightId,
		StartDate: args.Date,
		EndDate:   args.Date,
	}
	retVal, err := srv.addPaxosLogAndUpdateState(paxosOp)
	if err != nil {
		return err
	}

	success := retVal.(bool)
	if success {
		reply.Status = serverrpc.Ok
	} else {
		reply.Status = serverrpc.Reject
	}
	return nil
}

func (srv *server) EditFlight(args *serverrpc.EditFlightArgs, reply *serverrpc.EditFlightReply) error {
	paxosOp := paxosrpc.PaxosValue{
		Type:      paxosrpc.EditFlight,
		Timestamp: time.Now(),
		FlightId:  args.FlightId,
		StartDate: args.Date,
		EndDate:   args.Date,
		TicketNum: args.TicketNum,
	}
	retVal, err := srv.addPaxosLogAndUpdateState(paxosOp)
	if err != nil {
		return err
	}

	success := retVal.(bool)
	if success {
		reply.Status = serverrpc.Ok
	} else {
		reply.Status = serverrpc.Reject
	}
	return nil
}

func (srv *server) Deaf() {
	srv.paxosInstance.Deaf()
}

func (srv *server) Wake() {
	srv.paxosInstance.Wake()
}

// this is the helper function that each RPC function should invoke to catch up its own state
// using Paxos (by redoing 'left behind' logs in Paxos replica system) and after updating it
// self to the latest state, it will write the Paxos operation given into Paxos system and update
// its own state using this operation. the function will return an interface{} containing the
// desired return value for the Paxos operation given, and a non-nil error if there is some anomaly situation
// in the Paxos clusters (like the Paxos admin is trying to add new servers into the current Paxos cluster)
func (srv *server) addPaxosLogAndUpdateState(paxosOp paxosrpc.PaxosValue) (interface{}, error) {
	var err error
	// //fmt.Println("addPaxosLogAndUpdateState")
	srv.mutex.Lock()
	defer srv.mutex.Unlock()

	var decided bool
	var val, retVal interface{}
	var decidedOp paxosrpc.PaxosValue

	for {
		decided, val, err = srv.paxosInstance.Status(srv.currSlotNum)
		if err != nil {
			return nil, err
		}

		// do paxos on current slot until the value is decided (paxos is successful done)
		for !decided {
			// try to do a round of paxos using paxosOp given on current slot
			err = srv.paxosInstance.Start(srv.currSlotNum, paxosOp)
			// if err is not nil, which means the paxos admin is trying to add new paxos server and all paxos
			// replicas are now locked, we should return error directly so client will know it. In this case
			//the paxosOp given should not have been executed
			if err != nil {
				return nil, err
			}

			decided, val, err = srv.paxosInstance.Status(srv.currSlotNum)
			if err != nil {
				return nil, err
			}
		}

		// extract the underlying type
		decidedOp = val.(paxosrpc.PaxosValue)
		srv.currSlotNum++

		switch decidedOp.Type {
		case paxosrpc.MyReservation:
			if paxosOp == decidedOp {
				retVal = srv.myReservationHelper(decidedOp)
			}
		case paxosrpc.CheckTickets:
			if paxosOp == decidedOp {
				retVal = srv.checkTicketsHelper(decidedOp)
			}
		case paxosrpc.ReserveTicket:
			retVal = srv.reserveTicketHelper(decidedOp)
		case paxosrpc.AddFlight:
			retVal = srv.addFlightHelper(decidedOp)
		case paxosrpc.CancelFlight:
			retVal = srv.cancelFlightHelper(decidedOp)
		case paxosrpc.EditFlight:
			retVal = srv.editFlightHelper(decidedOp)
		case paxosrpc.NoOp:
		}

		// if the decided value on current time slot is just the paxosOp given, which means we have catch up to the
		// latest state and just added the paxosOp given into paxos and updated the server's state, return the execute result directly
		if paxosOp == decidedOp {
			return retVal, nil
		}
	}
}

func (srv *server) myReservationHelper(paxosOp paxosrpc.PaxosValue) []serverrpc.FlightInfo {
	userName := paxosOp.UserName
	flights, exist := srv.reservedFlights[userName]
	// if the user has not reserved any tickets yet, retuan an empty slice instead of nil
	if !exist {
		return make([]serverrpc.FlightInfo, 0)
	}

	recheckedFlights := make([]serverrpc.FlightInfo, 0)
	// since some flights may be canceld but still exist in the user account, so we have to double check it here to ensure
	// no canceled flights will be returned back to the client
	for _, fli := range flights {
		queryFlight := flight{flightId: fli.FlightId, date: fli.Date}
		_, exist := srv.flightStatus[queryFlight]

		if exist {
			recheckedFlights = append(recheckedFlights, fli)
		}
	}

	// update the reserved flights in user account
	// srv.reservedFlights[userName] = recheckedFlights
	return recheckedFlights
}

func (srv *server) checkTicketsHelper(paxosOp paxosrpc.PaxosValue) []serverrpc.FlightInfo {
	flightId := paxosOp.FlightId
	startDate := paxosOp.StartDate
	endDate := paxosOp.EndDate

	retFlightsInfo := make([]serverrpc.FlightInfo, 0)

	queryDate := startDate

	for !queryDate.After(endDate) {
		queryFlight := flight{flightId: flightId, date: queryDate}
		// check if the flight with the given flight id and date exists
		ticketNum, exist := srv.flightStatus[queryFlight]

		if exist && ticketNum > 0 {

			queryFlightInfo := serverrpc.FlightInfo{FlightId: flightId, Date: queryDate, TicketNum: ticketNum}
			retFlightsInfo = append(retFlightsInfo, queryFlightInfo)
		}

		queryDate = queryDate.AddDate(0, 0, 1)
	}

	return retFlightsInfo
}

func (srv *server) reserveTicketHelper(paxosOp paxosrpc.PaxosValue) bool {
	userName := paxosOp.UserName
	flightId := paxosOp.FlightId
	date := paxosOp.StartDate

	queryFlight := flight{flightId: flightId, date: date}
	ticketNum, exist := srv.flightStatus[queryFlight]

	if exist && ticketNum > 0 {
		// decrease the number of remaining tickets for the flight
		srv.flightStatus[queryFlight]--
		// add the flight info into the user's account
		addedFlightInfo := serverrpc.FlightInfo{FlightId: flightId, Date: date, TicketNum: 0}
		srv.reservedFlights[userName] = append(srv.reservedFlights[userName], addedFlightInfo)
		return true
	} else {
		return false
	}
}

func (srv *server) addFlightHelper(paxosOp paxosrpc.PaxosValue) bool {
	flightId := paxosOp.FlightId
	date := paxosOp.StartDate
	ticketNum := paxosOp.TicketNum

	addedFlight := flight{flightId: flightId, date: date}
	_, exist := srv.flightStatus[addedFlight]

	if !exist {
		srv.flightStatus[addedFlight] = ticketNum
		return true
	} else {
		return false
	}
}

func (srv *server) cancelFlightHelper(paxosOp paxosrpc.PaxosValue) bool {
	flightId := paxosOp.FlightId
	date := paxosOp.StartDate

	deleteFlight := flight{flightId: flightId, date: date}
	_, exist := srv.flightStatus[deleteFlight]

	if exist {
		delete(srv.flightStatus, deleteFlight)
		return true
	} else {
		return false
	}
}

func (srv *server) editFlightHelper(paxosOp paxosrpc.PaxosValue) bool {
	flightId := paxosOp.FlightId
	date := paxosOp.StartDate
	ticketNum := paxosOp.TicketNum

	editFlight := flight{flightId: flightId, date: date}
	_, exist := srv.flightStatus[editFlight]

	if exist {
		srv.flightStatus[editFlight] = ticketNum
		return true
	} else {
		return false
	}
}

// Runner of Paxos Admin

package main

import (
	"flag"
	"fmt"
	"github.com/cmu440/flight_reservation/paxosadmin"
	"github.com/cmu440/flight_reservation/rpc/paxosrpc"
	"log"
	"os"
	"strconv"
	"strings"
)

const shortForm = "2006.01.02"

var peers = flag.String("peers", "localhost:9000,localhost:9001,localhost:9002", "peers hostports")
var peerme = flag.String("me", "localhost:9000", "the hostport of paxos peer to be added/removed")
var sleep = flag.Int("sleep", 0, "specify number of seconds that Paxos Admin should sleep after setting all Paxos nodes to quiesce mode when adding/removing a Paxos node, this is for test purpose only")

type cmdInfo struct {
	cmdline  string
	funcname string
	nargs    int
}

func init() {
	log.SetFlags(log.Lshortfile | log.Lmicroseconds)
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "The padrunner program is a testing tool that that creates and runs an Paxos Admin")
		fmt.Fprintln(os.Stderr, "instance. You may use it to test the correctness of your Paxos Admin.\n")
		fmt.Fprintln(os.Stderr, "Usage: \n (Date formate: \"2006-01-02\")")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Possible commands:")
		fmt.Fprintln(os.Stderr, "  AddPeer:    	-peers=[hostports] 	-me=[hostport]	-sleep=[int(not required)]	add")
		fmt.Fprintln(os.Stderr, "  Remove:		-peers=[hostports] 	-me=[hostport]	-sleep=[int(not required)]	remove")
		fmt.Fprintln(os.Stderr, "  ShowPeers:  	-peers=[hostports]		show")
		fmt.Fprintln(os.Stderr, "  ShowSlot:   	-peers=[hostports]		showslot")
		fmt.Fprintln(os.Stderr, "  Change:   	-peers=[hostports]	-me=[hostport]	change")

	}
}

func main() {
	flag.Parse()
	fmt.Println("padmin begin connect to hostport:", *peers)

	peers := strings.Split(*peers, ",")

	me := *peerme
	cmd := flag.Arg(0)

	cmdlist := []cmdInfo{
		{"add", "paxosadmin.AddPeer", 1},
		{"show", "paxosadmin.ShowPeers", 1},
		{"showslot", "paxosadmin.ShowSlot", 1},
		{"remove", "paxosadmin.Remove", 1},
		{"change", "paxosadmin.ChangePeer", 1},
	}

	cmdmap := make(map[string]cmdInfo)
	for _, j := range cmdlist {
		cmdmap[j.cmdline] = j
	}

	ci, found := cmdmap[cmd]
	if !found {
		fmt.Println("error not found commmand")
		flag.Usage()
		os.Exit(1)
	}
	if flag.NArg() < (ci.nargs) {
		fmt.Println("error lose argument")
		flag.Usage()
		os.Exit(1)
	}

	switch cmd {
	case "change": // change peer
		err := paxosadmin.ChangePeer(peers, me)
		// printStatus(ci.funcname, status, err)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Successfully add a new peer to the old peers")
		}
	case "show": // user create
		goodpeers, badpeers, err := paxosadmin.ShowPeers(peers)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("live peers that are still working:")
			for key, value := range goodpeers {
				fmt.Println(key, value)
			}
			fmt.Println("bad peers that we cannot connect to:", badpeers)

			// fmt.Println("ok you already finish:", flag.Arg(0))
		}
		// printStatus(ci.funcname, status, err)
	case "add": // subscription list
		err := paxosadmin.AddPeer(peers, me, *sleep)
		// printStatus(ci.funcname, status, err)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Successfully add a new peer to the old peers")
		}
	case "remove": //remove  the server
		err := paxosadmin.RemovePeer(peers, me, *sleep)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Successfully remove a peer from the old peers")
		}
	case "showslot": //showslot on the server
		allSlotValue, err := paxosadmin.ShowSlot(peers)
		if err != nil {
			fmt.Println(err)
		} else {
			// following is to print the slot infomation
			for peer, slotvalues := range allSlotValue {
				fmt.Println("machine:", peer)
				for i := 0; i < len(slotvalues); i++ {
					fmt.Println("slot number:", i, ";slot decided:", slotvalues[i].IsDecided)
					if slotvalues[i].IsDecided == true {
						paxosValue := slotvalues[i].Value.(paxosrpc.PaxosValue)
						var opstring string
						switch paxosValue.Type {
						case paxosrpc.MyReservation:
							opstring = "MyReservation:  username: " + paxosValue.UserName
						case paxosrpc.CheckTickets:
							opstring = "CheckTickets: flightId:" + paxosValue.FlightId
							opstring += " startDate:" + paxosValue.StartDate.Format(shortForm)
							opstring += " endDate:" + paxosValue.EndDate.Format(shortForm)
						case paxosrpc.ReserveTicket:
							opstring = "ReserveTicket: username:" + paxosValue.UserName
							opstring += " flightId:" + paxosValue.FlightId
							opstring += " date:" + paxosValue.StartDate.Format(shortForm)
						case paxosrpc.AddFlight:
							opstring = "AddFlight:  flightId:" + paxosValue.FlightId
							opstring += " date:" + paxosValue.StartDate.Format(shortForm)
							opstring += " ticketnumber:" + strconv.Itoa(paxosValue.TicketNum)
						case paxosrpc.CancelFlight:
							opstring = "CancelFlight:  flightId:" + paxosValue.FlightId
							opstring += " date:" + paxosValue.StartDate.Format(shortForm)
						case paxosrpc.EditFlight:
							opstring = "EditFlight:  flightId:" + paxosValue.FlightId
							opstring += " date:" + paxosValue.StartDate.Format(shortForm)
							opstring += " ticketnumber:" + strconv.Itoa(paxosValue.TicketNum)
						}

						fmt.Println("  ", opstring)
					}
				}
				fmt.Println("=======================================================")
			}
			// fmt.Println("ok you already finish:", flag.Arg(0))
		}
	}
}

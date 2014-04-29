// Runner of client

package main

import (
	"flag"
	"fmt"
	"github.com/cmu440/flight_reservation/client"
	"github.com/cmu440/flight_reservation/rpc/serverrpc"
	"log"
	"os"
	"strconv"
)

const shortForm = "2006.1.2"

var serverHostPort = flag.String("server", "localhost:9000", "hostport of the server to connect to")

type cmdInfo struct {
	cmdline  string
	funcname string
	nargs    int
}

func init() {
	log.SetFlags(log.Lshortfile | log.Lmicroseconds)
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "The crunner program is a testing tool that that creates and runs an instance")
		fmt.Fprintln(os.Stderr, "of the Client. You may use it to test the correctness of your Client.\n")
		fmt.Fprintln(os.Stderr, "Usage: Date formate: \"2006.1.2\"")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Possible commands:")
		fmt.Fprintln(os.Stderr, "  MyReservation:	-server=[hostport] myreservation username")
		fmt.Fprintln(os.Stderr, "  CheckTickets:	-server=[hostport] check flightId startDate endDate")
		fmt.Fprintln(os.Stderr, "  ReserveTicket:	-server=[hostport] reserve username flightId date")
	}
}

func main() {

	flag.Parse()

	cmd := flag.Arg(0)
	client, err := client.NewClient(*serverHostPort)
	if err != nil {
		log.Fatalln("Failed to create Client:", err)
	}

	cmdlist := []cmdInfo{
		{"myreservation", "Server.MyReservation", 1},
		{"check", "Server.CheckTickets", 3},
		{"reserve", "Server.ReserveTicket", 3},
	}

	cmdmap := make(map[string]cmdInfo)
	for _, j := range cmdlist {
		cmdmap[j.cmdline] = j
	}

	ci, found := cmdmap[cmd]
	if !found {
		flag.Usage()
		os.Exit(1)
	}
	if flag.NArg() < (ci.nargs) {
		flag.Usage()
		os.Exit(1)
	}

	switch cmd {
	case "myreservation": // check the user's reservation records
		flightInfo, err := client.MyReservation(flag.Arg(1))
		if err != nil {
			fmt.Println(err)
		} else {
			printflightInfo(flightInfo, "myreservation")
		}
	case "check": // check flights availability
		flightInfo, err := client.CheckTickets(flag.Arg(1), flag.Arg(2), flag.Arg(3))
		if err != nil {
			fmt.Println(err)
		} else {
			printflightInfo(flightInfo, "check")
		}
	case "reserve": // reserve the flight for the user
		err := client.ReserveTicket(flag.Arg(1), flag.Arg(2), flag.Arg(3))
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Successfully reserved one ticket of flight " + flag.Arg(2) + " on " + flag.Arg(3) + ", for " + flag.Arg(1))
		}
	}
}

// print the air line information
func printflightInfo(flightsInfo []serverrpc.FlightInfo, op string) {
	if op == "myreservation" {
		if len(flightsInfo) == 0 {
			fmt.Println("Sorry user " + flag.Arg(1) + " has not booked any flight tickets yet.")
		} else {
			fmt.Println(flag.Arg(1) + "'s reserved tickets:")
			for _, flight := range flightsInfo {
				fmt.Println("flight id: " + flight.FlightId + "\t" + "date: " + flight.Date.Format(shortForm))
			}
		}
	} else {
		if len(flightsInfo) == 0 {
			fmt.Println("Sorry there is no available flights between " + flag.Arg(2) + " and " + flag.Arg(3))
		} else {
			fmt.Println("Flights that are available between " + flag.Arg(2) + " and " + flag.Arg(3))
			for _, flight := range flightsInfo {
				fmt.Println("flight id: " + flight.FlightId + "\t" + "date: " + flight.Date.Format(shortForm) + "\t" + "tickets remaining: " + strconv.Itoa(flight.TicketNum))
			}
		}
	}
}

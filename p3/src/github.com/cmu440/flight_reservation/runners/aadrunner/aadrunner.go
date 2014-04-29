// Runner of Airline Admin client

package main

import (
	"flag"
	"fmt"
	"github.com/cmu440/flight_reservation/airlineadmin"
	"log"
	"os"
)

var peer = flag.String("peer", "localhost:9000", "airlineadmin hostport")

type cmdInfo struct {
	cmdline  string
	funcname string
	nargs    int
}

func init() {
	log.SetFlags(log.Lshortfile | log.Lmicroseconds)
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "The aadrunner program is a testing tool that that creates and runs an instance")
		fmt.Fprintln(os.Stderr, "of the Airline Admin. You may use it to test the correctness of your Airline Admin.\n")
		fmt.Fprintln(os.Stderr, "Usage: \n (Date formate: \"2006-01-02\")")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Possible commands:")
		fmt.Fprintln(os.Stderr, "  AddFlight:		-peer=[hostport]  	add flightId date number")
		fmt.Fprintln(os.Stderr, "  CancelFlight: 	-peer=[hostport]	cancel flightId date")
		fmt.Fprintln(os.Stderr, "  EditFlight:   	-peer=[hostport]	edit flightId date number")
	}
}

func main() {
	flag.Parse()
	fmt.Println("Airlineadmin begin connect to hostport:", *peer)

	// fmt.Println(flag.Args())
	cmd := flag.Arg(0)
	client, err := airlineadmin.NewAirlineadmin(*peer)
	if err != nil {
		log.Fatalln("Failed to create Client:", err)
	}

	cmdlist := []cmdInfo{
		{"add", "Server.AddFlight", 3},
		{"cancel", "Server.CancelFlight", 2},
		{"edit", "Server.EditFlight", 3},
	}

	cmdmap := make(map[string]cmdInfo)
	for _, j := range cmdlist {
		cmdmap[j.cmdline] = j
	}

	ci, found := cmdmap[cmd]
	if !found {
		fmt.Println("error bad command")
		flag.Usage()
		os.Exit(1)
	}
	if flag.NArg() < (ci.nargs) {
		// fmt.Println(flag.NArg(), ci.nargs)
		fmt.Println("error lose argument")
		flag.Usage()
		os.Exit(1)
	}

	switch cmd {
	case "add": // user create
		err := client.AddFlight(flag.Arg(1), flag.Arg(2), flag.Arg(3))
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Successfully added", flag.Arg(3), "tickets for", " flight ", flag.Arg(1), "on ", flag.Arg(2))
		}
		// printStatus(ci.funcname, status, err)
	case "cancel": // subscription list
		err := client.CancelFlight(flag.Arg(1), flag.Arg(2))
		// printStatus(ci.funcname, status, err)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Successfully canceled", " flight ", flag.Arg(1), "on ", flag.Arg(2))
		}
	case "edit": // edit the flight list
		err := client.EditFlight(flag.Arg(1), flag.Arg(2), flag.Arg(3))
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Successfully edited", flag.Arg(3), "tickets for", " flight ", flag.Arg(1), "on ", flag.Arg(2))
		}
	}
}

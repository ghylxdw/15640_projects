// Runner of Airline Server

package main

import (
	"flag"
	"fmt"
	"github.com/cmu440/flight_reservation/server"
	"log"
	"os"
	"strings"
)

var (
	mode  = flag.Int("mode", 1, "mode of srunner, where 0 means running on foreground and supports further input command like 'deaf' and 'wake', and 1 means running in background")
	peers = flag.String("peers", "localhost:9000", "hostports of all peers in the Paxos cluster")
	me    = flag.Int("me", 0, "the index of current server in peers, which indicates the hostport of current server")
)

func init() {
	log.SetFlags(log.Lshortfile | log.Lmicroseconds)
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "The srunner program is a testing tool that that creates and runs an instance of Server ")
		fmt.Fprintln(os.Stderr, "and a Paxos instance inside it. You may use it to test the correctness of your Server.\n")
		fmt.Fprintln(os.Stderr, "You can type command \"deaf\" or \"wake\" to turn the server to deaf or awake mode\n")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Possible commands:")
		fmt.Fprintln(os.Stderr, "  CreateAndRunServer:	-peers=[hostports] 	-me=[int]")
	}
}

func main() {
	flag.Parse()
	fmt.Println("server is started")
	if *mode != 0 && *mode != 1 {
		log.Fatalln("mode invalid")
	}
	peersList := strings.Split(*peers, ",")

	// if the hostport of the current server is not in the peer list
	if *me < 0 || *me >= len(peersList) {
		log.Fatalln("\"me\" provided invalid")
	}

	// Create and start the server.
	server, err := server.NewServer(peersList, *me)
	if err != nil {
		log.Fatalln("Failed to create server:", err)
	}

	if *mode == 0 {
		fmt.Println("Please type 'deaf' or 'wake' to turn the paxos instance of the server to deaf or awake mode")
		// run the server forever and read input command
		var s string
		for {
			_, err = fmt.Scanln(&s)
			switch s {
			case "deaf":
				server.Deaf()
				fmt.Println("paxos instance is deaf")
			case "wake":
				server.Wake()
				fmt.Println("paxos instance is awaked")
			default:
				fmt.Println("command invalid")
			}
		}
	} else {
		select {}
	}

}

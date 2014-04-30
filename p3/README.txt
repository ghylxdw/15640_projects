README for Flight Reservation System on Paxos Project


HOW TO RUN THE PROGRAM:

We have written different runners for Client, Airline Server, Airline Admin and Paxos Admin, you can use these four runners to start every component of this project.
So before running these runners, you must first install them using the commands below: (assume your GOPATH is set correctly)

go install github.com/cmu440/flight_reservation/runners/aadrunner/
go install github.com/cmu440/flight_reservation/runners/padrunner/
go install github.com/cmu440/flight_reservation/runners/srunner/
go install github.com/cmu440/flight_reservation/runners/crunner/


USAGE OF RUNNERS: (assume all compiled runners are in the bin directory)

CLIENT:
1. Check reserved tickets for a specified user:

	./crunner -server=[hostport] myreservation username
	Example:  ./crunner -server=localhost:9000 myreservation chunchen

2. Check available tickets for a flight in a specified date range

	./crunner -server=[hostport] check flightId startDate endDate 
	Example:  ./crunner -server=localhost:9000 check AA100 2010.01.01 2010.01.05

3. Reserve a flight ticket given the flight id, date and username

	./crunner -server=[hostport] reserve username flightId date
	Example:  ./crunner -server=localhost:9000 reserve chunchen AA100 2010.01.01


AIRLINE SERVER RUNNER:
1. Start a airline server (-peers refers to the view of the whole Paxos cluster, and peers[me] refers to the hostport of current started server)

	./srunner -peers=[hostports] -me=[int]
	Example:  ./srunner -peers=localhost:9000,localhost:9001,localhost:9002 -me=0
	

AIRLINE ADMIN:
1. Add a flight given flight id, date and number of tickets remaining (-peer refers to the hostport of the airline server you want the airline admin to connect to)

	./aadrunner -peer=[hostport]  add flightId date number
	Example:  ./aadrunner -peer=localhost:9000 add AA100 2010.01.01 100
	
2. Edit the remaining ticket of an existing flight given the flight id, date and number of ticket to be edited to

	./aadrunner -peer=[hostport] edit flightId date number
	Example:  ./aadrunner -peer=localhost:9000 edit AA100 2010.01.01 1

3. Cancel an existing flight, given the flight id and date

	./aadrunner -peer=[hostport] cancel flightId date
	Example:  ./aadrunner -peer=localhost:9000 cancel AA100 2010.01.01


PAXOS ADMIN:
1. Show the working status of all Paxos peers in the cluster, along with its view of the Paxos cluster (you should specify the hostports of all Paxos peers manually, for typical workflow, you should first use this function to check the status of all Paxos peers in the cluster before adding or removing a Paxos node)
	
	./padrunner -peers=[hostports] show
	Example:  ./padrunner -peers=localhost:9000,localhost:9001,localhost:9002 show

2. Show values of all slots inside all Paxos peers in the cluster (this function is typically used for white-box test)

	./padrunner -peers=[hostports] showslot
	Example:  ./padrunner -peers=localhost:9000,localhost:9001,localhost:9002 showslot

3. Add a new Paxos node into the current Paxos cluster (for typical workflow of adding a new Paxos node, you should first use 'srunner' to start a new airline server and then use 'padrunner' to add it into the current Paxos cluster, here '-peers' refers to the hostpors of Paxos peers in the original cluster, the servers included in '-peers' should all be in working condition, otherwise you should first remove dead node from the cluster and then add new one, and '-me' refers to the hostport of the Paxos node to be added)

	./padrunner -peers=[hostports] -me=[hostport] add
	Example:  ./padrunner -peers=localhost:9000,localhost:9001,localhost:9002 -me=localhost:9003 add

4. Remove a Paxos node from the current Paxos cluster ('-peers' refers to the hostports of all peers in current Paxos cluster, and '-me' refers to the hostport of the Paxos node to be removed from the current Paxos cluster)
	
	./padrunner -peers=[hostports] -me=[hostport] remove
	Example: ./padrunner -peers=localhost:9000,localhost:9001,localhost:9002 -me=localhost:9000 remove



HOW TO RUN THE TESTS

We have written several shell-script tests which test most aspects of both Paxos and airline reservation system, all test scripts are located in the /tests directory.
Our test scripts don't support giving PASS/FAIL automatically, and we must manually determine whether the output results are the same as what we expect.

Our tests contains multiple aspects as below:

Basic test: 				test basic functionality of flight reservation on fixed-size Paxos
Fail tolerance test: 		test if system continues working after up to f nodes fail
Fail recover test: 			test if node continues working after fail-then-recover
Race condition test: 		test if system works when multiple clients try to make reservation on the same flight which has only 1 ticket remaining.
Add node test: 				test whether system is in the right status after adding a node manually (use both black-box and white-box test)
Remove node test: 			same as above.
Quiesce mode test: 			test if the system can be interrupted when it is adding/removing node
Stress test 1: 				This is to test that we run the paxos for many operations and it is still works 
Stress test 2(node fail): 	This is to test that we kill some server and the remaining server is still working fine

You can run the test scrips in the /tests directory directly, for example: ./basictest.sh

Note: please don't stop the test before finishing, since it may cause some servers not killed and still running in background after you stop the test.




 
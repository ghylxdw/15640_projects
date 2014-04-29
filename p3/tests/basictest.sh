go install github.com/cmu440/flight_reservation/runners/aadrunner/
go install github.com/cmu440/flight_reservation/runners/padrunner/
go install github.com/cmu440/flight_reservation/runners/srunner/
go install github.com/cmu440/flight_reservation/runners/crunner/

echo "--------Test basic functionality of servers on fixed-size Paxos (no Fail, no Fail-Recover)---------"

echo "STARTING 3 SERVERS"

SERVER_ID=('9000' '9001' '9002')

for i in `seq 0 2`
    do
        ../bin/srunner -mode=1 -peers=localhost:${SERVER_ID[0]},localhost:${SERVER_ID[1]},localhost:${SERVER_ID[2]} -me=$i &
        SERVER_PID[$i]=$!
    done

echo "--------ADD 3 FLIGHTS ON SERVER 1--------"
for i in `seq 1 3`
    do
    	echo "SERVER 1:  add flight AA100 on 2010.01.0$i with 100 tickets"
        ../bin/aadrunner -peer=localhost:${SERVER_ID[0]} add AA100 2010.01.0$i  100
    done

echo ""
echo ""
echo "--------TEST CHECK TICKETS ON DIFFERENT SERVERS--------"
echo "SERVER 1:  check available tickets for AA100 between 2010.01.01 and 2010.01.03"
../bin/crunner -server=localhost:${SERVER_ID[0]} check AA100 2010.01.01 2010.01.03
echo ""
echo "SERVER 2:  check available tickets for AA100 between 2010.01.01 and 2010.01.03"
../bin/crunner -server=localhost:${SERVER_ID[1]} check AA100 2010.01.01 2010.01.03
echo ""
echo "SERVER 3:  check available tickets for AA100 between 2010.01.01 and 2010.01.03"
../bin/crunner -server=localhost:${SERVER_ID[2]} check AA100 2010.01.01 2010.01.03
echo ""
echo ""
echo ""

echo "--------TEST CHECK TICKETS ON DATE RANGE WITH NO AVAILABLE TICKETS--------"
echo "SERVER 1:  check available tickets for AA100 between 2010.02.20 and 2010.03.20"
../bin/crunner -server=localhost:${SERVER_ID[0]} check AA100 2010.02.20 2010.03.20
echo ""
echo ""
echo ""

echo "--------TEST CHECK TICKETS ON UNAVAILABLE FLIGHT--------"
echo "SERVER 2:  check available tickets for AA200 between 2010.01.01 and 2010.01.03"
../bin/crunner -server=localhost:${SERVER_ID[1]} check AA200 2010.01.01 2010.01.03
echo ""
echo ""
echo ""

echo "--------TEST CHECK TICKETS AFTER FLIGHT CANCELLATION ON DIFFERENT SERVER--------"
echo "SERVER 3:  cancel flight AA100 on 2010.01.01"
../bin/aadrunner -peer=localhost:${SERVER_ID[2]} cancel AA100 2010.01.01
echo ""
echo "SERVER 2:  check available tickets for AA100 between 2010.01.01 and 2010.01.03"
../bin/crunner -server=localhost:${SERVER_ID[1]} check AA100 2010.01.01 2010.01.03
echo ""
echo ""
echo ""

echo "--------TEST CHECK TICKETS AFTER FLIGHT EDITING ON DIFFERENT SERVER--------"
echo "SERVER 3:  edit remaining tickets of flight AA100 on 2010.01.02 to 0"
../bin/aadrunner -peer=localhost:${SERVER_ID[2]} edit AA100 2010.01.02 0
echo ""
echo "SERVER 1:  check available tickets for AA100 between 2010.01.01 and 2010.01.03"
../bin/crunner -server=localhost:${SERVER_ID[0]} check AA100 2010.01.01 2010.01.03
echo ""
echo ""
echo ""


echo "--------TEST RESERVE TICKET ON DIFFERENT SERVERS--------"
echo "SERVER 1:  reserve a ticket of AA100 on 2010.01.03 for user bom"
../bin/crunner  -server=localhost:${SERVER_ID[0]} reserve  bom AA100 2010.01.03 
echo ""
echo "SERVER 2:  reserve a ticket of AA100 on 2010.01.03 for user bom"
../bin/crunner  -server=localhost:${SERVER_ID[1]} reserve  bom AA100 2010.01.03 
echo ""
echo "SERVER 3:  reserve a ticket of AA100 on 2010.01.03 for user bom"
../bin/crunner  -server=localhost:${SERVER_ID[2]} reserve  bom AA100 2010.01.03 
echo ""
echo ""
echo ""

echo "--------TEST RESERVE TICKET ON A FLIGHT WITH NO REMAINING TICKETS--------"
echo "SERVER 3:  edit remaining tickets of flight AA100 on 2010.01.02 to 0"
../bin/aadrunner -peer=localhost:${SERVER_ID[2]} edit AA100 2010.01.02 0
echo ""
echo "SERVER 1:  reserve a ticket of AA100 on 2010.01.02 for user bom"
../bin/crunner  -server=localhost:${SERVER_ID[0]} reserve  bom AA100 2010.01.02
echo ""
echo ""
echo ""

echo "--------TEST RESERVE TICKET ON A NON-EXIST OR CANCELLED FLIGHT--------"
echo "SERVER 1:  reserve a ticket of AA100 on 2010.01.01 for user bom (cancelled)"
../bin/crunner  -server=localhost:${SERVER_ID[0]} reserve  bom AA100 2010.01.01
echo ""
echo "SERVER 2:  reserve a ticket of AA100 on 2010.12.01 for user bom (non-exist)"
../bin/crunner  -server=localhost:${SERVER_ID[0]} reserve  bom AA100 2010.12.01
echo ""
echo ""
echo ""


echo "--------TEST CHECK MY RESERVATION ON DIFFERENT SERVERS--------"
echo "SERVER 1:  check reserved tickets for user bom"
../bin/crunner -server=localhost:${SERVER_ID[0]} myreservation bom
echo ""
echo "SERVER 2:  check reserved tickets for user bom"
../bin/crunner -server=localhost:${SERVER_ID[1]} myreservation bom
echo ""
echo "SERVER 3:  check reserved tickets for user bom"
../bin/crunner -server=localhost:${SERVER_ID[2]} myreservation bom
echo ""
echo ""
echo ""

echo "--------TEST CHECK MY RESERVATION AFTER FLIGHT CANCELLATION--------"
echo "SERVER 2: cancel flight AA100 on 2010.01.03"
../bin/aadrunner -peer=localhost:${SERVER_ID[1]} cancel AA100 2010.01.03
echo ""
echo "SERVER 1:  check reserved tickets for user bom"
../bin/crunner -server=localhost:${SERVER_ID[0]} myreservation bom
echo ""
echo ""
echo ""


echo "--------TEST ADD SAME FLIGHT TWICE ON DIFFERENT SERVERS--------"
echo "SERVER 1:  add flight AA200 on 2010.01.01 with 100 tickets"
../bin/aadrunner -peer=localhost:${SERVER_ID[0]} add AA200 2010.01.01  100
echo ""
echo "SERVER 2:  add flight AA200 on 2010.01.01 with 100 tickets"
../bin/aadrunner -peer=localhost:${SERVER_ID[1]} add AA200 2010.01.01  100
echo ""
echo ""
echo ""

echo "--------TEST CANCEL SAME FLIGHT TWICE ON DIFFERENT SERVERS--------"
echo "SERVER 3:  cancel flight AA200 on 2010.01.01"
../bin/aadrunner -peer=localhost:${SERVER_ID[2]} cancel AA200 2010.01.01
echo ""
echo "SERVER 2:  cancel flight AA200 on 2010.01.01"
../bin/aadrunner -peer=localhost:${SERVER_ID[1]} cancel AA200 2010.01.01


# Kill storage server.
kill -9 ${SERVER_PID[0]}
wait ${SERVER_PID[0]} 2> /dev/null
kill -9 ${SERVER_PID[1]}
wait ${SERVER_PID[1]} 2> /dev/null
kill -9 ${SERVER_PID[2]}
wait ${SERVER_PID[2]} 2> /dev/null




go install github.com/cmu440/flight_reservation/runners/aadrunner/
go install github.com/cmu440/flight_reservation/runners/padrunner/
go install github.com/cmu440/flight_reservation/runners/srunner/
go install github.com/cmu440/flight_reservation/runners/crunner/

echo "--------Test dynamic Paxos (Remove a Paxos node from the cluster)---------"
echo ""
echo "--------STARTING 3 SERVERS--------"
echo ""

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
echo "--------USER RESERVES ONE TICKET--------"
echo "SERVER 1:  user bom reservers one ticket for flight AA100 on 2010.01.01"
../bin/crunner  -server=localhost:${SERVER_ID[0]} reserve  bom AA100 2010.01.01
echo ""

echo "--------PAXOS PEERS' STATUS BEFORE REMOVING A PAXOS NODE--------"
../bin/padrunner -peers=localhost:${SERVER_ID[0]},localhost:${SERVER_ID[1]},localhost:${SERVER_ID[2]} show
echo ""

echo "--------REMOVE SERVER 3 (localhost:9002) FROM PAXOS CLUSTER--------"
../bin/padrunner -peers=localhost:${SERVER_ID[0]},localhost:${SERVER_ID[1]},localhost:${SERVER_ID[2]} -me=localhost:${SERVER_ID[2]} remove
echo ""
echo ""

echo "--------WHITE BOX TEST--------"
echo ""
echo "--------PAXOS PEERS' STATUS AFTER REMOVING ONE PAXOS NODE (locoalhost:9002)--------"
../bin/padrunner -peers=localhost:${SERVER_ID[0]},localhost:${SERVER_ID[1]},localhost:${SERVER_ID[2]} show
echo ""
echo "--------PAXOS PEERS' SLOTS STATUS AFTER REMOVING ONE PAXOS NODE (localhost:9002)--------"
../bin/padrunner -peers=localhost:${SERVER_ID[0]},localhost:${SERVER_ID[1]},localhost:${SERVER_ID[2]} showslot
echo ""
echo ""

echo "--------BLACK BOX TEST--------"
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


kill -9 ${SERVER_PID[0]}
wait ${SERVER_PID[0]} 2> /dev/null
kill -9 ${SERVER_PID[1]}
wait ${SERVER_PID[1]} 2> /dev/null
kill -9 ${SERVER_PID[2]}
wait ${SERVER_PID[2]} 2> /dev/null


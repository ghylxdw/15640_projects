go install github.com/cmu440/flight_reservation/runners/aadrunner/
go install github.com/cmu440/flight_reservation/runners/padrunner/
go install github.com/cmu440/flight_reservation/runners/srunner/
go install github.com/cmu440/flight_reservation/runners/crunner/

echo "--------Test dynamic Paxos (Client sends request to Server when Paxos Admin is adding new Paxos Node to the cluster)---------"
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
echo "--------ADD NEW PAXOS NODE INTO THE ORIGINAL PAXOS CLUSTER (SET ALL PAXOS NODES TO BE QUIESCE FOR 1 SECOND)--------"
echo "starting new server on localhost:9003"
../bin/srunner -mode=1 -peers=localhost:9003 -me=0 &
NEW_SERVER=$!
echo "add new server on localhost:9003 to paxos cluster, set all paxos nodes to be quiesce for 1 second"
../bin/padrunner -peers=localhost:${SERVER_ID[0]},localhost:${SERVER_ID[1]},localhost:${SERVER_ID[2]} -me=localhost:9003 -sleep=1 add &
sleep 0.1
echo ""
echo ""

echo "--------TEST CLIENT MAKE REQUEST WHEN ALL PAXOS NODES ARE IN QUIESCE MODE--------"
echo "SERVER 1:  check available tickets for AA100 between 2010.01.01 and 2010.01.03"
../bin/crunner -server=localhost:${SERVER_ID[0]} check AA100 2010.01.01 2010.01.03
echo ""
echo "SERVER 2:  check available tickets for AA100 between 2010.01.01 and 2010.01.03"
../bin/crunner -server=localhost:${SERVER_ID[1]} check AA100 2010.01.01 2010.01.03
echo ""
echo "SERVER 3:  check available tickets for AA100 between 2010.01.01 and 2010.01.03"
../bin/crunner -server=localhost:${SERVER_ID[2]} check AA100 2010.01.01 2010.01.03
echo ""
echo "SERVER 4:  check available tickets for AA100 between 2010.01.01 and 2010.01.03"
../bin/crunner -server=localhost:9003 check AA100 2010.01.01 2010.01.03
echo ""

echo "WAITING FOR 3 SECONDS TILL THE QUIESCE MODE IS DONE..."
sleep 1
echo ""
echo "--------TEST CLIENT MAKE REQUEST AFTER QUIESCE MODE IS DONE (PAXOS ADMIN FINISHES ADDING NEW PAXOS NODE)--------"
echo "SERVER 1:  check available tickets for AA100 between 2010.01.01 and 2010.01.03"
../bin/crunner -server=localhost:${SERVER_ID[0]} check AA100 2010.01.01 2010.01.03
echo ""
echo "SERVER 2:  check available tickets for AA100 between 2010.01.01 and 2010.01.03"
../bin/crunner -server=localhost:${SERVER_ID[1]} check AA100 2010.01.01 2010.01.03
echo ""
echo "SERVER 3:  check available tickets for AA100 between 2010.01.01 and 2010.01.03"
../bin/crunner -server=localhost:${SERVER_ID[2]} check AA100 2010.01.01 2010.01.03
echo ""
echo "SERVER 4:  check available tickets for AA100 between 2010.01.01 and 2010.01.03"
../bin/crunner -server=localhost:9003 check AA100 2010.01.01 2010.01.03
echo ""

kill -9 ${SERVER_PID[0]}
wait ${SERVER_PID[0]} 2> /dev/null
kill -9 ${SERVER_PID[1]}
wait ${SERVER_PID[1]} 2> /dev/null
kill -9 ${SERVER_PID[2]}
wait ${SERVER_PID[2]} 2> /dev/null
kill -9 ${NEW_SERVER}
wait ${NEW_SERVER} 2> /dev/null
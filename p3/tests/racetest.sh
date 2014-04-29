go install github.com/cmu440/flight_reservation/runners/aadrunner/
go install github.com/cmu440/flight_reservation/runners/padrunner/
go install github.com/cmu440/flight_reservation/runners/srunner/
go install github.com/cmu440/flight_reservation/runners/crunner/

echo "--------Test race functionality of servers on fixed-size Paxos (no Fail, no Fail-Recover)---------"

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
    	echo "SERVER 1:  add flight AA100 on 2010.01.0$i with 1 tickets"
        ../bin/aadrunner -peer=localhost:${SERVER_ID[0]} add AA100 2010.01.0$i  1
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
echo "--------TEST ON CLIENT RACE ON ON DATA, THE SERVER MAINTAIN CONSISTENCY--------"
echo "SERVER 1:  reserve a ticket of AA100 on 2010.01.01 for user bom"  
../bin/crunner  -server=localhost:${SERVER_ID[0]} reserve  bom AA100 2010.01.01 &
echo "SERVER 2:  reserve a ticket of AA100 on 2010.01.01 for user andy"  
../bin/crunner  -server=localhost:${SERVER_ID[1]} reserve  andy AA100 2010.01.01 &
echo "SERVER 3:  reserve a ticket of AA100 on 2010.01.01 for user dog"  
../bin/crunner  -server=localhost:${SERVER_ID[2]} reserve  dog AA100 2010.01.01 &
echo ""
echo "SERVER 1:  reserve a ticket of AA100 on 2010.01.02 for user david" 
../bin/crunner  -server=localhost:${SERVER_ID[0]} reserve  david AA100 2010.01.02 &
echo "SERVER 2:  reserve a ticket of AA100 on 2010.01.02 for user joy" 
../bin/crunner  -server=localhost:${SERVER_ID[1]} reserve  joy AA100 2010.01.02 &
echo "SERVER 3:  reserve a ticket of AA100 on 2010.01.02 for user frank" 
../bin/crunner  -server=localhost:${SERVER_ID[2]} reserve  frank AA100 2010.01.02 &
echo ""
echo ""
echo ""

sleep 1
# Kill storage server.
kill -9 ${SERVER_PID[0]}
wait ${SERVER_PID[0]} 2> /dev/null
kill -9 ${SERVER_PID[1]}
wait ${SERVER_PID[1]} 2> /dev/null
kill -9 ${SERVER_PID[2]}
wait ${SERVER_PID[2]} 2> /dev/null




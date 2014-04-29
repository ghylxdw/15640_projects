go install github.com/cmu440/flight_reservation/runners/aadrunner/
go install github.com/cmu440/flight_reservation/runners/padrunner/
go install github.com/cmu440/flight_reservation/runners/srunner/
go install github.com/cmu440/flight_reservation/runners/crunner/

echo "--------Test fail-tolerancetest functionality of servers on fixed-size Paxos (has Fail, no Fail-Recover)---------"

echo "STARTING 3 SERVERS"

SERVER_ID=('9000' '9001' '9002')

for i in `seq 0 2`
    do
        ../bin/srunner -mode=1 -peers=localhost:${SERVER_ID[0]},localhost:${SERVER_ID[1]},localhost:${SERVER_ID[2]} -me=$i &
        SERVER_PID[$i]=$!
    done
echo "--------TEST KILL A SERVER TO SHOW FAIL_TOLERANCE--------"
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
echo "--------TEST SOME SERVER IS KILLED, THE REMAINING SERVER IS STILL WORKING --------"
echo "SERVER 1: it is killed"
kill -9 ${SERVER_PID[0]}
wait ${SERVER_PID[0]} 2> /dev/null
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
echo "--------TEST THE REMAINING SERVER IS WORKING CORRECTLY--------"
echo "SERVER 3:  cancel flight AA100 on 2010.01.01"
../bin/aadrunner -peer=localhost:${SERVER_ID[2]} cancel AA100 2010.01.01
echo ""
echo "SERVER 2:  check available tickets for AA100 between 2010.01.01 and 2010.01.03"
../bin/crunner -server=localhost:${SERVER_ID[1]} check AA100 2010.01.01 2010.01.03
echo ""
echo "SERVER 2:  edit remaining tickets of flight AA100 on 2010.01.02 to 0"
../bin/aadrunner -peer=localhost:${SERVER_ID[1]} edit AA100 2010.01.02 0
echo ""
echo "SERVER 3:  check available tickets for AA100 between 2010.01.01 and 2010.01.03"
../bin/crunner -server=localhost:${SERVER_ID[2]} check AA100 2010.01.01 2010.01.03
echo ""
echo ""
echo ""



kill -9 ${SERVER_PID[1]}
wait ${SERVER_PID[1]} 2> /dev/null
kill -9 ${SERVER_PID[2]}
wait ${SERVER_PID[2]} 2> /dev/null




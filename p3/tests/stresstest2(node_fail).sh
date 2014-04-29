go install github.com/cmu440/flight_reservation/runners/aadrunner/
go install github.com/cmu440/flight_reservation/runners/padrunner/
go install github.com/cmu440/flight_reservation/runners/srunner/
go install github.com/cmu440/flight_reservation/runners/crunner/

echo "--------Test stresskill functionality of servers on fixed-size Paxos (has Fail, no Fail-Recover)---------"

echo "STARTING 3 SERVERS"
begin=2000
end=2300
SERVER_ID=('9000' '9001' '9002')
for i in `seq 0 2`
    do
        ../bin/srunner -mode=1 -peers=localhost:${SERVER_ID[0]},localhost:${SERVER_ID[1]},localhost:${SERVER_ID[2]} -me=$i &
        SERVER_PID[$i]=$!
    done

echo "ADD 3 FLIGHTS ON SERVER 1:"


echo "--------TEST Add MANY TICKETS --------"
for j in `seq $begin 3 $end`
    do

    	for i in `seq 0 2`
	    	do
		    	year=`expr $i + $j`
		    	day=`expr $i + 1`
		    	echo "SERVER $day:  add flight AA100 on $year.01.$day with 2 tickets"
		    	../bin/aadrunner -peer=localhost:${SERVER_ID[i]} add AA100 $year.01.$day 2 
	    	done
    done


echo "--------TEST  KILL A SERVER --------"
echo "SERVER 1: it is killed"
kill -9 ${SERVER_PID[0]}
wait ${SERVER_PID[0]} 2> /dev/null
sleep 2

echo "--------TEST EDIT 100 TICKETS --------"
for j in `seq $begin 3 $end`
    do

    	for i in `seq 1 2`
	    	do
		    	year=`expr $i + $j`
		    	day=`expr $i + 1`
		    	echo "SERVER $day:  add flight AA100 on $year.01.$day with 2 tickets"
		    	../bin/aadrunner -peer=localhost:${SERVER_ID[i]} edit AA100 $year.01.$day 1 
	    	done
	    year=`expr $j`
	    ../bin/aadrunner -peer=localhost:${SERVER_ID[1]} edit AA100 $year.01.1 1 
	    	
    done

echo "--------TEST CHECK TICKETS ON DIFFERENT SERVERS--------"
# echo "SERVER 1:  check available tickets for AA100 between  $begin.01.01 $end.01.03"
# ../bin/crunner -server=localhost:${SERVER_ID[0]} check AA100 $begin.01.01 $end.01.03
echo ""
echo "SERVER 2:  check available tickets for AA100 between $begin.01.02 and 2010.01.03"
../bin/crunner -server=localhost:${SERVER_ID[1]} check AA100  $begin.01.01 $end.01.03
echo ""
echo "SERVER 3:  check available tickets for AA100 between  $begin.01.01 $end.01.03"
../bin/crunner -server=localhost:${SERVER_ID[2]} check AA100  $begin.01.01 $end.01.03




# kill -9 ${SERVER_PID[0]}
# wait ${SERVER_PID[0]} 2> /dev/null
kill -9 ${SERVER_PID[1]}
wait ${SERVER_PID[1]} 2> /dev/null
kill -9 ${SERVER_PID[2]}
wait ${SERVER_PID[2]} 2> /dev/null




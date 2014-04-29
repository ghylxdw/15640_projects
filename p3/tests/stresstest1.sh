go install github.com/cmu440/flight_reservation/runners/aadrunner/
go install github.com/cmu440/flight_reservation/runners/padrunner/
go install github.com/cmu440/flight_reservation/runners/srunner/
go install github.com/cmu440/flight_reservation/runners/crunner/

echo "--------Test stress functionality of servers on fixed-size Paxos (no Fail, no Fail-Recover)---------"

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
# for i in `seq 1 3`
#     do
#     	echo "SERVER 1:  add flight AA100 on 1010.01.0$i with 100 tickets"
#         ../bin/aadrunner -peer=localhost:${SERVER_ID[0]} add AA100 2010.01.0$i  100
#     done


echo "--------TEST Add MANY TICKETS --------"
for j in `seq $begin 3 $end`
    do

    	for i in `seq 0 2`
	    	do
		    	year=`expr $i + $j`
		    	day=`expr $i + 1`
		    	echo "SERVER $day:  add flight AA100 on $year.01.$day with 2 tickets"
		    	../bin/aadrunner -peer=localhost:${SERVER_ID[i]} add AA100 $year.01.$day 2 &
	    	done
    done

echo "--------TEST EDIT 100 TICKETS --------"
for j in `seq $begin 3 $end`
    do

    	for i in `seq 0 2`
	    	do
		    	year=`expr $i + $j`
		    	day=`expr $i + 1`
		    	echo "SERVER $day:  add flight AA100 on $year.01.$day with 2 tickets"
		    	../bin/aadrunner -peer=localhost:${SERVER_ID[i]} edit AA100 $year.01.$day 1 &
	    	done
    done


echo "--------TEST CHECK TICKETS ON DIFFERENT SERVERS--------"
echo "SERVER 1:  check available tickets for AA100 between 2010.01.01 and 2010.01.03"
../bin/crunner -server=localhost:${SERVER_ID[0]} check AA100 2000.01.01 2010.01.03
echo ""
echo "SERVER 2:  check available tickets for AA100 between 2010.01.01 and 2010.01.03"
../bin/crunner -server=localhost:${SERVER_ID[1]} check AA100 2000.01.01 2010.01.03
echo ""
echo "SERVER 3:  check available tickets for AA100 between 2010.01.01 and 2010.01.03"
../bin/crunner -server=localhost:${SERVER_ID[2]} check AA100 2000.01.01 2010.01.03




# echo "--------TEST CANCEL 100 TICKETS --------"
# for j in `seq $begin 3 $end`
#     do

#     	for i in `seq 0 2`
# 	    	do
# 		    	year=`expr $i + $j`
# 		    	day=`expr $i + 1`
# 		    	echo "SERVER $day:  add flight AA100 on $year.01.$day with 2 tickets"
# 		    	../bin/aadrunner -peer=localhost:${SERVER_ID[i]} cancel AA100 $year.01.$day
# 	    	done
#     done

# echo "--------TEST CHECK TICKETS ON DIFFERENT SERVERS--------"
# echo "SERVER 1:  check available tickets for AA100 between 2010.01.01 and 2010.01.03"
# ../bin/crunner -server=localhost:${SERVER_ID[0]} check AA100 2000.01.01 2010.01.03
# echo ""
# echo "SERVER 2:  check available tickets for AA100 between 2010.01.01 and 2010.01.03"
# ../bin/crunner -server=localhost:${SERVER_ID[1]} check AA100 2000.01.01 2010.01.03
# echo ""
# echo "SERVER 3:  check available tickets for AA100 between 2010.01.01 and 2010.01.03"
# ../bin/crunner -server=localhost:${SERVER_ID[2]} check AA100 2000.01.01 2010.01.03




kill -9 ${SERVER_PID[0]}
wait ${SERVER_PID[0]} 2> /dev/null
kill -9 ${SERVER_PID[1]}
wait ${SERVER_PID[1]} 2> /dev/null
kill -9 ${SERVER_PID[2]}
wait ${SERVER_PID[2]} 2> /dev/null




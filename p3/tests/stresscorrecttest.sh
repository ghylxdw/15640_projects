go install github.com/cmu440/flight_reservation/runners/aadrunner/
go install github.com/cmu440/flight_reservation/runners/padrunner/
go install github.com/cmu440/flight_reservation/runners/srunner/
go install github.com/cmu440/flight_reservation/runners/crunner/

echo "--------Test stresscorrect functionality of servers on fixed-size Paxos (has Fail, no Fail-Recover)---------"

echo "STARTING 3 SERVERS"
begin=2000
end=2300
ticketnum=300
SERVER_ID=('9000' '9001' '9002')
for i in `seq 0 2`
    do
        ../bin/srunner -mode=1 -peers=localhost:${SERVER_ID[0]},localhost:${SERVER_ID[1]},localhost:${SERVER_ID[2]} -me=$i &
        SERVER_PID[$i]=$!
    done

# echo "ADD 3 FLIGHTS ON SERVER 1:"
# for i in `seq 1 3`
#     do
#     	echo "SERVER 1:  add flight AA100 on 1010.01.0$i with 100 tickets"
#         ../bin/aadrunner -peer=localhost:${SERVER_ID[0]} add AA100 2010.01.0$i  100
#     done


echo "--------TEST ADD $ticketnum TICKETS FOR SERVER 1 --------"
for i in `seq 0 0`
	do
    	day=`expr $i + 1`
    	echo "SERVER $day:  add flight AA100 on 2000.01.$day  with $ticketnum tickets"
    	../bin/aadrunner -peer=localhost:${SERVER_ID[i]} add AA100 2000.01.$day  $ticketnum
	done



echo "--------TEST CORRECT BOOK TICKETS --------"
sleep 1
no=0 
loop=`expr $ticketnum / 3 - 1` 
for j  in `seq 0 $loop`
	do
		for i in `seq 0 2`
			do
				echo "--------BOOK A TICKET --------"
				no=`expr $no + 1`
				user=bom$no
				echo "SERVER $day:  reserve a ticket of AA100 on 2010.01.01 for user $user"
				../bin/crunner  -server=localhost:${SERVER_ID[i]} reserve  $user AA100 2000.01.01
				day=`expr $i + 1`
		    	echo "--------CHECK THE REMAINING TICKETS --------"
		    	echo "SERVER $day:  check available tickets for AA100 between 2000.01.01 and 2000.01.01"
				../bin/crunner -server=localhost:${SERVER_ID[i]} check AA100 2000.01.01 2000.01.01
				echo ""
			done
	done





kill -9 ${SERVER_PID[0]}
wait ${SERVER_PID[0]} 2> /dev/null
kill -9 ${SERVER_PID[1]}
wait ${SERVER_PID[1]} 2> /dev/null
kill -9 ${SERVER_PID[2]}
wait ${SERVER_PID[2]} 2> /dev/null




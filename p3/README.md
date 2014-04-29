Flight Reservation System on Paxos
==================

Project 3 of 15-640 Distributed System

Flight Reservation Functionality:
Client:
Check available flights, in a specified date range
Reserve flight ticket, given flight id and date
List one user’s reserved flights, given username

Airline Admin:
Add flight, given flight id, date and number of tickets
Edit flight, given flight id, date and number of tickets
Cancel flight, given flight id and date


Paxos Features:
Failure tolerance
Fail recover
Handles network partition
Support adding/removing Paxos node, checking condition of all Paxos nodes manually by PaxosAdmin (set the cluster to quiesce mode during operation)


Test Structure:
Use shell scripts for testing, including black-box and white-box testing:
Basic test: test basic functionality of flight reservation on fixed-size Paxos
Fail tolerance test: test if system continues working after up to f nodes fail
Fail recover test: test if node continues working after fail-then-recover
Race condition test: test if system works when multiple clients try to make reservation on the same flight which has only 1 ticket remaining.
Add node test: test whether system is in the right status after adding a node manually (use both black-box and white-box test)
Remove node test: same as above.
Quiesce mode test: test if the system can be interrupted when it is adding/removing node


Stress Test Structure:
Use shell scripts for testing, including black-box and white-box testing:
Stress test 1: This is to test that we run the paxos for many operations and it is still works 
Stress test 2(node fail): This is to test that we kill some server and the remaining server is still working fine
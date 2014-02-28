package main

import (
	"container/list"
	"encoding/json"
	"fmt"
	"github.com/cmu440/bitcoin"
	"github.com/cmu440/lsp"
	"log"
	"math"
	"os"
	"strconv"
)

const (
	name                   = "log.txt"
	flag                   = os.O_RDWR | os.O_CREATE
	perm                   = os.FileMode(0666)
	MinerWorkloadThreshold = 10000
)

// unit of work that a miner is responsible for, including the client that this job belongs to
type workUnit struct {
	clientWorkFor int
	workMsg       *bitcoin.Message
}

type clientStatus struct {
	notFinishedJob int
	minHash        uint64
	nonce          uint64
}

var (
	minerCurrWork   map[int]*workUnit
	unOccupiedMiner map[int]bool
	clients         map[int]*clientStatus
	jobQ            *list.List
	lspServer       lsp.Server
	FLOG            *log.Logger
)

// return an initialized ClientStatus for client, given the number of jobs partitioned for the request message
func newClientStatus(partitionedJobNum int) *clientStatus {
	return &clientStatus{
		notFinishedJob: partitionedJobNum,
		minHash:        math.MaxUint64,
		nonce:          0,
	}
}

// marshall and send message to specified connection
func sendMessage(connId int, msg *bitcoin.Message) error {
	buf, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	err = lspServer.Write(connId, buf)
	if err != nil {
		return err
	}
	return nil
}

// partition/schedule request from a client
// this function will evenly partition the calculation to all connected miner if (upper - lower > MinerWorkloadThreshold)
// otherwise it will not partition the calculation but assign it to only one miner
func partitionJob(connId int, msg *bitcoin.Message) {
	data, lower, upper := msg.Data, msg.Lower, msg.Upper

	if upper-lower+1 <= MinerWorkloadThreshold {
		workU := &workUnit{
			clientWorkFor: connId,
			workMsg:       bitcoin.NewRequest(data, lower, upper),
		}
		jobQ.PushBack(workU)
		clients[connId] = newClientStatus(1)
	} else {
		numOfMiner := len(minerCurrWork)
		// if there is no miner at this moment, partition the job as if there is one miner
		if numOfMiner == 0 {
			numOfMiner = 1
		}
		clients[connId] = newClientStatus(numOfMiner)
		interval := (upper - lower + 1) / uint64(numOfMiner)
		// partition the job and push them into the job queue
		for i := 0; i < numOfMiner; i++ {
			currLower := lower + uint64(i)*interval
			currUpeer := currLower + interval - 1
			if i == numOfMiner-1 {
				currUpeer = upper
			}
			workU := &workUnit{
				clientWorkFor: connId,
				workMsg:       bitcoin.NewRequest(data, currLower, currUpeer),
			}
			jobQ.PushBack(workU)
		}
	}
}

// update calculation result of a client's request, when receiving calculation result from a miner.
// will send the final result back to client if all partitioned calculations for the client's request are done
func updateResult(connId int, msg *bitcoin.Message) {
	workU := minerCurrWork[connId]
	// mark the miner as unoccupied
	minerCurrWork[connId] = nil
	unOccupiedMiner[connId] = true
	clientStatus := clients[workU.clientWorkFor]
	// if the client has already lost connection, just ignore its result message
	if clientStatus != nil {
		clientStatus.notFinishedJob -= 1
		// update the calculation result on server
		if msg.Hash < clientStatus.minHash {
			clientStatus.minHash = msg.Hash
			clientStatus.nonce = msg.Nonce
		}
		// if all partitioned jobs of the client's request are done, return the result back to the client
		if clientStatus.notFinishedJob == 0 {
			resultMsg := bitcoin.NewResult(clientStatus.minHash, clientStatus.nonce)
			// if the connection of the client is lost, just ignore the result message
			FLOG.Println("send back result: ", resultMsg.Hash, resultMsg.Nonce)
			sendMessage(workU.clientWorkFor, resultMsg)
			delete(clients, workU.clientWorkFor)
		}
	}
}

// handle income message from client and miner
func handleIncomeMessage(connId int, payLoad []byte) {
	// unmarshall the message
	msg := &bitcoin.Message{}
	err := json.Unmarshal(payLoad, msg)
	if err != nil {
		fmt.Println("cannot unmarshall the message data")
		return
	}

	FLOG.Println("handle income message: ", msg)
	switch msg.Type {
	case bitcoin.Join:
		// register the joined miner
		minerCurrWork[connId] = nil
		unOccupiedMiner[connId] = true
	case bitcoin.Request:
		// if client sends new request while the previous request has not finished, just ignore the new request
		if clients[connId] == nil {
			partitionJob(connId, msg)
		}
	case bitcoin.Result:
		updateResult(connId, msg)
	}
}

// assign pending jobs in job queue to unoccupied miners
func assignJobsToUnoccupiedMiner() {
	if jobQ.Len() > 0 {
		for connId, _ := range unOccupiedMiner {

			// choose the first work unit that works for a active client
			for jobQ.Len() > 0 {
				workU := jobQ.Front().Value.(*workUnit)
				// if the client this work unit works for is active, assign it to unoccupied miner;
				// otherwise remove it from job queue since we don't need to do any computation for a lost client
				if clients[workU.clientWorkFor] != nil {
					err := sendMessage(connId, workU.workMsg)
					// if the job is assigned successfully
					if err == nil {
						jobQ.Remove(jobQ.Front())
						minerCurrWork[connId] = workU
						delete(unOccupiedMiner, connId)
					}
					break
				} else {
					jobQ.Remove(jobQ.Front())
				}
			}
			if jobQ.Len() == 0 {
				break
			}
		}
	}
}

// handle miner/client connection lost
func handleFailure(connId int) {
	// if a miner losts connection
	if currWork, ok := minerCurrWork[connId]; ok {
		// put the work that the lost miner was doing back to the job queue (at front)
		if currWork != nil {
			jobQ.PushFront(currWork)
		}
		// delete this miner from unOccupiedMiner and minerCurrWork
		delete(unOccupiedMiner, connId)
		delete(minerCurrWork, connId)
	} else if clients[connId] != nil {
		// if a client losts connection
		delete(clients, connId)
	}
}

func main() {
	const numArgs = 2
	var port int

	if len(os.Args) != numArgs {
		fmt.Println("Usage: ./server <port>")
		return
	}

	port, err := strconv.Atoi(os.Args[1])

	if err != nil {
		fmt.Println("error parsing port")
		return
	}

	params := lsp.NewParams()
	lspServer, err = lsp.NewServer(port, params)
	if err != nil {
		fmt.Println("Cannot create the server")
		return
	}

	minerCurrWork = make(map[int]*workUnit)
	unOccupiedMiner = make(map[int]bool)
	clients = make(map[int]*clientStatus)
	jobQ = list.New()

	// use logger to save logs into file
	file, err := os.OpenFile(name, flag, perm)
	if err != nil {
		return
	}
	defer file.Close()
	FLOG = log.New(file, "", log.Lshortfile|log.Lmicroseconds)

	for {
		// assign pending jobs from the job queue to unoccupied miners
		assignJobsToUnoccupiedMiner()
		connId, payLoad, err := lspServer.Read()
		// if error occurs when reading, it indicates some connections from client/miner get lost
		if err != nil {
			FLOG.Println("handle failure: ", connId)
			handleFailure(connId)
		} else {
			handleIncomeMessage(connId, payLoad)
		}
	}
}

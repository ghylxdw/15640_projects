package main

import (
	"encoding/json"
	"fmt"
	"github.com/cmu440/bitcoin"
	"github.com/cmu440/lsp"
	"math"
	"os"
)

var lspClient lsp.Client

// marshall and send message to the server
func sendMessage(msg *bitcoin.Message) error {
	buf, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	err = lspClient.Write(buf)
	if err != nil {
		return err
	}
	return nil
}

// calculate the minimum hash number and corresponding nonce in the given range
func calculateMinHash(data string, lower, upper uint64) (minHash, nonce uint64) {
	minHash = math.MaxUint64
	nonce = 0
	for i := lower; i <= upper; i++ {
		hash := bitcoin.Hash(data, i)
		if hash < minHash {
			minHash = hash
			nonce = i
		}
	}
	fmt.Println(minHash, nonce)
	return
}

func main() {
	var err error

	const numArgs = 2
	if len(os.Args) != numArgs {
		fmt.Println("Usage: ./miner <hostport>")
		return
	}

	hostport := os.Args[1]
	fmt.Println(hostport)
	params := lsp.NewParams()
	lspClient, err = lsp.NewClient(hostport, params)
	if err != nil {
		fmt.Println("cannot craete client or connect to the server")
		return
	}

	// send Join message to the server
	joinMsg := bitcoin.NewJoin()
	err = sendMessage(joinMsg)
	if err != nil {
		fmt.Println(err)
		return
	}

	for {
		buf, err := lspClient.Read()
		if err != nil {
			fmt.Println("connection lost when reading from server")
			return
		}

		reqMsg := &bitcoin.Message{}
		err = json.Unmarshal(buf, reqMsg)
		if err != nil {
			fmt.Println("error when unmarshalling message")
			return
		}

		// calculate the minhash and nonce and send the result back to server
		fmt.Println("calculating... ", reqMsg.Data, reqMsg.Lower, reqMsg.Upper)
		hash, nonce := calculateMinHash(reqMsg.Data, reqMsg.Lower, reqMsg.Upper)
		resultMsg := bitcoin.NewResult(hash, nonce)
		fmt.Println("sending back result: ", hash, nonce)
		sendMessage(resultMsg)
	}
}

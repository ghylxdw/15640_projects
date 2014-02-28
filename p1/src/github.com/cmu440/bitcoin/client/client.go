package main

import (
	"encoding/json"
	"fmt"
	"github.com/cmu440/bitcoin"
	"github.com/cmu440/lsp"
	"os"
	"strconv"
)

func main() {
	const numArgs = 4
	if len(os.Args) != numArgs {
		fmt.Println("Usage: ./client <hostport> <message> <maxNonce>")
		return
	}

	hostport := os.Args[1]
	data := os.Args[2]
	maxNonce, err := strconv.ParseUint(os.Args[3], 10, 64)

	if err != nil {
		fmt.Println("error when parsing maxNonce")
		return
	}

	params := lsp.NewParams()
	lspClient, err := lsp.NewClient(hostport, params)
	if err != nil {
		fmt.Println("cannot craete client or connect to the server")
		return
	}

	reqMsg := bitcoin.NewRequest(data, 0, maxNonce)
	buf, err := json.Marshal(reqMsg)
	if err != nil {
		fmt.Println("error when marshalling")
		return
	}

	err = lspClient.Write(buf)
	if err != nil {
		fmt.Println("unable to write to server, connection lost")
		return
	}

	buf, err = lspClient.Read()
	if err != nil {
		printDisconnected()
		return
	}

	resultMsg := &bitcoin.Message{}
	err = json.Unmarshal(buf, resultMsg)
	if err != nil {
		fmt.Println("error when unmarshalling")
		return
	}

	printResult(strconv.FormatUint(resultMsg.Hash, 10), strconv.FormatUint(resultMsg.Nonce, 10))
}

// printResult prints the final result to stdout.
func printResult(hash, nonce string) {
	fmt.Println("Result", hash, nonce)
}

// printDisconnected prints a disconnected message to stdout.
func printDisconnected() {
	fmt.Println("Disconnected")
}

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"blocks"
)

var quitReader = make(chan os.Signal, 1)

var clientType string

func main() {
	pServerEndpoint := flag.String("server", "localhost:80", "The log collector server. You must include also http:// and the port number")
	pClientName := flag.String("name", "NODE", "A custom name to assign to this node")
	flag.Parse()
	clientType = flag.Arg(0)

	serverEndpoint := strings.Join([]string{"http://", *pServerEndpoint}, "")

	fmt.Printf("Starting node with name %s\n", *pClientName)
	time.Sleep(1 * time.Second)
	entryChannel := make(chan *blocks.LogEntry, 10)
	go reader(bufio.NewReader(os.Stdin), entryChannel)
	go sender(serverEndpoint, *pClientName, entryChannel)
	fmt.Printf("Sending logs to %s \n", serverEndpoint)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	signal.Notify(quitReader, os.Interrupt)

	for range quit {
		fmt.Print("Ciao\n")
		close(quit)
		close(entryChannel)
	}
}

func reader(reader *bufio.Reader, entryChannel chan *blocks.LogEntry) {
	var blockSource blocks.BlockSource
	if clientType == "multichain" {
		blockSource = &blocks.MultichainBlockSource{}
	} else if clientType == "geth" {
		blockSource = &blocks.GethBlockSource{}
	}
	blockSource.Init()
	go blockSource.Start(reader, entryChannel)
	<-quitReader
	blockSource.Stop()
}

func sender(url string, clientName string, entryChannel chan *blocks.LogEntry) {
	for logEntry := range entryChannel {
		logEntry.NodeName = clientName
		fmt.Printf("%s %s %d\n", logEntry.Timestamp, logEntry.Verb, logEntry.Block.Height)
		logData, _ := json.Marshal(*logEntry)
		_, err := http.Post(url, "application/json", bytes.NewReader(logData))
		if err != nil {
			entryChannel <- logEntry
			time.Sleep(500000000)
		}
	}
}

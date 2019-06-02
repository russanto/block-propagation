package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"blocks"
)

var maxLogEntryPerWorker uint = 2
var initialBlockAllocation = 300
var nodesToWaitBeforePrint int //TODO convert to uint
var delayCsvFileName string
var logFlag bool
var exePath string

// LogEntry represents an entry in the debug.log
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Verb      string `json:"verb"`
	Block     struct {
		Height uint   `json:"height"`
		Hash   string `json:"hash"`
		NTx    uint   `json:"nTX"`
		Size   uint   `json:"size"`
	} `json:"block"`
	NodeName string `json:"nodeName"`
	fromIP   string
}

type blockDelayCount struct {
	block      *blocks.Block
	delayCount int
}

var computedDelayQueue = make(chan blockDelayCount, 100)

func main() {

	pLogFlag := flag.Bool("log", false, "If set logs all received messages")
	pNodesFlag := flag.Int("nodes", 1, "Nodes to wait before calculating delay")
	pCsvFileName := flag.String("csv", "delay.csv", "CSV file to write results to")
	flag.Parse()
	logFlag = *pLogFlag

	nodesToWaitBeforePrint = *pNodesFlag
	delayCsvFileName = *pCsvFileName

	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exePath = filepath.Dir(ex)

	logEntryProcessQueue := make(chan *LogEntry, 100)
	defer close(logEntryProcessQueue)

	logEndpointHandler := func(w http.ResponseWriter, req *http.Request) {
		body, err := ioutil.ReadAll(req.Body)
		defer req.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		var logEntry LogEntry
		err = json.Unmarshal(body, &logEntry)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		logEntry.fromIP = req.RemoteAddr
		logEntryProcessQueue <- &logEntry
	}

	hub := newHub()

	go sorter(logEntryProcessQueue)
	go delayPrinter(computedDelayQueue, hub.broadcast)
	go hub.run()

	http.HandleFunc("/", logEndpointHandler)
	http.HandleFunc("/follow", serveHome)
	http.HandleFunc("/delays", serveDelays)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})
	fmt.Printf("Starting server at port 80 for a %d node network\n", nodesToWaitBeforePrint)
	log.Print(http.ListenAndServe(":80", nil))
}

func sorter(processQueue chan *LogEntry) {
	workersCount := 0
	nodesCount := uint(0)
	nodes := make(map[string]chan *LogEntry, 10)
	var currentQueue chan *LogEntry
	for logEntry := range processQueue {
		queue, exists := nodes[logEntry.NodeName]
		if exists {
			queue <- logEntry
		} else {
			if nodesCount%maxLogEntryPerWorker == 0 {
				currentQueue = make(chan *LogEntry)
				workersCount++
				go logEntryWorker(workersCount, currentQueue)
			}
			nodes[logEntry.NodeName] = currentQueue
			currentQueue <- logEntry
			nodesCount++
		}
	}
}

func logEntryWorker(workerID int, processQueue chan *LogEntry) {
	nodes := make(map[string]*blocks.Node)
	var node *blocks.Node
	var exists bool
	var timestamp time.Time
	for logEntry := range processQueue {
		timestamp, _ = time.Parse("15:04:05.000", logEntry.Timestamp)

		node, exists = nodes[logEntry.NodeName]
		if !exists {
			sendLog(fmt.Sprintf("[Worker %d] Added node %s", workerID, logEntry.NodeName))
			node = &blocks.Node{
				Name: logEntry.NodeName,
				IP:   logEntry.fromIP}
			nodes[logEntry.NodeName] = node
		}

		block, exists := blocks.GetBlock(logEntry.Block.Hash)
		switch logEntry.Verb {
		case blocks.BlockMined:
			if !exists {
				block = blocks.NewBlock(logEntry.Block.Hash, logEntry.Block.Height, logEntry.Block.Size, node, node.LastBlock, timestamp)
				sendLog(fmt.Sprintf("[%s] Block %d (%s): Mined", node.Name, logEntry.Block.Height, logEntry.Block.Hash))
				computedDelayQueue <- blockDelayCount{
					block:      block,
					delayCount: block.CalculateDelay(logEntry.NodeName, timestamp)}
			} else {
				block.SetSize(logEntry.Block.Size)
				computedDelayQueue <- blockDelayCount{
					block:      block,
					delayCount: block.SetMiner(node, timestamp)}
				sendLog(fmt.Sprintf("[%s] Block %d (%s): Updated miner", node.Name, logEntry.Block.Height, logEntry.Block.Hash))
			}
		case blocks.BlockAdded:
			if exists {
				node.LastBlock = block
				sendLog(fmt.Sprintf("[%s] Block %d (%s): Sent update request", node.Name, logEntry.Block.Height, logEntry.Block.Hash))
				computedDelayQueue <- blockDelayCount{
					block:      block,
					delayCount: block.CalculateDelay(logEntry.NodeName, timestamp)}
			} else {
				block = blocks.NewBlock(logEntry.Block.Hash, logEntry.Block.Height, 0, nil, node.LastBlock, timestamp)
				node.LastBlock = block
				block.CalculateDelay(logEntry.NodeName, timestamp)
				sendLog(fmt.Sprintf("[%s] Block %d (%s): Created without miner", node.Name, logEntry.Block.Height, logEntry.Block.Hash))
			}
		}
	}
}

func delayPrinter(delayQueue chan blockDelayCount, delayAnnounceQueue chan MessageType) {
	// Init the csv writer
	csvDelayQueue := make(chan MessageType, 10)
	defer close(csvDelayQueue)
	go resultWriter(csvDelayQueue)

	var lastPrinted uint

	for delayCount := range delayQueue {
		if delayCount.delayCount >= nodesToWaitBeforePrint { // Se > allora c'è stato un rollback considerando che nodesToWaitBeforePrint sono tutti
			if lastPrinted >= delayCount.block.Heigth() {
				continue
			}
			lastPrinted = delayCount.block.Heigth()
			delays := delayCount.block.GetDelays()
			sendLog(fmt.Sprintf("------- Block %d ---------", delayCount.block.Heigth()))
			sendLog(fmt.Sprintf("# Size: %d", delayCount.block.Size()))
			for key, value := range delays {
				sendLog(fmt.Sprintf("- %s: %d", key, value/1000000))
			}
			sendLog(fmt.Sprintf("-------------------------"))
			delayAnnounceQueue <- MessageType{
				Size:   delayCount.block.Size(),
				Height: delayCount.block.Heigth(),
				Delays: delays}

			csvDelayQueue <- MessageType{
				Size:   delayCount.block.Size(),
				Height: delayCount.block.Heigth(),
				Delays: delayCount.block.GetDelays()}
		}
	}
}

func resultWriter(delayQueue chan MessageType) {
	file, err := os.Create(delayCsvFileName)
	if err != nil {
		log.Fatal("Cannot create delay file file", err)
	}
	defer file.Close()

	positions := make([]string, 0, 100)
	var bufferString string
	for delaysEntry := range delayQueue {
		bufferString = fmt.Sprintf("%d,%d", delaysEntry.Height, delaysEntry.Size)
		for _, node := range positions {
			bufferString += fmt.Sprintf(",%d", delaysEntry.Delays[node])
			delete(delaysEntry.Delays, node)
		}
		for node, delay := range delaysEntry.Delays {
			positions = append(positions, node)
			bufferString += fmt.Sprintf(",%d", delay)
		}
		bufferString += "\n"
		fmt.Fprint(file, bufferString)
	}
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	if r.URL.Path != "/follow" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, exePath+"/home.html")
}

func serveDelays(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, exePath+"/"+delayCsvFileName)
}

func sendLog(message string) {
	if logFlag {
		log.Println(message)
	}
}

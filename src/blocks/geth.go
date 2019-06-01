package blocks

import (
	"bufio"
	"strconv"
	"strings"
)

// GethBlockSource type to use to parse a geth client log
type GethBlockSource struct {
	quit chan bool
}

// Init initialize the type for the subsequent start and stop calls
func (gbs *GethBlockSource) Init() {
	if gbs.quit == nil {
		gbs.quit = make(chan bool)
	}
}

// Start starts log parsing.
func (gbs *GethBlockSource) Start(reader *bufio.Reader, target chan *LogEntry) {
	var hash, timestamp string // These are utility variables in order to not define a new one at each iteration
	spaceSplitter := func(c rune) bool {
		return c == ' '
	}
	for {
		select {
		case <-gbs.quit:
			break
		default:
			text, _ := reader.ReadString('\n')
			lineSlice := strings.Split(text, "[")
			switch lineSlice[0] {
			case "INFO ":
				infoSlice := strings.FieldsFunc(lineSlice[1], spaceSplitter)
				if infoSlice[2] == "mined" {
					timestamp = strings.Split(infoSlice[0], "|")[1]
					logEntry := &LogEntry{
						Timestamp: timestamp[:len(timestamp)-1],
						Verb:      BlockMined}
					logEntry.Block.Height, _ = strconv.Atoi(strings.Split(infoSlice[5], "=")[1])
					hash = strings.Split(infoSlice[6], "=")[1]
					logEntry.Block.Hash = hash[:len(hash)-1]
					target <- logEntry
				}
			case "DEBUG":
				debugSlice := strings.FieldsFunc(lineSlice[1], spaceSplitter)
				if debugSlice[1] == "Inserted" && debugSlice[2] != "forked" {
					timestamp := strings.Split(debugSlice[0], "|")[1]
					timestamp = timestamp[:len(timestamp)-1]
					logEntry := &LogEntry{
						Timestamp: timestamp,
						Verb:      BlockAdded}
					logEntry.Block.Height, _ = strconv.Atoi(strings.Split(debugSlice[4], "=")[1])
					logEntry.Block.Hash = strings.Split(debugSlice[5], "=")[1]
					logEntry.Block.NTx, _ = strconv.Atoi(strings.Split(debugSlice[7], "=")[1])
					target <- logEntry
				}
			}
		}
	}
}

// Stop stops log parsing
func (gbs *GethBlockSource) Stop() {
	gbs.quit <- true
}

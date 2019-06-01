package blocks

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

// MultichainBlockSource implements Multichain client log parsing
type MultichainBlockSource struct {
	logEntry      *LogEntry
	nextMinedSize int
	quit          chan bool
}

// Init initialize the type for the subsequent start and stop calls
func (mbs *MultichainBlockSource) Init() {
	if mbs.quit == nil {
		mbs.quit = make(chan bool)
	}
}

// Start starts log parsing.
func (mbs *MultichainBlockSource) Start(reader *bufio.Reader, target chan *LogEntry) {
	for {
		select {
		case <-mbs.quit:
			break
		default:
			if mbs.logEntry == nil {
				mbs.logEntry = &LogEntry{}
			}
			text, _ := reader.ReadString('\n')
			lineSlice := strings.Split(text, " ")
			if len(lineSlice) < 6 {
				continue
			}
			mbs.logEntry.Timestamp = strings.Join(lineSlice[:2], " ")
			switch lineSlice[2] {
			case "MultiChainMiner:":
				mbs.logEntry.Verb = BlockMined
				mbs.logEntry.Block.Hash = strings.Split(lineSlice[6], ",")[0]
				mbs.logEntry.Block.Height, _ = strconv.Atoi(strings.Split(lineSlice[10], ",")[0])
				mbs.logEntry.Block.NTx, _ = strconv.Atoi(lineSlice[12])
				mbs.logEntry.Block.Size = mbs.nextMinedSize
			case "UpdateTip:":
				mbs.logEntry.Verb = BlockAdded
				mbs.logEntry.Block.Hash = strings.Split(lineSlice[15], "=")[1]
				mbs.logEntry.Block.Height, _ = strconv.Atoi(strings.Split(lineSlice[17], "=")[1])
			case "CreateNewBlock():":
				mbs.nextMinedSize, _ = strconv.Atoi(strings.Split(lineSlice[5], "\n")[0])
				fmt.Printf("Size detected %d\n", mbs.nextMinedSize)
				continue
			default:
				continue
			}
			target <- mbs.logEntry
			mbs.logEntry = nil
		}
	}
}

// Stop stops log parsing
func (mbs *MultichainBlockSource) Stop() {
	mbs.quit <- true
}

package blocks

import "bufio"

// BlockSource is the interface to implement to add a new blockchain client
type BlockSource interface {
	Init()
	Start(source *bufio.Reader, target chan *LogEntry)
	Stop()
}

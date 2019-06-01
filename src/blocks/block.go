package blocks

import (
	"sync"
	"time"
)

// Verb constants
const (
	BlockMined = "block_mined" // New block mined
	BlockAdded = "block_added" // Block added to chain
)

// Block is the block struct info in which we are interested in
type Block struct {
	sync.RWMutex
	previous *Block
	heigth   uint
	hash     string
	size     uint
	miner    *Node
	minedAt  time.Time
	delays   map[string]time.Duration
}

var blocks = make(map[string]*Block)
var blocksRWMutex = sync.RWMutex{}

// NewBlock creates a new block and returns its pointer
func NewBlock(hash string, height uint, size uint, miner *Node, previous *Block, timestamp time.Time) *Block {
	block := &Block{
		previous: previous,
		hash:     hash,
		heigth:   height,
		size:     size,
		miner:    miner,
		minedAt:  timestamp,
		delays:   make(map[string]time.Duration)}
	blocksRWMutex.Lock()
	blocks[hash] = block
	blocksRWMutex.Unlock()
	return block
}

// GetBlock retrieves the block with the given hash
func GetBlock(hash string) (*Block, bool) {
	var block *Block
	var exists bool
	blocksRWMutex.RLock()
	block, exists = blocks[hash]
	blocksRWMutex.RUnlock()
	return block, exists
}

// Heigth height getter. It is thread safe because height can only be written when a block is created.
func (b *Block) Heigth() uint {
	return b.heigth
}

// Hash height getter. It is thread safe because hash can only be written when a block is created.
func (b *Block) Hash() string {
	return b.hash
}

// Previous height getter. It is thread safe because previous can only be written when a block is created.
func (b *Block) Previous() *Block {
	return b.previous
}

// Size of the block. It is thread safe.
func (b *Block) Size() uint {
	var size uint
	b.RLock()
	size = b.size
	b.RUnlock()
	return size
}

// SetSize sets the siza in a thread safe mode.
func (b *Block) SetSize(size uint) {
	b.Lock()
	b.size = size
	b.Unlock()
}

// SetMiner update delays with the right miner timestamp. It is thread safe.
func (b *Block) SetMiner(node *Node, minedAt time.Time) int {
	b.Lock()
	b.miner = node
	minerDelayOffset := b.minedAt.Sub(minedAt)
	b.minedAt = minedAt
	for nodeName, delay := range b.delays {
		b.delays[nodeName] = delay + minerDelayOffset
	}
	computedDelaysCount := len(b.delays)
	b.Unlock()
	return computedDelaysCount
}

// CalculateDelay puts the calculcated delay for the given node inside the block. It is thread safe.
func (b *Block) CalculateDelay(nodeIdentifier string, time time.Time) int {
	b.Lock()
	b.delays[nodeIdentifier] = time.Sub(b.minedAt)
	computedDelaysCount := len(b.delays)
	b.Unlock()
	return computedDelaysCount
}

// GetDelays gets a copy of computed delays of a block. It is Thread Safe.
func (b *Block) GetDelays() map[string]time.Duration {
	delays := make(map[string]time.Duration)
	b.RLock()
	for key, value := range b.delays {
		delays[key] = value
	}
	b.RUnlock()
	return delays
}

package blocks

// Node holds information about node partecipating in blockchain network
type Node struct {
	Name      string
	IP        string
	LastBlock *Block
}

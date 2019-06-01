package blocks

// LogEntry represents an entry in the debug.log
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Verb      string `json:"verb"`
	Block     struct {
		Height int    `json:"height"`
		Hash   string `json:"hash"`
		NTx    int    `json:"nTX"`
		Size   int    `json:"size"`
	} `json:"block"`
	NodeName string `json:"nodeName"`
}

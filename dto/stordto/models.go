package stordto

// ChCheckPoint is a struct with checkpoint data
type ChCheckPoint struct {
	// Ver is a checkpoint version
	Ver int64
	// SrcCollectFromBlockNums is a map with block numbers for each channel
	SrcCollectFromBlockNums map[string]uint64
	// MinExecBlockNum is a minimal block number for execution
	MinExecBlockNum uint64
}

package stordto

type ChCheckPoint struct {
	Ver                     int64
	SrcCollectFromBlockNums map[string]uint64
	MinExecBlockNum         uint64
}

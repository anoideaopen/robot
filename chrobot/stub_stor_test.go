package chrobot

import (
	"context"
	"fmt"
	"sync"

	"github.com/anoideaopen/common-component/testshlp"
	"github.com/anoideaopen/robot/dto/stordto"
)

type stubStor struct {
	callHlp testshlp.CallHlp

	lock sync.RWMutex
	cp   *stordto.ChCheckPoint
}

func (stor *stubStor) SaveCheckPoints(_ context.Context, cp *stordto.ChCheckPoint) (*stordto.ChCheckPoint, error) {
	stor.lock.Lock()
	defer stor.lock.Unlock()

	if err := stor.callHlp.Call(stor.SaveCheckPoints); err != nil {
		return nil, err
	}

	newVer := cp.Ver
	if stor.cp != nil {
		if stor.cp.Ver != cp.Ver {
			return nil, fmt.Errorf("invalid version for save current:%v, saving:%v", stor.cp.Ver, cp.Ver)
		}
		newVer++
	}
	stor.cp = cloneChCheckPoints(cp)
	stor.cp.Ver = newVer

	return cloneChCheckPoints(stor.cp), nil
}

func (stor *stubStor) LoadCheckPoints(_ context.Context) (*stordto.ChCheckPoint, bool, error) {
	stor.lock.RLock()
	defer stor.lock.RUnlock()

	if err := stor.callHlp.Call(stor.LoadCheckPoints); err != nil {
		return nil, false, err
	}

	return cloneChCheckPoints(stor.cp), stor.cp != nil, nil
}

func cloneChCheckPoints(cp *stordto.ChCheckPoint) *stordto.ChCheckPoint {
	if cp == nil {
		return nil
	}
	res := &stordto.ChCheckPoint{
		Ver:                     cp.Ver,
		SrcCollectFromBlockNums: make(map[string]uint64),
		MinExecBlockNum:         cp.MinExecBlockNum,
	}

	for k, v := range cp.SrcCollectFromBlockNums {
		res.SrcCollectFromBlockNums[k] = v
	}

	return res
}

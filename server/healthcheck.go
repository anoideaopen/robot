package server

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/newity/glog"
)

type RobotState string

const (
	robotCreating       RobotState = "creating"
	RobotStarted        RobotState = "started"
	RobotStopped        RobotState = "stopped"
	RobotStoppedWithErr RobotState = "stoppedWithErr"
)

type LivenessMng struct {
	log glog.Logger

	lock   sync.RWMutex
	robots map[string]RobotState
}

func (lm *LivenessMng) SetRobotState(chName string, state RobotState) {
	lm.lock.Lock()
	defer lm.lock.Unlock()

	lm.robots[chName] = state
}

func (lm *LivenessMng) summary() []byte {
	lm.lock.RLock()
	defer lm.lock.RUnlock()

	res, err := json.Marshal(lm.robots)
	if err != nil {
		lm.log.Errorf("marshal robots state error: %+v", err)
		return nil
	}
	return res
}

func healthChecksHandler(ctx context.Context, robots []string) (liveness http.HandlerFunc, readiness http.HandlerFunc, lm *LivenessMng) {
	log := glog.FromContext(ctx)

	lm = &LivenessMng{
		log: log,

		robots: make(map[string]RobotState),
	}

	for _, rName := range robots {
		lm.SetRobotState(rName, robotCreating)
	}

	liveness = func(w http.ResponseWriter, _ *http.Request) {
		resp := lm.summary()

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")

		if len(resp) == 0 {
			return
		}

		if _, err := w.Write(resp); err != nil {
			log.Errorf("write robots state response error: %+v", err)
		}
	}

	readiness = func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	return
}

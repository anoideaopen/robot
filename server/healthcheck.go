package server

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/newity/glog"
)

// RobotState is a state of robot
type RobotState string

const (
	robotCreating RobotState = "creating"
	// RobotStarted is a state of robot when it is started
	RobotStarted RobotState = "started"
	// RobotStopped is a state of robot when it is stopped
	RobotStopped RobotState = "stopped"
	// RobotStoppedWithErr is a state of robot when it is stopped with error
	RobotStoppedWithErr RobotState = "stoppedWithErr"
)

// LivenessMng is a manager of robots liveness
type LivenessMng struct {
	log glog.Logger

	lock   sync.RWMutex
	robots map[string]RobotState
}

// SetRobotState sets state of robot
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

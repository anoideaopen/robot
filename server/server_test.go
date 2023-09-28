//go:build !integration
// +build !integration

package server

import (
	"testing"
	"time"

	"github.com/atomyze-foundation/common-component/testshlp"
	"github.com/atomyze-foundation/robot/helpers/ntesting"
	"github.com/stretchr/testify/require"
)

func TestStartStopServer(t *testing.T) {
	_ = ntesting.CI(t)

	ctx, _ := testshlp.CreateCtxLogger(t)

	lm, serverShutdown, err := StartServer(ctx, 8080,
		&AppInfo{
			Ver:          "verX",
			VerSdkFabric: "verFabricY",
		},
		[]string{"fiat", "cc"}, nil)
	require.NoError(t, err)
	lm.SetRobotState("fiat", RobotStarted)

	serverShutdown()
	<-time.After(1 * time.Second)
}

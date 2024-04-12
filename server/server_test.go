package server

import (
	"testing"
	"time"

	"github.com/anoideaopen/common-component/testshlp"
	"github.com/anoideaopen/robot/helpers/ntesting"
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

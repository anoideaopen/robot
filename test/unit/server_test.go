package unit

import (
	"testing"
	"time"

	"github.com/anoideaopen/common-component/testshlp"
	"github.com/anoideaopen/robot/helpers/ntesting"
	"github.com/anoideaopen/robot/server"
	"github.com/stretchr/testify/require"
)

func TestStartStopServer(t *testing.T) {
	_ = ntesting.CI(t)

	ctx, _ := testshlp.CreateCtxLogger(t)

	lm, serverShutdown, err := server.StartServer(ctx, 8080,
		&server.AppInfo{
			Ver:          "verX",
			VerSdkFabric: "verFabricY",
		},
		[]string{"fiat", "cc"}, nil)
	require.NoError(t, err)
	lm.SetRobotState("fiat", server.RobotStarted)

	serverShutdown()
	<-time.After(1 * time.Second)
}

package relay

import (
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	tasksora "github.com/QuantumNous/new-api/relay/channel/task/sora"
	"github.com/stretchr/testify/require"
)

func TestGetTaskAdaptorReturnsSoraAdaptorForXAI(t *testing.T) {
	adaptor := GetTaskAdaptor(constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeXai)))

	require.IsType(t, &tasksora.TaskAdaptor{}, adaptor)
}

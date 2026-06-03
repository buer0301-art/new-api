package perfmetrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestQueryRangeUsesExplicitRange(t *testing.T) {
	startTs := int64(1716076800)
	endTs := int64(1716163199)

	gotStart, gotEnd := queryRange(startTs, endTs, 24)

	require.Equal(t, startTs, gotStart)
	require.Equal(t, endTs, gotEnd)
}

func TestQueryRangeSwapsReversedExplicitRange(t *testing.T) {
	startTs := int64(1716163199)
	endTs := int64(1716076800)

	gotStart, gotEnd := queryRange(startTs, endTs, 24)

	require.Equal(t, endTs, gotStart)
	require.Equal(t, startTs, gotEnd)
}

func TestQueryRangeFallsBackToHours(t *testing.T) {
	before := time.Now().Unix()
	gotStart, gotEnd := queryRange(0, 0, 2)
	after := time.Now().Unix()

	require.InDelta(t, after, gotEnd, 2)
	require.InDelta(t, before-2*3600, gotStart, 4)
}

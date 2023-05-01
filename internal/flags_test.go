package internal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTimeRangeValue(t *testing.T) {
	tr := &TimeRangeValue{}

	require.Error(t, tr.Set("a"), "Bad formats should return error")

	require.NoError(t, tr.Set("1970-01-01,"))
	require.True(t, tr.Start.Equal(time.Unix(0, 0).UTC()))
	require.Nil(t, tr.End)

	require.NoError(t, tr.Set("1970-01-01T00:00:00Z,"))
	require.True(t, tr.Start.Equal(time.Unix(0, 0).UTC()))
	require.Nil(t, tr.End)

	require.NoError(t, tr.Set("1970-01-01T00:00:00Z,1970-01-01T00:00:00Z"))
	require.True(t, tr.Start.Equal(time.Unix(0, 0).UTC()))
	require.True(t, tr.End.Equal(time.Unix(0, 0).UTC()))
}

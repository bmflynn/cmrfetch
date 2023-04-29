package internal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSearchCollectionParams(t *testing.T) {
  refTime := time.Unix(0, 0).UTC()
	params := NewSearchCollectionParams()
	q, err := params.
    Keyword("foo").
	  Providers("p1", "p2").
	  UpdatedSince(refTime).
    GranulesAdded(TimeRange{Start: refTime}).
    Instruments("viirs", "modis").
    Platforms("suomi-npp", "aqua").
    Title("pat?e*n").
    build()
	require.NoError(t, err)

	require.Equal(t, "foo", q.Get("keyword"))
	require.Equal(t, []string{"p1", "p2"}, q["provider_short_name"])
	require.Equal(t, "true", q.Get("options[provider_short_name][ignore_case]"))
  require.Equal(t, "pat?e*n", q.Get("entry_title"))
	require.Equal(t, "true", q.Get("options[entry_title][ignore_case]"))
	require.Equal(t, "true", q.Get("options[entry_title][pattern]"))
  require.Equal(t, "1970-01-01T00:00:00Z", q.Get("updated_since"))
  require.Equal(t, "1970-01-01T00:00:00Z,", q.Get("has_granules_revised_at"))
}     

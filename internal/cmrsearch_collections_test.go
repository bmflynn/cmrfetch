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
		ShortNames("s1", "s2").
		Platforms("suomi-npp", "aqua").
		Instruments("viirs", "modis").
		Title("pat?e*n").
		UpdatedSince(refTime).
		GranulesAdded(TimeRange{Start: refTime}).
		DataType("dt").
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
	require.Equal(t, "dt", q.Get("collection_data_type"))

	require.Equal(t, false, params.cloudHostedSet)
	q, err = params.CloudHosted(true).build()
	require.NoError(t, err)
	require.Equal(t, "true", q.Get("cloud_hosted"))

	require.Equal(t, false, params.standardSet)
	q, err = params.Standard(true).build()
	require.NoError(t, err)
	require.Equal(t, "true", q.Get("standard_product"))

	require.Equal(t, false, params.hasGranules)
	q, err = params.HasGranules(true).build()
	require.NoError(t, err)
	require.Equal(t, "true", q.Get("has_granules"))
}

package granules

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/bmflynn/cmrfetch/internal"
	"github.com/stretchr/testify/require"
)

func TestJsonDocWriter(t *testing.T) {
	granules := struct {
		Items []internal.Granule `json:"items"`
	}{
		Items: []internal.Granule{
			{Name: "granule1"},
			{Name: "granule2"},
		},
	}
	zult := internal.NewGranuleResult()
	go func() {
		defer close(zult.Ch)
		for _, g := range granules.Items {
			zult.Ch <- g
		}
	}()

	w := &bytes.Buffer{}
	err := jsonDocWriter(zult, w, []string{"name"})
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, json.Unmarshal(w.Bytes(), &doc), "output is not valid JSON")
	require.Contains(t, doc, "items", "output JSON does not contain 'items' key")

	items, ok := doc["items"].([]any)
	require.True(t, ok, "'items' is not an array")
	require.Len(t, items, len(granules.Items), "'items' length does not match expected")
	for _, item := range items {
		g, ok := item.(map[string]any)
		require.True(t, ok, "item is not a map")
		require.Contains(t, g, "name", "granule does not contain 'name' field")
	}
}

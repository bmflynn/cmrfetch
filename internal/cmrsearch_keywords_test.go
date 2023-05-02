package internal

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSearchFacets(t *testing.T) {
	newServer := func(t *testing.T, body string, status int, hits string) func() {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if hits != "" {
				w.Header().Set("cmr-hits", hits)
			}
			w.WriteHeader(status)
			w.Write([]byte(body))
		}))
		url := fmt.Sprintf("http://%s", ts.Listener.Addr())
		origURL := defaultCMRURL
		defaultCMRSearchURL = url
		return func() {
			defaultCMRSearchURL = origURL
			ts.Close()
		}
	}

	doGet := func(t *testing.T, val string, types []string) ScrollResult[Facet] {
		t.Helper()

		api := NewCMRSearchAPI(log.Default())
		// make sure we're not waiting long
		zult, err := api.SearchFacets(context.Background(), val, types)
		require.NoError(t, err)

		return zult
	}

	t.Run("get", func(t *testing.T) {
		body := `
{
  "feed": {
    "entry": [
      {
        "score": 1.0,
        "type": "TYPE",
        "fields": "FIELDS",
        "value": "VALUE"
      }
    ]
  }
}
    `

		cleanup := newServer(t, string(body), http.StatusOK, "1")
		defer cleanup()

		zult := doGet(t, "xxx", []string{"t1", "t2"})
		require.NoError(t, zult.Err())
		require.Equal(t, 1, zult.Hits())

		facets := []Facet{}
		for f := range zult.Ch {
			facets = append(facets, f)
		}

		require.NoError(t, zult.Err())
		require.Len(t, facets, 1)

		facet := facets[0]
		require.Equal(t, 1.0, facet.Score)
		require.Equal(t, "TYPE", facet.Type)
		require.Equal(t, "FIELDS", facet.Fields)
		require.Equal(t, "VALUE", facet.Value)
	})

	t.Run("error", func(t *testing.T) {
		cleanup := newServer(t, "{}", http.StatusBadRequest, "1")
		defer cleanup()

		zult := doGet(t, "xxx", []string{"t1", "t2"})
		require.NoError(t, zult.Err())
		require.Equal(t, 0, zult.Hits())

		facets := []Facet{}
		for f := range zult.Ch {
			facets = append(facets, f)
		}

		require.NoError(t, zult.Err())
		require.Len(t, facets, 0)
	})
}

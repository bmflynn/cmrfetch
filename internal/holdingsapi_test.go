package internal

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_readHoldings(t *testing.T) {
	r := bytes.NewBuffer([]byte(`
[
  {
    "concept-id": "C1",
    "entry-title": "TITLE",
    "provider-id": "PROVIDER1",
    "granule-count": 1
  },
  {
    "concept-id": "C2",
    "entry-title": "TITLE",
    "provider-id": "PROVIDER1",
    "granule-count": 1
  },
  {
    "concept-id": "C1",
    "entry-title": "TITLE",
    "provider-id": "PROVIDER2",
    "granule-count": 1
  }
]
`))

	zult, err := readHoldings(r)
	require.Len(t, zult, 2)
	require.NoError(t, err)

	providers := map[string]Provider{}
	for _, p := range zult {
		providers[p.ID] = p
	}

	prov := providers["PROVIDER1"]
	require.Len(t, prov.Collections, 2)
	require.Equal(t, int64(2), prov.GranuleCount)
	require.Equal(t, int64(1), prov.Collections[0].GranuleCount)
	require.Equal(t, int64(1), prov.Collections[1].GranuleCount)
	prov = providers["PROVIDER2"]
	require.Len(t, prov.Collections, 1)
	require.Equal(t, int64(1), prov.GranuleCount)
	require.Equal(t, int64(1), prov.Collections[0].GranuleCount)
}

func testCacheDir(t *testing.T) (string, func()) {
	t.Helper()

	dir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	t.Setenv("XDG_CACHE_HOME", dir)
	return dir, func() {
		os.RemoveAll(dir)
	}
}

func Test_getCachedProviderHoldngs(t *testing.T) {

	t.Run("cache dir missing is ok", func(t *testing.T) {
		_, cleanup := testCacheDir(t)
		defer cleanup()

		zult, mtime, err := getCachedProviderHoldings()
		t.Logf("%#v", zult)
		require.Nil(t, zult, fmt.Sprintf("arr should be nil: %#v", zult))
		require.True(t, mtime.Equal(time.Time{}))
		require.Nil(t, err)
	})

	t.Run("not a dir is err", func(t *testing.T) {
		dir, cleanup := testCacheDir(t)
		defer cleanup()
		err := ioutil.WriteFile(filepath.Join(dir, "cmrfetch"), nil, 0o644)
		require.NoError(t, err)

		zult, mtime, err := getCachedProviderHoldings()
		require.Nil(t, zult)
		require.True(t, mtime.Equal(time.Time{}))
		require.NotNil(t, err)
	})

	t.Run("cache file missing is ok", func(t *testing.T) {
		dir, cleanup := testCacheDir(t)
		defer cleanup()
		err := os.MkdirAll(filepath.Join(dir, "cmrfetch"), 0o755)
		require.NoError(t, err)

		zult, mtime, err := getCachedProviderHoldings()
		require.Nil(t, zult)
		require.True(t, mtime.Equal(time.Time{}))
		require.NoError(t, err)
	})

	t.Run("fail to open is err", func(t *testing.T) {
		dir, cleanup := testCacheDir(t)
		defer cleanup()
		err := os.MkdirAll(filepath.Join(dir, "cmrfetch"), 0o755)
		require.NoError(t, err)
		fpath := filepath.Join(dir, "cmrfetch", "provider_holdings.json")
		require.NoError(t, ioutil.WriteFile(fpath, nil, 0o000))

		zult, mtime, err := getCachedProviderHoldings()
		require.Nil(t, zult)
		require.True(t, mtime.Equal(time.Time{}))
		require.Error(t, err)
	})

	t.Run("nominal", func(t *testing.T) {
		dir, cleanup := testCacheDir(t)
		defer cleanup()
		err := os.MkdirAll(filepath.Join(dir, "cmrfetch"), 0o755)
		require.NoError(t, err)
		fpath := filepath.Join(dir, "cmrfetch", "provider_holdings.json")
		require.NoError(t, ioutil.WriteFile(fpath, []byte(`[]`), 0o644))

		zult, mtime, err := getCachedProviderHoldings()
		require.Len(t, zult, 0)
		require.Less(t, time.Since(mtime), time.Minute)
		require.NoError(t, err)
	})
}

func Test_writeCachedProviderHoldings(t *testing.T) {
	dir, cleanup := testCacheDir(t)
	defer cleanup()

	providers := []Provider{
		{
			ID: "PROVIDER1",
			Collections: []CollectionItem{
				{
					ConceptID: "C1",
					Title:     "TITLE",
				},
			},
			GranuleCount: 0,
		},
	}

	err := writeCachedProviderHoldings(providers)
	require.NoError(t, err)

	fpath := filepath.Join(dir, "cmrfetch", "provider_holdings.json")
	fi, err := os.Stat(fpath)
	require.NoError(t, err)
	require.Greater(t, fi.Size(), int64(0))
}

func TestGetProviderHoldings(t *testing.T) {
	dir, cleanup := testCacheDir(t)
	defer cleanup()

	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`
[
  {
    "concept-id": "C1",
    "entry-title": "TITLE",
    "provider-id": "PROVIDER1",
    "granule-count": 1
  },
  {
    "concept-id": "C2",
    "entry-title": "TITLE",
    "provider-id": "PROVIDER1",
    "granule-count": 1
  },
  {
    "concept-id": "C1",
    "entry-title": "TITLE",
    "provider-id": "PROVIDER2",
    "granule-count": 1
  }
]`))
	}))
	defer svr.Close()
	origURL := defaultHoldingsURL
	defaultHoldingsURL = fmt.Sprintf("http://%s/", svr.Listener.Addr())
	defer func() {
		defaultHoldingsURL = origURL
	}()

	providers, err := GetProviderHoldings()
	require.NoError(t, err)
	require.Len(t, providers, 2)

	fpath := filepath.Join(dir, "cmrfetch", "provider_holdings.json")
	_, err = os.Stat(fpath)
	require.NoError(t, err)
}

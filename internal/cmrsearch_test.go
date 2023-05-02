package internal

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestCMRSearchAPI(t *testing.T) {
	newServer := func(t *testing.T, body string, status int, hits string) (*httptest.Server, string) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if hits != "" {
				w.Header().Set("cmr-hits", hits)
			}
			w.WriteHeader(status)
			w.Write([]byte(body))
		}))
		url := fmt.Sprintf("http://%s", ts.Listener.Addr())
		return ts, url
	}

	doGet := func(t *testing.T, url string) ScrollResult[gjson.Result] {
		t.Helper()

		api := NewCMRSearchAPI(nil)
		// make sure we're not waiting long
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		zult, err := api.Get(ctx, url)
		require.NoError(t, err)

		return zult
	}

	t.Run("get", func(t *testing.T) {
		t.Run("non-200 is CMRError", func(t *testing.T) {
			body := `
      {
          "errors": [
            "Your request is borked"
          ]
      }`

			svr, url := newServer(t, body, http.StatusBadRequest, "")
			defer svr.Close()

			zult := doGet(t, url)
			_, more := <-zult.Ch
			require.False(t, more, "expected channel to be closed")

			var cmrErr *CMRError
			require.ErrorAs(t, zult.Err(), &cmrErr, "Expected CMRError")
      require.Contains(t, cmrErr.Error(), "Your request is borked")
		})

		t.Run("bad hits is error", func(t *testing.T) {
			svr, url := newServer(t, "", http.StatusBadRequest, "a")
			defer svr.Close()

			zult := doGet(t, url)

			require.Error(t, zult.Err(), "expected error for bad hits header")
		})

		t.Run("success", func(t *testing.T) {
			t.Run("items", func(t *testing.T) {
				body := `{"items": [1]}`
				svr, url := newServer(t, body, http.StatusOK, "1")
				defer svr.Close()

				zult := doGet(t, url)

				err := zult.Err()
				require.NoError(t, err, "expected no error for valid body: %#v", err)

				results := []gjson.Result{}
				for r := range zult.Ch {
					results = append(results, r)
				}

				require.Len(t, results, 1)
			})

			t.Run("feed.entry", func(t *testing.T) {
				body := `{"feed": {"entry": [1]}}`
				svr, url := newServer(t, body, http.StatusOK, "1")
				defer svr.Close()

				zult := doGet(t, url)

				require.NoError(t, zult.Err(), "expected no error for valid body")

				results := []gjson.Result{}
				for r := range zult.Ch {
					results = append(results, r)
				}

				require.Len(t, results, 1)
			})
		})
	})
}

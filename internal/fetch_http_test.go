package internal

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

type mockReadCloser struct {
	r   io.Reader
	err error
}

func (r *mockReadCloser) Close() error { return nil }
func (r *mockReadCloser) Read(buf []byte) (int, error) {
	if r.err != nil {
		return 0, r.err
	}
	return r.r.Read(buf)
}

func Test_newFailedDownloadError(t *testing.T) {
	t.Run("err reading body", func(t *testing.T) {
		resp := &http.Response{
			Body:    &mockReadCloser{err: fmt.Errorf("bogus error")},
			Request: httptest.NewRequest("", "/", nil),
		}
		fd := newFailedDownloadError(resp)
		require.Equal(t, "", fd.ResponseBody)
	})

	hdr := http.Header{}
	hdr.Set("request-id", "REQUESTID")
	resp := &http.Response{
		Header:  hdr,
		Status:  "STATUS",
		Body:    &mockReadCloser{r: bytes.NewBuffer([]byte("BODY"))},
		Request: httptest.NewRequest("", "/", nil),
	}

	fd := newFailedDownloadError(resp)

	require.Equal(t, "REQUESTID", fd.RequestID)
	require.Equal(t, "STATUS", fd.Status)
	require.Equal(t, "BODY", fd.ResponseBody)

	require.Contains(t, fd.Error(), "requestid")
}

func mockNetrc(t *testing.T) func() {
	t.Helper()

	netrc, err := os.CreateTemp("", "")
	require.NoError(t, err)
	ioutil.WriteFile(netrc.Name(), []byte("machine testhost.com login LOGIN password PASSWORD"), 0o644)

	orig := defaultNetrcFinder
	defaultNetrcFinder = func() (string, error) {
		return netrc.Name(), nil
	}
	return func() {
		defaultNetrcFinder = orig
		os.Remove(netrc.Name())
	}
}

func Test_newRedirectWithNetrcCredentials(t *testing.T) {
	defer mockNetrc(t)()

	redirect, err := newRedirectWithNetrcCredentials()
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "http://testhost.com/path", nil)
	err = redirect(req, []*http.Request{})
	require.NoError(t, err)

	user, passwd, ok := req.BasicAuth()
	require.True(t, ok, "request should have basic auth set")
	require.Equal(t, "LOGIN", user)
	require.Equal(t, "PASSWORD", passwd)
}

func TestHTTPFetcher(t *testing.T) {
	defer mockNetrc(t)()

	t.Run("ok", func(t *testing.T) {
		body := []byte("xxx")
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Write(body)
		}))
		defer svr.Close()

		fetcher, err := NewHTTPFetcher(true, nil)
		require.NoError(t, err)

		w := bytes.NewBuffer(nil)
		size, err := fetcher.Fetch(context.Background(), fmt.Sprintf("http://%s/", svr.Listener.Addr()), w)
		require.NoError(t, err)
		require.Equal(t, len(body), int(size))
	})

	t.Run("httperr", func(t *testing.T) {
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer svr.Close()

		fetcher, err := NewHTTPFetcher(true, nil)
		require.NoError(t, err)

		w := bytes.NewBuffer(nil)
		_, err = fetcher.Fetch(context.Background(), fmt.Sprintf("http://%s/", svr.Listener.Addr()), w)

		var dlErr *FailedDownload
		require.ErrorAs(t, err, &dlErr)
	})
}

func Test_validateResponse(t *testing.T) {
	req := httptest.NewRequest("GET", "http://localhost/", nil)
	resp := &http.Response{
		Request: httptest.NewRequest("GET", "http://nope/", nil),
	}

	require.Error(t, validateResponse(req, resp))
}

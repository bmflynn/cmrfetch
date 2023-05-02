package internal

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFetchConcurrent(t *testing.T) {
	dir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer func() {
		os.RemoveAll(dir)
	}()

	body := []byte("xxx")
	csum := hex.EncodeToString(md5.New().Sum(body))

	fetcher := func(ctx context.Context, url string, w io.Writer) (int64, error) {
		n, err := w.Write(body)
		return int64(n), err
	}

	requests := make(chan DownloadRequest, 1)
	requests <- DownloadRequest{
		URL:         "doesn't matter",
		ChecksumAlg: "md5",
		Checksum:    csum,
		Dest:        filepath.Join(dir, "testoutput.txt"),
	}
  close(requests)

	resultsCh, err := FetchConcurrent(requests, func() (Fetcher, error) { return fetcher, nil }, 1)
	require.NoError(t, err)

	results := []DownloadResult{}
	for x := range resultsCh {
		results = append(results, x)
	}

	require.Len(t, results, 1)
}

package internal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type FetchError struct {
	Request DownloadRequest
	Err     error
}

func (e *FetchError) Error() string {
	return fmt.Sprintf("fetching: %s", e.Err)
}

type FetcherFactory func() (Fetcher, error)

type FetchPoolFunc = func(reqs chan DownloadRequest, concurrency int) (chan DownloadResult, error)

func FetchConcurrent(reqs chan DownloadRequest, fetcherFactory FetcherFactory, concurrency int) (chan DownloadResult, error) {
	return FetchConcurrentWithContext(context.Background(), reqs, fetcherFactory, concurrency)
}

func FetchConcurrentWithContext(ctx context.Context, reqs chan DownloadRequest, fetcherFactory FetcherFactory, concurrency int) (chan DownloadResult, error) {
	results := make(chan DownloadResult)

	wg := &sync.WaitGroup{}
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		fetcher, err := fetcherFactory()
		if err != nil {
			return nil, fmt.Errorf("failed to init fetcher: %w", err)
		}
		go downloader(ctx, wg, reqs, results, fetcher)
	}

	// Close results once all the downloaders have exited
	go func() {
		wg.Wait()
		close(results)
	}()

	return results, nil
}

type DownloadRequest struct {
	URL         string
	ChecksumAlg string
	Checksum    string
	Dest        string
}

type DownloadResult struct {
	URL      string
	Path     string
	Checksum string
	Duration time.Duration
	Size     int64
	Err      error
}

// downloader downloads all requests using fetcher sending results to results. The provided context
// is used to cancel in-flight requests to the fetcher.
//
// All errors are of type *FetchError and will terminate the downloader.
func downloader(
	// Cancels in-flight download requests when canceled
	ctx context.Context,
	wg *sync.WaitGroup,
	requests chan DownloadRequest,
	results chan DownloadResult,
	fetch Fetcher,
) {
	defer wg.Done()

	for req := range requests {
		zult := DownloadResult{
			URL:  req.URL,
			Path: req.Dest,
		}

		// in a func so we can defer the close and make collecting the error easier
		err := func() error {
			start := time.Now()

			destdir, fname := filepath.Split(zult.Path)
			dest, err := os.CreateTemp(destdir, fmt.Sprintf(".%s.*", fname))
			if err != nil {
				return fmt.Errorf("creating dest: %w", err)
			}
			defer os.Remove(dest.Name())

			w := &writerHasher{Writer: dest}
			if req.ChecksumAlg != "" {
				w.hash, err = newHash(req.ChecksumAlg)
				if err != nil {
					return err
				}
			}
			_, err = fetch(ctx, zult.URL, w)
			if err != nil {
				return err
			}
			dest.Close() // Close before checksumming
			zult.Checksum = w.Checksum()
			zult.Size = w.size
			zult.Duration = time.Since(start)

			if zult.Checksum != req.Checksum {
				return fmt.Errorf("got checksum %s, expected %s", zult.Checksum, req.Checksum)
			}
			if err := os.Rename(dest.Name(), zult.Path); err != nil {
				return fmt.Errorf("failed to rename %s to %s: %w", dest.Name(), zult.Path, err)
			}
			if err := os.Chmod(zult.Path, 0o644); err != nil {
				return fmt.Errorf("failed to update permissions on %s: %w", zult.Path, err)
			}
			return nil
		}()
		if err != nil {
			zult.Err = &FetchError{Request: req, Err: err}
		}

		results <- zult // success!
	}
}

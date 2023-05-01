package granules

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/bmflynn/cmrfetch/internal"
)

const (
	maxResultsWithoutPrompt    = 1000
	defaultDownloadConcurrency = 4
)

func zultsToRequests(granules internal.GranuleResult, destdir string, clobber bool) chan internal.DownloadRequest {
	requests := make(chan internal.DownloadRequest)
	go func() {
		defer close(requests)
		for gran := range granules.Ch {
			request := internal.DownloadRequest{
				// Use grnaule name in dest, b/c who knows what the base of the URL will be
				Dest:        filepath.Join(destdir, gran.Name),
				URL:         gran.GetDataURL,
				Checksum:    gran.Checksum,
				ChecksumAlg: gran.ChecksumAlg,
			}
			if !clobber && internal.Exists(request.Dest) {
				log.Printf("exists %s", request.Dest)
				continue
			}
			requests <- request
		}
	}()
	return requests
}

func doDownload(
	ctx context.Context,
	api *internal.CMRSearchAPI,
	params internal.SearchGranuleParams,
	destdir string,
	netrc, clobber, yes, verbose bool,
	concurrency int,
) error {
	zult, err := api.SearchGranules(context.Background(), params)
	if err != nil {
		return err
	}

	if !yes && zult.Hits() > maxResultsWithoutPrompt {
		fmt.Printf("There are more than %v, CTRL-C to cancel or ENTER to continue\n", maxResultsWithoutPrompt)
		bufio.NewReader(os.Stdin).ReadBytes('\n')
	}

	if internal.Exists(destdir) {
		switch {
		case !internal.IsDir(destdir):
			return fmt.Errorf("download dir %s exists but is not a directory", destdir)
		case !internal.CanWrite(destdir):
			return fmt.Errorf("download dir %s exists but is not writable", destdir)
		}
	} else {
		err := os.MkdirAll(destdir, 0o755)
		if err != nil {
			return fmt.Errorf("making download dir: %w", err)
		}
	}

  fetcherFactory := func() (internal.Fetcher, error) {
    fetcher, err := internal.NewHTTPFetcher(netrc)
    return fetcher.Fetch, err
  }
	requests := zultsToRequests(zult, destdir, clobber)
	results, err := internal.FetchConcurrentWithContext(ctx, requests, fetcherFactory, concurrency)
	if err != nil {
		return fmt.Errorf("init fetcher: %s", err)
	}

	for zult := range results {
		switch {
		case zult.Err != nil:
			log.Printf("failed! %s error=%s", zult.URL, zult.Err)
			continue
		case zult.Err == nil:
			log.Printf(
				"fetched %s in %.1fs(%.1f Mb/s)", zult.URL,
				zult.Duration.Seconds(),
				(float64(zult.Size*8) / zult.Duration.Seconds() / (1024 * 1024)))
		}
	}
	return nil
}

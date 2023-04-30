package internal

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/tidwall/gjson"
)

const (
	defaultCMRURL       = "https://cmr.earthdata.nasa.gov"
	defaultCMRSearchURL = defaultCMRURL + "/search"
)

type CMRSearchAPI struct {
	url      string
	client   *http.Client
	pageSize int
	log      *log.Logger
}

func NewCMRSearchAPI(logger *log.Logger) *CMRSearchAPI {
	return &CMRSearchAPI{
		url:      defaultCMRSearchURL,
		client:   http.DefaultClient,
		pageSize: 200,
		log:      logger,
	}
}

func (api *CMRSearchAPI) debug(msg string, args ...any) {
	if api.log != nil {
		api.log.Printf(msg, args...)
	}
}

type ScrollResult[T Granule | Collection | gjson.Result | Facet] struct {
	Ch   chan T
	err  error
	hits int
	mu   *sync.Mutex
}

func newScrollResult[T Granule | Collection | gjson.Result | Facet]() ScrollResult[T] {
  return ScrollResult[T]{
    Ch: make(chan T, 1),
    mu: &sync.Mutex{},
  }
}

func (r ScrollResult[T]) Err() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.err
}

func (r ScrollResult[T]) Hits() int {
	return r.hits
}

func (r ScrollResult[T]) Close() {
	close(r.Ch)
}

func (api *CMRSearchAPI) hits(ctx context.Context, url string) (int, error) {
	api.debug("method=HEAD url=%s", url)
	zult, err := api.client.Head(url)
	if err != nil {
		return 0, fmt.Errorf("protocol error: %w", err)
	}
	hits, err := strconv.Atoi(zult.Header.Get("CMR-Hits"))
	if err != nil {
		return 0, fmt.Errorf("invalid cmr-hits value; wanted int, got %v", zult.Header.Get("cmr-hits"))
	}
	return hits, nil
}

func (api *CMRSearchAPI) Get(ctx context.Context, url string) (ScrollResult[gjson.Result], error) {
	api.debug("method=GET url=%s", url)

	result := newScrollResult[gjson.Result]()

	hitsCh := make(chan int, 1)
	go func() {
		defer result.Close()

		var searchAfter string
		for {
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				result.err = fmt.Errorf("create request: %w", err)
				return
			}
			if searchAfter != "" {
				req.Header.Set("cmr-search-after", searchAfter)
			}
			resp, err := api.client.Do(req)
			if err != nil {
				result.err = fmt.Errorf("protocol error: %w", err)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				result.err = newUMMError(resp.Body)
				return
			}

			hits, err := strconv.Atoi(resp.Header.Get("CMR-Hits"))
			if err != nil {
				result.err = fmt.Errorf("failed to parse cmr-hits header as int: %s", resp.Header.Get("cmr-hits"))
				return
			}
			if hitsCh != nil {
				api.debug("sending hits=%v", hits)
				hitsCh <- hits
				hitsCh = nil
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				result.err = fmt.Errorf("reading response: %w", err)
				return
			}
			items := gjson.Get(string(body), "items").Array()
      if len(items) == 0 {
        items = gjson.Get(string(body), "feed.entry").Array()
      }

			for _, item := range items {
				result.Ch <- item
			}

			// No results or empty search-after-header indicates pagination is done
			searchAfter = resp.Header.Get("cmr-search-after")
			if searchAfter == "" || len(items) == 0 {
				return
			}
		}
	}()

	// Block until we've retrieved the number of hits from the header. This gives
	// the client a chance to react to the number of hits before scrolling results
	result.hits = <-hitsCh

	return result, nil
}

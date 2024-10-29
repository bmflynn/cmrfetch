package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/bmflynn/cmrfetch/internal/log"
	"github.com/tidwall/gjson"
)

var (
	defaultCMRURL       = "https://cmr.earthdata.nasa.gov"
	defaultCMRSearchURL = defaultCMRURL + "/search"
)

type CMRSearchAPI struct {
	url      string
	client   *http.Client
	pageSize int
}

func NewCMRSearchAPI() *CMRSearchAPI {
	return &CMRSearchAPI{
		url:      defaultCMRSearchURL,
		client:   http.DefaultClient,
		pageSize: 200,
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
		Ch: make(chan T),
		mu: &sync.Mutex{},
	}
}

func (r *ScrollResult[T]) setErr(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.err = err
}

func (r *ScrollResult[T]) Err() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.err
}

func (r ScrollResult[T]) Hits() int {
	return r.hits
}

func (api *CMRSearchAPI) Get(ctx context.Context, url string) (ScrollResult[gjson.Result], error) {
	log.Debug("method=GET url=%s", url)

	result := newScrollResult[gjson.Result]()

	hitsCh := make(chan int, 1)
	go func() {
		defer close(result.Ch)
		defer close(hitsCh)

		var searchAfter string
		for {
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				result.setErr(fmt.Errorf("create request: %w", err))
				return
			}
			if searchAfter != "" {
				req.Header.Set("cmr-search-after", searchAfter)
			}
			resp, err := api.client.Do(req)
			if err != nil {
				result.setErr(fmt.Errorf("protocol error: %w", err))
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				result.setErr(api.newCMRError(resp))
				return
			}

			hits, err := strconv.Atoi(resp.Header.Get("cmr-hits"))
			if err != nil {
				result.setErr(fmt.Errorf("failed to parse cmr-hits header as int: %s", resp.Header.Get("cmr-hits")))
				return
			}
			if hitsCh != nil {
				hitsCh <- hits
				hitsCh = nil // set hits to nil so we don't send again
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				result.setErr(fmt.Errorf("reading response: %w", err))
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

func (api *CMRSearchAPI) newCMRError(resp *http.Response) error {
	cmrErr := &CMRError{
		Status:    resp.Status,
		RequestID: resp.Header.Get("cmr-request-id"),
	}
	// attempt to unmarshal what we think errors from CMR should look like
	body, _ := io.ReadAll(resp.Body)
	errs := struct {
		Errors []string `json:"errors"`
	}{}
	if err := json.Unmarshal(body, &errs); err == nil {
		cmrErr.Err = fmt.Errorf("%s", strings.Join(errs.Errors, "; "))
	} else {
		log.Debug("failed to unmarshal errors: %s: %s", err, body)
	}
	return cmrErr
}

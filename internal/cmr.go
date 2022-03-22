package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"time"
)

var defaultCMRAPIURL = "https://cmr.earthdata.nasa.gov/search"

const dataRel = "http://esipfed.org/ns/fedsearch/1.1/data#"

func init() {
	if s, ok := os.LookupEnv("EARTHDATA_CMR_API"); ok {
		defaultCMRAPIURL = s
	}
}

type Link struct {
	Rel  string `json:"rel"`
	Type string `json:"type"`
	HREF string `json:"href"`
}

type Granule struct {
	ID                string    `json:"id"`
	ProducerGranuleID string    `json:"producer_granule_id"` // a.k.a., filename
	Links             []Link    `json:"links"`
	Updated           time.Time `json:"updated"`
}

// DownloadURL attempts to locate the download url in our links by matching the file
// extension.
func (g Granule) DownloadURL() string {
	ext := filepath.Ext(g.ProducerGranuleID)
	for _, link := range g.Links {
		if link.Rel == dataRel && filepath.Ext(link.HREF) == ext {
			return link.HREF
		}
	}
	return ""
}

type Collection struct {
	ID        string    `json:"id"`
	ShortName string    `json:"short_name"`
	Version   string    `json:"version_id"`
	DatasetID string    `json:"dataset_id"`
	Platforms []string  `json:"platforms"`
	Links     []Link    `json:"links"`
	Updated   time.Time `json:"updated"`
}

func formatTime(t time.Time) string { return t.Format("2006-01-02T15:04:05Z") }

type Iter struct {
	current string
	err     error

	URL *url.URL
	Ch  chan json.RawMessage
}

func (i Iter) Err() error { return i.err }

type CMRAPI struct {
	url *url.URL

	client *http.Client
}

func NewCMRAPI() CMRAPI {
	u, err := url.Parse(defaultCMRAPIURL)
	if err != nil {
		panic("invalid CMR URL: " + defaultCMRAPIURL)
	}
	return CMRAPI{
		url:    u,
		client: http.DefaultClient,
	}
}

func (api *CMRAPI) get(u *url.URL, x interface{}) error {
	// do the query
	resp, err := api.client.Get(u.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected status 200, got %d for %s", resp.StatusCode, u)
	}

	return json.NewDecoder(resp.Body).Decode(x)
}

func (api *CMRAPI) scroll(u *url.URL, pageSize int) Iter {

	qry := u.Query()
	qry.Set("page_size", "500")
	u.RawQuery = qry.Encode()

	iter := Iter{
		URL: u,
		Ch:  make(chan json.RawMessage),
	}

	go func() {
		defer close(iter.Ch)

		for {
			// do the query
			req, err := http.NewRequest("GET", u.String(), nil)
			if err != nil {
				iter.err = err
				return
			}

			// set request header if it has been previously set
			if iter.current != "" {
				req.Header.Set("CMR-Search-After", iter.current)
			}

			resp, err := api.client.Do(req)
			if err != nil {
				iter.err = err
				return
			}
			if resp.StatusCode != http.StatusOK {
				iter.err = fmt.Errorf("expected 200, got %d", resp.StatusCode)
				return
			}

			// empty search-after header means results should be empty and we're done
			iter.current = resp.Header.Get("CMR-Search-After")
			if iter.current == "" {
				return
			}

			var doc struct {
				Feed struct {
					Entry json.RawMessage `json:"entry"`
				} `json:"feed"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
				iter.err = err
				resp.Body.Close()
				return
			}
			resp.Body.Close()

			iter.Ch <- doc.Feed.Entry
		}
	}()

	return iter
}

func (api *CMRAPI) Granules(conceptID string, temporal []time.Time, since *time.Time) (<-chan Granule, <-chan error) {
	// compile the URL
	u, _ := url.Parse(api.url.String())
	u.Path = path.Join(u.Path, "granules.json")
	qry := url.Values{}
	qry.Add("collection_concept_id", conceptID)
	if len(temporal) == 2 {
		qry.Add("temporal", fmt.Sprintf("%s,%s", formatTime(temporal[0]), formatTime(temporal[1])))
	}
	if since != nil {
		qry.Add("updated_since", formatTime(*since))
	}
	u.RawQuery = qry.Encode()

	ch := make(chan Granule, 1)
	errs := make(chan error, 1)
	go func() {
		defer close(ch)
		defer close(errs)

		iter := api.scroll(u, 500)
		for raw := range iter.Ch {
			granules := []Granule{}
			if err := json.Unmarshal(raw, &granules); err != nil {
				errs <- err
				return
			}
			for _, g := range granules {
				ch <- g
			}
		}
	}()

	return ch, errs
}

func (api *CMRAPI) Collection(provider, shortName, version string) (Collection, error) {
	// compile the URL
	u, _ := url.Parse(api.url.String())
	u.Path = path.Join(u.Path, "collections.json")
	qry := url.Values{}
	if provider != "" {
		qry.Add("provider", provider)
	}
	if shortName != "" {
		qry.Add("short_name", shortName)
	}
	if version != "" {
		qry.Add("version", version)
	}
	u.RawQuery = qry.Encode()

	var zult struct {
		Feed struct {
			Entry []Collection `json:"entry"`
		} `json:"feed"`
	}
	if err := api.get(u, &zult); err != nil {
		return Collection{}, err
	}

	if len(zult.Feed.Entry) == 0 {
		return Collection{}, fmt.Errorf("not found")
	}

	// Sort by updated
	sort.Slice(zult.Feed.Entry, func(i, j int) bool {
		return zult.Feed.Entry[i].Updated.Before(zult.Feed.Entry[j].Updated)
	})

	return zult.Feed.Entry[0], nil
}

func (api *CMRAPI) Collections(provider, shortName string) (<-chan Collection, <-chan error) {
	// compile the URL
	u, _ := url.Parse(api.url.String())
	u.Path = path.Join(u.Path, "collections.json")
	qry := url.Values{}
	if provider != "" {
		qry.Add("provider", provider)
	}
	if shortName != "" {
		qry.Add("short_name", shortName)
	}
	u.RawQuery = qry.Encode()

	ch := make(chan Collection, 1)
	errs := make(chan error, 1)
	go func() {
		defer close(ch)
		defer close(errs)

		iter := api.scroll(u, 500)
		for raw := range iter.Ch {
			collections := []Collection{}
			if err := json.Unmarshal(raw, &collections); err != nil {
				errs <- err
				return
			}
			for _, c := range collections {
				ch <- c
			}
		}
	}()
	return ch, errs
}

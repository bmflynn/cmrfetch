package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/apex/log"
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
	ID        string   `json:"id"`
	ShortName string   `json:"short_name"`
	Version   string   `json:"version_id"`
	DatasetID string   `json:"dataset_id"`
	Platforms []string `json:"platforms"`
	Links     []Link   `json:"links"`
}

func formatTime(t time.Time) string { return t.Format("2006-01-02T15:04:05Z") }

func decodeErrors(r io.Reader) string {
	zult := struct {
		Errors []string `json:"errors"`
	}{}
	if err := json.NewDecoder(r).Decode(&zult); err != nil {
		log.WithError(err).Debugf("unable to decode errors")
		return "unknown error"
	}
	return strings.Join(zult.Errors, "; ")
}

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

	if resp.StatusCode == http.StatusBadRequest {
		return fmt.Errorf("bad request: %s", decodeErrors(resp.Body))
	} else if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected status 200, got %d for %s", resp.StatusCode, u)
	}

	return json.NewDecoder(resp.Body).Decode(x)
}

func (api *CMRAPI) scroll(u *url.URL, pageSize int) ([]json.RawMessage, error) {

	qry := u.Query()
	qry.Set("page_size", "500")
	u.RawQuery = qry.Encode()

	current := ""
	data := []json.RawMessage{}

	for {
		// do the query
		req, err := http.NewRequest("GET", u.String(), nil)
		if err != nil {
			return data, fmt.Errorf("creating request: %w", err)
		}

		// set request header if it has been previously set
		if current != "" {
			req.Header.Set("CMR-Search-After", current)
		}

		resp, err := api.client.Do(req)
		if err != nil {
			return data, err
		}
		if resp.StatusCode == http.StatusBadRequest {
			return nil, fmt.Errorf("bad request: %s", decodeErrors(resp.Body))
		} else if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("expected status 200, got %d for %s", resp.StatusCode, u)
		}

		// empty search-after header means results should be empty and we're done
		current = resp.Header.Get("CMR-Search-After")
		if current == "" {
			return data, nil
		}

		var doc struct {
			Feed struct {
				Entry json.RawMessage `json:"entry"`
			} `json:"feed"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
			resp.Body.Close()
			return data, err
		}
		resp.Body.Close()

		data = append(data, doc.Feed.Entry)
	}
}

func (api *CMRAPI) Granules(conceptID string, temporal []time.Time, since *time.Time) ([]Granule, error) {
	if conceptID == "" {
		return nil, fmt.Errorf("concept id is required")
	}
	// compile the URL
	u, _ := url.Parse(api.url.String())
	u.Path = path.Join(u.Path, "granules.json")
	qry := url.Values{}
	qry.Add("collection_concept_id", conceptID)
	if len(temporal) == 2 {
		qry.Add("temporal", fmt.Sprintf("%v,%v", formatTime(temporal[0]), formatTime(temporal[1])))
	}
	if since != nil {
		qry.Add("updated_since", formatTime(*since))
	}
	u.RawQuery = qry.Encode()
	log.Debug(u.String())

	data, err := api.scroll(u, 500)
	if err != nil {
		return nil, err
	}

	granules := make([]Granule, 0, len(data))
	for _, raw := range data {
		fragment := []Granule{}
		if err := json.Unmarshal(raw, &granules); err != nil {
			return granules, fmt.Errorf("decoding %s: %w", string(raw), err)
		}
		granules = append(granules, fragment...)
	}

	// Sort by updated
	sort.Slice(granules, func(i, j int) bool {
		return granules[i].Updated.After(granules[j].Updated)
	})

	return granules, nil
}

func (api *CMRAPI) Collection(provider, shortName, version string) (Collection, error) {
	// compile the URL
	u, _ := url.Parse(api.url.String())
	u.Path = path.Join(u.Path, "collections.json")
	qry := url.Values{}
	qry.Set("sort_key[]", "-revision_date")
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
	log.Debug(u.String())

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
		return strings.Compare(zult.Feed.Entry[i].ShortName, zult.Feed.Entry[j].ShortName) < 0
	})

	return zult.Feed.Entry[0], nil
}

func (api *CMRAPI) Collections(provider, shortName string) ([]Collection, error) {
	// compile the URL
	u, _ := url.Parse(api.url.String())
	u.Path = path.Join(u.Path, "collections.json")
	qry := url.Values{}
	qry.Set("sort_key[]", "-revision_date")
	if provider != "" {
		qry.Add("provider", provider)
	}
	if shortName != "" {
		qry.Add("short_name", shortName)
	}
	u.RawQuery = qry.Encode()
	log.Debug(u.String())

	data, err := api.scroll(u, 500)
	if err != nil {
		return nil, err
	}

	collections := make([]Collection, 0, len(data))
	for _, raw := range data {
		fragment := []Collection{}
		if err := json.Unmarshal(raw, &collections); err != nil {
			return collections, fmt.Errorf("decoding %s: %w", string(raw), err)
		}
		collections = append(collections, fragment...)
	}

	// Sort by updated
	sort.Slice(collections, func(i, j int) bool {
		return strings.Compare(collections[i].ShortName, collections[j].ShortName) < 0
	})

	return collections, nil
}

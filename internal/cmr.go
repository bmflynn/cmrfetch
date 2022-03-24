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
	"strings"
	"time"

	"github.com/apex/log"
)

var defaultCMRAPIURL = "https://cmr.earthdata.nasa.gov/search"

type URL struct {
	URL            string
	Type           string
	MimeType       string
	URLContentType string
	Description    string
}

type Date struct {
	Date time.Time
	Type string
}

type CollectionRef struct {
	ShortName string
	Version   string
}

type Identifier struct {
	Identifier     string
	IdentifierType string
}

type Checksum struct {
	Algorithm string
	Value     string
}

type ArchiveInfo struct {
	Name        string
	SizeInBytes int64
	Checksum    Checksum
}

type RangeDateTime struct {
	BeginningDateTime time.Time
	EndingDateTime    time.Time
}

type DataGranule struct {
	DayNightFlag                      string
	Identifiers                       []Identifier
	ProductionDateTime                time.Time
	ArchiveAndDistributionInformation []ArchiveInfo
}

type Meta struct {
	RevisionID   int       `json:"revision-id"`
	Deleted      bool      `json:"deleted"`
	ProviderID   string    `json:"provider-id"`
	ConceptID    string    `json:"concept-id"`
	NativeID     string    `json:"native-id"`
	RevisionDate time.Time `json:"revision-date"`
}

type Granule struct {
	Meta                Meta
	GranuleUR           string
	SpatialExtent       map[string]interface{}
	ProviderDates       []Date
	RelatedUrls         []URL
	CollectionReference CollectionRef
	DataGranule         DataGranule
	TemporalExtent      struct {
		RangeDateTime RangeDateTime
	}
}

func (g Granule) Name() string {
	for _, i := range g.DataGranule.Identifiers {
		if i.IdentifierType == "ProducerGranuleId" {
			return i.Identifier
		}
	}
	return ""
}

func (g Granule) DownloadURL() string {
	name := g.Name()
	if name == "" {
		return ""
	}

	ext := filepath.Ext(name)
	for _, url := range g.RelatedUrls {
		if url.Type == "GET DATA" && filepath.Ext(url.URL) == ext {
			return url.URL
		}
	}

	return ""
}

func (g Granule) FindChecksum(name string) *Checksum {
	for _, info := range g.DataGranule.ArchiveAndDistributionInformation {
		if info.Name == name {
			return &info.Checksum
		}
	}
	return nil
}

type Instrument struct {
	ShortName string
	LongName  string
}

type Platform struct {
	ShortName   string
	LongName    string
	Type        string
	Instruments []Instrument
}

type Collection struct {
	Meta               Meta
	SpatialExtent      map[string]json.RawMessage
	CollectionProgress string
	ScienceKeywords    []map[string]string
	TemporalExtents    []map[string]json.RawMessage
	ProcessingLevel    struct {
		ID string
	}
	DOI struct {
		Authority string
		DOI       string
	}
	ShortName          string
	EntryTitle         string
	RelatedUrls        []URL
	Abstract           string
	VersionDescription string
	Version            string
	UserConstraints    map[string]json.RawMessage
	CollectionDataType string
	DataCenters        []json.RawMessage
}

type ummResponse struct {
	Hits   int             `json:"hits"`
	Took   int             `json:"took"`
	Items  json.RawMessage `json:"items"`
	Errors []string        `json:"errors"`
}

type collectionResponse struct {
	Meta Meta       `json:"meta"`
	UMM  Collection `json:"umm"`
}

type granuleResponse struct {
	Meta Meta    `json:"meta"`
	UMM  Granule `json:"umm"`
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

		zult := ummResponse{}
		if err := json.NewDecoder(resp.Body).Decode(&zult); err != nil {
			resp.Body.Close()
			return data, err
		}
		resp.Body.Close()

		data = append(data, zult.Items)
	}
}

func (api *CMRAPI) Granules(conceptID string, temporal []time.Time, since *time.Time) ([]Granule, error) {
	if conceptID == "" {
		return nil, fmt.Errorf("concept id is required")
	}
	// compile the URL
	u, _ := url.Parse(api.url.String())
	u.Path = path.Join(u.Path, "granules.umm_json")
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
		zults := []granuleResponse{}
		if err := json.Unmarshal(raw, &zults); err != nil {
			return granules, fmt.Errorf("decoding %s: %w", string(raw), err)
		}
		for _, z := range zults {
			z.UMM.Meta = z.Meta
			g := z.UMM
			g.Meta = z.Meta
			granules = append(granules, g)
		}
	}

	return granules, nil
}

func (api *CMRAPI) Collection(shortName, version string) (Collection, error) {
	// compile the URL
	u, _ := url.Parse(api.url.String())
	u.Path = path.Join(u.Path, "collections.umm_json")
	qry := url.Values{}
	qry.Set("sort_key[]", "short_name")
	if shortName != "" {
		qry.Add("short_name", shortName)
	}
	if version != "" {
		qry.Add("version", version)
	}
	u.RawQuery = qry.Encode()
	log.Debug(u.String())

	data, err := api.scroll(u, 500)
	if err != nil {
		return Collection{}, err
	}

	collections := make([]Collection, 0, len(data))
	for _, raw := range data {
		zults := []collectionResponse{}
		if err := json.Unmarshal(raw, &collections); err != nil {
			return Collection{}, fmt.Errorf("decoding %s: %w", string(raw), err)
		}
		for _, z := range zults {
			z.UMM.Meta = z.Meta
			collections = append(collections, z.UMM)
		}
	}

	if len(collections) == 0 {
		return Collection{}, fmt.Errorf("not found")
	}

	return collections[0], nil
}

func (api *CMRAPI) Collections(provider, shortName string) ([]Collection, error) {
	// compile the URL
	u, _ := url.Parse(api.url.String())
	u.Path = path.Join(u.Path, "collections.umm_json")
	qry := url.Values{}
	qry.Set("sort_key[]", "short_name")
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

	collections := []Collection{}
	for _, raw := range data {
		zults := []collectionResponse{}
		if err := json.Unmarshal(raw, &zults); err != nil {
			return collections, fmt.Errorf("decoding %s: %w", string(raw), err)
		}
		for _, z := range zults {
			z.UMM.Meta = z.Meta
			collections = append(collections, z.UMM)
		}
	}

	return collections, nil
}

func init() {
	if s, ok := os.LookupEnv("EARTHDATA_CMR_API"); ok {
		defaultCMRAPIURL = s
	}
}

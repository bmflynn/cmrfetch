package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/bmflynn/cmrfetch/internal/log"
	"github.com/tidwall/gjson"
)

func joinFloats(vals []float64) string {
	s := []string{}
	for _, v := range vals {
		s = append(s, fmt.Sprintf("%v", v))
	}
	return strings.Join(s, ",")
}

// SearchGranuleParams is a builder for collection search query params
type SearchGranuleParams struct {
	daynight      string
	shortnames    []string
	filenames     []string
	collectionIDs []string
	nativeIDs     []string
	boundingBox   []float64
	point         []float64
	circle        []float64
	polygon       []float64
	versions      []string

	timerangeStart *time.Time
	timerangeEnd   *time.Time
}

func NewSearchGranuleParams() *SearchGranuleParams {
	return &SearchGranuleParams{}
}

func (p *SearchGranuleParams) DayNightFlag(name string) *SearchGranuleParams {
	p.daynight = name
	return p
}

func (p *SearchGranuleParams) ShortNames(name ...string) *SearchGranuleParams {
	p.shortnames = name
	return p
}

func (p *SearchGranuleParams) Versions(version ...string) *SearchGranuleParams {
	p.versions = version
	return p
}

func (p *SearchGranuleParams) Filenames(name ...string) *SearchGranuleParams {
	p.filenames = name
	return p
}

func (p *SearchGranuleParams) Collections(id ...string) *SearchGranuleParams {
	p.collectionIDs = id
	return p
}

func (p *SearchGranuleParams) NativeIDs(id ...string) *SearchGranuleParams {
	p.nativeIDs = id
	return p
}

func (p *SearchGranuleParams) BoundingBox(vals []float64) *SearchGranuleParams {
	p.boundingBox = vals
	return p
}

func (p *SearchGranuleParams) Point(vals []float64) *SearchGranuleParams {
	p.point = vals
	return p
}

func (p *SearchGranuleParams) Circle(vals []float64) *SearchGranuleParams {
	p.circle = vals
	return p
}

func (p *SearchGranuleParams) Polygon(vals []float64) *SearchGranuleParams {
	p.polygon = vals
	return p
}

func (p *SearchGranuleParams) Timerange(start time.Time, end *time.Time) *SearchGranuleParams {
	p.timerangeStart = &start
	p.timerangeEnd = end
	return p
}

func (p *SearchGranuleParams) build() (url.Values, error) {
	query := url.Values{}
	if p.daynight != "" {
		query.Set("day_night_flag", p.daynight)
	}
	if p.shortnames != nil {
		query.Set("options[short_name][pattern]", "true")
		query.Set("options[short_name][ignore_case]", "true")
		for _, name := range p.shortnames {
			query.Add("short_name", name)
		}
	}
	if p.filenames != nil {
		for _, name := range p.filenames {
			query.Add("readable_granule_name", name)
		}
	}
	if p.collectionIDs != nil {
		for _, name := range p.collectionIDs {
			query.Add("collection_concept_id", name)
		}
	}
	if p.nativeIDs != nil {
		for _, name := range p.nativeIDs {
			query.Add("native_id", name)
		}
	}
	if p.timerangeStart != nil {
		s := p.timerangeStart.Format(time.RFC3339) + ","
		if p.timerangeEnd != nil {
			s += p.timerangeEnd.Format(time.RFC3339)
		}
		query.Set("temporal", s)
	}
	if len(p.polygon) > 0 {
		if len(p.polygon)%2 != 0 {
			return query, fmt.Errorf("number of polygon points must be divisible by 2")
		}
		query.Set("polygon", joinFloats(p.polygon))
	}
	if len(p.circle) != 0 {
		if len(p.circle) != 3 {
			return query, fmt.Errorf("wrong number of values for circle")
		}
		query.Set("circle", joinFloats(p.circle))
	}
	if len(p.boundingBox) != 0 {
		if len(p.boundingBox) != 4 {
			return query, fmt.Errorf("wrong number of values for bounding box")
		}
		query.Set("bounding_box", joinFloats(p.boundingBox))
	}
	if len(p.point) != 0 {
		if len(p.point) != 4 {
			return query, fmt.Errorf("wrong number of values for point")
		}
		query.Set("point", joinFloats(p.point))
	}
	if len(p.versions) > 0 {
		for _, version := range p.versions {
			query.Add("version", version)
		}
	}
	query.Set("sort_key", "-start_date")
	return query, nil
}

type Granule struct {
	Name          string            `json:"name"`
	Size          string            `json:"size"`
	Checksum      string            `json:"checksum"`
	ChecksumAlg   string            `json:"checksum_alg"`
	GetDataURL    string            `json:"download_url"`
	GetDataDAURL  string            `json:"download_direct_url"`
	NativeID      string            `json:"native_id"`
	RevisionID    string            `json:"revision_id"`
	ConceptID     string            `json:"concept_id"`
	Collection    string            `json:"collection"`
	DayNightFlag  string            `json:"daynight"`
	TimeRange     []string          `json:"timerange"`
	BoundingBox   []string          `json:"boundingbox"`
	ProviderDates map[string]string `json:"provider_dates"`
}

func findDownloadURLs(zult *gjson.Result, directAccess bool) map[string]string {
	typeKey := "GET DATA"
	if directAccess {
		typeKey = "GET DATA VIA DIRECT ACCESS"
	}
	urls := map[string]string{}
	for _, dat := range zult.Get("umm.RelatedUrls").Array() {
		url := dat.Get("URL").String()
		typ := dat.Get("Type").String()
		if typ != typeKey {
			continue
		}
		name := path.Base(url)
		urls[name] = url
	}
	return urls
}

func newGranulesFromUMM(zult gjson.Result) []Granule {
	granules := []Granule{}

	for _, gran := range findGranules(zult) {
		gran.ConceptID = zult.Get("meta.concept-id").String()
		gran.NativeID = zult.Get("meta.native-id").String()
		gran.RevisionID = zult.Get("meta.revision-id").String()
		col := zult.Get("umm.CollectionReference")
		if col.Exists() {
			gran.Collection = fmt.Sprintf(
				"%s/%s",
				col.Get("ShortName").String(),
				col.Get("Version").String(),
			)
		}

		gran.DayNightFlag = zult.Get("umm.DataGranule.DayNightFlag").String()
		gran.TimeRange = []string{
			zult.Get("umm.TemporalExtent.RangeDateTime.BeginningDateTime").String(),
			zult.Get("umm.TemporalExtent.RangeDateTime.EndingDateTime").String(),
		}
		gran.BoundingBox = []string{}
		for _, polygon := range zult.Get("umm.SpatialExtent.HorizontalSpatialDomain.Geometry.GPolygons").Array() {
			points := []string{}
			for _, point := range polygon.Get("Boundary.Points").Array() {
				points = append(points, fmt.Sprintf("%v", point.Get("Longitude").Float()))
				points = append(points, fmt.Sprintf("%v", point.Get("Latitude").Float()))
			}
			gran.BoundingBox = append(gran.BoundingBox, strings.Join(points, ","))
		}

		gran.ProviderDates = map[string]string{}
		for _, dt := range zult.Get("umm.ProviderDates").Array() {
			gran.ProviderDates[dt.Get("Type").String()] = dt.Get("Date").String()
		}
		granules = append(granules, gran)
	}

	return granules
}

func (api *CMRSearchAPI) SearchGranules(ctx context.Context, params *SearchGranuleParams) (ScrollResult[Granule], error) {
	query, err := params.build()
	if err != nil {
		return ScrollResult[Granule]{}, err
	}
	query.Set("page_size", fmt.Sprintf("%v", api.pageSize))
	url := fmt.Sprintf("%s/granules.umm_json?%s", defaultCMRSearchURL, query.Encode())

	zult, err := api.Get(ctx, url)
	if err != nil {
		return ScrollResult[Granule]{}, err
	}

	gzult := newScrollResult[Granule]()
	// hits is set before Get returns
	gzult.hits = zult.hits

	go func() {
		defer close(gzult.Ch)
		for gj := range zult.Ch {
			for _, gran := range newGranulesFromUMM(gj) {
				gzult.Ch <- gran
			}
		}
	}()

	return gzult, nil
}

type GranuleResult = ScrollResult[Granule]

type archiveInfo struct {
	Size        string
	Checksum    string
	ChecksumAlg string
}

func (a *archiveInfo) String() string {
	if b, err := json.Marshal(&a); err == nil {
		return string(b)
	}
	return "archiveInfo{ <unmarshal error> }"
}

var _ fmt.Stringer = (*archiveInfo)(nil)

// decodeArchiveInfo parses Size, Checksum and ChecksumAlg out of an array of archive info, iff
// the archive info has a name and it matches the download url name.
//
// If there are multiple archive infos with a matching name the first available values will be
// used in combination for size and checksum.
func decodeArchiveInfo(docs []gjson.Result) map[string]archiveInfo {
	infos := map[string]archiveInfo{}

	for _, ar := range docs {

		// Have to have a name
		name := ar.Get("Name").String()
		if name == "" {
			log.Debug("skipping archive info w/o name: %v", ar.Str)
			continue
		}

		var info archiveInfo
		if x, ok := infos[name]; ok {
			info = x
		} else {
			info = archiveInfo{}
		}

		if info.Size == "" {
			// Either SizeInBytes or Size w/ SizeUnit; SizeInBytes takes precedence
			sizeInBytes := ar.Get("SizeInBytes").Int()
			size := ar.Get("Size").Int()
			if sizeInBytes != 0 {
				info.Size = ByteCountSI(sizeInBytes)
			} else if size != 0 {
				info.Size = strings.TrimSpace(fmt.Sprintf("%v %v", size, ar.Get("SizeUnit").String()))
			}
		}

		if info.Checksum == "" {
			info.Checksum = strings.TrimSpace(ar.Get("Checksum.Value").String())
			info.ChecksumAlg = strings.TrimSpace(ar.Get("Checksum.Algorithm").String())
		}

		infos[name] = info
	}

	return infos
}

// Find all files contained in item that have either a download url.
//
// The download url may be of type GET DATA or GET DATA Via DIRECT DOWNLOAD. Archive info
// is added if there exists archive info metadata with the same basename as the download
// URL.
func findGranules(item gjson.Result) []Granule {
	files := map[string]Granule{}
	for name, url := range findDownloadURLs(&item, false) {
		log.Debug("found granule name=%s type=getdata url=%s", name, url)
		if _, ok := files[name]; !ok {
			files[name] = Granule{
				Name:       name,
				GetDataURL: url,
			}
		}
	}
	for name, url := range findDownloadURLs(&item, true) {
		log.Debug("found granule name=%s type=getdata_da url=%s", name, url)
		if _, ok := files[name]; !ok {
			files[name] = Granule{
				Name:         name,
				GetDataDAURL: url,
			}
		} else {
			log.Debug("")
			file := files[name]
			file.GetDataDAURL = url
			files[name] = file
		}
	}

	archiveInfos := decodeArchiveInfo(item.Get("umm.DataGranule.ArchiveAndDistributionInformation").Array())
	for name, gran := range files {
		if info, ok := archiveInfos[name]; ok {
			log.Debug("archive info for name=%s: %s", name, info.String())
			gran.Size = info.Size
			gran.Checksum = info.Checksum
			gran.ChecksumAlg = info.ChecksumAlg
			files[name] = gran
		} else {
			log.Debug("no archive info for name=%s", name)
		}
	}

	granules := []Granule{}
	for _, f := range files {
		granules = append(granules, f)
	}

	return granules
}

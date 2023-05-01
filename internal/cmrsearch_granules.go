package internal

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

const notProvided = "Not provided"

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
	collectionIDs []string
	nativeIDs     []string
	boundingBox   []float64
	point         []float64
	circle        []float64
	polygon       []float64

	timerangeStart *time.Time
	timerangeEnd   *time.Time
}

func NewSearchGranuleParams() SearchGranuleParams {
	return SearchGranuleParams{}
}

func (p *SearchGranuleParams) DayNightFlag(name string) *SearchGranuleParams {
	p.daynight = name
	return p
}

func (p *SearchGranuleParams) ShortName(name ...string) *SearchGranuleParams {
	p.shortnames = name
	return p
}

func (p *SearchGranuleParams) Collection(id ...string) *SearchGranuleParams {
	p.collectionIDs = id
	return p
}

func (p *SearchGranuleParams) NativeID(id ...string) *SearchGranuleParams {
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
	query.Set("sort_key", "-start_date")
	return query, nil
}

type Granule struct {
	Name         string   `json:"name"`
	Size         string   `json:"size"`
	Checksum     string   `json:"checksum"`
	ChecksumAlg  string   `json:"checksum_alg"`
	GetDataURL   string   `json:"download_url"`
	GetDataDAURL string   `json:"download_direct_url"`
	NativeID     string   `json:"native_id"`
	RevisionID   string   `json:"revision_id"`
	ConceptID    string   `json:"concept_id"`
	Collection   string   `json:"collection"`
	DayNightFlag string   `json:"daynight"`
	TimeRange    []string `json:"timerange"`
	BoundingBox  []string `json:"boundingbox"`
}

var dataFileRx = regexp.MustCompile(`\.(nc|hdf|h5|dat)`)

func findDownloadURLs(zult gjson.Result, directAccess bool) []string {
	typeKey := "GET DATA"
	if directAccess {
		typeKey = "GET DATA VIA DIRECT ACCESS"
	}
	urls := []string{}
	for _, dat := range zult.Get("umm.RelatedUrls").Array() {
		url := dat.Get("URL").String()
		typ := dat.Get("Type").String()
		if typ != typeKey {
			continue
		}
		urls = append(urls, url)
	}
	return urls
}

func newGranuleFromUMM(zult gjson.Result) Granule {
	gran := Granule{}


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

	// Not sure in which case where would be multiple infos, but it is an array so we
	// just separate all values by new lines.
	for _, ar := range zult.Get("umm.DataGranule.ArchiveAndDistributionInformation").Array() {
		size := ar.Get("SizeInBytes").Int()
		if size != 0 {
			gran.Size += fmt.Sprintf("%s\n", ByteCountSI(ar.Get("SizeInBytes").Int()))
		}
		gran.Checksum += fmt.Sprintf("%s\n", ar.Get("Checksum.Value").String())
		gran.ChecksumAlg += fmt.Sprintf("%s\n", ar.Get("Checksum.Algorithm").String())
	}
	gran.Size = strings.TrimSpace(gran.Size)
	gran.Checksum = strings.TrimSpace(gran.Checksum)
	gran.ChecksumAlg = strings.TrimSpace(gran.ChecksumAlg)

	gran.GetDataURL = strings.Join(findDownloadURLs(zult, false), "\n")
	if gran.GetDataURL != "" {
		gran.Name = path.Base(gran.GetDataURL)
	}
	gran.GetDataDAURL = strings.Join(findDownloadURLs(zult, true), "\n")

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

  // Find the granule name, which may be in one of several sources
  for _, elem := range zult.Get("umm.DataGranule.Identifiers").Array() {
    if elem.Get("IdentifierType").String() == "ProducerGranuleId" {
      gran.Name = elem.Get("Identifier").String()
      break
    }
  }
  // Subideally attempt to get the name of the first file in the archive info
  if gran.Name == "" {
    gran.Name = zult.Get("umm.DataGranule.ArchiveAndDistributionInformation.0.Name").String()
  }

	return gran
}

func (api *CMRSearchAPI) SearchGranules(ctx context.Context, params SearchGranuleParams) (ScrollResult[Granule], error) {
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
	gzult.hits = zult.hits

	go func() {
		defer gzult.Close()
		for gj := range zult.Ch {
			gzult.Ch <- newGranuleFromUMM(gj)
		}
	}()

	return gzult, nil
}

type GranuleResult = ScrollResult[Granule]

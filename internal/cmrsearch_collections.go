package internal

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

type Collection map[string]string

func parseTemporalExtents(gj gjson.Result) []string {
	temporalExtents := []string{}
	for _, te := range gj.Get("umm.TemporalExtents").Array() {
		for _, rng := range te.Get("RangeDateTimes").Array() {
			start := rng.Get("BeginningDateTime").String()
			end := rng.Get("EndingDateTime").String()
			if end == "" && te.Get("EndsAtPresentFlag").Bool() {
				end = "PRESENT"
			}
			temporalExtents = append(
				temporalExtents,
				fmt.Sprintf("%s / %s", start, end),
			)
		}
		if s := te.Get("SingleDateTime"); s.Exists() {
			temporalExtents = append(temporalExtents, s.String())
		}
	}
	return temporalExtents
}

func newCollectionFromUMM(gj gjson.Result) Collection {
	col := Collection{
		"shortname":        gj.Get("umm.ShortName").String(),
		"title":            gj.Get("umm.EntryTitle").String(),
		"version":          gj.Get("umm.Version").String(),
		"concept_id":       gj.Get("meta.concept-id").String(),
		"processing_level": gj.Get("umm.ProcessingLevel.Id").String(),
		"doi":              strings.TrimSpace(gj.Get("umm.DOI.DOI").String()),
		"provider":         gj.Get("meta.provider-id").String(),
		"revision_id":      gj.Get("meta.revision-id").String(),
		"revision_date":    gj.Get("meta.revision-date").String(),
		"abstract":         gj.Get("umm.Abstract").String(),
		"data_type":        gj.Get("umm.CollectionDataType").String(),
		"temporal_extents": strings.Join(parseTemporalExtents(gj), "\n"),
	}
	instruments := []string{}
	for _, plat := range gj.Get("umm.Platforms").Array() {
		platName := plat.Get("ShortName").String()
		for _, inst := range plat.Get("Instruments").Array() {
			instName := inst.Get("ShortName").String()
			instruments = append(instruments, fmt.Sprintf("%s/%s", platName, instName))
		}
	}
	col["instruments"] = strings.Join(instruments, "\n")
	urls := []string{}
	for _, urlInfo := range gj.Get("umm.RelatedUrls").Array() {
		if urlInfo.Get("Type").String() == "VIEW RELATED INFORMATION" {
			urls = append(urls, urlInfo.Get("URL").String())
		}
	}
	col["infourls"] = strings.Join(urls, "\n")
	return col
}

// SearchCollectionParams is a builder for collection search query params
type SearchCollectionParams struct {
	keyword        string
	providers      []string
	platforms      []string
	instruments    []string
	shortnames     []string
	titlePattern   string
	updatedSince   *time.Time
	granulesAdded  *TimeRange
	cloudHosted    bool
	cloudHostedSet bool
	hasGranules    bool
	hasGranulesSet bool
	standard       bool
	standardSet    bool
	sortField      string
	dataType       string
}

func NewSearchCollectionParams() *SearchCollectionParams {
	return &SearchCollectionParams{}
}

func (p *SearchCollectionParams) Keyword(kw string) *SearchCollectionParams {
	p.keyword = kw
	return p
}

func (p *SearchCollectionParams) Providers(names ...string) *SearchCollectionParams {
	p.providers = names
	return p
}

func (p *SearchCollectionParams) ShortNames(names ...string) *SearchCollectionParams {
	p.shortnames = names
	return p
}

func (p *SearchCollectionParams) Platforms(names ...string) *SearchCollectionParams {
	p.platforms = names
	return p
}

func (p *SearchCollectionParams) Instruments(names ...string) *SearchCollectionParams {
	p.instruments = names
	return p
}

func (p *SearchCollectionParams) Title(pattern string) *SearchCollectionParams {
	p.titlePattern = pattern
	return p
}

func (p *SearchCollectionParams) UpdatedSince(t time.Time) *SearchCollectionParams {
	p.updatedSince = &t
	return p
}

func (p *SearchCollectionParams) GranulesAdded(tr TimeRange) *SearchCollectionParams {
	p.granulesAdded = &tr
	return p
}

func (p *SearchCollectionParams) CloudHosted(b bool) *SearchCollectionParams {
	p.cloudHostedSet = true
	p.cloudHosted = b
	return p
}

func (p *SearchCollectionParams) Standard(b bool) *SearchCollectionParams {
	p.standardSet = true
	p.standard = b
	return p
}

func (p *SearchCollectionParams) HasGranules(b bool) *SearchCollectionParams {
	p.hasGranulesSet = true
	p.hasGranules = b
	return p
}

func (p *SearchCollectionParams) SortBy(field string) *SearchCollectionParams {
	switch field {
	case "shortname":
		field = "short_name"
	}
	p.sortField = field
	return p
}

func (p *SearchCollectionParams) DataType(val string) *SearchCollectionParams {
	p.dataType = val
	return p
}

func (p *SearchCollectionParams) build() (url.Values, error) {
	query := url.Values{}
	if p.keyword != "" {
		if ok, err := regexp.MatchString(`^[\w\s_-]+`, p.keyword); !ok {
			if err != nil {
				panic(err)
			}
			return query, fmt.Errorf("invalid keyword")
		}
		query.Set("keyword", p.keyword)
	}
	if len(p.providers) != 0 {
		query.Set("options[provider_short_name][ignore_case]", "true")
	}
	for _, name := range p.providers {
		query.Add("provider_short_name", name)
	}
	if len(p.platforms) != 0 {
		query.Set("options[platform][pattern]", "true")
		query.Set("options[platform][ignore_case]", "true")
	}
	for _, name := range p.platforms {
		query.Add("platform", name)
	}
	if len(p.instruments) != 0 {
		query.Set("options[instrument][pattern]", "true")
		query.Set("options[instrument][ignore_case]", "true")
	}
	for _, name := range p.instruments {
		query.Add("instrument", name)
	}
	if len(p.shortnames) != 0 {
		query.Set("options[short_name][pattern]", "true")
		query.Set("options[short_name][ignore_case]", "true")
	}
	for _, name := range p.shortnames {
		query.Add("short_name", name)
	}
	if p.titlePattern != "" {
		query.Set("options[entry_title][pattern]", "true")
		query.Set("options[entry_title][ignore_case]", "true")
		query.Set("entry_title", p.titlePattern)
	}
	if p.updatedSince != nil {
		query.Set("updated_since", encodeTime(*p.updatedSince))
	}
	if p.granulesAdded != nil {
		query.Set("has_granules_revised_at", encodeTimeRange(*p.granulesAdded))
	}
	if p.cloudHostedSet {
		query.Set("cloud_hosted", fmt.Sprintf("%v", p.cloudHosted))
	}
	if p.standardSet {
		query.Set("standard_product", fmt.Sprintf("%v", p.standard))
	}
	if p.hasGranulesSet {
		query.Set("has_granules", fmt.Sprintf("%v", p.hasGranules))
	}
	if p.sortField != "" {
		query.Set("sort_key", p.sortField)
	}
	if p.dataType != "" {
		query.Set("options[collection_data_type][ignore_case]", "true")
		query.Set("collection_data_type", p.dataType)
	}
	return query, nil
}

func (api *CMRSearchAPI) SearchCollections(ctx context.Context, params *SearchCollectionParams) (ScrollResult[Collection], error) {
	query, err := params.build()
	if err != nil {
		return ScrollResult[Collection]{}, err
	}
	query.Set("page_size", fmt.Sprintf("%v", api.pageSize))
	url := fmt.Sprintf("%s/collections.umm_json?%s", defaultCMRSearchURL, query.Encode())

	zult, err := api.Get(ctx, url)
	// FIXME: Get never returns an error
	if err != nil {
		return ScrollResult[Collection]{}, err
	}
	if zult.Err() != nil {
		return ScrollResult[Collection]{}, zult.Err()
	}

	gzult := newScrollResult[Collection]()
	gzult.hits = zult.hits

	go func() {
		defer close(gzult.Ch)
		for gj := range zult.Ch {
			gzult.Ch <- newCollectionFromUMM(gj)
		}
	}()

	return gzult, nil
}

type CollectionResult = ScrollResult[Collection]

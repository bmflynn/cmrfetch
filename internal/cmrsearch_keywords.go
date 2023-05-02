package internal

import (
	"context"
	"fmt"
	"net/url"
)

type Facet struct {
	Score  float64 `json:"score"`
	Type   string  `json:"type"`
	Fields string  `json:"fields"`
	Value  string  `json:"value"`
}

func (api *CMRSearchAPI) SearchFacets(ctx context.Context, val string, types []string) (ScrollResult[Facet], error) {
	query := url.Values{}
	query.Set("q", val)
	query.Set("page_size", fmt.Sprintf("%v", api.pageSize))
	for _, typ := range types {
		query.Add("type[]", typ)
	}
	url := fmt.Sprintf("%s/autocomplete?%s", defaultCMRSearchURL, query.Encode())

	zult, err := api.Get(ctx, url)
	if err != nil {
		return ScrollResult[Facet]{}, nil
	}

	gzult := newScrollResult[Facet]()
	gzult.hits = zult.hits

	go func() {
		defer close(gzult.Ch)
		for gj := range zult.Ch {
			gzult.Ch <- Facet{
				Score:  gj.Get("score").Float(),
				Type:   gj.Get("type").String(),
				Fields: gj.Get("fields").String(),
				Value:  gj.Get("value").String(),
			}
		}
	}()

	return gzult, nil
}

package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var defaultHoldingsURL = "https://cmr.earthdata.nasa.gov/search/provider_holdings.json"

type ProviderCollection struct {
}

/*
  {
    "concept-id": "C1214600516-SCIOPS",
    "entry-title": "U6: Deadhorse active-layer thickness",
    "provider-id": "SCIOPS",
    "granule-count": 0
  },
*/

type CollectionItem struct {
	ConceptID    string
	GranuleCount int64
	Title        string
}

type Provider struct {
	ID           string
	Collections  []CollectionItem
	GranuleCount int64
}

func readHoldings(r io.Reader) ([]Provider, error) {
	var dat []struct {
		ConceptID    string `json:"concept-id"`
		Title        string `json:"entry-title"`
		ProviderID   string `json:"provider-id"`
		GranuleCount int64  `json:"granule-count"`
	}
	if err := json.NewDecoder(r).Decode(&dat); err != nil {
		return nil, err
	}
	providers := map[string]Provider{}
	for _, x := range dat {
		var prov Provider
		if _, ok := providers[x.ProviderID]; ok {
			prov = providers[x.ProviderID]
		} else {
			prov = Provider{ID: x.ProviderID, Collections: []CollectionItem{}}
		}
		prov.GranuleCount += x.GranuleCount
		prov.Collections = append(prov.Collections, CollectionItem{
			ConceptID:    x.ConceptID,
			GranuleCount: x.GranuleCount,
			Title:        x.Title,
		})
		providers[x.ProviderID] = prov
	}

	zult := []Provider{}
	for _, p := range providers {
		zult = append(zult, p)
	}
	return zult, nil
}

func getCachedProviderHoldings() ([]Provider, time.Time, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return nil, time.Time{}, err
	}
	dir = filepath.Join(dir, "cmrfetch")
	if !Exists(dir) {
		return nil, time.Time{}, nil
	}
	if !IsDir(dir) {
		return nil, time.Time{}, fmt.Errorf("expected %s to be a dir", dir)
	}
	f, err := os.Open(filepath.Join(dir, "provider_holdings.json"))
	if err != nil {
		return nil, time.Time{}, err
	}
	fi, err := f.Stat()
	if err != nil {
		return nil, time.Time{}, err
	}

	var providers []Provider
	return providers, fi.ModTime(), json.NewDecoder(f).Decode(&providers)
}

func writeCachedProviderHoldings(providers []Provider) error {
	dir, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	dir = filepath.Join(dir, "cmrfetch")
	os.MkdirAll(dir, 0o755)
	f, err := os.Create(filepath.Join(dir, "provider_holdings.json"))
	if err != nil {
		return err
	}
	return json.NewEncoder(f).Encode(providers)
}

func GetProviderHoldings() ([]Provider, error) {
	providers, mtime, err := getCachedProviderHoldings()
	if err != nil {
		return nil, fmt.Errorf("loading cached holdings: %w", err)
	}
	if len(providers) > 0 && time.Since(mtime) < time.Hour*24*30 {
		return providers, nil
	}
	resp, err := http.Get(defaultHoldingsURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	providers, err = readHoldings(resp.Body)
	if err != nil {
		return nil, err
	}
	// intentionally ignoring error
	writeCachedProviderHoldings(providers)
	return providers, err
}

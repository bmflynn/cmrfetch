package internal

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestSearchCollectionParams(t *testing.T) {
	refTime := time.Unix(0, 0).UTC()
	params := NewSearchCollectionParams()
	q, err := params.
		Keyword("foo").
		Providers("p1", "p2").
		ShortNames("s1", "s2").
		Platforms("suomi-npp", "aqua").
		Instruments("viirs", "modis").
		Title("pat?e*n").
		UpdatedSince(refTime).
		GranulesAdded(TimeRange{Start: refTime}).
		DataType("dt").
		build()
	require.NoError(t, err)

	require.Equal(t, "foo", q.Get("keyword"))
	require.Equal(t, []string{"p1", "p2"}, q["provider_short_name"])
	require.Equal(t, "true", q.Get("options[provider_short_name][ignore_case]"))
	require.Equal(t, "pat?e*n", q.Get("entry_title"))
	require.Equal(t, "true", q.Get("options[entry_title][ignore_case]"))
	require.Equal(t, "true", q.Get("options[entry_title][pattern]"))
	require.Equal(t, "1970-01-01T00:00:00Z", q.Get("updated_since"))
	require.Equal(t, "1970-01-01T00:00:00Z,", q.Get("has_granules_revised_at"))
	require.Equal(t, "dt", q.Get("collection_data_type"))

	require.Equal(t, false, params.cloudHostedSet)
	q, err = params.CloudHosted(true).build()
	require.NoError(t, err)
	require.Equal(t, "true", q.Get("cloud_hosted"))

	require.Equal(t, false, params.standardSet)
	q, err = params.Standard(true).build()
	require.NoError(t, err)
	require.Equal(t, "true", q.Get("standard_product"))

	require.Equal(t, false, params.hasGranules)
	q, err = params.HasGranules(true).build()
	require.NoError(t, err)
	require.Equal(t, "true", q.Get("has_granules"))
}

func Test_newCollectionFromUMM(t *testing.T) {
	dat, err := ioutil.ReadFile("testdata/aerdt_collection.umm_json")
	require.NoError(t, err)

	col := newCollectionFromUMM(gjson.Parse(string(dat)).Get("items.0"))

	require.Equal(t, "AERDT_L2_VIIRS_SNPP", col["shortname"], col)
	require.Equal(t, "VIIRS/SNPP Dark Target Aerosol L2 6-Min Swath 6 km", col["title"])
	require.Equal(t, "1.1", col["version"])
	require.Equal(t, "C1688453112-LAADS", col["concept_id"])
	require.Equal(t, "2", col["processing_level"])
	require.Equal(t, "10.5067/VIIRS/AERDT_L2_VIIRS_SNPP.011", col["doi"])
	require.Equal(t, "LAADS", col["provider"])
	require.Equal(t, "7", col["revision_id"])
	require.Equal(t, "2023-04-12T15:03:44.726Z", col["revision_date"])
	require.Equal(t, "The VIIRS/SNPP Dark Target Aerosol L2 6-Min Swath 6 km product provides satellite-derived measurements of Aerosol Optical Thickness (AOT) and their properties over land and ocean, and spectral AOT and their size parameters over oceans every 6 minutes, globally.  The Suomi National Polar-orbiting Partnership (SNPP) Visible Infrared Imaging Radiometer Suite (VIIRS) incarnation of the dark target (DT) aerosol product is based on the same DT algorithm that was developed and used to derive products from the Terra and Aqua mission&#8217;s MODIS instruments.  Two separate and distinct DT algorithms exist.  One helps retrieve aerosol information over ocean (dark in visible and longer wavelengths), while the second aids retrievals over vegetated/dark-soiled land (dark in the visible).\r\n\r\nThis orbit-level product (Short-name: AERDT_L2_VIIRS_SNPP) has an at-nadir resolution of 6 km x 6 km, and progressively increases away from nadir given the sensor&#8217;s scanning geometry and Earth&#8217;s curvature.  Viewed differently, this product&#8217;s resolution accommodates 8 x 8 native VIIRS moderate-resolution (M-band) pixels that nominally have ~750 m horizontal pixel size.  Hence, the L2 DT AOT data product incorporates 64 (750 m) pixels over a 6-minute acquisition.\r\n\r\nIn contrast to collection 1 of this product, this collection 1.1 uses bowtie-restored pixels that are used for both cloud masking and for performing some retrievals using the restored pixels as well.\r\n\r\nFor more information consult LAADS product description page at:\r\n\r\nhttps://ladsweb.modaps.eosdis.nasa.gov/missions-and-measurements/products/AERDT_L2_VIIRS_SNPP\r\n\r\nOr, Dark Target aerosol team Page at: \r\nhttps://darktarget.gsfc.nasa.gov/", col["abstract"])
	require.Equal(t, "SCIENCE_QUALITY", col["data_type"])
	require.Equal(t, "Suomi-NPP/VIIRS", col["instruments"])
	require.Equal(t, "https://darktarget.gsfc.nasa.gov/pubs\nhttps://darktarget.gsfc.nasa.gov/sites/default/files/DT_Aerosol_UsersGuide_MODIS_VIIRS_v3.pdf\nhttps://darktarget.gsfc.nasa.gov/atbd/overview", col["infourls"])
}

func TestSearchCollections(t *testing.T) {
	newServer := func(t *testing.T, body string, status int, hits string) func() {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if hits != "" {
				w.Header().Set("cmr-hits", hits)
			}
			w.WriteHeader(status)
			w.Write([]byte(body))
		}))
		url := fmt.Sprintf("http://%s", ts.Listener.Addr())
		origURL := defaultCMRURL
		defaultCMRSearchURL = url
		return func() {
			defaultCMRSearchURL = origURL
			ts.Close()
		}
	}

	doGet := func(t *testing.T, params *SearchCollectionParams) ScrollResult[Collection] {
		t.Helper()

		api := NewCMRSearchAPI(log.Default())
		// make sure we're not waiting long
		zult, err := api.SearchCollections(context.Background(), params)
		require.NoError(t, err)

		return zult
	}

	t.Run("get", func(t *testing.T) {
		dat, err := ioutil.ReadFile("testdata/aerdt_collection.umm_json")
		require.NoError(t, err)
		require.True(t, gjson.Valid(string(dat)))

		cleanup := newServer(t, string(dat), http.StatusOK, "1880")
		defer cleanup()

		zult := doGet(t, NewSearchCollectionParams())
		require.NoError(t, zult.Err())
		require.Equal(t, 1880, zult.Hits())

		cols := []Collection{}
		for g := range zult.Ch {
			cols = append(cols, g)
		}

		require.NoError(t, zult.Err())
		require.Len(t, cols, 2)
	})
}

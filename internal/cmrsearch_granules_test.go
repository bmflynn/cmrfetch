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

func TestGranuleSearchParams(t *testing.T) {
	refTime := time.Unix(0, 0).UTC()
	params := NewSearchGranuleParams()
	q, err := params.
		DayNightFlag("day").
		ShortNames("s1", "s2").
		Filenames("f1", "f2").
		Collections("c1", "c2").
		NativeIDs("n1", "n2").
		BoundingBox([]float64{1, 2, 3, 4}).
		Point([]float64{1, 2, 3, 4}).
		Circle([]float64{1.1, 2.2, 3.3}).
		Polygon([]float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}).
		Timerange(refTime, nil).
		build()
	require.NoError(t, err)

	require.Equal(t, "day", q.Get("day_night_flag"))
	require.Equal(t, []string{"s1", "s2"}, q["short_name"])
	require.Equal(t, []string{"f1", "f2"}, q["readable_granule_name"])
	require.Equal(t, []string{"c1", "c2"}, q["collection_concept_id"])
	require.Equal(t, []string{"n1", "n2"}, q["native_id"])
	require.Equal(t, "1,2,3,4", q.Get("bounding_box"))
	require.Equal(t, "1,2,3,4", q.Get("point"))
	require.Equal(t, "1.1,2.2,3.3", q.Get("circle"))
	require.Equal(t, "1,2,3,4,5,6,7,8,9,0", q.Get("polygon"))

	require.Equal(t, "1970-01-01T00:00:00Z,", q.Get("temporal"))
	q, err = params.Timerange(refTime, &refTime).build()
	require.NoError(t, err)
	require.Equal(t, "1970-01-01T00:00:00Z,1970-01-01T00:00:00Z", q.Get("temporal"))
}

func Test_newGranuleFromUMM(t *testing.T) {
	t.Run("name lookup", func(t *testing.T) {
		expected := "I AM GRANULE NAME"
		cases := []struct {
			Name string
			Body string
		}{
			{
				"DataGranule",
				fmt.Sprintf(`
        {
          "meta": {},
          "umm": {
            "DataGranule": {
              "Identifiers": [
                {
                  "Identifier": "%s",
                  "IdentifierType": "ProducerGranuleId"
                }
              ]
            }
          }
        }
        `, expected),
			},
			{
				"ArchiveInfo",
				fmt.Sprintf(`
        {
          "meta": {},
          "umm": {
            "DataGranule": {
              "ArchiveAndDistributionInformation": [
                {
                  "Name": "%s"
                }
              ]
            }
          }
        }
        `, expected),
			},
		}

		for _, test := range cases {
			t.Run(test.Name, func(t *testing.T) {
				require.True(t, gjson.Valid(test.Body))
				zult := gjson.Parse(test.Body)
				gran := newGranuleFromUMM(zult)

				require.Equal(t, expected, gran.Name)
			})
		}
	})

	t.Run("", func(t *testing.T) {
		dat, err := ioutil.ReadFile("testdata/aerdt_granules.umm_json")
		require.NoError(t, err)
		body := string(dat)
		require.True(t, gjson.Valid(body))

		zult := gjson.Get(body, "items.0")
		require.True(t, zult.Exists())

		gran := newGranuleFromUMM(zult)

		require.Equal(t, "AERDT_L2_VIIRS_SNPP.A2023117.1654.011.nrt.nc", gran.Name)
		require.Equal(t, "7.0 MB", gran.Size)
		require.Equal(t, "3967c4c9d5768e4eff7e1b508b9011f2", gran.Checksum)
		require.Equal(t, "MD5", gran.ChecksumAlg)
		require.Equal(t, "https://sips-data.ssec.wisc.edu/nrt/47503027/AERDT_L2_VIIRS_SNPP.A2023117.1654.011.nrt.nc", gran.GetDataURL)
		require.Equal(t, "", gran.GetDataDAURL)
		require.Equal(t, "ASIPS:AERDT_L2_VIIRS_SNPP_NRT:1682614440", gran.NativeID)
		require.Equal(t, "1", gran.RevisionID)
		require.Equal(t, "G2669133699-ASIPS", gran.ConceptID)
		require.Equal(t, "AERDT_L2_VIIRS_SNPP_NRT/1.1", gran.Collection)
		require.Equal(t, "Day", gran.DayNightFlag)
		require.Equal(t, []string{"2023-04-27T16:54:00.000000Z", "2023-04-27T16:59:59.000000Z"}, gran.TimeRange)
		require.Equal(t, []string{
			"-131.310653687,66.963340759,-92.430793762,55.710681915,-37.703670502,63.907997131,-4.32655859,82.950004578,-131.310653687,66.963340759",
		}, gran.BoundingBox)
	})
}

func TestSearchGranules(t *testing.T) {
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

	doGet := func(t *testing.T, params *SearchGranuleParams) ScrollResult[Granule] {
		t.Helper()

		api := NewCMRSearchAPI(log.Default())
		// make sure we're not waiting long
		zult, err := api.SearchGranules(context.Background(), params)
		require.NoError(t, err)

		return zult
	}

	t.Run("get", func(t *testing.T) {
		dat, err := ioutil.ReadFile("testdata/aerdt_granules.umm_json")
		require.NoError(t, err)
		require.True(t, gjson.Valid(string(dat)))

		cleanup := newServer(t, string(dat), http.StatusOK, "1880")
		defer cleanup()

		zult := doGet(t, NewSearchGranuleParams())
		require.NoError(t, zult.Err())
		require.Equal(t, 1880, zult.Hits())

		granules := []Granule{}
		for g := range zult.Ch {
			granules = append(granules, g)
		}

		require.NoError(t, zult.Err())
		require.Len(t, granules, 10)
	})
}

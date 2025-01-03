package internal

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
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
		expected := "GRANULE_NAME"
		cases := []struct {
			Name string
			Body string
		}{
			{
				"DownloadURL",
				fmt.Sprintf(`
        {
          "meta": {},
          "umm": {
            "RelatedUrls": [
              {
                "URL": "https://sips-data.ssec.wisc.edu/nrt/47503027/%s",
                "Type": "GET DATA",
                "MimeType": "application/x-netcdf"
              }
            ]
          }
        }
        `, expected),
			},
		}

		for _, test := range cases {
			t.Run(test.Name, func(t *testing.T) {
				require.True(t, gjson.Valid(test.Body))
				zult := gjson.Parse(test.Body)
				grans := newGranulesFromUMM(zult)

				require.Len(t, grans, 1)

				require.Equal(t, expected, grans[0].Name)
			})
		}
	})

	t.Run("multi granule", func(t *testing.T) {
		t.Run("nominal", func(t *testing.T) {
			dat, err := os.ReadFile("testdata/aerdt_granules_multigranule1.umm_json")
			require.NoError(t, err)
			body := string(dat)
			require.True(t, gjson.Valid(body))

			zult := gjson.Get(body, "items.0")
			require.True(t, zult.Exists())

			grans := newGranulesFromUMM(zult)
			require.Len(t, grans, 2, "Expected a 2 granules, one for each url/archive info")
		})

		t.Run("ignores extra archive info", func(t *testing.T) {
			dat, err := os.ReadFile("testdata/aerdt_granules_multigranule2.umm_json")
			require.NoError(t, err)
			body := string(dat)
			require.True(t, gjson.Valid(body))

			zult := gjson.Get(body, "items.0")
			require.True(t, zult.Exists())

			grans := newGranulesFromUMM(zult)
			require.Len(t, grans, 2, "Expected a 2 granules, one for each url/archive info")
		})
	})

	t.Run("asips aerdt", func(t *testing.T) {
		dat, err := os.ReadFile("testdata/aerdt_granules.umm_json")
		require.NoError(t, err)
		body := string(dat)
		require.True(t, gjson.Valid(body))

		zult := gjson.Get(body, "items.0")
		require.True(t, zult.Exists())

		grans := newGranulesFromUMM(zult)

		require.Len(t, grans, 1, "Expected a single file for granule")
		gran := grans[0]

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
			_, err := w.Write([]byte(body))
			require.NoError(t, err)
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

		api := NewCMRSearchAPI()
		// make sure we're not waiting long
		zult, err := api.SearchGranules(context.Background(), params)
		require.NoError(t, err)

		return zult
	}

	t.Run("get", func(t *testing.T) {
		dat, err := os.ReadFile("testdata/aerdt_granules.umm_json")
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

func TestDecodeArchiveInfo(t *testing.T) {
	ar := gjson.Parse(`
    [
      {
        "Name": "CAL_LID_L1-Standard-V4-51.2016-08-31T23-21-32ZD.hdf"
      },
      {
        "Checksum": {
          "Algorithm": "MD5",
          "Value": "ffffffffffffffffffffffffffffffff"
        },
        "Name": "CAL_LID_L1-Standard-V4-51.2016-08-31T23-21-32ZD.hdf",
        "Size": 999,
        "SizeUnit": "MB"
      },
      {
        "Checksum": {
          "Algorithm": "MD5",
          "Value": "3e84cf5f8ffb0e97627ff9462cec8534"
        },
        "Name": "CAL_LID_L1-Standard-V4-51.2016-08-31T23-21-32ZD.hdf.met",
        "Size": 8.0,
        "SizeUnit": "KB"
      }
    ]
  `)
	infos := decodeArchiveInfo(ar.Array())

	require.Len(t, infos, 2)

	info := infos["CAL_LID_L1-Standard-V4-51.2016-08-31T23-21-32ZD.hdf"]
	require.Equal(t, "999 MB", info.Size)
	require.Equal(t, "MD5", info.ChecksumAlg)
	require.Equal(t, "ffffffffffffffffffffffffffffffff", info.Checksum)

	info = infos["CAL_LID_L1-Standard-V4-51.2016-08-31T23-21-32ZD.hdf.met"]
	require.Equal(t, "8 KB", info.Size)
	require.Equal(t, "MD5", info.ChecksumAlg)
	require.Equal(t, "3e84cf5f8ffb0e97627ff9462cec8534", info.Checksum)
}

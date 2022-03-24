package internal

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCollectionModel(t *testing.T) {
	t.Run("unmarshal json", func(t *testing.T) {
		dat, err := ioutil.ReadFile("testdata/collection.umm_json.json")
		require.NoError(t, err, "reading fixture data")

		zult := ummResponse{}
		err = json.Unmarshal(dat, &zult)
		require.NoError(t, err, "failed to decode response wrapper")

		items := []collectionResponse{}
		err = json.Unmarshal(zult.Items, &items)
		require.NoError(t, err, "failed to decode collection response")
	})
}

func TestGranuleModel(t *testing.T) {
	t.Run("unmarshal json", func(t *testing.T) {
		dat, err := ioutil.ReadFile("testdata/granule.umm_json.json")
		require.NoError(t, err, "reading fixture data")

		zult := ummResponse{}
		err = json.Unmarshal(dat, &zult)
		require.NoError(t, err, "failed to decode response wrapper")

		items := []granuleResponse{}
		err = json.Unmarshal(zult.Items, &items)
		require.NoError(t, err, "failed to decode granule response")
	})
}

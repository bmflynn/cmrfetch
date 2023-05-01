package internal

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUMMCollection(t *testing.T) {
	t.Run("unmarshal", func(t *testing.T) {
		dat, err := ioutil.ReadFile("testdata/aerdt_collection.umm_json")
		require.NoError(t, err)

		col := UMMCollection{}
		err = json.Unmarshal(dat, &col)
		require.NoError(t, err, "expected no error unmarshalling")
	})
}

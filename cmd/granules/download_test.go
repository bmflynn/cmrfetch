package granules

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/bmflynn/cmrfetch/internal"
	"github.com/stretchr/testify/require"
)

func Test_shouldDownload(t *testing.T) {
	tests := []struct {
		exists         bool
		checksumAlg    string
		checksum       string
		checksumErr    error
		clobber        bool
		skipByChecksum bool
		expected       bool
		name           string
	}{
		{true, "MD5", "", nil, true, false, true, "exists with clobber downloads"},
		{false, "MD5", "", nil, false, false, true, "doesn't exist downloads"},
		{true, "MD5", "", nil, false, false, false, "exists, noclobber does not download"},

		{true, "MD5", "xxx", nil, false, true, false, "exists, checksum matches, skipbychecksum, does not download"},
		{true, "MD5", "...", nil, false, true, true, "exists, checksum mismatch, skipbychecksum, downloads"},
		{true, "MD5", "xxx", fmt.Errorf("whoops"), false, true, false, "exists, checksum error, skipbychecksum, does not download"},
		{true, "BOGUS", "...", nil, false, true, false, "exists, unsupported alg, skipbychecksum, does not download"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			exister := func(string) bool { return test.exists }
			checksummer := func(string, string) (string, error) { return test.checksum, test.checksumErr }

			ok, _ := shouldDownload(&internal.DownloadRequest{
				URL:         "",
				ChecksumAlg: test.checksumAlg,
				Checksum:    "xxx",
				Dest:        "",
			}, test.clobber, test.skipByChecksum, checksummer, exister)

			require.Equal(t, test.expected, ok)
		})
	}
}

func Test_zultsToRequests(t *testing.T) {
	tmpdir, err := os.MkdirTemp("", "")
	require.NoError(t, err)

	granules := internal.GranuleResult{
		Ch: make(chan internal.Granule, 1),
	}
	granules.Ch <- internal.Granule{
		Name: "filename.ext",
	}

	req := <-zultsToRequests(granules, tmpdir, false, false)

	require.Equal(t, path.Base(req.Dest), "filename.ext")
}

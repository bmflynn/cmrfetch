package internal

import (
	"bytes"
	"crypto/md5"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitChecksum(t *testing.T) {
	_, _, err := SplitChecksum("xx")
	require.Error(t, err, "invalid format should fail")

	alg, val, err := SplitChecksum("xxx:yyy")
	require.NoError(t, err, "good format should succeed")
	require.Equal(t, "xxx", alg)
	require.Equal(t, "yyy", val)
}

func Test_newHash(t *testing.T) {
	goodCases := []string{"md5", "MD5", "sha256", "sha512"}
	for _, test := range goodCases {
		t.Run(test, func(t *testing.T) {
			hash, err := newHash(test)
			require.NotNil(t, hash)
			require.NoError(t, err)
		})
	}

	_, err := newHash("xxx")
	require.Error(t, err, "bad alg should be an error")
}

func Test_writerHasher(t *testing.T) {
	wh := writerHasher{
		Writer: bytes.NewBuffer(nil),
		hash:   md5.New(),
	}

	n, err := wh.Write([]byte("xxx"))
	require.NoError(t, err)
	require.Equal(t, 3, n)

	require.NotEmpty(t, wh.Checksum(), "checksum should be set after writing")

	t.Run("nil has", func(t *testing.T) {
		wh := writerHasher{
			Writer: bytes.NewBuffer(nil),
		}

		n, err := wh.Write([]byte("xxx"))
		require.NoError(t, err)
		require.Equal(t, 3, n)

		require.Empty(t, wh.Checksum(), "checksum should be empty when hash is nil")
	})
}

func Test_findNetrc(t *testing.T) {
	t.Run("not exist is err", func(t *testing.T) {
		t.Setenv("NETRC", "xxxx")

		_, err := findNetrc()
		require.Error(t, err, "expected error when file does not exist")
	})
	t.Run("NETRC var", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		t.Setenv("NETRC", tmpFile.Name())

		path, err := findNetrc()
		require.NoError(t, err, "expected no error looking up netrc")
		require.Equal(t, tmpFile.Name(), path, "Should have returned path to our temp file")
	})

	t.Run("HOME", func(t *testing.T) {
		dir, err := os.MkdirTemp("", "")
		require.NoError(t, err)
		defer os.RemoveAll(dir)

		path := filepath.Join(dir, ".netrc")
		err = os.WriteFile(path, []byte("xxx"), 0o644)
		require.NoError(t, err)
		t.Setenv("HOME", dir)

		gotpath, err := findNetrc()
		require.NoError(t, err, "expected no error looking up netrc")
		require.Equal(t, path, gotpath, "Should have returned path to netrc in our HOME: %s", path)
	})
}

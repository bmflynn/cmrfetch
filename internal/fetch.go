package internal

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type Fetcher func(ctx context.Context, url string, w io.Writer) (int64, error)

func ParseChecksum(s string) (string, string, error) {
	alg, val, ok := strings.Cut(strings.ToLower(s), ":")
	if !ok {
		return "", "", fmt.Errorf("expected <alg>:<val>, got %s", s)
	}
	return alg, val, nil
}

func newHash(alg string) (hash.Hash, error) {
	var hash hash.Hash
	switch strings.ToLower(alg) {
	case "md5":
		hash = md5.New()
	case "sha256":
		hash = sha256.New()
	case "shas512":
		hash = sha512.New()
	default:
		return nil, fmt.Errorf("expected one of md5, sha256, sha512, got %s", alg)
	}
	return hash, nil
}

// writerHasher wraps a writer to compute a hash as bytes are written
type writerHasher struct {
	io.Writer
	hash hash.Hash
	size int64
}

func (wh *writerHasher) Write(buf []byte) (int, error) {
	n, err := wh.Writer.Write(buf)
	if wh.hash != nil {
		wh.hash.Write(buf[:n])
	}
	wh.size += int64(n)
	return n, err
}

func (wh *writerHasher) Checksum() string { return hex.EncodeToString(wh.hash.Sum(nil)) }

// findNetrc trys to lookup .netrc(_netrc on windows) in the home dir
func findNetrc() (string, error) {
	var fpath string
	if s, ok := os.LookupEnv("NETRC"); ok {
		fpath = s
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("getting user home dir: %w", err)
		}
		name := ".netrc"
		if runtime.GOOS == "windows" {
			name = "_netrc"
		}
		fpath = filepath.Join(home, name)
	}
	_, err := os.Stat(fpath)
	if errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("%s: %w", fpath, err)
	}
	return fpath, nil
}

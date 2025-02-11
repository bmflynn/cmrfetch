package internal

import (
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
)

func ChecksumAlgSupported(alg string) bool {
	switch alg {
	case "SHA-256":
		return true
	case "SHA-384":
		return true
	case "SHA-512":
		return true
	case "MD5":
		return true
	default:
		return false
	}
}

// Checksum performs the alg checksum for the file at path. Alg must be one
// of SHA-256, SHA-384, SHA-512, or MD5.
func Checksum(alg, path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	var hash hash.Hash
	switch alg {
	case "SHA-256":
		hash = sha256.New()
	case "SHA-384":
		hash = sha512.New384()
	case "SHA-512":
		hash = sha512.New()
	case "MD5":
		hash = md5.New()
	default:
		return "", fmt.Errorf("%s checksum not supported", alg)
	}
	if _, err = io.Copy(hash, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

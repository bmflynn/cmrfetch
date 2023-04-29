package internal

import "os"

func CanWrite(path string) bool {
	if fi, err := os.Stat(path); err == nil {
		return fi.Mode()&0o300 == 0o300
	}
	return false
}

func Exists(path string) bool {
	_, err := os.Stat(path)
  return err == nil
}

func IsDir(path string) bool {
	if fi, err := os.Stat(path); err == nil {
		return fi.IsDir()
	}
	return false
}

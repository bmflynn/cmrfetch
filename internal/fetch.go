package internal

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
)

func Fetch(client *http.Client, downloadURL string, dir string) error {
	return FetchContext(context.Background(), client, downloadURL, dir)
}

func FetchContext(ctx context.Context, client *http.Client, downloadURL string, dir string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return err // shouldn't really happen
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("[%v] %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	fpath := filepath.Join(dir, path.Base(downloadURL))
	tmppath := fpath + ".tmp"
	f, err := os.OpenFile(tmppath, os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return fmt.Errorf("opening dest %s: %w", tmppath, err)
	}

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return fmt.Errorf("writing file %s: %w", tmppath, err)
	}

	return os.Rename(tmppath, fpath)
}

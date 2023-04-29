package internal

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"time"

	"github.com/jdxcode/netrc"
)

// Sets basic auth on redirect if the host is in the netrc file.
func newRedirectWithNetrcCredentials() (func(*http.Request, []*http.Request) error, error) {
	fpath, err := findNetrc()
	if err != nil {
		return nil, err
	}
	nc, err := netrc.Parse(fpath)
	if err != nil {
		return nil, fmt.Errorf("failed to read netrc: %w", err)
	}
	return func(req *http.Request, via []*http.Request) error {
		host := req.URL.Hostname()
		if machine := nc.Machine(host); machine != nil {
			req.SetBasicAuth(machine.Get("login"), machine.Get("password"))
		}
		return nil
	}, nil
}

// HTTPFetch is a Fetcher that supports basic file fetching. It supports netrc for authentication
// redirects and uses an in-memory cookie jar to save authentication cookies provided by
// authentication services such as NASA Einternal.
type HTTPFetcher struct {
	client   *http.Client
	readSize int64
}

func NewHTTPFetcher(netrc bool) (*HTTPFetcher, error) {
	client := &http.Client{
		Timeout: 20 * time.Minute,
	}
	if netrc {
		// Netrc needs a cookiejar so we don't have to do redirect everytime
		jar, err := cookiejar.New(nil)
		if err != nil {
			return nil, fmt.Errorf("creating cookiejar: %w", err)
		}
		client.Jar = jar
		client.CheckRedirect, err = newRedirectWithNetrcCredentials()
		if err != nil {
			return nil, err
		}
	}
	return &HTTPFetcher{
		client:   client,
		readSize: 2 << 19,
	}, nil
}

func (f *HTTPFetcher) newRequest(ctx context.Context, url string) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, "GET", url, nil)
}

// Fetch url to destdir using url's basename as the filename and update hash with the file
// bytes as they are read.
func (f *HTTPFetcher) Fetch(ctx context.Context, url string, w io.Writer) (int64, error) {
	req, err := f.newRequest(ctx, url)
	if err != nil {
		return 0, err
	}

	resp, err := f.client.Do(req)
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("server returned %s", resp.Status)
	}
	defer resp.Body.Close()

	var size int64
	buf := make([]byte, f.readSize)
	r := bufio.NewReader(resp.Body)
	for {
		n, rErr := r.Read(buf)
		_, wErr := w.Write(buf[:n])
		if wErr != nil {
			return size, fmt.Errorf("writing to file: %w", err)
		}

		size += int64(n)

		if errors.Is(rErr, io.EOF) {
			break
		}
		if rErr != nil {
			return size, fmt.Errorf("reading from remote: %w", rErr)
		}
	}
	return size, nil
}
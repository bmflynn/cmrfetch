package internal

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"sync"
	"time"

	"github.com/jdxcode/netrc"
)

var defaultNetrcFinder = findNetrc

type FailedDownload struct {
	RequestID    string
	ResponseBody string
	Status       string
	URL          string
}

func newFailedDownloadError(resp *http.Response) *FailedDownload {
	var body string
	dat, err := io.ReadAll(resp.Body)
	if err == nil {
		body = string(dat)
	}
	url := ""
	if resp.Request != nil {
		url = resp.Request.URL.String()
	}
	return &FailedDownload{
		RequestID:    resp.Header.Get("request-id"),
		ResponseBody: body,
		Status:       resp.Status,
		URL:          url,
	}
}

func (e *FailedDownload) Error() string {
	rid := e.RequestID
	if rid == "" {
		rid = "<unavailable>"
	}
	return fmt.Sprintf("%s requestid=%s", e.Status, rid)
}

func ResolveEDLToken(token string) string {
	// Check for token; commandline flag has priority over env var
	resolvedToken := token
	if resolvedToken == "" {
		// Check env var if commandline flag not set
		bearer, ok := os.LookupEnv("EDL_TOKEN")
		if ok && bearer != "" {
			resolvedToken = bearer
		}
	}
	return resolvedToken
}

// Sets basic auth on redirect if the host is in the netrc file.
func newRedirectWithNetrcCredentials() (func(*http.Request, []*http.Request) error, error) {
	fpath, err := defaultNetrcFinder()
	if err != nil {
		return nil, err
	}
	nc, err := netrc.Parse(fpath)
	if err != nil {
		return nil, fmt.Errorf("failed to read netrc: %w", err)
	}
	mu := &sync.Mutex{}
	return func(req *http.Request, via []*http.Request) error {
		host := req.URL.Hostname()
		mu.Lock()
		machine := nc.Machine(host)
		mu.Unlock()
		if machine != nil {
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
	// If provided an authorization header is added to every request
	bearerToken string
}

func NewHTTPFetcher(netrc bool, edlToken string) (*HTTPFetcher, error) {
	client := &http.Client{
		Timeout: 20 * time.Minute,
	}

	// Token has priority over netrc if set
	if edlToken == "" && netrc {
		// Netrc needs a cookiejar so we don't have to do redirect everytime
		jar, err := cookiejar.New(nil)
		if err != nil {
			return nil, fmt.Errorf("creating cookiejar: %w", err)
		}
		client.Jar = jar
		client.CheckRedirect, err = newRedirectWithNetrcCredentials()
		if err != nil {
			return nil, fmt.Errorf("configuring netrc token redirect: %w", err)
		}
	}
	return &HTTPFetcher{
		client:      client,
		readSize:    2 << 19,
		bearerToken: edlToken,
	}, nil
}

func (f *HTTPFetcher) newRequest(ctx context.Context, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	if f.bearerToken != "" {
		if req.URL.Scheme != "https" {
			return nil, fmt.Errorf("refusing to add bearer token to non-https url %s", req.URL)
		}
		req.Header.Add("Authorization", "Bearer "+f.bearerToken)
	}
	return req, nil
}

// Fetch url to destdir using url's basename as the filename and update hash with the file
// bytes as they are read.
func (f *HTTPFetcher) Fetch(ctx context.Context, url string, w io.Writer) (int64, error) {
	req, err := f.newRequest(ctx, url)
	if err != nil {
		return 0, err
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return 0, err
	}
	if resp.StatusCode != http.StatusOK {
		err = newFailedDownloadError(resp)
		resp.Body.Close()
		return 0, err
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

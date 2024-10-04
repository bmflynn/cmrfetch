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
var edlToken = ""

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

// Sets token auth header
func newRedirectWithToken(bearer string) (func(*http.Request, []*http.Request) error, error) {
	return func(req *http.Request, via []*http.Request) error {
		req.Header.Add("Authorization", "Bearer "+bearer)
		return nil
	}, nil
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

	if edlToken != "" {
		req.Header.Add("Authorization", "Bearer "+edlToken)
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

	// if err := validateResponse(req, resp); err != nil {
	// 	return 0, err
	// }

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

func validateResponse(req *http.Request, resp *http.Response) error {
	wanted := req.URL.Hostname()
	found := resp.Request.URL.Hostname()
	if wanted != found {
		return fmt.Errorf("probable auth redirect error; expected host %s, found %s", wanted, found)
	}
	return nil
}

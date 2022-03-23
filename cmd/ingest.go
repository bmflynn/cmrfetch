package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"sync"
	"time"

	"github.com/apex/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gitlab.ssec.wisc.edu/brucef/cmrfetch/internal"
)

var (
	ingestTemporalVal TimerangeVal
)

var Ingest = &cobra.Command{
	Use:   "cmrfetch {-c ID | -p PRODUCT}",
	Short: "Ingest files from CMR",
	Long: `Ingest granule files from NASA CMR (https://cmr.earthdata.nasa.gov).
	
This was implemented and tested using granules provided by LAADS and ASIPS, however, it
may work with other providers as well, but your mileage may vary.

Files for granules are ingested into the directory provided by --dir to temporary files
and renamed into place on successfull download. If a download fails an error file will be
created with the name of the granule and a .error extension.

On a successful granules listing state is written to --dir to keep track of the time of 
the last listing. This state is always written but only used if the --since-lastran flag
is used to limit the query to files since the last time ran.

If a file listed already exists by name in --dir it will by default be skipped. To force
the download of existing files use --clobber.

Authentication
==============
Generally, searching data from NASA CMR does not require an Earthdata login, however, in
most cases an Earthdata login is required to download data. To register for an Earthdata
account see https://urs.earthdata.nasa.gov/. 

The preferred way to provide credentials is using the environment variables EARTHDATA_USER 
and EARTHDATA_PASSWD. Both variables must be set or no credentials will be available.

You can also use a netrc file by either by giving the path directly via --netrc=<path> or 
by using -n in which case it will look for .netrc in the logged in user's home dir or ./netrc
if the user's home dir is not available.

`,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		opts, err := newIngestOpts(cmd.Flags())
		if err != nil {
			return err
		}
		if opts.Verbose {
			log.SetLevel(log.DebugLevel)
		}
		if err := doIngest(newAppContext(), opts); err != nil {
			log.WithError(err).Fatalf("ingest failed")
		}
		return nil
	},
	SilenceUsage: true,
}

func init() {
	flags := Ingest.Flags()
	flags.String("dir", "ingest", "Directory to ingest files to")
	flags.Bool("verbose", false, "Verbose output")
	flags.Bool("clobber", false, "Overwrite exiting files in --dir")
	flags.StringP("netrc", "n", "",
		"Use the netrc file at the provided path for Earthdata credentials. If provided w/o a value, "+
			"e.g., -n, try to set a reasonable default.")
	flags.Int("workers", 2, "Number of workers, up to 5")
	flags.StringP("concept-id", "c", "",
		"Concept ID of the collection the granule belongs to. See the collections sub-command "+
			"for a way to view collection concept ids for a provider.")
	flags.StringP("product", "p", "",
		"Forward slash separated provider, shortname, and version that will be used to lookup the concept id at runtime.")
	flags.BoolP(
		"since-lastran", "s",
		false,
		"Only query for granules updated since the last time run. The last time ran is determined "+
			"by the state file `last_ran` in the directory provided by --dir. If no state file exists "+
			"one will be created when a granule query returns successfully.",
	)
	flags.VarP(
		&ingestTemporalVal, "temporal", "t",
		"Comma separated granule start and end time to search over where time "+
			"format is <yyyy-mm-dd>T<hh:mm:ss>Z. "+
			"See https://cmr.earthdata.nasa.gov/search/site/docs/search/api.html#g-temporal.")

	// Determine default netrc location
	path := "netrc"
	usr, err := user.Current()
	if err == nil {
		path = filepath.Join(usr.HomeDir, ".netrc")
	}
	flags.Lookup("netrc").NoOptDefVal = path
}

type ingestOpts struct {
	Dir            string
	CollectionID   string
	Since          *time.Time
	Temporal       []time.Time
	CredentialFunc credentialFunc
	NumWorkers     int
	Clobber        bool
	Verbose        bool
}

const lastRanFile = "last_ran"

func readLastRan(dir string) (*time.Time, error) {
	path := filepath.Join(dir, lastRanFile)
	dat, err := ioutil.ReadFile(path)
	log.Debugf("last_ran %s", string(dat))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var t time.Time
	if err := json.Unmarshal(dat, &t); err != nil {
		return nil, nil
	}
	return &t, nil
}

func writeLastRan(dir string) error {
	path := filepath.Join(dir, lastRanFile)
	dat, err := json.Marshal(time.Now().UTC())
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, dat, 0o644)
}

func newIngestOpts(flags *pflag.FlagSet) (ingestOpts, error) {
	panerr := func(err error) {
		if err != nil {
			panic(err)
		}
	}

	var err error
	opts := ingestOpts{}

	opts.Verbose, err = flags.GetBool("verbose")
	panerr(err)
	opts.Dir, err = flags.GetString("dir")
	panerr(err)
	if err := os.MkdirAll(opts.Dir, 0o0755); err != nil && !errors.Is(err, os.ErrExist) {
		return opts, fmt.Errorf("failed to make working dir: %w", err)
	}
	opts.Clobber, err = flags.GetBool("clobber")
	panerr(err)

	opts.CollectionID, err = flags.GetString("concept-id")
	panerr(err)
	opts.NumWorkers, err = flags.GetInt("workers")
	panerr(err)
	if opts.NumWorkers < 1 || opts.NumWorkers > 5 {
		return opts, fmt.Errorf("workers must be 1 to 5")
	}
	if sinceLast, err := flags.GetBool("since-lastran"); sinceLast {
		panerr(err)
		opts.Since, err = readLastRan(opts.Dir)
		if err != nil {
			log.WithError(err).Warn("failed to read last_ran")
		}
	}

	opts.Temporal = ([]time.Time)(ingestTemporalVal)

	if f := flags.Lookup("netrc"); f != nil && f.Changed {
		netrc, err := flags.GetString("netrc")
		panerr(err)
		opts.CredentialFunc, err = newNetrcCredentialFunc(netrc)
		if err != nil {
			return opts, fmt.Errorf("failed to init netrc: %w", err)
		}
	}

	if opts.CredentialFunc == nil {
		opts.CredentialFunc, err = newEnvCredentialFunc()
		if err != nil {
			return opts, err
		}
	}

	return opts, nil
}

// exists returns true if a file exists or false on error or if it does not
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return false, nil
	case err != nil:
		return false, err
	}
	return true, nil
}

func doIngest(ctx context.Context, opts ingestOpts) error {
	if opts.Since != nil {
		log.Infof("querying for granules since %s", opts.Since)
	}
	api := internal.NewCMRAPI()
	granules, err := api.Granules(opts.CollectionID, opts.Temporal, opts.Since)
	if err != nil {
		return fmt.Errorf("querying granules: %w", err)
	}
	writeLastRan(opts.Dir)

	if len(granules) == 0 {
		log.Info("no granules")
		return nil
	}
	log.Infof("%d granules", len(granules))

	granCh := make(chan internal.Granule)
	go func() {
		defer close(granCh)
		for _, g := range granules {
			path := filepath.Join(opts.Dir, g.ProducerGranuleID)
			if !opts.Clobber {
				if ok, _ := exists(path); ok {
					log.Infof("exists, skipping %s", path)
					continue
				}
			}
			granCh <- g
		}
	}()

	ingestErrs := make(chan error)

	wg := sync.WaitGroup{}
	for i := 0; i < opts.NumWorkers; i++ {
		wg.Add(1)
		// connection cache per worker
		client, err := newClient(opts.CredentialFunc)
		if err != nil {
			return fmt.Errorf("unable to create storage for cookies: %w", err)
		}
		go func() {
			defer wg.Done()
			worker(ctx, client, granCh, ingestErrs, opts.Dir)
		}()
	}
	log.WithField("count", opts.NumWorkers).Debug("started workers")

	// Close the errors change when all the workers are done
	go func() {
		defer close(ingestErrs)
		wg.Wait()
	}()

	done := false
	for !done {
		select {
		case err, ok := <-ingestErrs:
			if !ok {
				done = true
				continue
			}
			log.WithError(err).Info("ingest failed")
			// TODO: persist error
		case <-ctx.Done():
			done = true
		}
	}

	return nil
}

// writeErr writes an <name>.error file in the case of a download error. An
// attempt is made to increment the count if an error file already exists,
// however, errors reading the existing error file are ignored so the count
// may be low if the error file cannot be decoded.
func writeErr(dlErr *downloadError, dir string) error {
	type fileError struct {
		Count int            `json:"count"`
		Last  time.Time      `json:"last"`
		Error *downloadError `json:"error"`
	}
	obj := fileError{1, time.Now().UTC(), dlErr}
	path := filepath.Join(dir, dlErr.Granule.ProducerGranuleID) + ".error"
	if _, err := os.Stat(path); errors.Is(err, os.ErrExist) {
		if dat, err := ioutil.ReadFile(path); err == nil {
			last := fileError{}
			if err := json.Unmarshal(dat, &last); err == nil {
				obj.Count = last.Count + 1
			}
		}
	}
	dat, err := json.MarshalIndent(obj, "", " ")
	if err != nil {
		return fmt.Errorf("serializing granule %v: %w", dlErr, err)
	}
	return ioutil.WriteFile(path, dat, 0o644)
}

func worker(ctx context.Context, client *http.Client, granules <-chan internal.Granule, errs chan error, dir string) {
	for {
		select {
		case <-ctx.Done():
			return
		case gran, ok := <-granules:
			if !ok {
				return
			}
			log.WithField("id", gran.ID).Debugf("ingesting %s", gran.DownloadURL())
			err := download(ctx, client, gran, dir)
			if err != nil {
				errs <- err
				var dlErr *downloadError
				if errors.As(err, &dlErr) {
					if err := writeErr(dlErr, dir); err != nil {
						log.WithError(err).Errorf("failed to write error for %v", gran)
					}
				} else {
					log.WithError(err).Errorf("worker exiting due to fatal error")
					return
				}
			} else {
				log.WithField("id", gran.ID).Infof("ingested %s", gran.DownloadURL())
			}
		}
	}
}

type downloadError struct {
	Reason  string           `json:"reason"`
	Granule internal.Granule `json:"granule"`
}

func (e *downloadError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Reason, e.Granule.DownloadURL())
}

func download(ctx context.Context, client *http.Client, g internal.Granule, dir string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", g.DownloadURL(), nil)
	if err != nil {
		return err // shouldn't really happen
	}

	resp, err := client.Do(req)
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Timeout() {
		return &downloadError{"timeout", g}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &downloadError{http.StatusText(resp.StatusCode), g}
	}

	path := filepath.Join(dir, g.ProducerGranuleID)
	tmppath := path + ".tmp"
	f, err := os.OpenFile(tmppath, os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return &downloadError{fmt.Sprintf("opening dest %s: %s", tmppath, err), g}
	}

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return &downloadError{fmt.Sprintf("writing file %s: %s", tmppath, err), g}
	}

	return os.Rename(tmppath, path)
}

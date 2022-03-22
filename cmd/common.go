package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/jdxcode/netrc"
)

func newAppContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt)
	go func() {
		log.WithField("signal", <-ch).Info("canceled")
		cancel()
	}()
	return ctx
}

type credentialFunc func(*http.Request) (string, string, bool)

func newNetrcCredentialFunc(path string) (credentialFunc, error) {
	if path == "" {
		usr, err := user.Current()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(usr.HomeDir, ".netrc")
	}
	n, err := netrc.Parse(path)
	if err != nil {
		return nil, err
	}
	return func(req *http.Request) (string, string, bool) {
		host := strings.SplitN(req.URL.Host, ":", 1)[0]
		if machine := n.Machine(host); machine != nil {
			user := machine.Get("login")
			passwd := machine.Get("password")
			return user, passwd, true
		}
		return "", "", false
	}, nil
}

func newEnvCredentialFunc() (credentialFunc, error) {
	user, haveUser := os.LookupEnv("EARTHDATA_USER")
	passwd, havePasswd := os.LookupEnv("EARTHDATA_PASSWD")
	if !haveUser || !havePasswd {
		return nil, fmt.Errorf("EARTHDATA_(USER|PASSWD) not set")
	}
	return func(req *http.Request) (string, string, bool) {
		return user, passwd, true
	}, nil
}

func newClient(getCreds credentialFunc) (*http.Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	return &http.Client{
		Transport: &http.Transport{},
		Timeout:   9 * time.Minute,
		Jar:       jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			user, passwd, ok := getCreds(req)
			if ok {
				req.SetBasicAuth(user, passwd)
			}
			return nil
		},
	}, nil
}

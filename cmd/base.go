package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/tsuru/acl-api/api/version"
)

const (
	defaultServiceName = "acl"
	userAgent          = "AclFromHell-Plugin-http-client/1.0"
)

var (
	baseClient = &http.Client{
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 10 * time.Second,
			IdleConnTimeout:     20 * time.Second,
		},
		Timeout: time.Minute,
	}

	warnOnce sync.Once
)

func serviceInstanceName(args []string, minArgs int) (string, string) {
	var instanceName string
	serviceName := defaultServiceName
	if len(args) == minArgs {
		instanceName = args[0]
	} else if len(args) > minArgs {
		serviceName = args[0]
		instanceName = args[1]
	}
	return serviceName, instanceName
}

func doProxyAdminRequest(method, service, path string, body io.Reader) (*http.Response, error) {
	baseURL := viper.GetString("tsuru.target")
	fullUrl := fmt.Sprintf("%s/services/proxy/service/%s?callback=%s",
		strings.TrimSuffix(baseURL, "/"),
		service,
		path,
	)
	return doProxyURLRequest(method, fullUrl, body)
}

func doProxyRequest(method, service, instance, path string, body io.Reader) (*http.Response, error) {
	baseURL := viper.GetString("tsuru.target")
	fullUrl := fmt.Sprintf("%s/services/%s/proxy/%s?callback=%s",
		strings.TrimSuffix(baseURL, "/"),
		service,
		instance,
		path,
	)
	return doProxyURLRequest(method, fullUrl, body)
}

func doProxyURLRequest(method, fullUrl string, body io.Reader) (*http.Response, error) {
	token := viper.GetString("tsuru.token")
	req, err := http.NewRequest(method, fullUrl, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "bearer "+token)
	req.Header.Set("User-Agent", userAgent)
	rsp, err := baseClient.Do(req)
	if err != nil {
		return nil, err
	}
	if rsp.StatusCode < 200 || rsp.StatusCode >= 400 {
		data, _ := ioutil.ReadAll(rsp.Body)
		rsp.Body.Close()
		return nil, errors.Errorf("invalid status code %d: %q", rsp.StatusCode, string(data))
	}
	warnOnce.Do(func() {
		warnVersion(rsp.Header)
	})
	return rsp, nil
}

func warnVersion(headers http.Header) {
	versionHeader := headers.Get(version.VersionHeader)
	serverVersion, _ := semver.NewVersion(versionHeader)
	clientVersion, _ := semver.NewVersion(version.Version)
	if serverVersion == nil {
		serverVersion = &semver.Version{}
	}
	if clientVersion == nil {
		clientVersion = &semver.Version{}
	}
	if clientVersion.LessThan(serverVersion) {
		fmt.Fprintln(os.Stderr, "There is a new version of the acl plugin available. Please update it.")
	}
}

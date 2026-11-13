package cmd

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	defaultServiceName = "acl"
	userAgent          = "AclFromHell-Plugin-http-client/1.0"
)

var baseClient = &http.Client{
	Transport: &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
		IdleConnTimeout:     20 * time.Second,
	},
	Timeout: time.Minute,
}

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
	fullUrl := fmt.Sprintf("%s/1.20/services/%s/resources/%s%s",
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
		data, _ := io.ReadAll(rsp.Body)
		_ = rsp.Body.Close()
		return nil, fmt.Errorf("invalid status code %d: %q", rsp.StatusCode, string(data))
	}
	return rsp, nil
}

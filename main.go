// Copyright 2023 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/tsuru/acl-api/api/version"
)

const (
	userAgent          = "AclFromHell-Plugin-http-client/1.0"
	defaultServiceName = "acl"
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

func initConfig(rootCmd *cobra.Command) func() {
	return func() {
		viper.SetConfigName(".acl")
		viper.AddConfigPath("$HOME")
		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		viper.AutomaticEnv()

		if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
			log.Fatalf("unable to bind flags to viper: %v\n", err)
		}

		if err := viper.ReadInConfig(); err == nil {
			log.Printf("Using config file: %v", viper.ConfigFileUsed())
		}
	}
}

func main() {
	rootCmd := &cobra.Command{
		Version: version.Version,
	}
	cobra.OnInitialize(initConfig(rootCmd))
	rulesCmd := &cobra.Command{
		Use: "rules",
	}
	rootCmd.AddCommand(rulesCmd)
	rulesCmd.AddCommand(addRuleCmd)
	rulesCmd.AddCommand(removeRuleCmd)
	rulesCmd.AddCommand(listRuleCmd)
	rulesCmd.AddCommand(forceSyncCmd)
	rulesCmd.AddCommand(syncDNSCmd)

	adminCmd := &cobra.Command{
		Use: "admin",
	}
	rootCmd.AddCommand(adminCmd)
	adminCmd.AddCommand(listAllRulesCmd)
	adminCmd.AddCommand(addCustomRuleCmd)

	rootCmd.PersistentFlags().String("tsuru.target", "", "Tsuru Target URL")
	rootCmd.PersistentFlags().String("tsuru.token", "", "Tsuru Token")

	dstFlags := pflag.NewFlagSet("", pflag.ExitOnError)
	dstFlags.IPNet("ip", net.IPNet{}, "Destination IP Network [10.0.0.1/32]")
	dstFlags.String("dns", "", "Destination DNS name [example.org]")
	dstFlags.String("app", "", "Destination Tsuru App Name [myapp]")
	dstFlags.String("app-pool", "", "Destination Tsuru Pool Name [dev]")
	dstFlags.String("rpaas", "", "Destination RPAAS ServiceName/Instance [rpaasv2-be/myrpaas]")
	dstFlags.String("service", "", "Destination Kubernetes Service [namespace/service]")
	dstFlags.StringSlice("port", nil, "Destination Ports [tcp:443]")

	adminFlags := pflag.NewFlagSet("", pflag.ExitOnError)
	adminFlags.AddFlagSet(dstFlags)

	adminFlags.String("src-app", "", "Source Tsuru App Name [myapp]")
	adminFlags.String("src-app-pool", "", "Source Tsuru Pool Name [dev]")
	adminFlags.String("owner", "", "Rule owner")

	addRuleCmd.Flags().AddFlagSet(dstFlags)
	addCustomRuleCmd.Flags().AddFlagSet(adminFlags)

	listRuleCmd.Flags().Bool("show-sync", false, "Show rules latest sync attempt")
	listRuleCmd.Flags().Bool("show-extra-sync", false, "Show rules with latest sync attempt details.")
	listAllRulesCmd.Flags().Bool("show-extra-sync", false, "Show rules with latest sync attempt details.")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

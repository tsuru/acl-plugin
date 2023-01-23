// Copyright 2023 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"net"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/tsuru/acl-api/api/version"
	"github.com/tsuru/acl-plugin/cmd"
)

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
	rulesCmd.AddCommand(cmd.AddRuleCmd)
	rulesCmd.AddCommand(cmd.RemoveRuleCmd)
	rulesCmd.AddCommand(cmd.ListRuleCmd)
	rulesCmd.AddCommand(cmd.ForceSyncCmd)
	rulesCmd.AddCommand(cmd.SyncDNSCmd)

	adminCmd := &cobra.Command{
		Use: "admin",
	}
	rootCmd.AddCommand(adminCmd)
	adminCmd.AddCommand(cmd.ListAllRulesCmd)
	adminCmd.AddCommand(cmd.AddCustomRuleCmd)

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

	cmd.AddRuleCmd.Flags().AddFlagSet(dstFlags)
	cmd.AddCustomRuleCmd.Flags().AddFlagSet(adminFlags)

	cmd.ListRuleCmd.Flags().Bool("show-sync", false, "Show rules latest sync attempt")
	cmd.ListRuleCmd.Flags().Bool("show-extra-sync", false, "Show rules with latest sync attempt details.")
	cmd.ListAllRulesCmd.Flags().Bool("show-extra-sync", false, "Show rules with latest sync attempt details.")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

// Copyright 2018 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/tsuru/acl-api/api/types"
	"github.com/tsuru/tablecli"
)

type serviceRuleData struct {
	ServiceInstance types.ServiceInstance
	ExpandedRules   []types.Rule
	RulesSync       []types.RuleSyncInfo
}

var listAllRulesCmd = &cobra.Command{
	Use:   "list [service name]",
	Short: "List all rules",
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceName, _ := serviceInstanceName(args, 1)
		rsp, err := doProxyAdminRequest(http.MethodGet, serviceName, "/rules", nil)
		if err != nil {
			return err
		}
		defer rsp.Body.Close()
		data, err := ioutil.ReadAll(rsp.Body)
		if err != nil {
			return err
		}
		var rules []types.Rule
		err = json.Unmarshal(data, &rules)
		if err != nil {
			return errors.Wrapf(err, "unable to unmarshal %q", string(data))
		}
		rsp, err = doProxyAdminRequest(http.MethodGet, serviceName, "/rules/sync", nil)
		if err != nil {
			return err
		}
		defer rsp.Body.Close()
		data, err = ioutil.ReadAll(rsp.Body)
		if err != nil {
			return err
		}
		var rulesSync []types.RuleSyncInfo
		err = json.Unmarshal(data, &rulesSync)
		if err != nil {
			return errors.Wrapf(err, "unable to unmarshal %q", string(data))
		}
		rsp, err = doProxyAdminRequest(http.MethodGet, serviceName, "/services", nil)
		if err != nil {
			return err
		}
		defer rsp.Body.Close()
		data, err = ioutil.ReadAll(rsp.Body)
		if err != nil {
			return err
		}
		var serviceInstances []types.ServiceInstance
		err = json.Unmarshal(data, &serviceInstances)
		if err != nil {
			return errors.Wrapf(err, "unable to unmarshal %q", string(data))
		}
		fmt.Println("Service Rules:")
		renderServiceRules(serviceInstances, true)
		fmt.Println("Expanded Rules:")
		renderExpandedRules(rules, rulesSync)
		renderSyncInfo(rulesSync)
		extraSync, _ := cmd.Flags().GetBool("show-extra-sync")
		if extraSync {
			renderExtraSyncInfo(rules, rulesSync)
		}
		return nil
	},
}

var listRuleCmd = &cobra.Command{
	Use:   "list [service name] [instance name]",
	Short: "List rules",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceName, instanceName := serviceInstanceName(args, 1)
		rsp, err := doProxyRequest(http.MethodGet, serviceName, instanceName, "/rule", nil)
		if err != nil {
			return err
		}
		defer rsp.Body.Close()
		data, err := ioutil.ReadAll(rsp.Body)
		if err != nil {
			return err
		}
		var ruleData serviceRuleData
		err = json.Unmarshal(data, &ruleData)
		if err != nil {
			return errors.Wrapf(err, "unable to unmarshal %q", string(data))
		}
		fmt.Println("Rules:")
		renderServiceRules([]types.ServiceInstance{ruleData.ServiceInstance}, false)
		fmt.Println("Expanded Rules (for each bound app):")
		renderExpandedRules(ruleData.ExpandedRules, ruleData.RulesSync)
		showSync, _ := cmd.Flags().GetBool("show-sync")
		extraSync, _ := cmd.Flags().GetBool("show-extra-sync")
		if showSync || extraSync {
			renderSyncInfo(ruleData.RulesSync)
		}
		if extraSync {
			renderExtraSyncInfo(ruleData.ExpandedRules, ruleData.RulesSync)
		}
		return nil
	},
}

func renderExtraSyncInfo(rules []types.Rule, rulesSync []types.RuleSyncInfo) {
	rulesSyncMap := map[string][]types.RuleSyncInfo{}
	for _, rs := range rulesSync {
		rulesSyncMap[rs.RuleID] = append(rulesSyncMap[rs.RuleID], rs)
	}
	fmt.Println("\nDetailed sync results:")
	for _, r := range rules {
		ruleSyncs, ok := rulesSyncMap[r.RuleID]
		if !ok {
			continue
		}
		sort.Slice(ruleSyncs, func(i, j int) bool {
			return ruleSyncs[i].Engine < ruleSyncs[j].Engine
		})
		synced := true
		for _, rs := range ruleSyncs {
			latestSync := rs.LatestSync()
			if latestSync == nil || !latestSync.Successful {
				synced = false
				break
			}
		}
		fields := []string{
			"ID", r.RuleID,
			"Source", r.Source.String(),
			"Destination", r.Destination.String(),
			"Deleted", strconv.FormatBool(r.Removed),
			"Synced", strconv.FormatBool(synced),
		}
		fmt.Print("----------------------\n")
		for i := 0; i < len(fields); i += 2 {
			fmt.Printf("%v: %v\n", fields[i], fields[i+1])
		}
		for _, rs := range ruleSyncs {
			latestSync := rs.LatestSync()
			if latestSync == nil {
				continue
			}
			if latestSync.Error != "" {
				fmt.Printf("Sync error in engine %q: %v\n", rs.Engine, latestSync.Error)
			}
			var data interface{} = map[string]interface{}{}
			err := json.Unmarshal([]byte(latestSync.SyncResult), &data)
			if err != nil {
				data = []map[string]interface{}{}
				_ = json.Unmarshal([]byte(latestSync.SyncResult), &data)
			}
			indented, _ := json.MarshalIndent(data, "", "  ")
			fmt.Printf("Sync result in engine %q: %v\n", rs.Engine, string(indented))
		}
	}
}

func renderSyncInfo(rulesSync []types.RuleSyncInfo) {
	fmt.Println("\nSync result summary:")
	table := tablecli.NewTable()
	table.Headers = tablecli.Row{"Rule ID", "Engine", "Start (duration)", "Success", "Error"}
	for _, rs := range rulesSync {
		success := "✓"
		latestSync := rs.LatestSync()
		if latestSync == nil {
			continue
		}
		if !latestSync.Successful {
			success = ""
		}
		tsRule := fmt.Sprintf("%s (%v)",
			latestSync.StartTime.Local().Format(time.RFC3339),
			latestSync.EndTime.Sub(latestSync.StartTime),
		)
		table.AddRow(tablecli.Row{
			rs.RuleID,
			rs.Engine,
			tsRule,
			success,
			latestSync.Error,
		})
	}
	fmt.Print(table.String())
}

func renderServiceRules(serviceInstances []types.ServiceInstance, renderName bool) {
	table := tablecli.NewTable()
	table.Headers = tablecli.Row{"ID", "Destination", "Creator"}
	if renderName {
		table.Headers = append(tablecli.Row{"Instance"}, table.Headers...)
	}
	for _, si := range serviceInstances {
		for _, r := range si.BaseRules {
			row := tablecli.Row{
				r.RuleID,
				r.Destination.String(),
				r.Creator,
			}
			if renderName {
				row = append(tablecli.Row{si.InstanceName}, row...)
			}
			table.AddRow(row)
		}
	}
	fmt.Println(table.String())
}

func renderExpandedRules(rules []types.Rule, rulesSync []types.RuleSyncInfo) {
	ruleSyncByRule := map[string][]types.RuleSyncInfo{}
	for _, rs := range rulesSync {
		ruleSyncByRule[rs.RuleID] = append(ruleSyncByRule[rs.RuleID], rs)
	}
	table := tablecli.NewTable()
	table.Headers = tablecli.Row{"ID", "Source", "Destination", "Deleted", "Synced"}
	for _, r := range rules {
		deleted := ""
		if r.Removed {
			deleted = "✓"
		}
		var synced string
		for _, rs := range ruleSyncByRule[r.RuleID] {
			latestSync := rs.LatestSync()
			if latestSync != nil && latestSync.Successful {
				synced = "✓"
			} else {
				synced = ""
				break
			}
		}
		table.AddRow(tablecli.Row{
			r.RuleID,
			r.Source.String(),
			r.Destination.String(),
			deleted,
			synced,
		})
	}
	fmt.Print(table.String())
}

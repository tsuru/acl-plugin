// Copyright 2023 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/spf13/cobra"
	"github.com/tsuru/acl-api/api/types"
)

type forceResponse struct {
	Count int `json:"count"`
}

var forceSyncCmd = &cobra.Command{
	Use:   "sync [app name]",
	Short: "Force sync rules (for debug/troubleshooting purpose)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appName := args[0]
		rsp, err := doProxyAdminRequest(http.MethodPost, defaultServiceName, "/apps/"+appName+"/sync", nil)
		if err != nil {
			return err
		}
		defer rsp.Body.Close()

		var response forceResponse
		json.NewDecoder(rsp.Body).Decode(&response)

		fmt.Printf("Sync request sent, %d rules synced\n", response.Count)
		return nil
	},
}

var syncDNSCmd = &cobra.Command{
	Use:   "sync-dns [cname]",
	Short: "Force sync rules (for debug/troubleshooting purpose)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dns := args[0]

		ruleIDs, err := rulesIDFromDNS(dns)
		if err != nil {
			return err
		}

		for i, ruleID := range ruleIDs {
			fmt.Printf("%d/%d Syncing rule %s\n", i+1, len(ruleIDs), ruleID)
			err = forceSyncRuleID(ruleID)
			if err != nil {
				fmt.Printf("Error Syncing rule %s, error: %s", ruleID, err)
			}
		}

		return nil
	},
}

func rulesIDFromDNS(dns string) ([]string, error) {
	var rules []types.Rule

	q := url.Values{}
	q.Set("destination.externaldns.name", dns)

	rsp, err := doProxyAdminRequest(http.MethodGet, defaultServiceName, "/rules?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	err = json.NewDecoder(rsp.Body).Decode(&rules)
	if err != nil {
		return nil, err
	}

	ruleIDs := []string{}
	for _, r := range rules {
		if r.Removed {
			continue
		}
		ruleIDs = append(ruleIDs, r.RuleID)
	}

	return ruleIDs, nil
}

func forceSyncRuleID(ruleID string) error {
	rsp, err := doProxyAdminRequest(http.MethodPost, defaultServiceName, "/rules/"+ruleID+"/sync", nil)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()

	if rsp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("invalid status code: %d", rsp.StatusCode)
	}

	return nil
}

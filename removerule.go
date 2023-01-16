// Copyright 2023 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var removeRuleCmd = &cobra.Command{
	Use:   "remove [service name] [instance name] [id]",
	Short: "Remove rule",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceName, instanceName := serviceInstanceName(args, 2)
		ruleID := args[len(args)-1]
		rsp, err := doProxyRequest(http.MethodDelete, serviceName, instanceName, "/rule/"+ruleID, nil)
		if err != nil {
			return err
		}
		defer rsp.Body.Close()
		fmt.Println("Rule successfully removed.")
		return nil
	},
}

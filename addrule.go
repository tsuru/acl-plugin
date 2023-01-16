// Copyright 2023 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/tsuru/acl-api/api/types"
)

var addCustomRuleCmd = &cobra.Command{
	Use:   "add",
	Short: "Add new rule custom rule",
	RunE: func(cmd *cobra.Command, args []string) error {
		src, err := parseSourceRuleType(cmd.Flags())
		if err != nil {
			return err
		}
		dst, err := parseRuleType(cmd.Flags())
		if err != nil {
			return err
		}
		owner, _ := cmd.Flags().GetString("owner")
		if owner == "" {
			return errors.New("--owner argument is mandatory")
		}

		serviceName, _ := serviceInstanceName(args, 1)
		data, err := json.Marshal(types.Rule{
			Source:      *src,
			Destination: *dst,
			Metadata: map[string]string{
				"owner": owner,
			},
		})
		if err != nil {
			return err
		}
		body := bytes.NewReader(data)

		rsp, err := doProxyAdminRequest(http.MethodPost, serviceName, "/rules", body)
		if err != nil {
			return err
		}
		defer rsp.Body.Close()

		return nil
	},
}

var addRuleCmd = &cobra.Command{
	Use:   "add [service name] [instance name]",
	Short: "Add new rule",
	Example: `
# Add ACL to a destination tsuru app
tsuru acl rules add <ACL SERVICE> --app <MY DESTINATION APP>

# Add ACL to a destination tsuru pool (prefer to use fine grained by each app)
tsuru acl rules add <ACL SERVICE> --app-pool <MY DESTINATION APP POOL>

# Add ACL to a destination RPASS
tsuru acl rules add <ACL SERVICE> --rpaas "<RPAAS SERVICE NAME>/<RPAAS SERVICE INSTANCE>"

# Add ACL to a destination service by DNS
tsuru acl rules add <ACL SERVICE> --dns mydomain.globoi.com --port tcp:443

# Add ACL to a destination service by IP (prefer by DNS over IP)
tsuru acl rules add <ACL SERVICE> --ip MYIP/32 --port tcp:443
	`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rt, err := parseRuleType(cmd.Flags())
		if err != nil {
			return err
		}
		serviceName, instanceName := serviceInstanceName(args, 1)
		data, err := json.Marshal(types.Rule{Destination: *rt})
		if err != nil {
			return err
		}
		body := bytes.NewReader(data)
		rsp, err := doProxyRequest(http.MethodPost, serviceName, instanceName, "/rule", body)
		if err != nil {
			return err
		}
		defer rsp.Body.Close()
		fmt.Println("Rule successfully added.")
		return nil
	},
}

func parseRuleType(flags *pflag.FlagSet) (*types.RuleType, error) {
	rawPorts, _ := flags.GetStringSlice("port")
	var ports []types.ProtoPort
	for _, p := range rawPorts {
		parts := strings.Split(p, ":")
		if len(parts) != 2 {
			return nil, errors.New("--port arguments must be in the format <protocol>:<port>, e.g. \"--port tcp:443\"")
		}
		portInt, err := strconv.ParseUint(parts[1], 10, 16)
		if err != nil {
			return nil, err
		}
		ports = append(ports, types.ProtoPort{
			Protocol: parts[0],
			Port:     uint16(portInt),
		})
	}
	ip, _ := flags.GetIPNet("ip")
	dns, _ := flags.GetString("dns")
	app, _ := flags.GetString("app")
	appPool, _ := flags.GetString("app-pool")
	service, _ := flags.GetString("service")

	rpaas, _ := flags.GetString("rpaas")

	count := 0
	rt := types.RuleType{}
	if ip.IP != nil {
		count++
		rt.ExternalIP = &types.ExternalIPRule{
			IP:    ip.String(),
			Ports: ports,
		}
	}
	if dns != "" {
		count++
		rt.ExternalDNS = &types.ExternalDNSRule{
			Name:  dns,
			Ports: ports,
		}
	}
	if app != "" {
		count++
		rt.TsuruApp = &types.TsuruAppRule{
			AppName: app,
		}
	}
	if appPool != "" {
		count++
		rt.TsuruApp = &types.TsuruAppRule{
			PoolName: appPool,
		}
	}
	if rpaas != "" {
		count++
		parts := strings.SplitN(rpaas, "/", 2)
		if len(parts) != 2 {
			return nil, errors.New("--rpaas argument must be in the format serviceName/serviceInstance, e.g. \"rpaasv2-be/myinstance\"")
		}

		rt.RpaasInstance = &types.RpaasInstanceRule{
			ServiceName: parts[0],
			Instance:    parts[1],
		}
	}
	if service != "" {
		count++
		parts := strings.SplitN(service, "/", 2)
		ns := "default"
		var svcName string
		if len(parts) == 2 {
			ns, svcName = parts[0], parts[1]
		} else {
			svcName = parts[0]
		}
		rt.KubernetesService = &types.KubernetesServiceRule{
			Namespace:   ns,
			ServiceName: svcName,
		}
	}

	if count != 1 {
		return nil, errors.New("only one of --ip, --dns, --app, --app-pool, --rpaas, --service must be set")
	}
	if (app != "" || appPool != "" || service != "") && len(ports) > 0 {
		return nil, errors.New("--port is not supported with --app, --app-pool or --service")
	}
	return &rt, nil
}

func parseSourceRuleType(flags *pflag.FlagSet) (*types.RuleType, error) {
	count := 0
	rt := types.RuleType{}
	app, _ := flags.GetString("src-app")
	appPool, _ := flags.GetString("src-app-pool")
	if app != "" {
		count++
		rt.TsuruApp = &types.TsuruAppRule{
			AppName: app,
		}
	}
	if appPool != "" {
		count++
		rt.TsuruApp = &types.TsuruAppRule{
			PoolName: appPool,
		}
	}

	if count != 1 {
		return nil, errors.New("only one of --app, --app-pool must be set")
	}

	return &rt, nil
}

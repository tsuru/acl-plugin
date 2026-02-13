// Copyright 2026 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"os"

	"github.com/tsuru/tablecli"
)

func InitTableClI() {
	tableColor := os.Getenv("TSURU_TABLE_COLOR")

	if tableColor != "" {
		tablecli.SetBorderColorByString(tableColor)
	}

	tableUTF8 := os.Getenv("TSURU_TABLE_UTF8")
	if tableUTF8 == "true" {
		tablecli.TableConfig.UseUTF8Borders = true
	} else {
		tablecli.TableConfig.UseUTF8Borders = false
	}
}

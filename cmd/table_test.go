// Copyright 2026 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"testing"

	"github.com/tsuru/tablecli"
)

func TestInitTableClI_UTF8(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		want     bool
	}{
		{
			name:     "utf8 enabled",
			envValue: "true",
			want:     true,
		},
		{
			name:     "utf8 disabled",
			envValue: "false",
			want:     false,
		},
		{
			name:     "utf8 unset",
			envValue: "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TSURU_TABLE_UTF8", tt.envValue)
			t.Setenv("TSURU_TABLE_COLOR", "")

			InitTableClI()

			if tablecli.TableConfig.UseUTF8Borders != tt.want {
				t.Errorf("UseUTF8Borders = %v, want %v", tablecli.TableConfig.UseUTF8Borders, tt.want)
			}
		})
	}
}

func TestInitTableClI_NamedColor(t *testing.T) {
	for _, name := range []string{"red", "blue", "yellow"} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("TSURU_TABLE_COLOR", name)
			t.Setenv("TSURU_TABLE_UTF8", "")

			tablecli.TableConfig.BorderColorFunc = nil

			InitTableClI()

			if tablecli.TableConfig.BorderColorFunc == nil {
				t.Errorf("BorderColorFunc should be set for color %q", name)
			}
		})
	}
}

func TestInitTableClI_HexColor(t *testing.T) {
	t.Setenv("TSURU_TABLE_COLOR", "#ff0000")
	t.Setenv("TSURU_TABLE_UTF8", "")

	tablecli.TableConfig.BorderColorFunc = nil

	InitTableClI()

	if tablecli.TableConfig.BorderColorFunc == nil {
		t.Error("BorderColorFunc should be set for hex color")
	}
}

func TestInitTableClI_InvalidHexColor(t *testing.T) {
	t.Setenv("TSURU_TABLE_COLOR", "#zzzzzz")
	t.Setenv("TSURU_TABLE_UTF8", "")

	tablecli.TableConfig.BorderColorFunc = nil

	InitTableClI()

	if tablecli.TableConfig.BorderColorFunc != nil {
		t.Error("BorderColorFunc should not be set for invalid hex color")
	}
}

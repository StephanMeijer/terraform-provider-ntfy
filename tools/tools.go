//go:build tools

package tools

import (
	// tfplugindocs generates documentation from schema annotations
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)

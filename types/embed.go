// Package bridgetypes provides the declaration shipped by the CLI initializer.
package bridgetypes

import _ "embed"

//go:embed velox.d.ts
var declaration string

func Declaration() string { return declaration }

// Package brand embeds the Pomme artwork used by the Windows launcher.
package brand

import _ "embed"

// Logo is the existing Pomme apple-and-terminal icon.
//
//go:embed pomme.png
var Logo []byte

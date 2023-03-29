//go:build !prod

package wesplot

import "embed"

var webuiFiles embed.FS

func openBrowser(url string) {
	// In dev mode we don't actually want to open the browser. That's up to the
	// developer as it will be in a different port anyway.
}

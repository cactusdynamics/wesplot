//go:build prod

package wesplot

import (
	"embed"
)

//go:embed webui
var webuiFiles embed.FS

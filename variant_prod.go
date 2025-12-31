//go:build prod

package wesplot

import (
	"embed"
	"log/slog"
	"os/exec"
	"runtime"
)

//go:embed webui
var webuiFiles embed.FS

func openBrowser(url string) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	err := exec.Command(cmd, args...).Start()
	if err != nil {
		slog.Warn("failed to start web browser automatically")
	}
}

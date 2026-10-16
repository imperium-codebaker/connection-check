package main

import (
	"fmt"
	"os/exec"
	"runtime"
)

// openBrowser opens url in the default browser, or prints a hint if that fails.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		fmt.Println("Open this address in your browser:", url)
		return
	}
	go cmd.Wait()
}

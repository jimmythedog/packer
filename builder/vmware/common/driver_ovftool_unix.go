// +build !windows

// These functions are compatible with ovftool on *NIX
package common

import (
	"log"
	"os/exec"
	"runtime"
)

func ovftoolFindOvfTool() (string, error) {
	log.Printf("ovftoolFindOvfTool ENTRY")
	path, err := exec.LookPath("ovftool")
	if err != nil {
		if runtime.GOOS == "darwin" {
			appPath := "/Applications/VMware OVF Tool/ovftool"
			log.Printf("Attempting to lookup ovftool in %s", appPath)
			path, err = exec.LookPath(appPath)
		}
	}

	log.Printf("ovftoolFindOvfTool EXIT: path=%s, err=%s", path, err)
	return path, err
}

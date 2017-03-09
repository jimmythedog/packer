// +build !windows

// These functions are compatible with ovftool on *NIX
package common

import (
	"os/exec"
)

func ovftoolFindOvfTool() (string, error) {
	return exec.LookPath("ovftool")
}

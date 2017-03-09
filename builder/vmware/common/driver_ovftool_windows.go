// +build windows

// These functions are compatible with ovftool on Windows
package common

import (
	"os"
	"os/exec"
	"path/filepath"
)

func ovftoolFindOvfTool() (string, error) {
	path, err := exec.LookPath("ovftool.exe")
	if err == nil {
		return path, nil
	}

	return findFile("ovftool.exe", ovftoolProgramFilePaths()), nil
}

func ovftoolProgramFilePaths() []string {
	paths := make([]string, 0, 2)

	if os.Getenv("ProgramFiles(x86)") != "" {
		paths = append(paths,
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "/VMware/VMware OVF Tool"))
	}

	if os.Getenv("ProgramFiles") != "" {
		paths = append(paths,
			filepath.Join(os.Getenv("ProgramFiles"), "/VMware/VMware OVF Tool"))
	}

	return paths
}

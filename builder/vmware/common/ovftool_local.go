package common

import (
	"errors"
	"fmt"
	"log"
	"os/exec"
	"regexp"
)

func importOvfToVmx(ovfPath string, vmName string, outputPath string) (vmxPath, diskPath string, err error) {
	log.Printf("ImportOvf Entry: ovfPath=%s, vmName=%s, outputPath=%s", ovfPath, vmName, outputPath)
	ovftool, err := ovftoolFindOvfTool()
	if err != nil {
		log.Printf("Could not find ovftool: %s", err)
		return
	}

	cmd := exec.Command(ovftool,
		"-tt=vmx",
		"--name="+vmName,
		"--machineOutput", //required, or extractVmxPath will not work!
		ovfPath,
		outputPath)

	stdOut, stdErr, err := runAndLog(cmd)
	if err != nil {
		log.Printf("Error importing ovf/ova: %s\n%s\n", err, stdErr)
		return
	}

	vmxPath, err = extractVmxPath(stdOut)
	if err != nil {
		return
	}
	log.Printf("vmxPath=%s", vmxPath)

	diskPath, err = FindRootDiskFilename(vmxPath)
	if err != nil {
		return
	}
	if diskPath == "" {
		return "", "", fmt.Errorf("Root disk filename could not be found!")
	}

	log.Printf("ImportOvf Exit: vmxPath=%s, diskPath=%s", vmxPath, diskPath)
	return
}

func extractVmxPath(output string) (string, error) {
	r := regexp.MustCompile(`\+ (?P<vmxPath>.*\.vmx)`)
	matched := r.FindStringSubmatch(output)
	if matched == nil {
		return "", errors.New(fmt.Sprintf("Could not find vmx path in ovftool output: %s", output))
	}
	return matched[1], nil
}

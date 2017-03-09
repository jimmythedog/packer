package common

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
)

func importOvfToVmx(ovfPath string, vmName string, outputPath string) (vmxPath, diskPath string, err error) {
	log.Printf("ImportOvf Entry: ovfPath=%s, vmName=%s", ovfPath, vmName)
	ovftool, err := ovftoolFindOvfTool()
	if err != nil {
		log.Printf("Could not find ovftool: %s", err)
		return
	}

	cmd := exec.Command(ovftool, "-tt=vmx", "--name="+vmName, ovfPath, outputPath)

	_, stdErr, err := runAndLog(cmd)
	if err != nil {
		log.Printf("Error importing ovf/ova: %s\n%s\n", err, stdErr)
		return
	}

	vmxPath = filepath.Join(outputPath, vmName, vmName+".vmx")

	diskName, err := FindRootDiskFilename(vmxPath)
	if err != nil {
		return
	}
	if diskName == "" {
		return "", "", fmt.Errorf("Root disk filename could not be found!")
	}
	diskPath = filepath.Join(outputPath, vmName, diskName)

	log.Printf("ImportOvf Exit: vmxPath=%s, diskPath=%s", vmxPath, diskPath)
	return
}

package ovf

import (
	"fmt"
	"github.com/mitchellh/multistep"
	vmwcommon "github.com/mitchellh/packer/builder/vmware/common"
	"github.com/mitchellh/packer/packer"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

// This step download the VMX to the remote host
//
// Uses:
//	 driver Driver
//	 ui	 packer.Ui
//	 vmx_path string
//
// Produces:
//	 <nothing>
type StepDownloadVMX struct {
	RemoteType string
	tempDir    string
}

func (s *StepDownloadVMX) Run(state multistep.StateBag) multistep.StepAction {
	log.Printf("StepDownloadVMX.Run entry: RemoteType=%s", s.RemoteType)

	if s.RemoteType != "esx5" {
		log.Printf("Will not download VMX due to incorrect remote type (%s)", s.RemoteType)
		return multistep.ActionContinue
	}

	driver := state.Get("driver").(vmwcommon.Driver)
	remoteDriver, ok := driver.(vmwcommon.RemoteDriver)
	if !ok {
		state.Put("error", "Could not get RemoteDriver")
		return multistep.ActionHalt
	}

	remoteVmxPath := state.Get("vmx_path").(string)
	tempDir, err := ioutil.TempDir("", "packer-vmx")
	if err != nil {
		state.Put("error", fmt.Errorf("Error creating temp dir (%s): %s", tempDir, err))
		return multistep.ActionHalt
	}
	s.tempDir = tempDir
	vmxPath := filepath.Join(s.tempDir, filepath.Base(remoteVmxPath))

	ui := state.Get("ui").(packer.Ui)
	ui.Say(fmt.Sprintf("Downloading VMX from: %s to: %s", remoteVmxPath, vmxPath))

	if err = remoteDriver.Download(remoteVmxPath, vmxPath); err != nil {
		state.Put("error", fmt.Errorf("Error writing VMX: %s", err))
		return multistep.ActionHalt
	}
	ui.Say(fmt.Sprintf("VMX downloaded to: %s", vmxPath))
	state.Put("vmx_path", vmxPath)
	log.Printf("StepDownloadVMX.Run exit")

	return multistep.ActionContinue
}

func (s *StepDownloadVMX) Cleanup(multistep.StateBag) {
	log.Printf("StepDownloadVMX.Cleanup entry")
	if s.tempDir != "" {
		log.Printf("Deleting %s", s.tempDir)
		if err := os.RemoveAll(s.tempDir); err != nil {
			log.Printf("Failed to remove local VMX dir(%s): %s", s.tempDir, err)
		}
	}
	log.Printf("StepDownloadVMX.Cleanup exit")
}

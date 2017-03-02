package ovf

import (
	"fmt"
	"github.com/mitchellh/multistep"
	vmwcommon "github.com/mitchellh/packer/builder/vmware/common"
	"github.com/mitchellh/packer/packer"
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
	vmxPath    string
}

func (s *StepDownloadVMX) Run(state multistep.StateBag) multistep.StepAction {
	log.Printf("StepDownloadVMX.Run entry: RemoteType=%s, vmxPath=%s", s.RemoteType, s.vmxPath)

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

	s.vmxPath = state.Get("vmx_path").(string)
	ui := state.Get("ui").(packer.Ui)
	ui.Say(fmt.Sprintf("Downloading VMX from: %s to: %s", s.vmxPath, s.vmxPath))

	if err := os.MkdirAll(filepath.Dir(s.vmxPath), 0700); err != nil {
		state.Put("error", fmt.Errorf("Error making directory for local VMX: %s", err))
		return multistep.ActionHalt
	}

	if err := remoteDriver.Download(s.vmxPath, s.vmxPath); err != nil {
		state.Put("error", fmt.Errorf("Error writing VMX: %s", err))
		return multistep.ActionHalt
	}
	ui.Say(fmt.Sprintf("VMX downloaded to: %s", s.vmxPath))
	state.Put("vmx_path", s.vmxPath)
	log.Printf("StepDownloadVMX.Run exit")

	return multistep.ActionContinue
}

func (s *StepDownloadVMX) Cleanup(multistep.StateBag) {
	log.Printf("StepDownloadVMX.Cleanup entry")
	if s.vmxPath != "" {
		log.Printf("Deleting %s", s.vmxPath)
		if err := os.Remove(s.vmxPath); err != nil {
			log.Printf("Failed to remove local VMX file(%s): %s", s.vmxPath, err)
		}
		vmxDir := filepath.Dir(s.vmxPath)
		if err := os.Remove(vmxDir); err != nil {
			log.Printf("Failed to remove local VMX dir(%s): %s", vmxDir, err)
		}
	}
	log.Printf("StepDownloadVMX.Cleanup exit")
}

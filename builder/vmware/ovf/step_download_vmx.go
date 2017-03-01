package ovf

import (
	"fmt"
	"log"
	"github.com/mitchellh/multistep"
	vmwcommon "github.com/mitchellh/packer/builder/vmware/common"
	"github.com/mitchellh/packer/packer"
	"io/ioutil"
	"os"
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
	vmxPath string
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

	tempFile, err := ioutil.TempFile("", ".vmx")
	if err != nil {
		state.Put("error", fmt.Errorf("Error creating temp vmx file: %s", err))
		return multistep.ActionHalt
	}

	s.vmxPath = tempFile.Name()
	remoteVmxPath := state.Get("remote_vmx_path").(string)
	ui := state.Get("ui").(packer.Ui)
	ui.Say(fmt.Sprintf("Downloading VMX from: %s to: %s", remoteVmxPath, s.vmxPath))
	if err := remoteDriver.Download(remoteVmxPath, s.vmxPath); err != nil {
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
		os.RemoveAll(s.vmxPath)
	}
	log.Printf("StepDownloadVMX.Cleanup exit")
}

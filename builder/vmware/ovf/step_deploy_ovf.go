package ovf

import (
	"github.com/mitchellh/multistep"
	vmwcommon "github.com/mitchellh/packer/builder/vmware/common"
	"github.com/mitchellh/packer/packer"
	"log"
	"path/filepath"
)

type StepDeployOvf struct {
	Format string
}

func (s *StepDeployOvf) Run(state multistep.StateBag) multistep.StepAction {
	log.Printf("StepDeployOvf.Run entry: Format=%s", s.Format)

	c := state.Get("esxova_config").(*Config)
	ui := state.Get("ui").(packer.Ui)

	if c.Config.RemoteType != "esx5" {
		log.Printf("Will not deploy ova/ovf: RemoteType=%s", c.Config.RemoteType)
		return multistep.ActionContinue
	}

	driver := state.Get("driver").(vmwcommon.Driver)
	remoteDriver, ok := driver.(vmwcommon.RemoteDriver)
	if !ok {
		log.Printf("Could not get remote driver")
		state.Put("error", "Could not get RemoteDriver")
		return multistep.ActionHalt
	}

	ui.Say("Deploying ova/ovf...")
	remoteVmxPath, err := remoteDriver.DeployOvf(c.SourcePath, c.Config.VMName+"-ovf") //returns "dir/file.vmx"
	if err != nil {
		log.Printf("Error during ova/ovf deploy: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	log.Printf("Remote VMX Path = %s", remoteVmxPath)
	state.Put("source_path", remoteVmxPath)

	log.Printf("StepDeployOvf.Run exit")
	return multistep.ActionContinue
}

func (s *StepDeployOvf) Cleanup(state multistep.StateBag) {
	log.Printf("StepDeployOvf.Cleanup entry")
	c := state.Get("esxova_config").(*Config)
	if c.Config.RemoteType != "esx5" {
		log.Printf("Will not cleanup - remote_type = %s", c.Config.RemoteType)
		return
	}

	//TODO diff logic for remote/local?
	dir := state.Get("dir").(vmwcommon.OutputDir)
	ovfImportedDir := filepath.Dir(state.Get("source_path").(string))
	log.Printf("Removing the remote directory that the ovf was imported to (%s)", ovfImportedDir)
	if err := dir.RemoveTree(ovfImportedDir); err != nil {
		log.Printf("Error deleting ovf directory: %s", err)
	}
	log.Printf("StepDeployOvf.Cleanup exit")
}

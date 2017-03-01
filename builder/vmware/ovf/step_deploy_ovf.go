package ovf

import (
	"fmt"
	"log"
	"github.com/mitchellh/multistep"
	"github.com/mitchellh/packer/packer"
	vmwcommon "github.com/mitchellh/packer/builder/vmware/common"
	iso "github.com/mitchellh/packer/builder/vmware/iso"
	"time"
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
	if err := remoteDriver.DeployOvf(c.SourcePath, c.Config.VMName); err != nil {
        log.Printf("Error during ova/ovf deploy: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	remoteVmxPath, err := remoteDriver.GetVmxPath() //returns "dir/file.vmx"
	if err != nil {
        log.Printf("Error getting remote path to VMX: %s", err)
		state.Put("error", fmt.Errorf("Error getting remote path to VMX: %s", err))
		return multistep.ActionHalt
	}
	log.Printf("Remote VMX Path = %s", remoteVmxPath)
	state.Put("remote_vmx_path", remoteVmxPath)

	remoteVmdkPath, err := remoteDriver.GetVmdkPath() //returns "dir/file.vmdk"
	if err != nil {
        log.Printf("Error getting remote path to VMDK: %s", err)
		state.Put("error", fmt.Errorf("Error getting remote path to VMDK: %s", err))
		return multistep.ActionHalt
	}
	log.Printf("Remote VMDK Path = %s", remoteVmdkPath)
	state.Put("full_disk_path", remoteVmdkPath)

	log.Printf("StepDeployOvf.Run exit")
	return multistep.ActionContinue
}

func (s *StepDeployOvf) Cleanup(state multistep.StateBag) {
	log.Printf("StepDeployOvf.Cleanup entry")
	driver := state.Get("driver").(vmwcommon.Driver)
	ui := state.Get("ui").(packer.Ui)
	config := state.Get("config").(*iso.Config)

	_, cancelled := state.GetOk(multistep.StateCancelled)
	_, halted := state.GetOk(multistep.StateHalted)
	if (config.KeepRegistered) && (!cancelled && !halted) {
		ui.Say("Keeping virtual machine registered with ESX host (keep_registered = true)")
		return
	}

	if remoteDriver, ok := driver.(vmwcommon.RemoteDriver); ok {
		if s.Format == "" {
			ui.Say("Unregistering virtual machine...")
			if err := remoteDriver.Unregister(""); err != nil {
				ui.Error(fmt.Sprintf("Error unregistering VM: %s", err))
			}
		} else {
			ui.Say("Destroying virtual machine...")
			if err := remoteDriver.Destroy(); err != nil {
				ui.Error(fmt.Sprintf("Error destroying VM: %s", err))
			}
			log.Printf("Waiting for machine to actually destroy...")
			maxAttempts := 200
			for i := 0; i <= maxAttempts; i++ {
				destroyed, _ := remoteDriver.IsDestroyed()
				if destroyed {
					log.Printf("Machine destroyed cleanly")
					break
				}
				if (i < maxAttempts) {
					time.Sleep(150 * time.Millisecond)
				} else {
					log.Printf("Machine has not destroyed cleanly - manually deleting...")
					dir := state.Get("dir").(vmwcommon.OutputDir)
					dir.RemoveAll()
				}
			}
		}
	}
	log.Printf("StepDeployOvf.Cleanup exit")
}

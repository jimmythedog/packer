package ovf

import (
	"fmt"
	"github.com/mitchellh/multistep"
	vmwcommon "github.com/mitchellh/packer/builder/vmware/common"
	"github.com/mitchellh/packer/packer"
	"log"
	"os"
	"path/filepath"
	"time"
)

type StepImportOvf struct {
	Format string
}

func (s *StepImportOvf) Run(state multistep.StateBag) multistep.StepAction {
	log.Printf("StepImportOvf.Run entry: Format=%s", s.Format)

	c := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)

	driver := state.Get("driver").(vmwcommon.Driver)
	ui.Say("Importing ova/ovf...")
	dir := state.Get("dir").(vmwcommon.OutputDir)
	vmxPath, vmdkPath, err := driver.ImportOvf(c.SourcePath, c.VMName, dir.String())
	if err != nil {
		log.Printf("Error during ova/ovf deploy: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	state.Put("full_disk_path", vmdkPath)
	state.Put("vmx_path", vmxPath)

	log.Printf("StepImportOvf.Run exit")
	return multistep.ActionContinue
}

func (s *StepImportOvf) Cleanup(state multistep.StateBag) {
	log.Printf("StepImportOvf.Cleanup entry")
	driver := state.Get("driver").(vmwcommon.Driver)
	ui := state.Get("ui").(packer.Ui)
	config := state.Get("config").(*Config)

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
				if i < maxAttempts {
					time.Sleep(150 * time.Millisecond)
				} else {
					log.Printf("Machine has not destroyed cleanly - manually deleting...")
					dir := state.Get("dir").(vmwcommon.OutputDir)
					dir.RemoveAll()
				}
			}
		}
		tempVmxPath := state.Get("vmx_path").(string)
		os.RemoveAll(filepath.Dir(tempVmxPath))
	}
	log.Printf("StepImportOvf.Cleanup exit")
}

package common

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/mitchellh/multistep"
	"github.com/mitchellh/packer/packer"
)

type StepExport struct {
	Format         string
	SkipExport     bool
	VMName         string
	OVFToolOptions []string
	OutputDir      string
}

func (s *StepExport) generateArgs(c *DriverConfig, hidePassword bool) []string {
	password := url.QueryEscape(c.RemotePassword)
	if hidePassword {
		password = "****"
	}
	args := []string{
		"--noSSLVerify=true",
		"--skipManifestCheck",
		"-tt=" + s.Format,
		"vi://" + c.RemoteUser + ":" + password + "@" + c.RemoteHost + "/" + s.VMName,
		s.OutputDir,
	}
	return append(s.OVFToolOptions, args...)
}

func (s *StepExport) Run(state multistep.StateBag) multistep.StepAction {
	c := state.Get("driverConfig").(*DriverConfig)
	ui := state.Get("ui").(packer.Ui)

	// Skip export if requested
	if s.SkipExport {
		ui.Say("Skipping export of virtual machine...")
		return multistep.ActionContinue
	}

	if c.RemoteType != "esx5" || s.Format == "" {
		return multistep.ActionContinue
	}

	ovftool, err := ovftoolFindOvfTool()
	if err != nil {
		err = fmt.Errorf("Could not find ovftool: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	// Export the VM
	if s.OutputDir == "" {
		s.OutputDir = s.VMName + "." + s.Format
	}

	if s.Format == "ova" {
		os.MkdirAll(s.OutputDir, 0755)
	}

	ui.Say("Exporting virtual machine...")
	ui.Message(fmt.Sprintf("Executing: %s %s", ovftool, strings.Join(s.generateArgs(c, true), " ")))
	var out bytes.Buffer
	cmd := exec.Command(ovftool, s.generateArgs(c, false)...)
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		err := fmt.Errorf("Error exporting virtual machine: %s\n%s\n", err, out.String())
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	ui.Message(fmt.Sprintf("%s", out.String()))

	return multistep.ActionContinue
}

func (s *StepExport) Cleanup(state multistep.StateBag) {}

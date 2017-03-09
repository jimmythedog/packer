package ovf

import (
	"path/filepath"
	"testing"

	"github.com/mitchellh/multistep"
	vmwcommon "github.com/mitchellh/packer/builder/vmware/common"
)

func TestStepImportOvf_impl(t *testing.T) {
	var _ multistep.Step = new(StepImportOvf)
}

func TestStepImportOvf(t *testing.T) {
	state := testState(t)

	dir := new(vmwcommon.LocalOutputDir)
	dir.SetOutputDir("foo")
	state.Put("dir", dir)

	// Create the source
	sourcePath := filepath.Join(dir.String(), "source.vmx")

	var config Config
	config.SourcePath = sourcePath
	config.VMName = "bar"
	state.Put("config", &config)
	driver := state.Get("driver").(*vmwcommon.DriverMock)

	driver.ImportOvfVmxPathResult = "abc"
	driver.ImportOvfVmdkPathResult = "def"

	step := new(StepImportOvf)

	// Test the run
	if action := step.Run(state); action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatal("should NOT have error")
	}

	// Test the driver
	if !driver.ImportOvfCalled {
		t.Fatal("should've called")
	}
	if driver.ImportOvfOvfPath != sourcePath {
		t.Fatal("should call with right ovf path")
	}
	if state.Get("full_disk_path") != driver.ImportOvfVmdkPathResult {
		t.Fatal("Incorrect full_disk_path value")
	}
	if state.Get("vmx_path") != driver.ImportOvfVmxPathResult {
		t.Fatal("Incorrect vmx_path value")
	}
}

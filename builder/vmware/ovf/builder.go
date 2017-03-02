package ovf

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/mitchellh/multistep"
	vmwcommon "github.com/mitchellh/packer/builder/vmware/common"
	iso "github.com/mitchellh/packer/builder/vmware/iso"
	"github.com/mitchellh/packer/common"
	"github.com/mitchellh/packer/helper/communicator"
	"github.com/mitchellh/packer/helper/config"
	"github.com/mitchellh/packer/packer"
	"github.com/mitchellh/packer/template/interpolate"
)

type Builder struct {
	config Config
	runner multistep.Runner
}

type Config struct {
	iso.Config `mapstructure:",squash"`

	SourcePath string `mapstructure:"source_path"`
	ctx        interpolate.Context
}

func (b *Builder) Prepare(raws ...interface{}) ([]string, error) {
	err := config.Decode(&b.config, &config.DecodeOpts{
		Interpolate:        true,
		InterpolateContext: &b.config.ctx,
		InterpolateFilter: &interpolate.RenderFilter{
			Exclude: []string{
				"boot_command",
				"tools_upload_path",
			},
		},
	}, raws...)
	if err != nil {
		return nil, err
	}

	// Accumulate any errors and warnings
	var errs *packer.MultiError
	warnings := make([]string, 0)

	errs = packer.MultiErrorAppend(errs, b.config.HTTPConfig.Prepare(&b.config.ctx)...)
	errs = packer.MultiErrorAppend(errs, b.config.DriverConfig.Prepare(&b.config.ctx)...)
	errs = packer.MultiErrorAppend(errs,
		b.config.OutputConfig.Prepare(&b.config.ctx, &b.config.PackerConfig)...)
	errs = packer.MultiErrorAppend(errs, b.config.RunConfig.Prepare(&b.config.ctx)...)
	errs = packer.MultiErrorAppend(errs, b.config.ShutdownConfig.Prepare(&b.config.ctx)...)
	errs = packer.MultiErrorAppend(errs, b.config.SSHConfig.Prepare(&b.config.ctx)...)
	errs = packer.MultiErrorAppend(errs, b.config.ToolsConfig.Prepare(&b.config.ctx)...)
	errs = packer.MultiErrorAppend(errs, b.config.VMXConfig.Prepare(&b.config.ctx)...)
	errs = packer.MultiErrorAppend(errs, b.config.FloppyConfig.Prepare(&b.config.ctx)...)

	if b.config.SourcePath == "" {
		errs = packer.MultiErrorAppend(errs, fmt.Errorf("source_path is blank, but is required"))
	} else {
		if _, err := os.Stat(b.config.SourcePath); err != nil {
			errs = packer.MultiErrorAppend(errs,
				fmt.Errorf("source_path is invalid: %s", err))
		}
	}

	if b.config.GuestOSType == "" {
		b.config.GuestOSType = "other"
	}

	if b.config.VMName == "" {
		b.config.VMName = fmt.Sprintf("packer-%s", b.config.PackerBuildName)
	}

	if b.config.Version == "" {
		b.config.Version = "9"
	}

	if b.config.VMXTemplatePath != "" {
		if err := b.validateVMXTemplatePath(); err != nil {
			errs = packer.MultiErrorAppend(
				errs, fmt.Errorf("vmx_template_path is invalid: %s", err))
		}

	}

	// Remote configuration validation
	if b.config.RemoteType != "" {
		if b.config.RemoteHost == "" {
			errs = packer.MultiErrorAppend(errs,
				fmt.Errorf("remote_host must be specified"))
		}
	}

	if b.config.Format != "" {
		if !(b.config.Format == "ova" || b.config.Format == "ovf" || b.config.Format == "vmx") {
			errs = packer.MultiErrorAppend(errs,
				fmt.Errorf("format must be one of ova, ovf, or vmx"))
		}
	}

	// Warnings
	if b.config.ShutdownCommand == "" {
		warnings = append(warnings,
			"A shutdown_command was not specified. Without a shutdown command, Packer\n"+
				"will forcibly halt the virtual machine, which may result in data loss.")
	}

	if errs != nil && len(errs.Errors) > 0 {
		return warnings, errs
	}

	return warnings, nil
}

func (b *Builder) Run(ui packer.Ui, hook packer.Hook, cache packer.Cache) (packer.Artifact, error) {
	driver, err := vmwcommon.NewDriver(&b.config.DriverConfig, &b.config.SSHConfig, &b.config.CommConfig, b.config.VMName)
	if err != nil {
		return nil, fmt.Errorf("Failed creating VMware driver: %s", err)
	}

	// Determine the output dir implementation
	var dir vmwcommon.OutputDir
	switch d := driver.(type) {
	case vmwcommon.OutputDir:
		dir = d
	default:
		dir = new(vmwcommon.LocalOutputDir)
	}
	if b.config.RemoteType != "" && b.config.Format != "" {
		b.config.OutputDir = b.config.VMName
	}
	dir.SetOutputDir(b.config.OutputDir)

	// Setup the state bag
	state := new(multistep.BasicStateBag)
	state.Put("cache", cache)
	state.Put("esxova_config", &b.config)
	state.Put("config", &b.config.Config)
	state.Put("debug", b.config.PackerDebug)
	state.Put("dir", dir)
	state.Put("driver", driver)
	state.Put("hook", hook)
	state.Put("ui", ui)
	state.Put("sshConfig", &b.config.SSHConfig)

	steps := []multistep.Step{
		&vmwcommon.StepPrepareTools{
			RemoteType:        b.config.Config.RemoteType,
			ToolsUploadFlavor: b.config.Config.ToolsUploadFlavor,
		},
		&common.StepCreateFloppy{
			Files:       b.config.Config.FloppyConfig.FloppyFiles,
			Directories: b.config.Config.FloppyConfig.FloppyDirectories,
		},
		&StepDeployOvf{
			Format: b.config.Config.Format,
		},
		&StepDownloadVMX{
			RemoteType: b.config.Config.RemoteType,
		},
		&vmwcommon.StepConfigureVMX{
			CustomData: b.config.Config.VMXData,
			VMName:     b.config.Config.VMName,
		},
		&vmwcommon.StepSuppressMessages{},
		&common.StepHTTPServer{
			HTTPDir:     b.config.Config.HTTPDir,
			HTTPPortMin: b.config.Config.HTTPPortMin,
			HTTPPortMax: b.config.Config.HTTPPortMax,
		},
		&vmwcommon.StepConfigureVNC{
			VNCBindAddress:     b.config.Config.VNCBindAddress,
			VNCPortMin:         b.config.Config.VNCPortMin,
			VNCPortMax:         b.config.Config.VNCPortMax,
			VNCDisablePassword: b.config.Config.VNCDisablePassword,
		},
		&vmwcommon.StepUploadVMX{
			RemoteType: b.config.Config.RemoteType,
		},
		&vmwcommon.StepRun{
			BootWait:           b.config.Config.BootWait,
			DurationBeforeStop: 5 * time.Second,
			Headless:           b.config.Config.Headless,
		},
		&vmwcommon.StepTypeBootCommand{
			BootCommand: b.config.Config.BootCommand,
			VMName:      b.config.Config.VMName,
			Ctx:         b.config.ctx,
		},
		&communicator.StepConnect{
			Config:    &b.config.Config.SSHConfig.Comm,
			Host:      driver.CommHost,
			SSHConfig: vmwcommon.SSHConfigFunc(&b.config.Config.SSHConfig),
		},
		&vmwcommon.StepUploadTools{
			RemoteType:        b.config.Config.RemoteType,
			ToolsUploadFlavor: b.config.Config.ToolsUploadFlavor,
			ToolsUploadPath:   b.config.Config.ToolsUploadPath,
			Ctx:               b.config.ctx,
		},
		&common.StepProvision{},
		&vmwcommon.StepShutdown{
			Command: b.config.Config.ShutdownCommand,
			Timeout: b.config.Config.ShutdownTimeout,
		},
		&vmwcommon.StepCleanFiles{},
		&vmwcommon.StepCompactDisk{
			Skip: b.config.Config.SkipCompaction,
		},
		&vmwcommon.StepConfigureVMX{
			CustomData: b.config.Config.VMXDataPost,
			SkipFloppy: true,
		},
		&vmwcommon.StepCleanVMX{},
		&iso.StepExport{
			Format: b.config.Config.Format,
		},
	}

	// Run!
	b.runner = common.NewRunnerWithPauseFn(steps, b.config.PackerConfig, ui, state)
	b.runner.Run(state)

	// If there was an error, return that
	if rawErr, ok := state.GetOk("error"); ok {
		return nil, rawErr.(error)
	}

	// If we were interrupted or cancelled, then just exit.
	if _, ok := state.GetOk(multistep.StateCancelled); ok {
		return nil, errors.New("Build was cancelled.")
	}

	if _, ok := state.GetOk(multistep.StateHalted); ok {
		return nil, errors.New("Build was halted.")
	}

	// Compile the artifact list
	var files []string
	if b.config.RemoteType != "" && b.config.Format != "" {
		dir = new(vmwcommon.LocalOutputDir)
		dir.SetOutputDir(b.config.OutputDir)
		files, err = dir.ListFiles()
	} else {
		files, err = state.Get("dir").(vmwcommon.OutputDir).ListFiles()
	}
	if err != nil {
		return nil, err
	}
	return vmwcommon.NewArtifact(dir, files, b.config.RemoteType != ""), nil
}

func (b *Builder) Cancel() {
	if b.runner != nil {
		log.Println("Cancelling the step runner...")
		b.runner.Cancel()
	}
}

func (b *Builder) validateVMXTemplatePath() error {
	f, err := os.Open(b.config.VMXTemplatePath)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	return interpolate.Validate(string(data), &b.config.ctx)
}

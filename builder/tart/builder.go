//go:generate packer-sdc mapstructure-to-hcl2 -type Config

package tart

import (
	"context"
	"errors"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/packer-plugin-sdk/bootcommand"
	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

const BuilderId = "tart.builder"

type Config struct {
	common.PackerConfig   `mapstructure:",squash"`
	bootcommand.VNCConfig `mapstructure:",squash"`
	FromIPSW              string              `mapstructure:"from_ipsw" required:"true"`
	VMName                string              `mapstructure:"vm_name" required:"true"`
	VMBaseName            string              `mapstructure:"vm_base_name" required:"true"`
	CpuCount              uint8               `mapstructure:"cpu_count" required:"false"`
	MemoryGb              uint16              `mapstructure:"memory_gb" required:"false"`
	Display               string              `mapstructure:"display" required:"false"`
	DiskSizeGb            uint16              `mapstructure:"disk_size_gb" required:"false"`
	Headless              bool                `mapstructure:"headless" required:"false"`
	Comm                  communicator.Config `mapstructure:",squash"`

	ctx interpolate.Context
}

type Builder struct {
	config Config
	runner multistep.Runner
}

func (b *Builder) ConfigSpec() hcldec.ObjectSpec { return b.config.FlatMapstructure().HCL2Spec() }

func (b *Builder) Prepare(raws ...interface{}) (generatedVars []string, warnings []string, err error) {
	err = config.Decode(&b.config, &config.DecodeOpts{
		PluginType:  "packer.builder.tart",
		Interpolate: true,
	}, raws...)
	if err != nil {
		return nil, nil, err
	}
	var errs *packer.MultiError
	errs = packer.MultiErrorAppend(errs, b.config.Comm.Prepare(&b.config.ctx)...)
	return nil, nil, nil
}

func (b *Builder) Run(ctx context.Context, ui packer.Ui, hook packer.Hook) (packer.Artifact, error) {
	steps := []multistep.Step{}

	if b.config.FromIPSW != "" {
		steps = append(steps, new(stepCreateVM))
	} else {
		steps = append(steps, new(stepCloneVM))
	}

	steps = append(steps,
		new(stepSetVM),
		new(stepDiskFilePrepare),
		new(stepRun),
		&communicator.StepConnect{
			Config:    &b.config.Comm,
			Host:      TartMachineIP(b.config.VMName),
			SSHConfig: b.config.Comm.SSHConfigFunc(),
		},
		new(stepResize),
		&commonsteps.StepProvision{},
	)

	// Setup the state bag and initial state for the steps
	state := new(multistep.BasicStateBag)
	state.Put("config", &b.config)
	state.Put("debug", b.config.PackerDebug)
	state.Put("hook", hook)
	state.Put("ui", ui)

	// Run
	b.runner = commonsteps.NewRunnerWithPauseFn(steps, b.config.PackerConfig, ui, state)
	b.runner.Run(ctx, state)

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

	artifact := &TartVMArtifact{
		VMName:    b.config.VMName,
		StateData: map[string]interface{}{"generated_data": state.Get("generated_data")},
	}
	return artifact, nil
}

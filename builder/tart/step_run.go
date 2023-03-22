package tart

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/hashicorp/packer-plugin-sdk/bootcommand"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	"github.com/mitchellh/go-vnc"
	"net"
	"os/exec"
	"regexp"
	"time"
)

var ErrFailedToDetectHostIP = errors.New("failed to detect host IP")

var vncRegexp = regexp.MustCompile("vnc://.*:(.*)@(.*):([0-9]{1,5})")

type stepRun struct {
	vmName string
}

type bootCommandTemplateData struct {
	HTTPIP   string
	HTTPPort int
}

func (s *stepRun) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packersdk.Ui)

	ui.Say("Starting the virtual machine...")
	runArgs := []string{"run", config.VMName}
	if config.Headless {
		runArgs = append(runArgs, "--no-graphics")
	} else {
		runArgs = append(runArgs, "--graphics")
	}
	if !config.DisableVNC {
		runArgs = append(runArgs, "--vnc-experimental")
	}
	if config.Recovery {
		runArgs = append(runArgs, "--recovery")
	}
	if config.Rosetta != "" {
		runArgs = append(runArgs, fmt.Sprintf("--rosetta=%s", config.Rosetta))
	}
	cmd := exec.Command("tart", runArgs...)
	stdout := bytes.NewBufferString("")
	cmd.Stdout = stdout
	cmd.Stderr = uiWriter{ui: ui}

	// Prevent the Tart from opening the Screen Sharing
	// window connected to the VNC server we're starting
	if !config.DisableVNC {
		cmd.Env = cmd.Environ()
		cmd.Env = append(cmd.Env, "CI=true")
	}

	if err := cmd.Start(); err != nil {
		err = fmt.Errorf("Error starting VM: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	state.Put("tart-cmd", cmd)

	if (len(config.FromISO) == 0) && !config.DisableVNC {
		if !typeBootCommandOverVNC(ctx, state, config, ui, stdout) {
			return multistep.ActionHalt
		}
	}

	ui.Say("Successfully started the virtual machine...")

	return multistep.ActionContinue
}

type uiWriter struct {
	ui packersdk.Ui
}

func (u uiWriter) Write(p []byte) (n int, err error) {
	u.ui.Say(string(p))
	return len(p), nil
}

// Cleanup stops the VM.
func (s *stepRun) Cleanup(state multistep.StateBag) {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packersdk.Ui)

	communicator := state.Get("communicator")
	if communicator != nil {
		ui.Say("Gracefully shutting down the VM...")

		shutdownCmd := packersdk.RemoteCmd{
			Command: fmt.Sprintf("echo %s | sudo -S shutdown -h now", config.Comm.Password()),
		}

		err := shutdownCmd.RunWithUi(context.Background(), communicator.(packersdk.Communicator), ui)
		if err != nil {
			ui.Say("Failed to gracefully shutdown VM...")
			ui.Error(err.Error())
		}
	}

	cmd := state.Get("tart-cmd").(*exec.Cmd)

	if cmd != nil {
		ui.Say("Waiting for the tart process to exit...")
		_, _ = cmd.Process.Wait()
	}
}

func typeBootCommandOverVNC(
	ctx context.Context,
	state multistep.StateBag,
	config *Config,
	ui packersdk.Ui,
	tartRunStdout *bytes.Buffer,
) bool {
	if config.HTTPDir != "" || len(config.HTTPContent) != 0 {
		ui.Say("Detecting host IP...")

		hostIP, err := detectHostIP(config)
		if err != nil {
			err := fmt.Errorf("Failed to detect the host IP address: %v", err)
			state.Put("error", err)
			ui.Error(err.Error())

			return false
		}

		ui.Say(fmt.Sprintf("Host IP is assumed to be %s", hostIP))

		// Should be already filled by the Packer's commonsteps.StepHTTPServer
		httpPort := state.Get("http_port").(int)

		config.ctx.Data = &bootCommandTemplateData{
			HTTPIP:   hostIP,
			HTTPPort: httpPort,
		}
	}

	ui.Say("Waiting for the VNC server credentials from Tart...")

	var vncPassword string
	var vncHost string
	var vncPort string

	for {
		matches := vncRegexp.FindStringSubmatch(tartRunStdout.String())
		if len(matches) == 1+vncRegexp.NumSubexp() {
			vncPassword = matches[1]
			vncHost = matches[2]
			vncPort = matches[3]

			break
		}

		time.Sleep(time.Second)
	}

	ui.Say("Retrieved VNC credentials, connecting...")

	netConn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", vncHost, vncPort))
	if err != nil {
		err := fmt.Errorf("Failed to connect to the Tart's VNC server: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())

		return false
	}
	defer netConn.Close()

	vncClient, err := vnc.Client(netConn, &vnc.ClientConfig{
		Auth: []vnc.ClientAuth{
			&vnc.PasswordAuth{Password: vncPassword},
		},
	})
	if err != nil {
		err := fmt.Errorf("Failed to connect to the Tart's VNC server: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())

		return false
	}
	defer vncClient.Close()

	ui.Say("Connected to the VNC!")

	vncDriver := bootcommand.NewVNCDriver(vncClient, config.BootKeyInterval)

	ui.Say("Typing the commands over VNC...")

	command, err := interpolate.Render(config.VNCConfig.FlatBootCommand(), &config.ctx)
	if err != nil {
		err := fmt.Errorf("Failed to render the boot command: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())

		return false
	}

	seq, err := bootcommand.GenerateExpressionSequence(command)
	if err != nil {
		err := fmt.Errorf("Failed to parse the boot command: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())

		return false
	}

	if err := seq.Do(ctx, vncDriver); err != nil {
		err := fmt.Errorf("Failed to run the boot command: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())

		return false
	}

	return true
}

func detectHostIP(config *Config) (string, error) {
	if config.HTTPAddress != "" {
		return config.HTTPAddress, nil
	}

	vmIPRaw, err := TartExec("ip", "--wait", "120", config.VMName)
	if err != nil {
		return "", fmt.Errorf("%w: while running \"tart ip\": %v",
			ErrFailedToDetectHostIP, err)
	}
	vmIP := net.ParseIP(vmIPRaw)

	// Find the interface that has this IP
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("%w: while retrieving interfaces: %v",
			ErrFailedToDetectHostIP, err)
	}

	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return "", fmt.Errorf("%w: while retrieving interface addresses: %v",
				ErrFailedToDetectHostIP, err)
		}

		for _, addr := range addrs {
			_, net, err := net.ParseCIDR(addr.String())
			if err != nil {
				return "", fmt.Errorf("%w: while parsing interface CIDR: %v",
					ErrFailedToDetectHostIP, err)
			}

			if net.Contains(vmIP) {
				gatewayIP, err := cidr.Host(net, 1)
				if err != nil {
					return "", fmt.Errorf("%w: while calculating gateway IP: %v",
						ErrFailedToDetectHostIP, err)
				}

				return gatewayIP.String(), nil
			}
		}
	}

	return "", fmt.Errorf("%w: no suitable interface found", ErrFailedToDetectHostIP)
}

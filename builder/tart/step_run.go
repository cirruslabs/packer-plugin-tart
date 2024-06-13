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
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var ErrFailedToDetectHostIP = errors.New("failed to detect host IP")

var vncRegexp = regexp.MustCompile("vnc://.*:(.*)@(.*):([0-9]{1,5})")

type stepRun struct{}

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
	for _, iso := range config.FromISO {
		runArgs = append(runArgs, fmt.Sprintf("--disk=%s:ro", iso))
	}
	if len(config.RunExtraArgs) > 0 {
		runArgs = append(runArgs, config.RunExtraArgs...)
	}
	cmd := exec.CommandContext(ctx, tartCommand, runArgs...)
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

	ui.Say("Successfully started the virtual machine...")

	if len(config.BootCommand) > 0 && !config.DisableVNC {
		if !typeBootCommandOverVNC(ctx, state, config, ui, stdout) {
			return multistep.ActionHalt
		}
	}

	return multistep.ActionContinue
}

type uiWriter struct {
	ui packersdk.Ui
}

func (u uiWriter) Write(p []byte) (n int, err error) {
	u.ui.Error(strings.TrimSpace(string(p)))
	return len(p), nil
}

// Cleanup stops the VM.
func (s *stepRun) Cleanup(state multistep.StateBag) {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packersdk.Ui)
	cmd := state.Get("tart-cmd").(*exec.Cmd)
	if cmd == nil {
		return // Nothing to shut down
	}

	communicator := state.Get("communicator")
	if communicator != nil {
		ui.Say("Gracefully shutting down the VM...")
		shutdownCmd := packersdk.RemoteCmd{
			Command: fmt.Sprintf("echo %s | sudo -S -p '' shutdown -h now", config.CommunicatorConfig.Password()),
		}

		err := shutdownCmd.RunWithUi(context.Background(), communicator.(packersdk.Communicator), ui)
		if err != nil {
			ui.Say("Failed to gracefully shutdown VM...")
			ui.Error(err.Error())
		}
	} else {
		ui.Say("Shutting down the VM...")
		err := cmd.Process.Signal(os.Interrupt)
		if err != nil {
			ui.Say("Failed to shutdown VM...")
			ui.Error(err.Error())
		}
	}

	// Always wait, even if we didn't initiate shutdown,
	// so that we properly read and close stdout/stderr.
	ui.Say("Waiting for the tart process to exit...")
	_, _ = cmd.Process.Wait()
}

func typeBootCommandOverVNC(
	ctx context.Context,
	state multistep.StateBag,
	config *Config,
	ui packersdk.Ui,
	tartRunStdout *bytes.Buffer,
) bool {
	ui.Say("Typing boot commands over VNC...")

	if config.HTTPDir != "" || len(config.HTTPContent) != 0 {
		ui.Say("Detecting host IP...")

		hostIP, err := detectHostIP(ctx, config)
		if err != nil {
			err := fmt.Errorf("Failed to detect the host IP address: %v", err)
			state.Put("error", err)
			ui.Error(err.Error())

			return false
		}

		ui.Say(fmt.Sprintf("Host IP is assumed to be %s", hostIP))
		state.Put("http_ip", hostIP)

		// Should be already filled by the Packer's commonsteps.StepHTTPServer
		httpPort := state.Get("http_port").(int)

		config.ctx.Data = &bootCommandTemplateData{
			HTTPIP:   hostIP,
			HTTPPort: httpPort,
		}
	}

	ui.Say("Waiting for the VNC server credentials from Tart...")

	vncCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

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

		select {
		case <-vncCtx.Done():
			return false
		case <-time.After(time.Second):
			// continue
		}
	}

	ui.Say("Retrieved VNC credentials, connecting...")
	ui.Message(fmt.Sprintf(
		"If you want to view the screen of the VM, connect via VNC with the password \"%s\" to\n"+
			"vnc://%s:%s", vncPassword, vncHost, vncPort))

	dialer := net.Dialer{}
	netConn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:%s", vncHost, vncPort))
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

	if config.VNCConfig.BootWait > 0 {
		message := fmt.Sprintf("Waiting %v after the VM has booted...", config.VNCConfig.BootWait)
		ui.Say(message)
		time.Sleep(config.VNCConfig.BootWait)
	}

	message := fmt.Sprintf("Typing commands with key interval %v...", config.BootKeyInterval)
	ui.Say(message)

	vncDriver := bootcommand.NewVNCDriver(vncClient, config.BootKeyInterval)

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

	ui.Say("Done typing commands!")

	return true
}

func detectHostIP(ctx context.Context, config *Config) (string, error) {
	if config.HTTPAddress != "0.0.0.0" {
		return config.HTTPAddress, nil
	}

	vmIPRaw, err := TartMachineIP(ctx, config.VMName, config.IpExtraArgs)
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

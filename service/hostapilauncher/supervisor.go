// Package hostapilauncher only supervises the independently packaged Host API.
// It neither parses stream tickets nor duplicates Host API authorization.
package hostapilauncher

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rancher/log"
)

const RequiredVersion = "0.38.5"

type Settings struct {
	Binary string
	Values map[string]string
}

type supervisor struct {
	validate func(context.Context) error
	execute  func(context.Context) error
	pause    func(context.Context) bool
	warning  func()
}

// Run never falls back to the embedded legacy implementation. A missing or
// mismatched producer can be repaired by the existing configcontent update;
// event processing stays available while only Host API admission is closed.
func Run(ctx context.Context, settings Settings) {
	environment := childEnvironment(os.Environ(), settings.Values)
	runner := supervisor{
		validate: func(ctx context.Context) error { return validateProducer(ctx, settings.Binary, environment) },
		execute: func(ctx context.Context) error {
			command := producerCommand(ctx, settings.Binary, environment)
			command.Stdout, command.Stderr = os.Stdout, os.Stderr
			return command.Run()
		},
		pause: func(ctx context.Context) bool {
			timer := time.NewTimer(2 * time.Second)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return false
			case <-timer.C:
				return true
			}
		},
		warning: func() { log.Warn("Host API producer unavailable; retrying without a legacy fallback") },
	}
	runner.run(ctx)
}

func (runner supervisor) run(ctx context.Context) {
	for ctx.Err() == nil {
		if runner.validate(ctx) == nil {
			_ = runner.execute(ctx)
		}
		if ctx.Err() != nil {
			return
		}
		runner.warning()
		if !runner.pause(ctx) {
			return
		}
	}
}

func producerCommand(ctx context.Context, binary string, environment []string, arguments ...string) *exec.Cmd {
	command := exec.CommandContext(ctx, binary, arguments...)
	command.Env = environment
	bindParentLifetime(command)
	return command
}

func validateProducer(ctx context.Context, binary string, environment []string) error {
	if !filepath.IsAbs(binary) {
		return errors.New("producer path must be absolute")
	}
	info, err := os.Lstat(binary)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0022 != 0 {
		return errors.New("producer must be an owned non-writable regular executable")
	}
	if err := validateProducerOwner(info); err != nil {
		return err
	}
	versionContext, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	command := producerCommand(versionContext, binary, environment, "--version")
	output := &boundedOutput{}
	command.Stdout, command.Stderr = output, io.Discard
	if err := command.Run(); err != nil || output.overflow || !validVersion(output.String()) {
		return errors.New("producer version is unavailable or incompatible")
	}
	return nil
}

func validVersion(value string) bool {
	return value == RequiredVersion+"\n" || value == RequiredVersion
}

type boundedOutput struct {
	bytes.Buffer
	overflow bool
}

func (output *boundedOutput) Write(data []byte) (int, error) {
	if len(data) > 128-output.Len() {
		output.overflow = true
		return 0, io.ErrShortWrite
	}
	return output.Buffer.Write(data)
}

func childEnvironment(original []string, values map[string]string) []string {
	result := make([]string, 0, len(original)+len(values))
	for _, entry := range original {
		name, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, "HOST_API_") {
			continue // Configuration comes from the verified current Node identity.
		}
		if _, replaced := values[name]; !replaced {
			result = append(result, entry)
		}
	}
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		result = append(result, name+"="+values[name])
	}
	return result
}

//go:build linux

package hostapilauncher

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestProducerIsBoundToLinuxParentDeathAndRejectsUnsafeExecutable(t *testing.T) {
	command := producerCommand(context.Background(), "/var/lib/cattle/bin/host-api", nil)
	if command.SysProcAttr == nil || command.SysProcAttr.Pdeathsig != syscall.SIGKILL {
		t.Fatal("orphan producer is possible")
	}
	directory := t.TempDir()
	file := filepath.Join(directory, "host-api")
	if err := os.WriteFile(file, []byte("#!/bin/sh\nprintf '0.38.5\\n'\n"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := validateProducer(context.Background(), file, os.Environ()); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(file, 0777); err != nil {
		t.Fatal(err)
	}
	if err := validateProducer(context.Background(), file, os.Environ()); err == nil {
		t.Fatal("writable producer accepted")
	}
	if err := os.Chmod(file, 0755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(directory, "link")
	if err := os.Symlink(file, link); err != nil {
		t.Fatal(err)
	}
	if err := validateProducer(context.Background(), link, os.Environ()); err == nil {
		t.Fatal("symlink producer accepted")
	}
}

func TestNodeCancellationStopsAnActualProducerProcessAtReadyBarrier(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	input, inputWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	defer inputWriter.Close()
	output, outputWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	defer outputWriter.Close()
	command := producerCommand(ctx, "/bin/sh", os.Environ(), "-c", "printf ready; read task_line")
	command.Stdin, command.Stdout = input, outputWriter
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	ready := make([]byte, 5)
	if _, err := io.ReadFull(output, ready); err != nil || string(ready) != "ready" {
		t.Fatal("producer ready barrier failed")
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	cancel()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("producer unexpectedly exited without cancellation")
		}
	case <-deadline.C:
		_ = command.Process.Kill()
		t.Fatal("producer survived Node cancellation")
	}
}

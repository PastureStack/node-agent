package hostapilauncher

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestProducerVersionIsExactAndBounded(t *testing.T) {
	for _, version := range []string{"0.38.5", "0.38.5\n"} {
		if !validVersion(version) {
			t.Fatal("current producer rejected")
		}
	}
	for _, version := range []string{"0.38.4", "0.38.5-dirty", "0.38.5\nextra", ""} {
		if validVersion(version) {
			t.Fatal("incompatible producer accepted")
		}
	}
	output := &boundedOutput{}
	if _, err := output.Write(make([]byte, 129)); err == nil || !output.overflow {
		t.Fatal("version output was unbounded")
	}
}

func TestNodeIdentityUsesEnvironmentAndNeverSecretArguments(t *testing.T) {
	values := map[string]string{"HOST_API_PLATFORM_SECRET_KEY": "agent-secret", "HOST_API_PLATFORM_URL": "http://engine:8080/v2-beta", "PASTURESTACK_HOME": "/var/lib/cattle"}
	environment := childEnvironment([]string{"PATH=/usr/bin", "HOST_API_CONFIG_FILE=/untrusted/config", "HOST_API_PLATFORM_URL=http://foreign", "PASTURESTACK_HOME=/wrong"}, values)
	command := producerCommand(context.Background(), "/var/lib/cattle/bin/host-api", environment)
	if !reflect.DeepEqual(command.Args, []string{"/var/lib/cattle/bin/host-api"}) {
		t.Fatal("producer received extra arguments")
	}
	for _, entry := range environment {
		if strings.HasPrefix(entry, "HOST_API_CONFIG_FILE=") || strings.Contains(entry, "foreign") || strings.Contains(entry, "/wrong") {
			t.Fatal("untrusted configuration inherited")
		}
	}
	if !strings.Contains(strings.Join(environment, "\n"), "HOST_API_PLATFORM_SECRET_KEY=agent-secret") {
		t.Fatal("agent identity was not forwarded in memory")
	}
}

func TestWrongVersionNeverStartsOrFallsBack(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started, warnings := 0, 0
	runner := supervisor{
		validate: func(context.Context) error { return errors.New("wrong version") },
		execute:  func(context.Context) error { started++; return nil },
		warning:  func() { warnings++ },
		pause:    func(context.Context) bool { cancel(); return false },
	}
	runner.run(ctx)
	if started != 0 || warnings != 1 {
		t.Fatal("missing producer did not fail closed")
	}
}

func TestChildExitRestartsOnlyAtControlledBarrierAndStopsWithNode(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	retry := make(chan struct{})
	started := make(chan int, 2)
	done := make(chan struct{})
	count := 0
	runner := supervisor{
		validate: func(context.Context) error { return nil },
		execute: func(childContext context.Context) error {
			count++
			started <- count
			if count == 1 {
				return errors.New("child exit")
			}
			<-childContext.Done()
			return childContext.Err()
		},
		warning: func() {},
		pause: func(ctx context.Context) bool {
			select {
			case <-retry:
				return true
			case <-ctx.Done():
				return false
			}
		},
	}
	go func() { runner.run(ctx); close(done) }()
	if <-started != 1 {
		t.Fatal("first child missing")
	}
	select {
	case <-started:
		t.Fatal("child restarted before barrier")
	default:
	}
	retry <- struct{}{}
	if <-started != 2 {
		t.Fatal("replacement child missing")
	}
	cancel()
	<-done
	if count != 2 {
		t.Fatal("child restarted after Node cancellation")
	}
}

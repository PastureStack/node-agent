package main

import (
	"os"
	"testing"
)

func TestFixturePathsIgnoreHostEnvironment(t *testing.T) {
	for name, value := range map[string]string{
		"CONVOY_DRIVER_NAME": "../../unsafe",
		"CONVOY_SOCKET":      "/etc/unsafe.sock",
		"CONVOY_DATA_DIR":    "/etc/unsafe",
	} {
		previous, existed := os.LookupEnv(name)
		if err := os.Setenv(name, value); err != nil {
			t.Fatal(err)
		}
		defer func(name, previous string, existed bool) {
			if existed {
				_ = os.Setenv(name, previous)
			} else {
				_ = os.Unsetenv(name)
			}
		}(name, previous, existed)
	}

	driver, socket, root := fixturePaths()
	if driver != testDriver || socket != testSocket || root != testRoot {
		t.Fatalf("test fixture paths changed through the environment: %q %q %q", driver, socket, root)
	}
}

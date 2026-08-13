package handlers

import "testing"

func TestApprovedConfigItemPreservesKnownFirstPartyItems(t *testing.T) {
	for _, input := range []string{
		"configscripts",
		"host-config",
		"metadata-answers",
		"psk",
		"pyagent",
		"system-stacks",
	} {
		got, ok := approvedConfigItem(input)
		if !ok || got != input {
			t.Fatalf("approvedConfigItem(%q) = %q, %v", input, got, ok)
		}
	}
}

func TestApprovedConfigItemRejectsCommandOptionsAndPaths(t *testing.T) {
	for _, input := range []string{
		"--archive-url=https://attacker.invalid/payload",
		"--force",
		"../../tmp/payload",
		"host-config;touch /tmp/pwned",
		"host-config\nmetadata",
		"unknown-extension",
		"",
	} {
		if got, ok := approvedConfigItem(input); ok || got != "" {
			t.Fatalf("approvedConfigItem(%q) unexpectedly accepted %q", input, got)
		}
	}
}

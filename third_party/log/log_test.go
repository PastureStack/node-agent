package log

import "testing"

func TestSafeMessageProducesSingleRecord(t *testing.T) {
	if got := safeMessage("first\r\nforged\nthird"); got != "first forged third" {
		t.Fatalf("unexpected safe log value: %q", got)
	}
}

package exec

import "testing"

func TestParseTTYResizeAcceptsNormalDimensions(t *testing.T) {
	options, err := parseTTYResize(":resizeTTY:160,48")
	if err != nil {
		t.Fatal(err)
	}
	if options.Width != 160 || options.Height != 48 {
		t.Fatalf("unexpected dimensions: %#v", options)
	}
}

func TestParseTTYResizeRejectsInvalidDimensions(t *testing.T) {
	for _, value := range []string{
		":resizeTTY:18446744073709551615,48",
		":resizeTTY:160,18446744073709551615",
		":resizeTTY:0,48",
		":resizeTTY:-1,48",
		":resizeTTY:160",
	} {
		if _, err := parseTTYResize(value); err == nil {
			t.Fatalf("unsafe dimensions were accepted: %q", value)
		}
	}
}

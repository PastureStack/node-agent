package ping

import "testing"

func TestLocalStorageMiB(t *testing.T) {
	const mib = uint64(1024 * 1024)
	for _, test := range []struct {
		name      string
		freeBytes uint64
		want      uint64
	}{
		{name: "zero"},
		{name: "less than one MiB", freeBytes: mib - 1},
		{name: "one MiB", freeBytes: mib, want: 1},
		{name: "one GiB", freeBytes: 1024 * mib, want: 1024},
		{name: "fraction floors", freeBytes: 3*mib + mib/2, want: 3},
		{name: "maximum uint64", freeBytes: ^uint64(0), want: (uint64(1) << 44) - 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := localStorageMiB(test.freeBytes); got != test.want {
				t.Fatalf("%d free bytes reported as %d MiB, want %d", test.freeBytes, got, test.want)
			}
		})
	}
}

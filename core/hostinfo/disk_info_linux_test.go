//go:build linux

package hostinfo

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
)

func diskFixture(t *testing.T, sameNamespace bool) (string, string) {
	t.Helper()
	root := t.TempDir()
	proc := filepath.Join(root, "proc")
	hostRoot := filepath.Join(root, "host-root")
	for _, dir := range []string{hostRoot, filepath.Join(proc, "1", "ns"), filepath.Join(proc, "self", "ns")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	writeDiskFixture(t, filepath.Join(proc, "filesystems"), "ext4\nsquashfs\n")
	selfNamespace := filepath.Join(proc, "self", "ns", "mnt")
	if err := os.Symlink("mnt:[1]", selfNamespace); err != nil {
		t.Fatal(err)
	}
	hostNamespace := "mnt:[2]"
	if sameNamespace {
		hostNamespace = "mnt:[1]"
	}
	if err := os.Symlink(hostNamespace, filepath.Join(proc, "1", "ns", "mnt")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(hostRoot, filepath.Join(proc, "1", "root")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOST_PROC", proc)
	t.Setenv("HOST_PROC_MOUNTINFO", "")
	return proc, hostRoot
}

func writeDiskFixture(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestDiskCollectorUsesEnumeratedHostRoot(t *testing.T) {
	proc, hostRoot := diskFixture(t, false)
	mountpoint := "/pasturestack-host-only-" + filepath.Base(hostRoot)
	if err := os.Mkdir(filepath.Join(hostRoot, mountpoint), 0700); err != nil {
		t.Fatal(err)
	}
	writeDiskFixture(t, filepath.Join(proc, "1", "mountinfo"),
		fmt.Sprintf("1 0 8:1 / %s rw - ext4 /dev/disk-a rw\n", mountpoint))

	data, err := (DiskCollector{Unit: 1048576}).GetData()
	if err != nil {
		t.Fatal(err)
	}
	filesystems := data["fileSystems"].(map[string]interface{})
	if len(filesystems) != 1 || filesystems["/dev/disk-a"].(map[string]interface{})["capacity"].(float64) <= 0 {
		t.Fatalf("host-only mount capacity missing: %v", filesystems)
	}
	if got := diskUsageRoot(); got != filepath.Join(proc, "1", "root") {
		t.Fatalf("wrong host root: %s", got)
	}
}

func TestDiskCollectorPreservesPartialDataAndSkipsSnapCapacity(t *testing.T) {
	proc, hostRoot := diskFixture(t, false)
	if err := os.MkdirAll(filepath.Join(hostRoot, "snap", "core24", "1643"), 0700); err != nil {
		t.Fatal(err)
	}
	writeDiskFixture(t, filepath.Join(proc, "1", "mountinfo"),
		"1 0 7:0 / /snap/core24/1643 ro - squashfs /dev/loop0 ro\n"+
			"2 0 8:2 / /missing rw - ext4 /dev/disk-missing rw\n"+
			"3 0 8:1 / / rw - ext4 /dev/disk-a rw\n")

	data, err := (DiskCollector{Unit: 1048576}).GetData()
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial collection must report the unreadable mount: %v", err)
	}
	for _, field := range []string{"fileSystems", "mountPoints"} {
		entries := data[field].(map[string]interface{})
		if len(entries) != 1 || entries["/dev/disk-a"] == nil {
			t.Fatalf("%s lost valid data or counted Snap: %v", field, entries)
		}
	}
}

func TestDiskCollectorPreservesDirectHostMode(t *testing.T) {
	proc, _ := diskFixture(t, true)
	mountpoint := t.TempDir()
	writeDiskFixture(t, filepath.Join(proc, "1", "mountinfo"),
		fmt.Sprintf("1 0 8:1 / %s rw - ext4 /dev/disk-a rw\n", mountpoint))
	if err := os.Remove(filepath.Join(proc, "1", "root")); err != nil {
		t.Fatal(err)
	}

	data, err := (DiskCollector{Unit: 1048576}).GetData()
	if err != nil || len(data["fileSystems"].(map[string]interface{})) != 1 {
		t.Fatalf("direct host mode must not need PID 1 root access: %v %v", data, err)
	}
	if got := diskUsageRoot(); got != "" {
		t.Fatalf("direct host mode unexpectedly rebased: %s", got)
	}
}

func TestDiskUsageRootHonorsMountInfoOverrideAndSelfFallback(t *testing.T) {
	proc, _ := diskFixture(t, false)
	writeDiskFixture(t, filepath.Join(proc, "self", "mountinfo"), "")
	if got := diskUsageRoot(); got != "" {
		t.Fatalf("missing PID 1 table must use self: %s", got)
	}

	custom := filepath.Join(proc, "custom")
	if err := os.MkdirAll(filepath.Join(custom, "ns"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("mnt:[3]", filepath.Join(custom, "ns", "mnt")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOST_PROC_MOUNTINFO", filepath.Join(custom, "mountinfo"))
	if got := diskUsageRoot(); got != filepath.Join(custom, "root") {
		t.Fatalf("override root differs from enumerated process: %s", got)
	}
}

func TestDiskNamespaceReadsOnlyAnIdentifier(t *testing.T) {
	for _, target := range []string{"/etc/passwd", "../other/ns/mnt", "mnt:[1]/../../other", "mnt:[+1]", "mnt:[-1]", "mnt:[]", "mnt:[0]", "mnt:[18446744073709551616]", "mnt:[1]\n"} {
		t.Run(target, func(t *testing.T) {
			process := t.TempDir()
			if err := os.Mkdir(filepath.Join(process, "ns"), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, filepath.Join(process, "ns", "mnt")); err != nil {
				t.Fatal(err)
			}
			if _, err := diskMountNamespace(process); !errors.Is(err, os.ErrInvalid) {
				t.Fatalf("non-namespace link must be rejected, got %v", err)
			}
		})
	}
}

func TestDiskNamespaceReadFailureDoesNotAssumeSameNamespace(t *testing.T) {
	proc, _ := diskFixture(t, false)
	writeDiskFixture(t, filepath.Join(proc, "1", "mountinfo"), "")
	if err := os.Remove(filepath.Join(proc, "1", "ns", "mnt")); err != nil {
		t.Fatal(err)
	}
	if got := diskUsageRoot(); got != filepath.Join(proc, "1", "root") {
		t.Fatalf("unavailable namespace must keep the enumerated process root: %s", got)
	}
}

func TestDiskNamespaceSupportsRelativeProcSelfAlias(t *testing.T) {
	proc, _ := diskFixture(t, true)
	writeDiskFixture(t, filepath.Join(proc, "1", "mountinfo"), "")
	if err := os.Rename(filepath.Join(proc, "self"), filepath.Join(proc, "123")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("123", filepath.Join(proc, "self")); err != nil {
		t.Fatal(err)
	}
	if got := diskUsageRoot(); got != "" {
		t.Fatalf("a proc self alias in the same namespace must retain direct paths: %s", got)
	}
}

func TestDiskNamespaceSupportsRealProcMagicLinks(t *testing.T) {
	if _, err := diskMountNamespace("/proc/self"); err != nil {
		t.Fatalf("the current process namespace must be readable without following its magic target: %v", err)
	}
}

func TestDiskNamespaceRejectsParentSymlinkEscape(t *testing.T) {
	process := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink("mnt:[1]", filepath.Join(outside, "mnt")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(process, "ns")); err != nil {
		t.Fatal(err)
	}
	if _, err := diskMountNamespace(process); err == nil {
		t.Fatal("a namespace directory outside the selected process root must not be read")
	}
}

func TestDiskMountTableFallbackMatchesGopsutilPathErrors(t *testing.T) {
	table := filepath.Join(t.TempDir(), "table")
	writeDiskFixture(t, table, "")
	for _, test := range []struct {
		name         string
		mountInfoErr error
		mountsErr    error
		want         bool
		wantFiles    []string
	}{
		{
			name:         "permission denied falls back to mounts",
			mountInfoErr: &os.PathError{Op: "open", Path: "mountinfo", Err: syscall.EACCES},
			want:         true,
			wantFiles:    []string{"mountinfo", "mounts"},
		},
		{
			name:         "both tables unreadable stay unavailable",
			mountInfoErr: &os.PathError{Op: "open", Path: "mountinfo", Err: syscall.EACCES},
			mountsErr:    &os.PathError{Op: "open", Path: "mounts", Err: syscall.EACCES},
			wantFiles:    []string{"mountinfo", "mounts"},
		},
		{
			name:         "non path error does not read mounts",
			mountInfoErr: errors.New("read failed"),
			wantFiles:    []string{"mountinfo"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var files []string
			openFile := func(path string) (*os.File, error) {
				name := filepath.Base(path)
				files = append(files, name)
				err := test.mountsErr
				if name == "mountinfo" {
					err = test.mountInfoErr
				}
				if err != nil {
					return nil, err
				}
				return os.Open(table)
			}
			if got := hasDiskMountTableWithOpen("/proc/1", openFile); got != test.want {
				t.Fatalf("mount table availability = %v, want %v", got, test.want)
			}
			if !reflect.DeepEqual(files, test.wantFiles) {
				t.Fatalf("read tables %v, want %v", files, test.wantFiles)
			}
		})
	}
}

func TestDiskCollectorUsesHostMountsAfterMountInfoPathError(t *testing.T) {
	proc, hostRoot := diskFixture(t, false)
	notDirectory := filepath.Join(proc, "not-a-directory")
	writeDiskFixture(t, notDirectory, "")
	if err := os.Symlink(filepath.Join(notDirectory, "mountinfo"), filepath.Join(proc, "1", "mountinfo")); err != nil {
		t.Fatal(err)
	}
	mountpoint := "/pasturestack-host-mounts-only"
	if err := os.Mkdir(filepath.Join(hostRoot, mountpoint), 0700); err != nil {
		t.Fatal(err)
	}
	writeDiskFixture(t, filepath.Join(proc, "1", "mounts"),
		fmt.Sprintf("/dev/disk-a %s ext4 rw 0 0\n", mountpoint))

	data, err := (DiskCollector{Unit: 1048576}).GetData()
	if err != nil || len(data["fileSystems"].(map[string]interface{})) != 1 {
		t.Fatalf("mounts fallback lost the host root view: %v %v", data, err)
	}
	if got := diskUsageRoot(); got != filepath.Join(proc, "1", "root") {
		t.Fatalf("mounts fallback selected a different namespace: %s", got)
	}
}

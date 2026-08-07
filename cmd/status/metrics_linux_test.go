package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestTrashPathsForPlatform(t *testing.T) {
	home := "/home/user"

	if runtime.GOOS == "darwin" {
		got := trashPathsForPlatform(home)
		want := []string{filepath.Join(home, ".Trash")}
		if len(got) != 1 || got[0] != want[0] {
			t.Fatalf("trashPathsForPlatform() = %v, want %v", got, want)
		}
		return
	}

	t.Setenv("XDG_DATA_HOME", "/var/data")
	got := trashPathsForPlatform(home)
	want := []string{filepath.Join("/var/data", "Trash", "files")}
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("trashPathsForPlatform() with XDG_DATA_HOME = %v, want %v", got, want)
	}

	t.Setenv("XDG_DATA_HOME", "")
	got = trashPathsForPlatform(home)
	want = []string{filepath.Join(home, ".local", "share", "Trash", "files")}
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("trashPathsForPlatform() default = %v, want %v", got, want)
	}
}

func TestReadFirstLine(t *testing.T) {
	if got := readFirstLine("/nonexistent/path"); got != "" {
		t.Fatalf("readFirstLine(missing) = %q, want empty", got)
	}
	if got := readFirstLine("/etc/hostname"); got == "" {
		// /etc/hostname can be missing in some containers; skip rather than fail.
		t.Log("hostname empty or missing")
	}
}

func TestReadCPUModel(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("requires /proc/cpuinfo")
	}
	if _, err := os.Stat("/proc/cpuinfo"); err != nil {
		t.Skip("/proc/cpuinfo not readable")
	}
	model := readCPUModel()
	if model == "" {
		t.Fatal("readCPUModel() returned empty on Linux with /proc/cpuinfo")
	}
}

func TestReadOSRelease(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("requires /etc/os-release")
	}
	if _, err := os.Stat("/etc/os-release"); err != nil {
		t.Skip("/etc/os-release missing")
	}
	distro := readOSRelease()
	if distro == "" {
		t.Fatal("readOSRelease() returned empty on Linux with /etc/os-release")
	}
	if strings.Contains(distro, "\"") {
		t.Fatalf("readOSRelease() left quotes intact: %q", distro)
	}
}

func TestCollectLinuxHardware(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only")
	}
	info := collectLinuxHardware(16<<30, "1 TB")
	if !strings.Contains(info.TotalRAM, "GB") {
		t.Fatalf("TotalRAM = %q, want a GB label", info.TotalRAM)
	}
	if info.DiskSize != "1 TB" {
		t.Fatalf("DiskSize = %q, want 1 TB", info.DiskSize)
	}
	// Model falls back to a non-empty placeholder; CPU/OS fill from procfs.
	if info.Model == "" {
		t.Fatal("collectLinuxHardware() left Model empty")
	}
	if info.CPUModel == "" {
		t.Fatal("collectLinuxHardware() left CPUModel empty")
	}
	if info.OSVersion == "" {
		t.Fatal("collectLinuxHardware() left OSVersion empty")
	}
}

func TestCollectLinuxThermalBounds(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only")
	}
	thermal := collectLinuxThermal()
	for name, v := range map[string]float64{
		"CPUTemp": thermal.CPUTemp,
		"GPUTemp": thermal.GPUTemp,
	} {
		if v < 0 || v > 125 {
			t.Fatalf("%s = %v out of sane range", name, v)
		}
	}
}

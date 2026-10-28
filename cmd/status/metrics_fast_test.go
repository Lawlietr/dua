package main

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/shirou/gopsutil/v4/disk"
	gopsutilnet "github.com/shirou/gopsutil/v4/net"
)

func TestCollectFastAvoidsExternalCommands(t *testing.T) {
	origRunCmd := runCmd
	origCommandExists := commandExists
	origPartitions := diskPartitionsFunc
	origUsage := diskUsageFunc
	origIOCounters := ioCountersFunc
	t.Cleanup(func() {
		runCmd = origRunCmd
		commandExists = origCommandExists
		diskPartitionsFunc = origPartitions
		diskUsageFunc = origUsage
		ioCountersFunc = origIOCounters
	})

	var externalCalls atomic.Int32
	runCmd = func(ctx context.Context, name string, args ...string) (string, error) {
		externalCalls.Add(1)
		return "", errors.New("unexpected command")
	}
	commandExists = func(name string) bool {
		externalCalls.Add(1)
		return false
	}
	diskPartitionsFunc = func(all bool) ([]disk.PartitionStat, error) {
		return []disk.PartitionStat{
			{Device: "/dev/disk3s1s1", Mountpoint: "/", Fstype: "apfs"},
		}, nil
	}
	diskUsageFunc = func(path string) (*disk.UsageStat, error) {
		return &disk.UsageStat{
			Path:        path,
			Fstype:      "apfs",
			Total:       2 * 1024 * 1024 * 1024,
			Used:        1024 * 1024 * 1024,
			UsedPercent: 50,
		}, nil
	}
	ioCountersFunc = func(bool) ([]gopsutilnet.IOCountersStat, error) {
		return []gopsutilnet.IOCountersStat{
			{Name: "en0", BytesRecv: 1024, BytesSent: 2048},
		}, nil
	}

	collector := NewCollector(ProcessWatchOptions{})
	if _, err := collector.CollectFast(); err != nil {
		t.Fatalf("CollectFast() error = %v", err)
	}
	if externalCalls.Load() != 0 {
		t.Fatalf("CollectFast() made %d external command calls", externalCalls.Load())
	}
}

func TestCollectProcessesKeepsLiveProcessesWithCachedEnrichment(t *testing.T) {
	origPartitions := diskPartitionsFunc
	origUsage := diskUsageFunc
	origIOCounters := ioCountersFunc
	origCollectProcesses := collectProcessesFunc
	t.Cleanup(func() {
		diskPartitionsFunc = origPartitions
		diskUsageFunc = origUsage
		ioCountersFunc = origIOCounters
		collectProcessesFunc = origCollectProcesses
	})

	diskPartitionsFunc = func(all bool) ([]disk.PartitionStat, error) {
		return []disk.PartitionStat{
			{Device: "/dev/disk3s1s1", Mountpoint: "/", Fstype: "apfs"},
		}, nil
	}
	diskUsageFunc = func(path string) (*disk.UsageStat, error) {
		return &disk.UsageStat{
			Path:        path,
			Fstype:      "apfs",
			Total:       2 * 1024 * 1024 * 1024,
			Used:        1024 * 1024 * 1024,
			UsedPercent: 50,
		}, nil
	}
	ioCountersFunc = func(bool) ([]gopsutilnet.IOCountersStat, error) {
		return []gopsutilnet.IOCountersStat{{Name: "en0", BytesRecv: 1024, BytesSent: 2048}}, nil
	}
	collectProcessesFunc = func() ([]ProcessInfo, error) {
		return []ProcessInfo{
			{PID: 200, PPID: 1, Name: "new-hot-process", Command: "/usr/bin/new-hot-process", CPU: 240, Memory: 1.5},
		}, nil
	}

	collector := NewCollector(ProcessWatchOptions{Enabled: true, CPUThreshold: 50})
	collector.cacheEnrichment(MetricsSnapshot{
		Hardware:  HardwareInfo{Model: "MacBook Pro"},
		TrashSize: 99,
		TopProcesses: []ProcessInfo{
			{PID: 100, Name: "old-process", CPU: 10},
		},
		ProcessAlerts: []ProcessAlert{
			{PID: 100, Name: "old-process", Status: "active"},
		},
	})

	snapshot, err := collector.CollectProcesses()
	if err != nil {
		t.Fatalf("CollectProcesses() error = %v", err)
	}

	if snapshot.Hardware.Model != "MacBook Pro" || snapshot.TrashSize != 99 {
		t.Fatalf("expected cached enrichment to be preserved, got hardware=%#v trash=%d", snapshot.Hardware, snapshot.TrashSize)
	}
	if len(snapshot.TopProcesses) != 1 || snapshot.TopProcesses[0].Name != "new-hot-process" {
		t.Fatalf("expected live top process data, got %#v", snapshot.TopProcesses)
	}
	if len(snapshot.ProcessAlerts) != 1 || snapshot.ProcessAlerts[0].Name != "new-hot-process" {
		t.Fatalf("expected live process alert data, got %#v", snapshot.ProcessAlerts)
	}
}

// A transient process-collection failure must not blank the process card: the
// previous full refresh's sample stays, while every other field of this refresh
// stays fresh. The full path is the one that renders --json and --watch frames.
func TestCollectFullKeepsLastProcessSampleWhenCollectionFails(t *testing.T) {
	original := collectProcessesFunc
	t.Cleanup(func() { collectProcessesFunc = original })

	collector := NewCollector(ProcessWatchOptions{})
	collector.cacheEnrichment(MetricsSnapshot{
		TopProcesses:  []ProcessInfo{{PID: 4242, Name: "cached-process", CPU: 12.5}},
		ProcessAlerts: []ProcessAlert{{PID: 4242, Name: "cached-process", CPU: 140, Threshold: 100, Window: "5m"}},
		Thermal:       ThermalStatus{AdapterPower: 987654},
		Sensors:       []SensorReading{{Label: "cached-sensor", Value: 42, Unit: "C"}},
	})

	collectProcessesFunc = func() ([]ProcessInfo, error) {
		return nil, errors.New("simulated transient ps failure")
	}

	snapshot, err := collector.Collect()
	if err == nil {
		t.Fatal("Collect() error = nil, want the process failure reported")
	}
	if len(snapshot.TopProcesses) != 1 || snapshot.TopProcesses[0].PID != 4242 {
		t.Fatalf("last process sample was dropped: %#v", snapshot.TopProcesses)
	}
	if len(snapshot.ProcessAlerts) != 1 || snapshot.ProcessAlerts[0].PID != 4242 {
		t.Fatalf("last process alerts were dropped: %#v", snapshot.ProcessAlerts)
	}

	// The narrow half of the fix: the cached values of tasks that did run must
	// not be copied over this refresh's own measurements.
	if snapshot.Thermal.AdapterPower == 987654 {
		t.Fatalf("cached thermal overwrote the fresh sample: %#v", snapshot.Thermal)
	}
	for _, sensor := range snapshot.Sensors {
		if sensor.Label == "cached-sensor" {
			t.Fatalf("cached sensors overwrote the fresh sample: %#v", snapshot.Sensors)
		}
	}

	// The cache itself must not be overwritten by the failed frame, or the
	// second frame would be empty and --watch would flicker.
	second, err := collector.Collect()
	if err == nil {
		t.Fatal("second Collect() error = nil, want the process failure reported")
	}
	if len(second.TopProcesses) != 1 || second.TopProcesses[0].PID != 4242 {
		t.Fatalf("process cache was overwritten by the first failed frame: %#v", second.TopProcesses)
	}
}

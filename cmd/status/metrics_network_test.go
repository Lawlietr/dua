package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	gopsutilnet "github.com/shirou/gopsutil/v4/net"
)

func TestCollectProxyFromEnvSupportsAllProxy(t *testing.T) {
	env := map[string]string{
		"ALL_PROXY": "socks5://127.0.0.1:7890",
	}
	getenv := func(key string) string {
		return env[key]
	}

	got := collectProxyFromEnv(getenv)
	if !got.Enabled {
		t.Fatalf("expected proxy enabled")
	}
	if got.Type != "SOCKS" {
		t.Fatalf("expected SOCKS type, got %s", got.Type)
	}
	if got.Host != "127.0.0.1:7890" {
		t.Fatalf("unexpected host: %s", got.Host)
	}
}

func TestCollectProxyFromScutilOutputPAC(t *testing.T) {
	out := `
<dictionary> {
  ProxyAutoConfigEnable : 1
  ProxyAutoConfigURLString : http://127.0.0.1:6152/proxy.pac
}`
	got := collectProxyFromScutilOutput(out)
	if !got.Enabled {
		t.Fatalf("expected proxy enabled")
	}
	if got.Type != "PAC" {
		t.Fatalf("expected PAC type, got %s", got.Type)
	}
	if got.Host != "127.0.0.1:6152" {
		t.Fatalf("unexpected host: %s", got.Host)
	}
}

func TestCollectProxyFromScutilOutputHTTPHostPort(t *testing.T) {
	out := `
<dictionary> {
  HTTPEnable : 1
  HTTPProxy : 127.0.0.1
  HTTPPort : 7890
}`
	got := collectProxyFromScutilOutput(out)
	if !got.Enabled {
		t.Fatalf("expected proxy enabled")
	}
	if got.Type != "HTTP" {
		t.Fatalf("expected HTTP type, got %s", got.Type)
	}
	if got.Host != "127.0.0.1:7890" {
		t.Fatalf("unexpected host: %s", got.Host)
	}
}

func TestCollectIOCountersSafelyRecoversPanic(t *testing.T) {
	original := ioCountersFunc
	ioCountersFunc = func(bool) ([]gopsutilnet.IOCountersStat, error) {
		panic("boom")
	}
	t.Cleanup(func() { ioCountersFunc = original })

	stats, err := collectIOCountersSafely()
	if err == nil {
		t.Fatalf("expected error from panic recovery")
	}
	if !strings.Contains(err.Error(), "panic collecting network counters") {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stats) != 0 {
		t.Fatalf("expected empty stats when panic recovered")
	}
}

func TestCollectIOCountersSafelyReturnsData(t *testing.T) {
	original := ioCountersFunc
	want := []gopsutilnet.IOCountersStat{
		{Name: "en0", BytesRecv: 1, BytesSent: 2},
	}
	ioCountersFunc = func(bool) ([]gopsutilnet.IOCountersStat, error) {
		return want, nil
	}
	t.Cleanup(func() { ioCountersFunc = original })

	got, err := collectIOCountersSafely()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Name != "en0" {
		t.Fatalf("unexpected stats: %+v", got)
	}
}

func TestCollectNetworkFirstSampleReturnsZeroRateInterfaces(t *testing.T) {
	original := ioCountersFunc
	ioCountersFunc = func(bool) ([]gopsutilnet.IOCountersStat, error) {
		return []gopsutilnet.IOCountersStat{
			{Name: "en0", BytesRecv: 1000, BytesSent: 2000},
		}, nil
	}
	t.Cleanup(func() { ioCountersFunc = original })

	c := &Collector{}
	got := c.collectNetwork(time.Now())
	if len(got) != 1 {
		t.Fatalf("expected first sample to render one interface, got %+v", got)
	}
	if got[0].RxRateMBs != 0 || got[0].TxRateMBs != 0 {
		t.Fatalf("expected first sample zero rates, got %+v", got[0])
	}
	if len(c.rxHistoryBuf.Slice()) != 1 || len(c.txHistoryBuf.Slice()) != 1 {
		t.Fatalf("expected history to be seeded on first sample")
	}
}

func TestCollectNetworkUsesPrimedCountersForInitialRates(t *testing.T) {
	original := ioCountersFunc
	calls := 0
	samples := [][]gopsutilnet.IOCountersStat{
		{{Name: "en0", BytesRecv: 1024 * 1024, BytesSent: 0}},
		{{Name: "en0", BytesRecv: 2 * 1024 * 1024, BytesSent: 512 * 1024}},
	}
	ioCountersFunc = func(bool) ([]gopsutilnet.IOCountersStat, error) {
		if calls >= len(samples) {
			return samples[len(samples)-1], nil
		}
		got := samples[calls]
		calls++
		return got, nil
	}
	t.Cleanup(func() { ioCountersFunc = original })

	c := NewCollector(ProcessWatchOptions{})
	got := c.collectNetwork(c.lastNetAt.Add(time.Second))
	if len(got) != 1 {
		t.Fatalf("expected one interface, got %+v", got)
	}
	if got[0].RxRateMBs != 1.0 {
		t.Fatalf("expected 1 MB/s down, got %v", got[0].RxRateMBs)
	}
	if got[0].TxRateMBs != 0.5 {
		t.Fatalf("expected 0.5 MB/s up, got %v", got[0].TxRateMBs)
	}
}

func TestCollectNetworkClampsCounterReset(t *testing.T) {
	original := ioCountersFunc
	ioCountersFunc = func(bool) ([]gopsutilnet.IOCountersStat, error) {
		return []gopsutilnet.IOCountersStat{
			{Name: "en0", BytesRecv: 10, BytesSent: 20},
		}, nil
	}
	t.Cleanup(func() { ioCountersFunc = original })

	base := time.Now()
	c := &Collector{
		prevNet: map[string]gopsutilnet.IOCountersStat{
			"en0": {Name: "en0", BytesRecv: 1024 * 1024, BytesSent: 1024 * 1024},
		},
		lastNetAt:    base,
		rxHistoryBuf: NewRingBuffer(NetworkHistorySize),
		txHistoryBuf: NewRingBuffer(NetworkHistorySize),
	}

	got := c.collectNetwork(base.Add(time.Second))
	if len(got) != 1 {
		t.Fatalf("expected one interface, got %+v", got)
	}
	if got[0].RxRateMBs != 0 || got[0].TxRateMBs != 0 {
		t.Fatalf("expected reset counters to clamp to zero, got %+v", got[0])
	}
}

func TestTunnelInterfaceIsNotReportedAsAProxy(t *testing.T) {
	// A machine with no configured proxy but an active utun (iCloud Private
	// Relay, a corporate VPN, or a TUN-mode client) must not be told it has a
	// proxy. The reading is still surfaced, just honestly labelled.
	original := ioCountersFunc
	ioCountersFunc = func(bool) ([]gopsutilnet.IOCountersStat, error) {
		return []gopsutilnet.IOCountersStat{
			{Name: "en0", BytesRecv: 100},
			{Name: "utun4", BytesRecv: 20, BytesSent: 30},
		}, nil
	}
	t.Cleanup(func() { ioCountersFunc = original })

	got := collectProxyFromTunInterfaces()
	if !got.Enabled || !got.IsTunnel || got.Type != "TUN" || got.Host != "utun4" {
		t.Fatalf("unexpected tunnel status: %+v", got)
	}
	card := renderNetworkCard(
		[]NetworkStatus{{Name: "en0", IP: "192.0.2.10"}},
		NetworkHistory{}, got, 40,
	)
	rendered := strings.Join(card.lines, "\n")
	if !strings.Contains(rendered, "Tunnel") || strings.Contains(rendered, "Proxy Tunnel") {
		t.Fatalf("tunnel must be rendered without a proxy claim: %q", rendered)
	}
}

func TestNetworkTotalsPrefersRoutedTunnel(t *testing.T) {
	tunnels := []NetworkStatus{
		{Name: "en0", RxRateMBs: 10, TxRateMBs: 12},
		{Name: "wg0", RxRateMBs: 99, TxRateMBs: 101, defaultTunnel: true},
	}
	rx, tx := networkTotals(tunnels)
	if rx != 99 || tx != 101 {
		t.Fatalf("routed-tunnel totals = (%v,%v), want (99,101)", rx, tx)
	}

	pure := []NetworkStatus{
		{Name: "en0", RxRateMBs: 10, TxRateMBs: 12},
		{Name: "eth1", RxRateMBs: 5, TxRateMBs: 7},
	}
	rx2, tx2 := networkTotals(pure)
	if rx2 != 15 || tx2 != 19 {
		t.Fatalf("plain totals = (%v,%v), want (15,19)", rx2, tx2)
	}
}

func TestIsTunnelInterfaceMatchesCommonPrefixes(t *testing.T) {
	tunnels := []string{"utun4", "tun0", "wg0", "wgcf1", "ipsec0", "ppp1", "gre0", "sit1"}
	for _, name := range tunnels {
		if !isTunnelInterface(name) {
			t.Errorf("isTunnelInterface(%q) = false, want true", name)
		}
	}
	physical := []string{"en0", "eth0", "wlan0", "lo", "docker0", "veth1"}
	for _, name := range physical {
		if isTunnelInterface(name) {
			t.Errorf("isTunnelInterface(%q) = true, want false", name)
		}
	}
}

func TestCollectNetworkKeepsDefaultRouteTunnelVisible(t *testing.T) {
	original := ioCountersFunc
	ioCountersFunc = func(bool) ([]gopsutilnet.IOCountersStat, error) {
		return []gopsutilnet.IOCountersStat{
			{Name: "en0", BytesRecv: 1000, BytesSent: 1200},
			{Name: "wg0", BytesRecv: 5000, BytesSent: 6000},
		}, nil
	}
	t.Cleanup(func() { ioCountersFunc = original })

	base := time.Now()
	c := &Collector{
		prevNet: map[string]gopsutilnet.IOCountersStat{
			"en0": {Name: "en0", BytesRecv: 0, BytesSent: 0},
			"wg0": {Name: "wg0", BytesRecv: 0, BytesSent: 0},
		},
		lastNetAt:           base,
		defaultNetInterface: "wg0",
		rxHistoryBuf:        NewRingBuffer(NetworkHistorySize),
		txHistoryBuf:        NewRingBuffer(NetworkHistorySize),
	}

	got := c.collectNetwork(base.Add(time.Second))
	if len(got) != 2 {
		t.Fatalf("expected tunnel + carrier, got %+v", got)
	}
	if got[0].Name != "wg0" || !got[0].defaultTunnel {
		t.Fatalf("routed tunnel must be first and marked defaultTunnel, got %+v", got[0])
	}
	rx, tx := networkTotals(got)
	if rx <= 0 || tx <= 0 {
		t.Fatalf("expected positive tunnel totals, got rx=%v tx=%v", rx, tx)
	}
}

func TestCollectNetworkFiltersNonDefaultTunnel(t *testing.T) {
	original := ioCountersFunc
	ioCountersFunc = func(bool) ([]gopsutilnet.IOCountersStat, error) {
		return []gopsutilnet.IOCountersStat{
			{Name: "en0", BytesRecv: 1000, BytesSent: 1200},
			{Name: "wg0", BytesRecv: 5000, BytesSent: 6000},
		}, nil
	}
	t.Cleanup(func() { ioCountersFunc = original })

	base := time.Now()
	c := &Collector{
		prevNet: map[string]gopsutilnet.IOCountersStat{
			"en0": {Name: "en0", BytesRecv: 0, BytesSent: 0},
			"wg0": {Name: "wg0", BytesRecv: 0, BytesSent: 0},
		},
		lastNetAt:           base,
		defaultNetInterface: "en0",
		rxHistoryBuf:        NewRingBuffer(NetworkHistorySize),
		txHistoryBuf:        NewRingBuffer(NetworkHistorySize),
	}

	got := c.collectNetwork(base.Add(time.Second))
	if len(got) != 1 || got[0].Name != "en0" {
		t.Fatalf("non-default tunnel must be filtered, got %+v", got)
	}
}

func TestTunnelHintDoesNotExpandProxyJSONContract(t *testing.T) {
	encoded, err := json.Marshal(ProxyStatus{
		Enabled:  true,
		Type:     "TUN",
		Host:     "utun4",
		IsTunnel: true,
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	const want = `{"enabled":true,"type":"TUN","host":"utun4"}`
	if string(encoded) != want {
		t.Fatalf("proxy JSON contract changed: got %s, want %s", encoded, want)
	}
}

// Package metrics collects VPS system metrics from /proc without any
// external dependencies. It auto-detects the primary network interface
// (skipping lo, docker*, veth*) and the primary block device (skipping
// loop* and sr*).
package metrics

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Collector reads /proc files and returns deltas between calls.
// It is safe for concurrent use.
type Collector struct {
	mu       sync.Mutex
	prevCPU  cpuRaw
	prevDisk diskRaw
	prevNet  netRaw
	prevTime time.Time
	netIface string
	diskDev  string
}

type cpuRaw struct {
	user, nice, system, idle, iowait, irq, softirq, steal, total uint64
}

type diskRaw struct {
	reads, writes uint64
}

type netRaw struct {
	rxBytes, txBytes, rxPackets, txPackets, rxDrops, txDrops uint64
}

// NewCollector creates and warms up a Collector, reading initial /proc values
// so the first call to Collect() returns meaningful deltas instead of zeros.
func NewCollector() *Collector {
	c := &Collector{
		netIface: detectNetIface(),
		diskDev:  detectDiskDev(),
		prevTime: time.Now(),
	}
	c.prevCPU, _ = readCPURaw()
	c.prevDisk, _ = readDiskRaw(c.diskDev)
	c.prevNet, _ = readNetRaw(c.netIface)
	return c
}

// Collect returns a fresh Snapshot. Deltas are computed against the previous call.
func (c *Collector) Collect() (*Snapshot, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(c.prevTime).Seconds()
	if elapsed < 0.1 {
		elapsed = 1
	}

	// CPU
	curCPU, err := readCPURaw()
	if err != nil {
		return nil, fmt.Errorf("metrics cpu: %w", err)
	}
	cpuStats := diffCPU(c.prevCPU, curCPU)
	c.prevCPU = curCPU

	// Memory
	memStats, swapStats, err := readMemInfo()
	if err != nil {
		return nil, fmt.Errorf("metrics mem: %w", err)
	}

	// Disk
	curDisk, _ := readDiskRaw(c.diskDev)
	diskStats := DiskStats{
		Device:       c.diskDev,
		ReadsPerSec:  safeRate(curDisk.reads, c.prevDisk.reads, elapsed),
		WritesPerSec: safeRate(curDisk.writes, c.prevDisk.writes, elapsed),
	}
	c.prevDisk = curDisk

	// Network
	curNet, _ := readNetRaw(c.netIface)
	netStats := NetStats{
		Interface:       c.netIface,
		RxBytesPerSec:   safeRate(curNet.rxBytes, c.prevNet.rxBytes, elapsed),
		TxBytesPerSec:   safeRate(curNet.txBytes, c.prevNet.txBytes, elapsed),
		RxPacketsPerSec: safeRate(curNet.rxPackets, c.prevNet.rxPackets, elapsed),
		TxPacketsPerSec: safeRate(curNet.txPackets, c.prevNet.txPackets, elapsed),
		RxDropsTotal:    curNet.rxDrops,
		TxDropsTotal:    curNet.txDrops,
	}
	c.prevNet = curNet
	c.prevTime = now

	// Load
	loadStats, _ := readLoadAvg()

	return &Snapshot{
		CollectedAt: now.Unix(),
		CPU:         cpuStats,
		Memory:      memStats,
		Swap:        swapStats,
		Disk:        diskStats,
		Net:         netStats,
		Load:        loadStats,
	}, nil
}

// --- /proc readers ---

func readCPURaw() (cpuRaw, error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return cpuRaw{}, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 8 {
			break
		}
		parseu := func(i int) uint64 {
			v, _ := strconv.ParseUint(fields[i], 10, 64)
			return v
		}
		r := cpuRaw{
			user:    parseu(1),
			nice:    parseu(2),
			system:  parseu(3),
			idle:    parseu(4),
			iowait:  parseu(5),
			irq:     parseu(6),
			softirq: parseu(7),
		}
		if len(fields) > 8 {
			r.steal = parseu(8)
		}
		r.total = r.user + r.nice + r.system + r.idle + r.iowait + r.irq + r.softirq + r.steal
		return r, nil
	}
	return cpuRaw{}, fmt.Errorf("cpu line not found in /proc/stat")
}

func diffCPU(prev, cur cpuRaw) CPUStats {
	totalDelta := float64(cur.total - prev.total)
	if totalDelta == 0 {
		return CPUStats{IdlePercent: 100}
	}
	pct := func(a, b uint64) float64 {
		if a <= b {
			return 0
		}
		return float64(a-b) / totalDelta * 100
	}
	return CPUStats{
		UsPercent:   pct(cur.user+cur.nice, prev.user+prev.nice),
		SyPercent:   pct(cur.system, prev.system),
		IdlePercent: pct(cur.idle, prev.idle),
		WaPercent:   pct(cur.iowait, prev.iowait),
		StPercent:   pct(cur.steal, prev.steal),
	}
}

func readMemInfo() (MemStats, SwapStats, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return MemStats{}, SwapStats{}, err
	}
	defer f.Close()

	kv := map[string]uint64{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		parts := strings.Fields(sc.Text())
		if len(parts) < 2 {
			continue
		}
		key := strings.TrimSuffix(parts[0], ":")
		val, _ := strconv.ParseUint(parts[1], 10, 64)
		kv[key] = val * 1024 // kB → bytes
	}

	memTotal := kv["MemTotal"]
	memAvail := kv["MemAvailable"]
	memUsed := memTotal - memAvail
	var memPct float64
	if memTotal > 0 {
		memPct = float64(memUsed) / float64(memTotal) * 100
	}

	swapTotal := kv["SwapTotal"]
	swapFree := kv["SwapFree"]
	swapUsed := swapTotal - swapFree
	var swapPct float64
	if swapTotal > 0 {
		swapPct = float64(swapUsed) / float64(swapTotal) * 100
	}

	return MemStats{
			TotalBytes:     memTotal,
			UsedBytes:      memUsed,
			AvailableBytes: memAvail,
			UsedPercent:    memPct,
		}, SwapStats{
			TotalBytes:  swapTotal,
			UsedBytes:   swapUsed,
			UsedPercent: swapPct,
		}, nil
}

func readDiskRaw(dev string) (diskRaw, error) {
	f, err := os.Open("/proc/diskstats")
	if err != nil {
		return diskRaw{}, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 14 || fields[2] != dev {
			continue
		}
		reads, _ := strconv.ParseUint(fields[3], 10, 64)
		writes, _ := strconv.ParseUint(fields[7], 10, 64)
		return diskRaw{reads: reads, writes: writes}, nil
	}
	return diskRaw{}, fmt.Errorf("device %q not found in /proc/diskstats", dev)
}

func readNetRaw(iface string) (netRaw, error) {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return netRaw{}, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, iface+":") {
			continue
		}
		line = strings.SplitN(line, ":", 2)[1]
		fields := strings.Fields(line)
		if len(fields) < 12 {
			continue
		}
		parseu := func(i int) uint64 {
			v, _ := strconv.ParseUint(fields[i], 10, 64)
			return v
		}
		return netRaw{
			rxBytes:   parseu(0),
			rxPackets: parseu(1),
			rxDrops:   parseu(3),
			txBytes:   parseu(8),
			txPackets: parseu(9),
			txDrops:   parseu(11),
		}, nil
	}
	return netRaw{}, fmt.Errorf("interface %q not found in /proc/net/dev", iface)
}

func readLoadAvg() (LoadStats, error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return LoadStats{}, err
	}
	fields := strings.Fields(string(data))
	if len(fields) < 4 {
		return LoadStats{}, fmt.Errorf("unexpected /proc/loadavg format")
	}
	l1, _ := strconv.ParseFloat(fields[0], 64)
	l5, _ := strconv.ParseFloat(fields[1], 64)
	l15, _ := strconv.ParseFloat(fields[2], 64)
	parts := strings.SplitN(fields[3], "/", 2)
	procs, _ := strconv.Atoi(parts[0])
	return LoadStats{Load1: l1, Load5: l5, Load15: l15, Procs: procs}, nil
}

// --- auto-detection ---

// detectNetIface returns the first non-loopback, non-container interface
// found in /proc/net/dev (skips lo, docker*, veth*).
func detectNetIface() string {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return "eth0"
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.Contains(line, ":") {
			continue
		}
		name := strings.TrimSpace(strings.SplitN(line, ":", 2)[0])
		if name == "lo" ||
			strings.HasPrefix(name, "docker") ||
			strings.HasPrefix(name, "veth") ||
			strings.HasPrefix(name, "br-") {
			continue
		}
		return name
	}
	return "eth0"
}

// detectDiskDev returns the first real whole-disk block device from /proc/diskstats
// (skips loop*, sr*, and partitions that end in a digit).
func detectDiskDev() string {
	f, err := os.Open("/proc/diskstats")
	if err != nil {
		return "vda"
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 3 {
			continue
		}
		dev := fields[2]
		if strings.HasPrefix(dev, "loop") || strings.HasPrefix(dev, "sr") {
			continue
		}
		last := dev[len(dev)-1]
		if last >= '0' && last <= '9' {
			continue // skip partitions (vda1, sda1, …)
		}
		return dev
	}
	return "vda"
}

// safeRate computes (cur-prev)/elapsed, returning 0 on counter reset.
func safeRate(cur, prev uint64, elapsed float64) uint64 {
	if cur < prev {
		return 0
	}
	return uint64(float64(cur-prev) / elapsed)
}

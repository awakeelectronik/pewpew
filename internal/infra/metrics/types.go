package metrics

// Snapshot holds a point-in-time snapshot of VPS system metrics.
type Snapshot struct {
	CollectedAt int64     `json:"collected_at"` // Unix seconds
	CPU         CPUStats  `json:"cpu"`
	Memory      MemStats  `json:"memory"`
	Swap        SwapStats `json:"swap"`
	Disk        DiskStats `json:"disk"`
	Net         NetStats  `json:"net"`
	Load        LoadStats `json:"load"`
}

// CPUStats represents CPU usage percentages derived from /proc/stat deltas.
type CPUStats struct {
	UsPercent   float64 `json:"us_percent"`
	SyPercent   float64 `json:"sy_percent"`
	IdlePercent float64 `json:"idle_percent"`
	WaPercent   float64 `json:"wa_percent"`
	StPercent   float64 `json:"st_percent"` // steal — important on VPS hypervisors
}

// MemStats represents memory usage derived from /proc/meminfo.
type MemStats struct {
	TotalBytes     uint64  `json:"total_bytes"`
	UsedBytes      uint64  `json:"used_bytes"`
	AvailableBytes uint64  `json:"available_bytes"`
	UsedPercent    float64 `json:"used_percent"`
}

// SwapStats represents swap usage derived from /proc/meminfo.
type SwapStats struct {
	TotalBytes  uint64  `json:"total_bytes"`
	UsedBytes   uint64  `json:"used_bytes"`
	UsedPercent float64 `json:"used_percent"`
}

// DiskStats represents disk I/O rates (ops/s) derived from /proc/diskstats deltas.
type DiskStats struct {
	Device       string `json:"device"`
	ReadsPerSec  uint64 `json:"reads_per_sec"`
	WritesPerSec uint64 `json:"writes_per_sec"`
}

// NetStats represents network interface counters derived from /proc/net/dev.
// RxBytesPerSec / TxBytesPerSec are deltas; RxDrops / TxDrops are lifetime totals.
type NetStats struct {
	Interface     string `json:"interface"`
	RxBytesPerSec uint64 `json:"rx_bytes_per_sec"`
	TxBytesPerSec uint64 `json:"tx_bytes_per_sec"`
	RxPacketsPerSec uint64 `json:"rx_packets_per_sec"`
	TxPacketsPerSec uint64 `json:"tx_packets_per_sec"`
	RxDropsTotal  uint64 `json:"rx_drops_total"`
	TxDropsTotal  uint64 `json:"tx_drops_total"`
}

// LoadStats represents system load averages derived from /proc/loadavg.
type LoadStats struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
	Procs  int     `json:"procs"`
}

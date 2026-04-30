package worker

import (
	"log"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
)

type Stats struct {
	MemStats  *mem.VirtualMemoryStat
	DiskStats *disk.UsageStat
	CpuStats  *cpu.TimesStat
	LoadStats *load.AvgStat
	TaskCount int
}

// gopsutil reports memory in bytes; book API was in KB so we convert here.
func (s *Stats) MemTotalKb() uint64 {
	return s.MemStats.Total / 1024
}

func (s *Stats) MemAvailableKb() uint64 {
	return s.MemStats.Available / 1024
}

func (s *Stats) MemUsedKb() uint64 {
	return s.MemStats.Used / 1024
}

func (s *Stats) MemUsedPercent() float64 {
	return s.MemStats.UsedPercent
}

func (s *Stats) DiskTotal() uint64 {
	return s.DiskStats.Total
}

func (s *Stats) DiskFree() uint64 {
	return s.DiskStats.Free
}

func (s *Stats) DiskUsed() uint64 {
	return s.DiskStats.Used
}

func (s *Stats) CpuUsage() float64 {
	idle := s.CpuStats.Idle + s.CpuStats.Iowait
	nonIdle := s.CpuStats.User + s.CpuStats.Nice + s.CpuStats.System + s.CpuStats.Irq + s.CpuStats.Guest + s.CpuStats.Softirq + s.CpuStats.Steal
	total := idle + nonIdle
	if total == 0 {
		return 0.00
	}
	return (total - idle) / total
}

func GetStats() *Stats {
	return &Stats{
		MemStats:  GetMemoryInfo(),
		DiskStats: GetDiskInfo(),
		CpuStats:  GetCpuInfo(),
		LoadStats: GetLoadAvg(),
	}
}

// GetMemoryInfo See https://pkg.go.dev/github.com/shirou/gopsutil/v3/mem#VirtualMemoryStat
func GetMemoryInfo() *mem.VirtualMemoryStat {
	v, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("[Worker] Error reading memory info: %v", err)
		return &mem.VirtualMemoryStat{}
	}
	return v
}

// GetDiskInfo See https://pkg.go.dev/github.com/shirou/gopsutil/v3/disk#UsageStat
func GetDiskInfo() *disk.UsageStat {
	d, err := disk.Usage("/")
	if err != nil {
		log.Printf("[Worker] Error reading disk info: %v", err)
		return &disk.UsageStat{}
	}
	return d
}

// GetCpuInfo See https://pkg.go.dev/github.com/shirou/gopsutil/v3/cpu#TimesStat
func GetCpuInfo() *cpu.TimesStat {
	times, err := cpu.Times(false)
	if err != nil || len(times) == 0 {
		log.Printf("[Worker] Error reading CPU info: %v", err)
		return &cpu.TimesStat{}
	}
	return &times[0]
}

// GetLoadAvg See https://pkg.go.dev/github.com/shirou/gopsutil/v3/load#AvgStat
func GetLoadAvg() *load.AvgStat {
	l, err := load.Avg()
	if err != nil {
		log.Printf("[Worker] Error reading load avg: %v", err)
		return &load.AvgStat{}
	}
	return l
}

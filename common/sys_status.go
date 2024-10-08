package common

import (
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
)

func CpuLoad1Usage() float64 {
	var (
		err         error
		cpuCount    int
		avgLoadStat *load.AvgStat
	)
	cpuCount, err = cpu.Counts(true)
	if err != nil {
		return -1
	}
	avgLoadStat, err = load.Avg()
	if err != nil {
		return -1
	}
	return convertFloatDecimal(avgLoadStat.Load1 / float64(cpuCount))
}

func DiskUsagePercent() float64 {
	dir, _ := os.Getwd()
	usage, err := disk.Usage(dir)
	if err != nil {
		return -1
	}
	return convertFloatDecimal(usage.UsedPercent)
}

func MemoryUsagePercent() float64 {
	vmStatus, err := mem.VirtualMemory()
	if err != nil {
		return -1
	}
	if cMem, err := CGroupMem(); err == nil && cMem.LimitUsage < vmStatus.Total {
		// 由此可判断，程序运行在容器内，且有内存限制
		return convertFloatDecimal(cMem.Percent())
	}
	return convertFloatDecimal(vmStatus.UsedPercent)
}

func convertFloatDecimal(value float64) float64 {
	result, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", value), 64)
	return result
}

func CurrentLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

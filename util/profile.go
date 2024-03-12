package util

import (
	"fmt"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

const (
	Cpu       = "cpu"
	Memory    = "mem"
	Disk      = "disk"
	Goroutine = "goroutine"
)

func GetCpuPercent() (float64, *Err) {
	percent, e := cpu.Percent(time.Second, false)
	if e != nil {
		return 0, WrapErr(EcServiceErr, e)
	}
	return percent[0], nil
}

func GetMemPercent() float64 {
	memInfo, _ := mem.VirtualMemory()
	return memInfo.UsedPercent
}

func GetDiskPercent() float64 {
	parts, _ := disk.Partitions(true)
	diskInfo, _ := disk.Usage(parts[0].Mountpoint)
	return diskInfo.UsedPercent
}

func StartProfile(dur time.Duration, receiver chan<- M) {
	go func() {
		sampling(receiver)
		ticker := time.NewTicker(dur)
		for {
			select {
			case <-_Ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				sampling(receiver)
			}
		}
	}()
}

func sampling(receiver chan<- M) {
	status := M{
		Memory:    float32(GetMemPercent()),
		Disk:      float32(GetDiskPercent()),
		Goroutine: uint32(runtime.NumGoroutine()),
	}
	if runtime.GOOS != "darwin" { //暂时不支持
		cp, err := GetCpuPercent()
		if err == nil {
			status[Cpu] = float32(cp)
		} else {
			fmt.Println("get cpu fail:")
			fmt.Println(err.Error())
		}
	}
	receiver <- status
}

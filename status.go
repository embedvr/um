package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"golang.org/x/sys/unix"

	"github.com/Ultramarine-Linux/um/util"
	"github.com/acobaugh/osrelease"
	"github.com/charmbracelet/lipgloss"
	"github.com/jaypipes/ghw"

	mem "github.com/mackerelio/go-osstat/memory"
	"github.com/mackerelio/go-osstat/uptime"
	"github.com/urfave/cli/v2"

	"github.com/Wifx/gonetworkmanager/v2"
)

var listHeader = lipgloss.NewStyle().
	Foreground(purple).
	MarginRight(2).
	MarginTop(1).
	Bold(true).
	Render

var listItem = lipgloss.NewStyle().PaddingLeft(2).Render

func networkInfo() ([]string, error) {
	nm, err := gonetworkmanager.NewNetworkManager()
	if err != nil {
		return nil, err
	}

	devices, err := nm.GetPropertyAllDevices()
	if err != nil {
		return nil, err
	}

	devicesInfo := []string{
		listHeader("Network"),
	}

	for _, device := range devices {
		deviceInterface, err := device.GetPropertyInterface()
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
		connection, err := device.GetPropertyActiveConnection()
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
		if connection == nil {
			continue
		}

		status, err := connection.GetPropertyState()
		if err != nil {
			fmt.Println(err.Error())
			continue
		}

		proptype, err := connection.GetPropertyType()
		if err != nil {
			fmt.Println(err.Error())
			continue
		}

		statusString := ""
		switch status {
		case gonetworkmanager.NmActiveConnectionStateActivated:
			statusString = "Connected"
		default:
			statusString = "Unknown"
		}
		devicesInfo = append(devicesInfo, listItem(fmt.Sprintf("%s (%s): %s", deviceInterface, proptype, statusString)))
	}

	return devicesInfo, nil
}

func statusInfo() ([]string, error) {
	dur, err := uptime.Get()
	if err != nil {
		return nil, err

	}

	u := unix.Utsname{}
	err = unix.Uname(&u)
	if err != nil {
		return nil, err
	}

	rpmCount := util.GetInstalledRpmCount()
	systemFlatpakCount := util.GetInstalledSystemFlatpakCount()
	userFlatpakCount := util.GetInstalledUserFlatpakCount()

	return []string{
		listHeader("Status"),
		listItem("Uptime: " + dur.String()),
		listItem("Kernel: " + string(u.Release[:])),
		listItem(fmt.Sprintf("Packages: %d rpms, %d system flatpaks, %d user flatpaks", rpmCount, systemFlatpakCount, userFlatpakCount)),
	}, nil
}

func gatherOsInfo() (result []string, err error) {
	release, err := osrelease.Read()
	if err != nil {
		return nil, err
	}

	var atomicValue string

	if strings.HasPrefix(release["VARIANT"], "Atomic") {
		atomicValue = "True"
	} else {
		atomicValue = "False"
	}

	return []string{
		listHeader("System"),
		listItem("Name: " + release["NAME"]),
		listItem("Version: " + release["VERSION"]),
		listItem("Variant: " + release["VARIANT"]),
		listItem("Atomic: " + atomicValue),
	}, nil
}

func gatherHwInfo() (result []string, err error) {
	cpu, err := ghw.CPU()
	if err != nil {
		return nil, err
	}

	gpu, err := ghw.GPU()
	if err != nil {
		return nil, err
	}

	result = []string{
		listHeader("Hardware"),
	}

	baseboard, err := ghw.Baseboard(ghw.WithDisableWarnings())
	if err != nil {
		fmt.Printf("Error getting baseboard info: %v", err)
	}
	result = append(result, listItem(fmt.Sprintf("Vendor: %s", baseboard.Vendor)))
	result = append(result, listItem(fmt.Sprintf("Product: %s", baseboard.Product)))

	memory, err := ghw.Memory()
	if err != nil {
		return nil, err
	}

	memoryStats, err := mem.Get()
	if err != nil {
		return nil, err
	}

	result = append(result, listItem(fmt.Sprintf("Memory: %s (physical), %s (usuable)",
		util.FormatBytes(int64(memory.TotalPhysicalBytes)),
		util.FormatBytes(int64(memory.TotalUsableBytes)))))

	result = append(result, listItem(fmt.Sprintf("Swap: %s", util.FormatBytes(int64(memoryStats.SwapTotal)))))

	for i, processor := range cpu.Processors {
		title := "CPU"
		if len(cpu.Processors) > 1 {
			title = title + string(i)
		}

		result = append(result, listItem(fmt.Sprintf("%s: %s (%s)", title, processor.Model, runtime.GOARCH)))
	}

	for i, card := range gpu.GraphicsCards {
		title := "GPU"
		if len(cpu.Processors) > 1 {
			title = title + string(i)
		}

		result = append(result, listItem(fmt.Sprintf("%s: %s", title, card.DeviceInfo.Product.Name)))
		result = append(result, listItem(fmt.Sprintf("%s Driver: %s", title, card.DeviceInfo.Driver))) //?
	}

	var stat unix.Statfs_t
	wd, err := os.Getwd()
	unix.Statfs(wd, &stat)
	diskFree := int64(stat.Bavail) * int64(stat.Bsize)
	result = append(result, listItem(fmt.Sprintf("Disk Free: %s", util.FormatBytes(diskFree))))

	block, err := ghw.Block()
	if err != nil {
		return nil, err
	}

	for _, disk := range block.Disks {
		for _, part := range disk.Partitions {
			if part.MountPoint == "/" {
				result = append(result, listItem(fmt.Sprintf("Disk Type: %s", disk.StorageController.String())))
				result = append(result, listItem(fmt.Sprintf("Filesystem: %s", part.Type)))
			}
		}
	}

	return
}

func gatherDiskInfo() (result []string, err error) {
	result = []string{
		listHeader("Disk"),
	}

	block, err := ghw.Block()
	if err != nil {
		return nil, err
	}

	for i, disk := range block.Disks {
		if disk.BusPath == "unknown" {
			continue
		}

		title := "Disk"
		if len(block.Disks) > 1 {
			title = title + string(i)
		}

		result = append(result, listItem(fmt.Sprintf("%s: %s (%s)", title, disk.Model, disk.Name)))
		result = append(result, listItem(fmt.Sprintf("%s Type: %s", title, disk.DriveType.String())))
		result = append(result, listItem(fmt.Sprintf("%s Controler: %s", title, disk.StorageController.String())))
	}

	return
}

func gatherDesktop() (result []string, err error) {
	var protocol string

	if s := os.Getenv("WAYLAND_DISPLAY"); s != "" {
		protocol = "Wayland"
	} else if s := os.Getenv("DISPLAY"); s != "" {
		protocol = "X11"
	} else {
		protocol = "Unknown"
	}

	result = []string{
		listHeader("Desktop"),
		listItem("Name: " + os.Getenv("XDG_CURRENT_DESKTOP")),
		listItem("Protocol: " + protocol),
	}
	return
}

func status(c *cli.Context) error {
	osinfo, err := gatherOsInfo()
	if err != nil {
		return err
	}
	fmt.Println(lipgloss.JoinVertical(lipgloss.Left, osinfo...))

	hwinfo, err := gatherHwInfo()
	if err != nil {
		return err
	}
	fmt.Println(lipgloss.JoinVertical(lipgloss.Left, hwinfo...))

	diskinfo, err := gatherDiskInfo()
	if err != nil {
		return err
	}
	fmt.Println(lipgloss.JoinVertical(lipgloss.Left, diskinfo...))

	desktopinfo, err := gatherDesktop()
	if err != nil {
		return err
	}
	fmt.Println(lipgloss.JoinVertical(lipgloss.Left, desktopinfo...))

	statusinfo, err := statusInfo()
	if err != nil {
		return err
	}
	fmt.Println(lipgloss.JoinVertical(lipgloss.Left, statusinfo...))

	networkinfo, err := networkInfo()
	if err != nil {
		return err
	}
	fmt.Println(lipgloss.JoinVertical(lipgloss.Left, networkinfo...))

	return nil
}

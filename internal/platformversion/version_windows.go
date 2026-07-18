//go:build windows

package platformversion

import "golang.org/x/sys/windows"

const workstationProductType = 1

func Current() Version {
	value := windows.RtlGetVersion()
	return Version{
		Major:    value.MajorVersion,
		Minor:    value.MinorVersion,
		Build:    value.BuildNumber,
		IsServer: value.ProductType != workstationProductType,
	}
}

//go:build windows

package privacy

import (
	"golang.org/x/sys/windows/registry"
)

func DisableFeature(feature string, disable bool) error {
	switch feature {
	case "copilot":
		return disableCopilot(disable)
	case "onedrive":
		return disableOneDrive(disable)
	case "telemetry":
		return disableTelemetry(disable)
	case "ads":
		return disableAds(disable)
	case "cortana":
		return disableCortana(disable)
	case "widgets":
		return disableWidgets(disable)
	}
	return nil
}

func setDWORD(path, name string, val uint32) error {
	k, _, err := registry.CreateKey(registry.LOCAL_MACHINE, path, registry.SET_VALUE)
	if err != nil {
		k2, _, err2 := registry.CreateKey(registry.CURRENT_USER, path, registry.SET_VALUE)
		if err2 != nil {
			return err
		}
		defer k2.Close()
		return k2.SetDWordValue(name, val)
	}
	defer k.Close()
	return k.SetDWordValue(name, val)
}

func disableCopilot(disable bool) error {
	v := uint32(0)
	if disable {
		v = 1
	}
	return setDWORD(`SOFTWARE\Policies\Microsoft\Windows\WindowsCopilot`, "TurnOffWindowsCopilot", v)
}

func disableOneDrive(disable bool) error {
	v := uint32(0)
	if disable {
		v = 1
	}
	return setDWORD(`SOFTWARE\Policies\Microsoft\Windows\OneDrive`, "DisableFileSyncNGSC", v)
}

func disableTelemetry(disable bool) error {
	v := uint32(1)
	if disable {
		v = 0
	}
	return setDWORD(`SOFTWARE\Policies\Microsoft\Windows\DataCollection`, "AllowTelemetry", v)
}

func disableAds(disable bool) error {
	v := uint32(1)
	if disable {
		v = 0
	}
	k, _, err := registry.CreateKey(registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\AdvertisingInfo`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetDWordValue("Enabled", v)
}

func disableCortana(disable bool) error {
	v := uint32(0)
	if disable {
		v = 1
	}
	return setDWORD(`SOFTWARE\Policies\Microsoft\Windows\Windows Search`, "AllowCortana", v)
}

func disableWidgets(disable bool) error {
	v := uint32(1)
	if disable {
		v = 0
	}
	return setDWORD(`SOFTWARE\Policies\Microsoft\Dsh`, "AllowNewsAndInterests", v)
}

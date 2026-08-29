//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

func ramGBOS() float64 {
	var m memoryStatusEx
	m.Length = uint32(unsafe.Sizeof(m))
	k := syscall.NewLazyDLL("kernel32.dll")
	proc := k.NewProc("GlobalMemoryStatusEx")
	r, _, _ := proc.Call(uintptr(unsafe.Pointer(&m)))
	if r == 0 || m.TotalPhys == 0 {
		return 16
	}
	return float64(m.TotalPhys) / (1024 * 1024 * 1024)
}

func availGB() float64 {
	var m memoryStatusEx
	m.Length = uint32(unsafe.Sizeof(m))
	k := syscall.NewLazyDLL("kernel32.dll")
	proc := k.NewProc("GlobalMemoryStatusEx")
	r, _, _ := proc.Call(uintptr(unsafe.Pointer(&m)))
	if r == 0 {
		return 4
	}
	return float64(m.AvailPhys) / (1024 * 1024 * 1024)
}

func setUTF8Console() {
	k := syscall.NewLazyDLL("kernel32.dll")
	k.NewProc("SetConsoleOutputCP").Call(65001)
	k.NewProc("SetConsoleCP").Call(65001)
}

func osUILang() string {
	k := syscall.NewLazyDLL("kernel32.dll")
	r, _, _ := k.NewProc("GetUserDefaultUILanguage").Call()
	if uint32(r)&0x3FF == 0x12 {
		return "ko"
	}
	return "en"
}

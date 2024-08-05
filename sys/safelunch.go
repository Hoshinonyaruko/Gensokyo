//go:build windows
// +build windows

package sys

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// RunningByDoubleClick 检查是否通过双击直接运行
func RunningByDoubleClick() bool {
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	lp := kernel32.NewProc("GetConsoleProcessList")
	if lp != nil {
		var ids [2]uint32
		var maxCount uint32 = 2
		ret, _, _ := lp.Call(uintptr(unsafe.Pointer(&ids)), uintptr(maxCount))
		if ret > 1 {
			return false
		}
	}
	return true
}

// windows
func setConsoleTitleWindows(title string) error {
	kernel32, err := syscall.LoadDLL("kernel32.dll")
	if err != nil {
		return err
	}
	proc, err := kernel32.FindProc("SetConsoleTitleW")
	if err != nil {
		return err
	}
	p0, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return err
	}
	r1, _, err := proc.Call(uintptr(unsafe.Pointer(p0)))
	if r1 == 0 {
		return err
	}
	return nil
}

// NoMoreDoubleClick 提示用户不要双击运行，并生成安全启动脚本
func NoMoreDoubleClick() error {
	toHighDPI()
	r := boxW(getConsoleWindows(), "请勿通过双击直接运行本程序, 这将导致一些非预料的后果.\n请在shell中运行./gensokyo.exe\n点击确认将释出安全启动脚本，点击取消则关闭程序", "警告", 0x00000030|0x00000001)
	if r == 2 {
		return nil
	}
	r = boxW(0, "点击确认将覆盖gensokyo.bat，点击取消则关闭程序", "警告", 0x00000030|0x00000001)
	if r == 2 {
		return nil
	}
	f, err := os.OpenFile("gensokyo.bat", os.O_CREATE|os.O_RDWR, 0o666)
	if err != nil {
		return err
	}
	if err != nil {
		fmt.Printf("打开gensokyo.bat失败: %v", err)
		return nil
	}
	_ = f.Truncate(0)

	ex, _ := os.Executable()
	exPath := filepath.Base(ex)
	_, err = f.WriteString("%Created by gensokyo. DO NOT EDIT ME!%\nstart cmd /K \"" + exPath + "\"")
	if err != nil {
		fmt.Printf("写入gensokyo.bat失败: %v", err)
		return nil
	}
	f.Close()
	boxW(0, "安全启动脚本已生成，请双击gensokyo.bat启动", "提示", 0x00000040|0x00000000)
	return nil
}

// BoxW of Win32 API. Check https://docs.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-messageboxw for more detail.
func boxW(hwnd uintptr, caption, title string, flags uint) int {
	captionPtr, _ := windows.UTF16PtrFromString(caption)
	titlePtr, _ := windows.UTF16PtrFromString(title)
	u32 := windows.NewLazySystemDLL("user32.dll")
	ret, _, _ := u32.NewProc("MessageBoxW").Call(
		hwnd,
		uintptr(unsafe.Pointer(captionPtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		uintptr(flags))

	return int(ret)
}

// GetConsoleWindows retrieves the window handle used by the console associated with the calling process.
func getConsoleWindows() (hWnd uintptr) {
	hWnd, _, _ = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetConsoleWindow").Call()
	return
}

// toHighDPI tries to raise DPI awareness context to DPI_AWARENESS_CONTEXT_UNAWARE_GDISCALED
func toHighDPI() {
	systemAware := ^uintptr(2) + 1
	unawareGDIScaled := ^uintptr(5) + 1
	u32 := windows.NewLazySystemDLL("user32.dll")
	proc := u32.NewProc("SetThreadDpiAwarenessContext")
	if proc.Find() != nil {
		return
	}
	for i := unawareGDIScaled; i <= systemAware; i++ {
		_, _, _ = u32.NewProc("SetThreadDpiAwarenessContext").Call(i)
	}
}

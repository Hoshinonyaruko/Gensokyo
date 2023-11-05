//go:build !windows
// +build !windows

package sys

// RunningByDoubleClick 检查是否通过双击直接运行
func RunningByDoubleClick() bool {
	// 非Windows平台下此功能不可用
	return false
}

// NoMoreDoubleClick 提示用户不要双击运行，并生成安全启动脚本
func NoMoreDoubleClick() error {
	// 非Windows平台下此功能不可用
	return nil
}

// boxW 是一个空操作函数，因为它只在Windows上有意义
func boxW(hwnd uintptr, caption, title string, flags uint) int {
	// 非Windows平台下此功能不可用
	return 0
}

// GetConsoleWindows retrieves the window handle used by the console associated with the calling process.
func getConsoleWindows() (hWnd uintptr) {
	// 非Windows平台下此功能不可用
	return 0
}

// toHighDPI tries to raise DPI awareness context to DPI_AWARENESS_CONTEXT_UNAWARE_GDISCALED
func toHighDPI() {
	// 非Windows平台下此功能不可用
}

// windows
func setConsoleTitleWindows(title string) error {
	// 非Windows平台下此功能不可用
	return nil
}

package crossplatform

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	WindowsOS = "windows"
	LinuxOS   = "linux"
)

// PathSeparator returns the OS-specific path separator
func PathSeparator() string {
	return string(os.PathSeparator)
}

// IsWindows checks if the current OS is Windows
func IsWindows() bool {
	return runtime.GOOS == WindowsOS
}

// IsLinux checks if the current OS is Linux
func IsLinux() bool {
	return runtime.GOOS == LinuxOS
}

// NormalizePath normalizes a path to use the correct OS-specific separators
func NormalizePath(path string) string {
	if path == "" {
		return ""
	}
	
	if IsWindows() {
		return strings.ReplaceAll(path, "/", "\\")
	}
	return strings.ReplaceAll(path, "\\", "/")
}

// JoinPaths joins multiple path components with the correct OS-specific separator
func JoinPaths(components ...string) string {
	return filepath.Join(components...)
}

// GetShell returns the appropriate shell for the current OS
func GetShell() string {
	if IsWindows() {
		return "cmd"
	}
	return "sh"
}

// GetShellArgs returns the appropriate shell arguments for the current OS
func GetShellArgs() []string {
	if IsWindows() {
		return []string{"/c"}
	}
	return []string{"-c"}
}

// GetDefaultPath returns the default PATH environment variable value for the current OS
func GetDefaultPath() string {
	if IsWindows() {
		return "C:\\Windows\\system32;C:\\Windows;C:\\Windows\\System32\\Wbem;C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\;C:\\Program Files\\nodejs\\;C:\\Program Files\\Go\\bin;C:\\Users\\%USERNAME%\\AppData\\Local\\Programs\\Python\\Python310;C:\\Users\\%USERNAME%\\AppData\\Local\\Programs\\Python\\Python310\\Scripts"
	}
	return "/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin:/usr/local/go/bin"
}

// GetTempDir returns the system temporary directory
func GetTempDir() string {
	return os.TempDir()
}

// Exists checks if a file or directory exists
func Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// IsDirectory checks if a path is a directory
func IsDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

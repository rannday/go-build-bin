package build

func isWindows(goos string) bool {
	return goos == "windows"
}
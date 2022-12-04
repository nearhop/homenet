//go:build windows
// +build windows

package nebula

func GetConfigFileDir() string {
	return "./nearhop/configs/"
}

func GetLogsFileDir() string {
	return "./nearhop/logs/"
}

func GetNearhopDir() string {
	return "./nearhop/"
}

func IsRootUser() bool {
	return true
}

func GetModel() string {
	return "windows"
}

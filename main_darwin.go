//go:build darwin
// +build darwin

package nebula

func GetConfigFileDir() string {
	return "/etc/nearhop/configs/"
}

func GetLogsFileDir() string {
	return "/etc/nearhop/logs/"
}

func GetNearhopDir() string {
	return "/etc/nearhop/"
}

func IsRootUser() bool {
	return true
}

func GetModel() string {
	return "darwin"
}

//go:build asus
// +build asus

package nebula

func GetConfigFileDir() string {
	return "/jffs/nearhop/configs/"
}

func GetLogsFileDir() string {
	return "/etc/nearhop/logs/"
}

func GetNearhopDir() string {
	return "/jffs/nearhop/"
}

func IsRootUser() bool {
	return true
}

func GetModel() string {
	// ToDo: Revisit this when we add more asus models
	return "asus"
}

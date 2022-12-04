//go:build windows
// +build windows

package platform

func Get_osname() string {
	return "windows"
}

func GetDefaultInterfaceName() (str string, err error) {
	return "", nil
}

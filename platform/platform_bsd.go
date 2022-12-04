//go:build freebsd
// +build freebsd

package platform

func Get_osname() string {
	return "freebsd"
}

func GetDefaultInterfaceName() (str string, err error) {
	return "", nil
}

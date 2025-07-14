package testplugin

import "fmt"

type address struct {
	ip   string
	port int
}

func (a address) Address() string {
	return fmt.Sprintf("%s:%d", a.ip, a.port)
}

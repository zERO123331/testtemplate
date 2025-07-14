package testplugin

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type address struct {
	ip   string
	port int
}

func (a address) String() string {
	return fmt.Sprintf("%s:%d", a.ip, a.port)
}

func (a address) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

func (a *address) UnmarshalJSON(data []byte) error {
	unquotedJSONValue, err := strconv.Unquote(string(data))
	if err != nil {
		return err
	}

	parts := strings.Split(unquotedJSONValue, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid address: %s", unquotedJSONValue)
	}
	a.ip = parts[0]
	a.port, err = strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("invalid port: %s", parts[1])
	}
	return nil
}

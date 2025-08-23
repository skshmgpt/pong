package utils

import (
	"encoding/base64"
	"fmt"
)

func Decode(id string) string {
	buf, _ := base64.RawURLEncoding.DecodeString(id)
	ip := fmt.Sprintf("192.168.%d.%d", buf[0], buf[1])
	port := int(buf[2])<<8 | int(buf[3])
	port_s := fmt.Sprintf(":%d", port)
	return ip + port_s
}

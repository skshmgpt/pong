package utils

import (
	"encoding/base64"
	"strconv"
	"strings"
)

func Encode(ip string, port int) string {
	parts := strings.Split(ip, ".")
	a, _ := strconv.Atoi(parts[2])
	b, _ := strconv.Atoi(parts[3])

	buf := make([]byte, 4)
	buf[0] = byte(a)
	buf[1] = byte(b)
	buf[2] = byte(port >> 8)
	buf[3] = byte(port & 0xff)

	return base64.RawURLEncoding.EncodeToString(buf)

}

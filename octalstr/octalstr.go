package octalstr

import (
	"bytes"
	"fmt"
	"strconv"
)

func Parse(s string) ([]byte, error) {
	in := []byte(s[1 : len(s)-1]) // strip quotes
	var out bytes.Buffer
	i := 0
	for i < len(in) {
		switch in[i] {
		case '\\':
			esc := string(in[i+1 : i+4])
			o, err := strconv.ParseUint(esc, 8, 8)
			if err != nil {
				return out.Bytes(), fmt.Errorf("parse octal sequence %s: %w", esc, err)
			}
			out.WriteByte(byte(o))
			i += 4
		default:
			out.WriteByte(in[i])
			i++
		}
	}
	return out.Bytes(), nil
}

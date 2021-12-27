//go:generate curl -so ref/mal.csv  http://standards-oui.ieee.org/oui/oui.csv
//go:generate curl -so ref/mam.csv  http://standards-oui.ieee.org/oui28/mam.csv
//go:generate curl -so ref/mas.csv  http://standards-oui.ieee.org/oui36/oui36.csv
package macvendors

import (
	"bytes"
	_ "embed"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"net"
)

//go:embed ref/mal.csv
var malCSV []byte

//go:embed ref/mam.csv
var mamCSV []byte

//go:embed ref/mas.csv
var masCSV []byte

var MacVendors *macVendors = NewMacVendors()

func init() {
	for _, b := range [][]byte{malCSV, mamCSV, masCSV} {
		f := bytes.NewBuffer(b)
		if err := MacVendors.Parse(f); err != nil {
			panic(err)
		}
	}
}

type macVendors struct {
	large  map[[3]byte]string
	medium map[[4]byte]string
	small  map[[5]byte]string
}

func NewMacVendors() *macVendors {
	return &macVendors{
		large:  map[[3]byte]string{},
		medium: map[[4]byte]string{},
		small:  map[[5]byte]string{},
	}
}

func (v *macVendors) Parse(f io.Reader) error {
	r := csv.NewReader(f)
	for {
		row, err := r.Read()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		switch row[0] {
		case "MA-L":
			// ex: 50CEE3, first 24 bits
			b, err := hex.DecodeString(row[1])
			if err != nil {
				return fmt.Errorf("decode ma-l mac prefix hex: %w", err)
			}
			var key [3]byte
			copy(key[:], b)
			v.large[key] = row[2]
		case "MA-M":
			// ex: 9806371, first 28 bits
			b, err := hex.DecodeString(row[1] + "0")
			if err != nil {
				return fmt.Errorf("decode ma-m mac prefix hex: %w", err)
			}
			var key [4]byte
			copy(key[:], b)
			v.medium[key] = row[2]
		case "MA-S":
			// ex: 70B3D562F, first 36 bits
			b, err := hex.DecodeString(row[1] + "0")
			if err != nil {
				return fmt.Errorf("decode ma-m mac prefix hex: %w", err)
			}
			var key [5]byte
			copy(key[:], b)
			v.small[key] = row[2]
		}
	}
}

func (v *macVendors) Lookup(mac net.HardwareAddr) string {
	bits := []byte(mac)
	var mal [3]byte
	copy(mal[:], bits)
	vend, ok := v.large[mal]
	if ok {
		return vend
	}
	var mam [4]byte
	copy(mal[:], bits)
	mam[3] = mam[3] & 0xF // top 4 bits
	vend, ok = v.medium[mam]
	if ok {
		return vend
	}
	var mas [5]byte
	copy(mal[:], bits)
	mas[4] = mas[4] & 0xF // top 4 bits
	vend, ok = v.small[mas]
	if ok {
		return vend
	}
	return ""
}

func Lookup(mac net.HardwareAddr) string {
	return MacVendors.Lookup(mac)
}

func LookupString(mac string) (string, error) {
	hw, err := net.ParseMAC(mac)
	if err != nil {
		return "", err
	}
	return MacVendors.Lookup(hw), nil
}

func IsLocal(mac net.HardwareAddr) bool {
	bits := []byte(mac)
	return bits[0]&0b01 != 0
}

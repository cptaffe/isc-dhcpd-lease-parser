package duid

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/cptaffe/isc-dhcpd-lease-parser/enterprisenumbers"
)

// IAID for IA_NA is 4 bytes, see: https://datatracker.ietf.org/doc/html/rfc3315#section-22.4
// DUID is one of three kinds, see: https://datatracker.ietf.org/doc/html/rfc3315#section-9.1

type DUIDType uint16

const (
	DUIDTypeLLT DUIDType = iota + 1
	DUIDTypeEN
	DUIDTypeLL
)

func (t DUIDType) String() string {
	switch t {
	case DUIDTypeLLT:
		return "DUID-LLT"
	case DUIDTypeEN:
		return "DUID-EN"
	case DUIDTypeLL:
		return "DUID-LL"
	default:
		return "DUID-??"
	}
}

type HardwareType uint16

const (
	HardwareTypeEthernet HardwareType = 1
)

type DUIDLLT struct {
	HardwareType HardwareType `json:"hwtype"`
	Time         time.Time    `json:"time"`
	HardwareAddr string       `json:"hwaddr"`
}

type DUIDEN struct {
	EN           enterprisenumbers.EN `json:"en"`
	HardwareAddr string               `json:"hwaddr"`
}

type DUIDLL struct {
	HardwareType HardwareType `json:"hwtype"`
	HardwareAddr string       `json:"hwaddr"`
}

type DUID struct {
	Type DUIDType `json:"type"`
	LL   *DUIDLL  `json:"ll,omitempty"`
	EN   *DUIDEN  `json:"en,omitempty"`
	LLT  *DUIDLLT `json:"llt,omitempty"`
}

type IAIDDUID struct {
	IAID []byte `json:"iaid"` // for IA_NA this is 4 bytes
	DUID *DUID  `json:"duid"`
}

var duidEpoch = time.Date(2000, time.December, 30, 0, 0, 0, 0, time.UTC)

func ParseDUID(duid []byte) (*DUID, error) {
	duidtype := DUIDType(binary.BigEndian.Uint16(duid[0:2]))
	res := &DUID{
		Type: duidtype,
	}

	switch duidtype {
	case DUIDTypeLLT:
		hwtype := HardwareType(binary.BigEndian.Uint16(duid[2:4]))
		time := duidEpoch.Add(time.Duration(binary.BigEndian.Uint32(duid[4:8])))
		mac := net.HardwareAddr(duid[8:])
		res.LLT = &DUIDLLT{
			HardwareType: hwtype,
			Time:         time,
			HardwareAddr: mac.String(),
		}

	case DUIDTypeEN:
		en := enterprisenumbers.EN(binary.BigEndian.Uint32(duid[2:6]))
		// EN can be anything, in the case of the HP JetDirect 635n it is a MAC
		hwaddr := net.HardwareAddr(duid[6:])
		res.EN = &DUIDEN{
			EN:           en,
			HardwareAddr: hwaddr.String(),
		}
	case DUIDTypeLL:
		hwtype := HardwareType(binary.BigEndian.Uint16(duid[2:4]))
		switch hwtype {
		case HardwareTypeEthernet:
			mac := net.HardwareAddr(duid[4:])
			res.LL = &DUIDLL{
				HardwareType: hwtype,
				HardwareAddr: mac.String(),
			}
		default:
			return nil, fmt.Errorf("unsupported hardware type %d", hwtype)
		}

	default:
		return nil, fmt.Errorf("unexpected DUID type value: %d", duidtype)
	}

	return res, nil
}

// TODO: Respect IA_NA/TA/PD
func ParseIAIDDUID(combined []byte) (*IAIDDUID, error) {
	duid := combined[4:]

	res, err := ParseDUID(duid)
	if err != nil {
		return nil, fmt.Errorf("parse duid: %w", err)
	}

	// Assume IA_NA
	return &IAIDDUID{
		IAID: combined[0:4],
		DUID: res,
	}, nil
}

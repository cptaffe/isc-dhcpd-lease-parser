package duid

import (
	"testing"

	"github.com/cptaffe/isc-dhcpd-lease-parser/octalstr"
)

func TestParseOctal(t *testing.T) {
	in := "\"\\276\\257\\244\\320\\000\\003\\000\\001 \\311\\320\\244\\257\\276\""

	actual, err := octalstr.Parse(in)
	if err != nil {
		t.Errorf("parse octal input: %v\n", err)
	}

	aaidduid, _ := ParseIAIDDUID(actual)

	duid := aaidduid.DUID

	if duid.Type != DUIDTypeLL {
		t.Errorf("expected LL but was %d", DUIDTypeLL)
	}

	if duid.LL == nil {
		var typ string
		switch {
		case duid.LLT != nil:
			typ = "LLT"
		case duid.EN != nil:
			typ = "EN"
		default:
			typ = "empty"
		}
		t.Errorf("expected LL but was %s", typ)
	}

	if duid.LL.HardwareType != HardwareTypeEthernet {
		t.Errorf("expected hardware type Ethernet but was %d", duid.LL.HardwareType)
	}

	expectedMAC := "20:c9:d0:a4:af:be"
	if duid.LL.HardwareAddr != expectedMAC {
		t.Errorf("expected MAC to be %s but was %s", expectedMAC, duid.LL.HardwareAddr)
	}
}

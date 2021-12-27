package octalstr

import (
	"encoding/hex"
	"testing"
)

func TestParseOctal(t *testing.T) {
	in := "\"\\276\\257\\244\\320\\000\\003\\000\\001 \\311\\320\\244\\257\\276\""
	out := "beafa4d00003000120c9d0a4afbe"

	actual, err := Parse(in)
	if err != nil {
		t.Errorf("parse octal input: %v\n", err)
	}

	got := hex.EncodeToString(actual)

	if got != out {
		t.Errorf("Octal decode incorrect.\nExpected: %s\nFound: %s\n", out, got)
	}
}

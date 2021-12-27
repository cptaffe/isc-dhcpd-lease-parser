//go:generate curl -so ref/enterprise-numbers.txt http://www.iana.org/assignments/enterprise-numbers/enterprise-numbers
package enterprisenumbers

import (
	"bufio"
	"bytes"
	_ "embed"
	"io"
	"strconv"
	"strings"
)

// TODO: Find a better way of doing this than loading it into memory once as bytes,
// and once again as parsed. Download at runtime at a set location and index the file?
//go:embed ref/enterprise-numbers.txt
var enterpriseNumbersTxt []byte

var ENTable *enTable = NewENTable()

func init() {
	f := bytes.NewBuffer(enterpriseNumbersTxt)
	if err := ENTable.Parse(f); err != nil {
		panic(err)
	}
}

type enTable struct {
	numbers map[uint32]string
}

func NewENTable() *enTable {
	return &enTable{
		numbers: map[uint32]string{},
	}
}

func (e *enTable) Parse(input io.Reader) error {
	s := bufio.NewScanner(input)
	var current uint32
	for s.Scan() {
		line := s.Text()
		if len(line) == 0 {
			continue
		}
		switch {
		case line[0] >= '0' && line[0] <= '9':
			i, err := strconv.ParseInt(line, 10, 32)
			if err != nil {
				return err
			}
			current = uint32(i) // store current number
		case strings.HasPrefix(line, "      "): // contact
		case strings.HasPrefix(line, "    "): // email
		case strings.HasPrefix(line, "  "): // organization
			e.numbers[current] = strings.Trim(line, " \n")
		}
	}
	return nil
}

func (e *enTable) Lookup(en uint32) (string, bool) {
	info, ok := e.numbers[en]
	return info, ok
}

func Lookup(en uint32) (string, bool) {
	return ENTable.Lookup(en)
}

type EN uint32

func (e EN) Organization() (string, bool) {
	return Lookup(uint32(e))
}

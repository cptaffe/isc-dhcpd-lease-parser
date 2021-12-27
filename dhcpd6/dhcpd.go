//go:generate goyacc -p Lease -o lease.go lease.y
package dhcpd6

import (
	"io"
	"log"
	"net"
	"time"

	"github.com/cptaffe/isc-dhcpd-lease-parser/duid"
	"github.com/cptaffe/isc-dhcpd-lease-parser/lex"
)

// TODO: Handle top-level declarations like `server-duid``

type DHCPv6LeaseType string

const (
	DHCPv6LeaseTypeTemporary        DHCPv6LeaseType = "ia-ta"
	DHCPv6LeaseTypeNonTemporary     DHCPv6LeaseType = "ia-na"
	DHCPv6LeaseTypePrefixDelegation DHCPv6LeaseType = "ia-pd"
)

type DHCPv6LeaseAddr struct {
	IP            net.IP     `json:"ip"`
	BindingState  string     `json:"binding-state,omitempty"`
	PreferredLife int        `json:"preferred-life,omitempty"`
	MaxLife       int        `json:"max-life,omitempty"`
	Ends          *time.Time `json:"ends,omitempty"`
}

type DHCPv6Lease struct {
	Type  DHCPv6LeaseType    `json:"type"`
	IAID  []byte             `json:"iaid"`           // Identity Associated ID
	DUID  *duid.DUID         `json:"duid"`           // DHCP Unique ID
	CLTT  *time.Time         `json:"cltt,omitempty"` // Client's Last Transaction Time
	Addrs []*DHCPv6LeaseAddr `json:"addrs,omitempty"`
}

// Allows us to pile up modifications to lease lazily and then
// evaluate them at once, that way we don't have to write the action
// twice -- once for when it is first and once for when a option precedes it
// and the lease object is already constructed.
type DHCPv6LeaseOption interface {
	Apply(lease *DHCPv6Lease)
}

type DHCPv6LeaseOptionCLTT time.Time

func (t *DHCPv6LeaseOptionCLTT) Apply(lease *DHCPv6Lease) {
	lease.CLTT = (*time.Time)(t)
}

type DHCPv6LeaseOptionAddr DHCPv6LeaseAddr

func (addr *DHCPv6LeaseOptionAddr) Apply(lease *DHCPv6Lease) {
	lease.Addrs = append(lease.Addrs, (*DHCPv6LeaseAddr)(addr))
}

type DHCPv6LeaseAddrOption interface {
	Apply(addr *DHCPv6LeaseAddr)
}

type DHCPv6LeaseAddrOptionBindingState string

func (bs DHCPv6LeaseAddrOptionBindingState) Apply(addr *DHCPv6LeaseAddr) {
	addr.BindingState = (string)(bs)
}

type DHCPv6LeaseAddrOptionEnds time.Time

func (ends *DHCPv6LeaseAddrOptionEnds) Apply(addr *DHCPv6LeaseAddr) {
	addr.Ends = (*time.Time)(ends)
}

type DHCPv6LeaseAddrOptionPreferredLife int

func (life DHCPv6LeaseAddrOptionPreferredLife) Apply(addr *DHCPv6LeaseAddr) {
	addr.PreferredLife = int(life)
}

type DHCPv6LeaseAddrOptionMaxLife int

func (life DHCPv6LeaseAddrOptionMaxLife) Apply(addr *DHCPv6LeaseAddr) {
	addr.MaxLife = int(life)
}

type LeaseLex struct {
	DHCPv6Leases chan *DHCPv6Lease
	Line         int
	LineTokens   []lex.Token
	Tokens       chan lex.Token
}

func (l *LeaseLex) Lex(lval *LeaseSymType) int {
	token, ok := <-l.Tokens
	if !ok {
		return 0
	}
	if l.Line != token.Pos.Line {
		l.LineTokens = []lex.Token{}
		l.Line = token.Pos.Line
	}
	l.LineTokens = append(l.LineTokens, token)
	lval.s = token.Val
	switch token.Typ {
	case lex.ItemBeginBlock:
		return BEGINBLOCK
	case lex.ItemEndBlock:
		return ENDBLOCK
	case lex.ItemWord:
		return WORD
	case lex.ItemString:
		return STRING
	case lex.ItemSemicolon:
		return SEMICOLON
	case lex.ItemAssign:
		return ASSIGN
	case lex.ItemSet:
		return SET
	case lex.ItemLease:
		return LEASE
	default:
		log.Fatalf("unknown token: %+v\n", token)
	}
	return 0
}

func (l *LeaseLex) Error(e string) {
	log.Fatalf("at %+v: %v\n", l.LineTokens, e)
}

func Parse(input io.Reader) chan *DHCPv6Lease {
	tokens := lex.Lex(input)
	leases := make(chan *DHCPv6Lease)
	go func() {
		LeaseParse(&LeaseLex{Tokens: tokens, DHCPv6Leases: leases})
		close(leases)
	}()
	return leases
}

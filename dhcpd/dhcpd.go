//go:generate goyacc -p Lease -o lease.go lease.y
package dhcpd

import (
	"io"
	"log"
	"net"
	"time"

	"github.com/cptaffe/isc-dhcpd-lease-parser/lex"
)

type DHCPv4Lease struct {
	IP                 net.IP     `json:"ip"`
	Starts             *time.Time `json:"starts,omitempty"`
	Ends               *time.Time `json:"ends,omitempty"`
	TSTP               *time.Time `json:"tstp,omitempty"`
	TSFP               *time.Time `json:"tsfp,omitempty"`
	ATSFP              *time.Time `json:"atsfp,omitempty"`
	CLTT               *time.Time `json:"cltt,omitempty"` // Client's Last Transaction Time
	BindingState       string     `json:"binding-state,omitempty"`
	NextBindingState   string     `json:"next-binding-state,omitempty"`
	RewindBindingState string     `json:"rewind-binding-state,omitempty"`
	HardwareEthernet   string     `json:"hardware-ethernet,omitempty"`
	ClientHostname     string     `json:"client-hostname,omitempty"`
	UID                []byte     `json:"uid,omitempty"`
	// set
	VendorClassIdentifier string `json:"vendor-class-identifier,omitempty"`
	DDNSFwdName           string `json:"ddns-fwd-name,omitempty"`
	DDNSTxt               string `json:"ddns-txt,omitempty"`
	DDNSRevName           string `json:"ddns-rev-name,omitempty"`
}

// Allows us to pile up modifications to lease lazily and then
// evaluate them at once, that way we don't have to write the action
// twice -- once for when it is first and once for when a option precedes it
// and the lease object is already constructed.
type DHCPv4LeaseOption interface {
	Apply(lease *DHCPv4Lease)
}

type DHCPv4LeaseOptionStarts time.Time

func (t *DHCPv4LeaseOptionStarts) Apply(lease *DHCPv4Lease) {
	lease.Starts = (*time.Time)(t)
}

type DHCPv4LeaseOptionEnds time.Time

func (t *DHCPv4LeaseOptionEnds) Apply(lease *DHCPv4Lease) {
	lease.Ends = (*time.Time)(t)
}

type DHCPv4LeaseOptionTSTP time.Time

func (t *DHCPv4LeaseOptionTSTP) Apply(lease *DHCPv4Lease) {
	lease.TSTP = (*time.Time)(t)
}

type DHCPv4LeaseOptionTSFP time.Time

func (t *DHCPv4LeaseOptionTSFP) Apply(lease *DHCPv4Lease) {
	lease.TSFP = (*time.Time)(t)
}

type DHCPv4LeaseOptionATSFP time.Time

func (t *DHCPv4LeaseOptionATSFP) Apply(lease *DHCPv4Lease) {
	lease.ATSFP = (*time.Time)(t)
}

type DHCPv4LeaseOptionCLTT time.Time

func (t *DHCPv4LeaseOptionCLTT) Apply(lease *DHCPv4Lease) {
	lease.CLTT = (*time.Time)(t)
}

type DHCPv4LeaseOptionUID []byte

func (uid DHCPv4LeaseOptionUID) Apply(lease *DHCPv4Lease) {
	lease.UID = []byte(uid)
}

type DHCPv4LeaseOptionBindingState string

func (bs DHCPv4LeaseOptionBindingState) Apply(lease *DHCPv4Lease) {
	lease.BindingState = string(bs)
}

type DHCPv4LeaseOptionNextBindingState string

func (nbs DHCPv4LeaseOptionNextBindingState) Apply(lease *DHCPv4Lease) {
	lease.NextBindingState = string(nbs)
}

type DHCPv4LeaseOptionRewindBindingState string

func (rbs DHCPv4LeaseOptionRewindBindingState) Apply(lease *DHCPv4Lease) {
	lease.RewindBindingState = string(rbs)
}

type DHCPv4LeaseOptionHardwareEthernet string

func (eth DHCPv4LeaseOptionHardwareEthernet) Apply(lease *DHCPv4Lease) {
	lease.HardwareEthernet = string(eth)
}

type DHCPv4LeaseOptionClientHostname string

func (hostname DHCPv4LeaseOptionClientHostname) Apply(lease *DHCPv4Lease) {
	lease.ClientHostname = string(hostname)
}

type DHCPv4LeaseOptionVendorClassIdentifier string

func (vci DHCPv4LeaseOptionVendorClassIdentifier) Apply(lease *DHCPv4Lease) {
	lease.VendorClassIdentifier = string(vci)
}

type DHCPv4LeaseOptionDDNSFwdName string

func (dfn DHCPv4LeaseOptionDDNSFwdName) Apply(lease *DHCPv4Lease) {
	lease.DDNSFwdName = string(dfn)
}

type DHCPv4LeaseOptionDDNSTxt string

func (dt DHCPv4LeaseOptionDDNSTxt) Apply(lease *DHCPv4Lease) {
	lease.DDNSTxt = string(dt)
}

type DHCPv4LeaseOptionDDNSRevName string

func (drn DHCPv4LeaseOptionDDNSRevName) Apply(lease *DHCPv4Lease) {
	lease.DDNSRevName = string(drn)
}

type LeaseLex struct {
	DHCPv4Leases chan *DHCPv4Lease
	CurrentToken lex.Token
	Tokens       chan lex.Token
}

func (l *LeaseLex) Lex(lval *LeaseSymType) int {
	token, ok := <-l.Tokens
	if !ok {
		return 0
	}
	l.CurrentToken = token
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
	log.Fatalf("at %+v: %v\n", l.CurrentToken, e)
}

func Parse(input io.Reader) chan *DHCPv4Lease {
	tokens := lex.Lex(input)
	leases := make(chan *DHCPv4Lease)
	go func() {
		LeaseParse(&LeaseLex{Tokens: tokens, DHCPv4Leases: leases})
		close(leases)
	}()
	return leases
}

%{

package dhcpd6

import (
	"log"
	"time"
	"net"
	"strconv"

	"github.com/cptaffe/isc-dhcpd-lease-parser/octalstr"
	"github.com/cptaffe/isc-dhcpd-lease-parser/duid"
)

%}

// fields inside this union end up as the fields in a structure known
// as ${PREFIX}SymType, of which a reference is passed to the lexer.
%union{
	s string
	lease_addr_detail DHCPv6LeaseAddrOption
	lease_addr_details []DHCPv6LeaseAddrOption
	lease_detail DHCPv6LeaseOption
	lease_details []DHCPv6LeaseOption
}

// any non-terminal which returns a value needs a type, which is
// really a field name in the above union struct
%type <lease_details> lease_details
%type <lease_detail> lease_detail
%type <lease_addr_details> lease_addr_details
%type <lease_addr_detail> lease_addr_detail

// same for terminals
%token <s> BEGINBLOCK ENDBLOCK WORD STRING SEMICOLON ASSIGN SET LEASE

%%
statements:
	statement
	| statements statement;

statement:
	WORD WORD SEMICOLON
	{
		switch {
		
		// authoring-byte-order little-endian;
		case $1 == "authoring-byte-order":
			// do nothing
		
		default:
			log.Fatalf("unknown top-level directive: %s %s;\n", $1, $2)
		}
	}

	| WORD STRING SEMICOLON
	{
		switch {

		// server-duid "\000\001\000\001)Yc\234\000\014),\357u";
		case $1 == "server-duid":
			// do nothing
		
		default:
			log.Fatalf("unknown top-level directive: %s %s;\n", $1, $2)
		}
	}

	| LEASE STRING BEGINBLOCK lease_details ENDBLOCK
	{
		l := &DHCPv6Lease{Type: DHCPv6LeaseType($1)}
		comb, err := octalstr.Parse($2)
		if err != nil {
			log.Fatalf("lease iaid-duid string parse: %v\n", err)
		}
		iaidduid, err := duid.ParseIAIDDUID(comb)
		if err != nil {
			log.Fatalf("lease iaid-duid parse: %v\n", err)
		}
		l.IAID = iaidduid.IAID
		l.DUID = iaidduid.DUID
		for _, opt := range $4 {
			opt.Apply(l)
		}
		Leaselex.(*LeaseLex).DHCPv6Leases <- l
	};

lease_details: lease_detail { $$ = []DHCPv6LeaseOption{$1} }
	| lease_details lease_detail { $$ = append($1, $2) };

lease_detail:
	WORD WORD WORD WORD SEMICOLON
	{
		switch {
		// cltt 6 2021/12/25 22:24:37;
		case $1 == "cltt":
			// default db-time-format is weekday year/month/day hour:minute:second
			t, err := time.Parse("2006/01/02 15:04:05", $3 + " " + $4)
			if err != nil {
				log.Fatalf("lease detail cltt: %v\n", err)
			}
			opt := DHCPv6LeaseOptionCLTT(t)
			$$ = &opt

		default:
			log.Fatalf("unknown lease detail: %s %s %s %s;\n", $1, $2, $3, $4)
		}
	}

	| WORD WORD BEGINBLOCK lease_addr_details ENDBLOCK
	{
		switch {
		case $1 == "iaaddr":
			addr := &DHCPv6LeaseAddr{IP: net.ParseIP($2)}
			for _, opt := range $4 {
				opt.Apply(addr)
			}
			$$ = (*DHCPv6LeaseOptionAddr)(addr)

		default:
			log.Fatalf("unknown lease detail: %s %s %s %s;\n", $1, $2, $3, $4)
		}
	};

lease_addr_details: lease_addr_detail { $$ = []DHCPv6LeaseAddrOption{$1} }
	| lease_addr_details lease_addr_detail { $$ = append($1, $2) };

lease_addr_detail:
	WORD WORD SEMICOLON
	{
		switch {
		
		// preferred-life 375;
		case $1 == "preferred-life":
			i, err := strconv.Atoi($2)
			if err != nil {
				log.Fatalf("lease addr preferred-life parse int: %v\n", err)
			}
			$$ = DHCPv6LeaseAddrOptionPreferredLife(i)

		// max-life 600;
		case $1 == "max-life":
			i, err := strconv.Atoi($2)
			if err != nil {
				log.Fatalf("lease addr max-life parse int: %v\n", err)
			}
			$$ = DHCPv6LeaseAddrOptionMaxLife(i)

		default:
			log.Fatalf("unknown lease addr detail: %s %s;\n", $1, $2)
		}
	}

	| WORD WORD WORD SEMICOLON
	{
		switch {
		
		// binding state free;
		case $1 == "binding" && $2 == "state":
			$$ = DHCPv6LeaseAddrOptionBindingState($3)

		default:
			log.Fatalf("unknown lease addr detail: %s %s %s;\n", $1, $2, $3)
		}
	}
	
	| WORD WORD WORD WORD SEMICOLON
	{
		switch {
		
		// ends 6 2021/12/25 22:34:37;
		case $1 == "ends":
			// default db-time-format is weekday year/month/day hour:minute:second
			t, err := time.Parse("2006/01/02 15:04:05", $3 + " " + $4)
			if err != nil {
				log.Fatalf("lease addr detail ends: %v\n", err)
			}
			opt := DHCPv6LeaseAddrOptionEnds(t)
			$$ = &opt

		default:
			log.Fatalf("unknown lease addr detail: %s %s %s %s;\n", $1, $2, $3, $4)
		}
	};

%%
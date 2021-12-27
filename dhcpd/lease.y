%{

package dhcpd

import (
	"log"
	"net"
	"time"

	"github.com/cptaffe/isc-dhcpd-lease-parser/octalstr"
)

%}

// fields inside this union end up as the fields in a structure known
// as ${PREFIX}SymType, of which a reference is passed to the lexer.
%union{
	s string
	lease_detail DHCPv4LeaseOption
	lease_details []DHCPv4LeaseOption
	lease *DHCPv4Lease
}

// any non-terminal which returns a value needs a type, which is
// really a field name in the above union struct
%type <lease> lease
%type <lease_details> lease_details
%type <lease_detail> lease_detail

// same for terminals
%token <s> BEGINBLOCK ENDBLOCK WORD STRING SEMICOLON ASSIGN SET LEASE

%left ASSIGN

%%
leases:
	lease
	{
		Leaselex.(*LeaseLex).DHCPv4Leases <- $1
	}
	| WORD WORD SEMICOLON
	{
		switch {
		
		// authoring-byte-order little-endian;
		case $1 == "authoring-byte-order":
			// do nothing
		
		default:
			log.Fatalf("unknown top-level directive: %s %s;\n", $1, $2)
		}
	}
	| leases lease
	{
		Leaselex.(*LeaseLex).DHCPv4Leases <- $2
	};

lease:
	LEASE WORD BEGINBLOCK lease_details ENDBLOCK
	{
		$$ = &DHCPv4Lease{IP: net.ParseIP($2)}
		for _, opt := range $4 {
			opt.Apply($$)
		}
	};

lease_details: lease_detail { $$ = []DHCPv4LeaseOption{$1} }
	| lease_details lease_detail { $$ = append($1, $2) };

lease_detail:
	WORD STRING SEMICOLON
	{
		switch {
		
		// uid "\001\264\231\272\003\217\346";
		case $1 == "uid":
			unquote, err := octalstr.Parse($2)
			if err != nil {
				log.Fatalf("lease detail uid string unquote: %v\n", err)
			}
			$$ = DHCPv4LeaseOptionUID(unquote)

		// client-hostname "wopr";
		case $1 == "client-hostname":
			$$ = DHCPv4LeaseOptionClientHostname($2[1:len($2)-1])
		
		default:
			log.Fatalf("unknown lease detail: %s %s;\n", $1, $2)
		}
	};

	| WORD WORD WORD SEMICOLON
	{
		switch {
		
		// binding state active;
		case $1 == "binding" && $2 == "state":
			$$ = DHCPv4LeaseOptionBindingState($3)

		// hardware ethernet 8c:dc:d4:2b:ec:6c;
		case $1 == "hardware" && $2 == "ethernet":
			// TODO: MAC serializes to JSON as base64 by default
			// ha, err := net.ParseMAC($3)
			// if err != nil {
			// 	log.Fatalf("lease detail hardware ethernet parse mac: %v\n", err)
			// }
			$$ = DHCPv4LeaseOptionHardwareEthernet($3)

		default:
			log.Fatalf("unknown lease detail: %s %s %s;\n", $1, $2, $3)
		}
	}

	| WORD WORD WORD WORD SEMICOLON
	{
		switch {
		
		// starts 6 2021/12/25 22:27:49;
		case $1 == "starts":
			// default db-time-format is weekday year/month/day hour:minute:second
			t, err := time.Parse("2006/01/02 15:04:05", $3 + " " + $4)
			if err != nil {
				log.Fatalf("lease detail starts: %v\n", err)
			}
			opt := DHCPv4LeaseOptionStarts(t)
			$$ = &opt
		
		// ends 6 2021/12/25 22:34:37;
		case $1 == "ends":
			// default db-time-format is weekday year/month/day hour:minute:second
			t, err := time.Parse("2006/01/02 15:04:05", $3 + " " + $4)
			if err != nil {
				log.Fatalf("lease detail ends: %v\n", err)
			}
			opt := DHCPv4LeaseOptionEnds(t)
			$$ = &opt

		// tstp 0 2021/12/26 05:36:57;
		case $1 == "tstp":
			// default db-time-format is weekday year/month/day hour:minute:second
			t, err := time.Parse("2006/01/02 15:04:05", $3 + " " + $4)
			if err != nil {
				log.Fatalf("lease detail tstp: %v\n", err)
			}
			opt := DHCPv4LeaseOptionTSTP(t)
			$$ = &opt

		// tsfp
		case $1 == "tsfp":
			// default db-time-format is weekday year/month/day hour:minute:second
			t, err := time.Parse("2006/01/02 15:04:05", $3 + " " + $4)
			if err != nil {
				log.Fatalf("lease detail tsfp: %v\n", err)
			}
			opt := DHCPv4LeaseOptionTSFP(t)
			$$ = &opt

		// atsfp
		case $1 == "atsfp":
			// default db-time-format is weekday year/month/day hour:minute:second
			t, err := time.Parse("2006/01/02 15:04:05", $3 + " " + $4)
			if err != nil {
				log.Fatalf("lease detail atsfp: %v\n", err)
			}
			opt := DHCPv4LeaseOptionATSFP(t)
			$$ = &opt
		
		// cltt 6 2021/12/25 22:24:37;
		case $1 == "cltt":
			// default db-time-format is weekday year/month/day hour:minute:second
			t, err := time.Parse("2006/01/02 15:04:05", $3 + " " + $4)
			if err != nil {
				log.Fatalf("lease detail cltt: %v\n", err)
			}
			opt := DHCPv4LeaseOptionCLTT(t)
			$$ = &opt
		
		// next binding state free;
		case $1 == "next" && $2 == "binding" && $3 == "state":
			$$ = DHCPv4LeaseOptionNextBindingState($4)

		// rewind binding state free;
		case $1 == "rewind" && $2 == "binding" && $3 == "state":
			$$ = DHCPv4LeaseOptionRewindBindingState($4)

		default:
			log.Fatalf("unknown lease detail: %s %s %s %s;\n", $1, $2, $3, $4)
		}
	}

	| SET WORD ASSIGN STRING SEMICOLON
	{
		switch {

		// set vendor-class-identifier = "MSFT 5.0";
		case $2 == "vendor-class-identifier":
			$$ = DHCPv4LeaseOptionVendorClassIdentifier($4[1:len($4)-1])

		// set ddns-fwd-name = "wopr.heavy.computer";
		case $2 == "ddns-fwd-name":
			$$ = DHCPv4LeaseOptionDDNSFwdName($4[1:len($4)-1])
		
		// set ddns-txt = "311faf8c3f99c3c50ad3a775ea6d108052";
		case $2 == "ddns-txt":
			$$ = DHCPv4LeaseOptionDDNSTxt($4[1:len($4)-1])

		// set ddns-rev-name = "107.1.168.192.in-addr.arpa";
		case $2 == "ddns-rev-name":
			$$ = DHCPv4LeaseOptionDDNSRevName($4[1:len($4)-1])

		default:
			log.Fatalf("unknown lease detail: set %s = %s;\n", $2, $4)
		}
	}
%%
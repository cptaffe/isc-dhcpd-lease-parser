package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"math"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"time"

	autoneg "github.com/adjust/goautoneg"
	"github.com/cptaffe/isc-dhcpd-lease-parser/dhcpd"
	"github.com/cptaffe/isc-dhcpd-lease-parser/dhcpd6"
	"github.com/cptaffe/isc-dhcpd-lease-parser/macvendors"
)

//go:embed templates
var content embed.FS
var funcs = template.FuncMap{
	"since": func(t time.Time) time.Duration {
		return time.Since(t)
	},
	"until": func(t time.Time) time.Duration {
		return time.Until(t)
	},
	"isPast": func(t time.Time) bool {
		return time.Since(t) > 0
	},
	"isFuture": func(t time.Time) bool {
		return time.Until(t) > 0
	},
	// Format a human-readable order of magnitude for duations, e.g. 2 weeks or 1 hour
	"duration": func(t time.Duration) string {
		var b strings.Builder
		hours := int(math.Abs(t.Hours()))
		years := hours / 8760
		weeks := hours / 168
		days := hours / 24
		hours = hours % 24
		minutes := int(math.Abs(t.Minutes())) % 60
		seconds := int(math.Abs(t.Seconds())) % 60
		switch {
		case years > 0:
			if years == 1 {
				b.WriteString("1 year")
			} else {
				fmt.Fprintf(&b, "%d year", years)
			}
		case weeks > 0:
			if weeks == 1 {
				b.WriteString("1 week")
			} else {
				fmt.Fprintf(&b, "%d weeks", weeks)
			}
		case days > 0:
			if days == 1 {
				b.WriteString("1 day")
			} else {
				fmt.Fprintf(&b, "%d days", days)
			}
		case hours > 0:
			if hours == 1 {
				b.WriteString("1 hour")
			} else {
				fmt.Fprintf(&b, "%d hours", hours)
			}
		case minutes > 0:
			if minutes == 1 {
				b.WriteString("1 minute")
			} else {
				fmt.Fprintf(&b, "%d minutes", minutes)
			}
		case seconds > 0:
			if seconds == 1 {
				b.WriteString("1 second")
			} else {
				fmt.Fprintf(&b, "%d seconds", seconds)
			}
		}
		return b.String()
	},
	"vendor": func(mac string) string {
		if mac == "" {
			return ""
		}
		hw, err := net.ParseMAC(mac)
		if err != nil {
			log.Printf("parse mac %s: %v\n", mac, err)
		}
		if macvendors.IsLocal(hw) {
			return "Local"
		}
		vendor := macvendors.Lookup(hw)
		if vendor == "" {
			return "Private"
		}
		return vendor
	},
	"revdns": func(ip net.IP) string {
		hosts, err := net.LookupAddr(ip.String())
		if err != nil || len(hosts) == 0 {
			return ""
		}
		return hosts[0]
	},
	"dnseq": func(name1 string, name2 string) bool {
		return strings.EqualFold(strings.TrimSuffix(name1, "."), strings.TrimSuffix(name2, "."))
	},
	"title": strings.Title,
}
var leasesTemplate = template.Must(template.New("leases.html").Funcs(funcs).ParseFS(content, "templates/leases.html"))
var v4LeaseFileFlag = flag.String("v4f", "/var/lib/dhcp/dhcpd.leases", "Path to dhcpd.leases file")
var v6LeaseFileFlag = flag.String("v6f", "/var/lib/dhcp/dhcpd6.leases", "Path to dhcpd6.leases file")
var listenFlag = flag.String("l", ":8080", "Listen interface e.g. :80 or 192.168.1.1:80")

type V1Leases struct {
	DHCPv4Leases []dhcpd.DHCPv4Lease  `json:"v4Leases"`
	DHCPv6Leases []dhcpd6.DHCPv6Lease `json:"v6Leases"`
}

func main() {
	flag.Parse()

	// Convenience redirect
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/v1/leases", http.StatusMovedPermanently)
	})

	http.HandleFunc("/v1/leases", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Only GET requests are supported", http.StatusMethodNotAllowed)
			return
		}
		ct := autoneg.Negotiate(r.Header.Get("Accept"), []string{"application/json", "text/html"})
		v4leases, err := fetchDHCPv4Leases()
		if err != nil {
			log.Println(err)
			http.Error(w, "Failed to fetch v4 leases", http.StatusInternalServerError)
			return
		}
		v6leases, err := fetchDHCPv6Leases()
		if err != nil {
			log.Println(err)
			http.Error(w, "Failed to fetch v6 leases", http.StatusInternalServerError)
			return
		}
		leases := V1Leases{DHCPv4Leases: v4leases, DHCPv6Leases: v6leases}
		switch ct {
		case "application/json":
			json.NewEncoder(w).Encode(leases)
		case "text/html":
			err = leasesTemplate.Execute(w, leases)
			if err != nil {
				log.Println(err)
				http.Error(w, "Failed to present leases", http.StatusInternalServerError)
				return
			}
		}
	})
	log.Fatal(http.ListenAndServe(*listenFlag, nil)) // CAP_NET_BIND_SERVICE
}

func fetchDHCPv4Leases() ([]dhcpd.DHCPv4Lease, error) {
	var leases []dhcpd.DHCPv4Lease
	cmd := exec.Command("dhcpd2json", "-f", *v4LeaseFileFlag)
	stdout, err := cmd.StdoutPipe()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err != nil {
		return leases, fmt.Errorf("dhcpd2json pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return leases, fmt.Errorf("dhcpd2json start: %w", err)
	}
	dec := json.NewDecoder(stdout)
	for dec.More() {
		var lease dhcpd.DHCPv4Lease
		dec.Decode(&lease)
		// Reverse to put the newest (by start date) on top
		leases = append([]dhcpd.DHCPv4Lease{lease}, leases...)
	}
	if err := cmd.Wait(); err != nil {
		return leases, fmt.Errorf("dhcpd2json wait: %w, stderr: %s", err, stderr.String())
	}
	return leases, nil
}

func fetchDHCPv6Leases() ([]dhcpd6.DHCPv6Lease, error) {
	var leases []dhcpd6.DHCPv6Lease
	cmd := exec.Command("dhcpd62json", "-f", *v6LeaseFileFlag)
	stdout, err := cmd.StdoutPipe()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err != nil {
		return leases, fmt.Errorf("dhcpd62json pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return leases, fmt.Errorf("dhcpd62json start: %w", err)
	}
	dec := json.NewDecoder(stdout)
	for dec.More() {
		var lease dhcpd6.DHCPv6Lease
		dec.Decode(&lease)
		// Reverse to put the newest (by start date) on top
		leases = append([]dhcpd6.DHCPv6Lease{lease}, leases...)
	}
	if err := cmd.Wait(); err != nil {
		return leases, fmt.Errorf("dhcpd62json wait: %w, stderr: %s", err, stderr.String())
	}
	return leases, nil
}

# ISC DHCP Daemon Lease Database Parser

This module consists of three binaries:

- `dhcpd2json`, the `dhcpd.leases` parser
- `dhcpd62json`, the `dhcpd6.leases` parser
- `dhcp-httpd`, the DHCP lease server

The `dhcp-httpd` server module executes the `dhcpd2json` and `dhcpd62json` commands to fetch the leases in JSON format and then provide them via HTTP either in JSON or as an HTML page. The separation exists partly as a division of labor (although the DHCP lease data structures are currently in the same library that provides parsing functionality), and partly to protect the server from the `log.Fatal`s scattered throughout the parser code.

This package also provides several adjacent pieces of functionality, as libraries:

- Parsers for both the `dhcp.leases` and `dhcp6.leases` files (they are quite different)
- A parser (`duid`) for the IAID+DUID string which ISC DHCP places after `ia-na` or similar blocks in the `dhcp6.leases` file. The string is made up of escaped octets which represent a binary four byte IAID (in the case of `ia-na`) followed by a DUID of one of [three flavors](https://datatracker.ietf.org/doc/html/rfc3315#section-9.1).
- A utility library (`macvendor`) to lookup the vendor name from the IEEE prefix database files given a MAC address.
- A utility library (`enterprisenumbers`) to lookup the organization name from the IANA database file given an enterprise number, this could be useuful when DUIDs are of the DUID-EN variety.

## Installation

- First run `go generate` in the `dhcpd` and `dhcpd6` libraries to generate the parsers, this requires `goyacc`.
- Then, run `go generate` in the `macvendors` and `enterprisenumbers` libraries to pull down the latest IEEE and IANA database files.
- Then, build the `dhcpd2json`, `dhcpd62json`, and `dhcp-httpd` binaries for your target platform, e.g. `GOOS=linux GOARCH=amd64 go build .` in those directories.

Once the build is done, then:

1. Copy binaries to `/usr/bin` on the target system.
2. Change SELinux context using e.g. `chcon -u system_u -t bin_t /usr/bin/dhcpd2json` and `chcon -u system_u -t bin_t /usr/bin/dhcp-httpd`.

To see the lease listing visit the URL e.g. http://localhost:8080, or to see the JSON response:

```sh
$ curl -sL http://localhost:8080 | jq
```

Here is an example systemd unit file:

```
[Unit]
Description=DHCPv4 HTTP UI Server
Wants=network-online.target
After=network-online.target
After=time-sync.target

[Service]
Type=simple
User=dhcpd
Group=dhcpd
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
AmbientCapabilities=CAP_NET_BIND_SERVICE
#EnvironmentFile=-/etc/sysconfig/dhcpd
ExecStart=/usr/bin/dhcp-httpd -v4f /var/lib/dhcpd/dhcpd.leases -v6f /var/lib/dhcpd/dhcpd6.leases -l :80

[Install]
WantedBy=multi-user.target
```
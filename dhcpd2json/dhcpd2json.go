package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/cptaffe/isc-dhcpd-lease-parser/dhcpd"
)

var leaseFileFlag = flag.String("f", "", "Path to dhcpd.leases file")
var outputFileFlag = flag.String("o", "", "Path to write ouput")

func main() {
	flag.Parse()

	outputFile := os.Stdout
	if *outputFileFlag != "" {
		f, err := os.Create(*outputFileFlag)
		if err != nil {
			log.Fatal(err)
		}
		outputFile = f
	}
	leasesFile := os.Stdin
	if *leaseFileFlag != "" {
		f, err := os.Open(*leaseFileFlag)
		if err != nil {
			log.Fatal(err)
		}
		leasesFile = f
	}

	for lease := range dhcpd.Parse(leasesFile) {
		err := json.NewEncoder(outputFile).Encode(lease)
		if err != nil {
			log.Fatal(err)
		}
	}
}

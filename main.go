package main

import (
	"log"
	"os"
	"strings"
)

const zoneNameSuffix = "."

func normalizeZoneName(z string) string {
	if strings.HasSuffix(z, zoneNameSuffix) {
		return z
	}
	return z + zoneNameSuffix
}

func main() {
	if len(os.Args) != 3 {
		log.Fatalln("args length not 2")
	}
	var (
		zoneName = os.Args[1]
		hostName = os.Args[2]
	)

	if err := update(normalizeZoneName(zoneName), hostName); err != nil {
		log.Fatalf("failed to update, err %+v", err)
	}
}

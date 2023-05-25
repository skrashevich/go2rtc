package mdns

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
)

const (
	homekitService = "_hap._tcp"
	homekitDomain  = ".local"
	Suffix         = homekitDomain + "."
)

// StatusFlags captures Bonjour TXT status flags
type StatusFlags byte

func (s StatusFlags) Bool() bool {

	if s&1 == 1 {
		return false
	} else {
		return true
	}

}
func ParseTXT(txts []string) map[string]string {
	mapped := make(map[string]string)

	for _, txt := range txts {
		parts := strings.SplitN(txt, "=", 2)
		if len(parts) == 2 {
			key := strings.ToLower(parts[0])
			value := parts[1]

			mapped[key] = value
		}
	}

	return mapped
}
func ParseFlag(v string) byte {
	if v == "" {
		return 0
	}
	f, err := strconv.ParseUint(v, 10, 8)
	if err != nil {
		panic(err)
	}

	return byte(f)
}

func GetAll(timeoutOpt ...time.Duration) ([]zeroconf.ServiceEntry, error) {
	timeout := time.Second
	if len(timeoutOpt) > 0 {
		timeout = timeoutOpt[0]
	}

	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	entries := make(chan *zeroconf.ServiceEntry)
	go func() {
		err = resolver.Browse(ctx, homekitService, homekitDomain, entries)
		if err != nil {
			close(entries)
		}
	}()

	result := []zeroconf.ServiceEntry{}
	for entry := range entries {
		//fmt.Printf("\n%v\n\n\n", entry)
		result = append(result, *entry)
	}

	return result, nil
}

func GetAddress(deviceID string) string {
	entries, err := GetAll()
	fmt.Printf("%v", entries)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if strings.Contains(strings.Join(entry.Text, ""), deviceID) {
			return fmt.Sprintf("%s:%d", entry.AddrIPv4[0].String(), entry.Port)
		}
	}

	return ""
}

func GetEntry(deviceID string) *zeroconf.ServiceEntry {
	entries, err := GetAll()
	fmt.Printf("%v", entries)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		if strings.Contains(strings.Join(entry.Text, ""), deviceID) {
			return &entry
		}
	}

	return nil
}

// err := resolver.Browse(ctx, homekitService, homekitDomain, entries)

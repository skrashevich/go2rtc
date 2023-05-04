package core

import (
	cryptorand "crypto/rand"
	"github.com/rs/zerolog/log"
	"net"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Now90000 - timestamp for Video (clock rate = 90000 samples per second)
// same as: uint32(time.Duration(time.Now().UnixNano()) * 90000 / time.Second)
func Now90000() uint32 {
	return uint32(time.Duration(time.Now().UnixMilli()) * 90)
}

const symbols = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ-_"

// RandString base10 - numbers, base16 - hex, base36 - digits+letters, base64 - URL safe symbols
func RandString(size, base byte) string {
	b := make([]byte, size)
	if _, err := cryptorand.Read(b); err != nil {
		panic(err)
	}
	for i := byte(0); i < size; i++ {
		b[i] = symbols[b[i]%base]
	}
	return string(b)
}

func Any(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func Between(s, sub1, sub2 string) string {
	i := strings.Index(s, sub1)
	if i < 0 {
		return ""
	}
	s = s[i+len(sub1):]

	if len(sub2) == 1 {
		i = strings.IndexByte(s, sub2[0])
	} else {
		i = strings.Index(s, sub2)
	}
	if i >= 0 {
		return s[:i]
	}

	return s
}

func Atoi(s string) (i int) {
	i, _ = strconv.Atoi(s)
	return
}

func Assert(ok bool) {
	if !ok {
		_, file, line, _ := runtime.Caller(1)
		panic(file + ":" + strconv.Itoa(line))
	}
}

func Caller() string {
	log.Error().Caller(0).Send()
	_, file, line, _ := runtime.Caller(1)
	return file + ":" + strconv.Itoa(line)
}
func GetMinimumMTU() uint16 {
	const DefaultMTU uint16 = ^uint16(0)
	interfaces, err := net.Interfaces()
	if err != nil {
		return DefaultMTU
	}

	var minMTU uint16 = DefaultMTU
	for _, iface := range interfaces {
		// Skip tunnel and loopback interfaces
		if (iface.Flags&net.FlagLoopback != 0) || (iface.Flags&net.FlagPointToPoint != 0) || (iface.Flags&net.FlagRunning == 0) {
			continue
		}

		// Get the MTU of the current interface
		mtu := uint16(iface.MTU)

		// Update the minimum MTU if it hasn't been set yet or if the current MTU is smaller
		if minMTU == DefaultMTU || mtu < minMTU {
			minMTU = mtu
			log.Debug().Uint16("mtu", minMTU).Str("iface", iface.Name).Msg("[net] ")
		}
	}

	log.Debug().Uint16("mtu", minMTU).Msg("[net] calculated ")

	return minMTU
}

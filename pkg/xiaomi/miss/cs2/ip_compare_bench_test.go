package cs2

import (
	"net"
	"testing"
)

var ipEqualSink bool

func mustParseIP(b *testing.B, s string) net.IP {
	ip := net.ParseIP(s)
	if ip == nil {
		b.Fatalf("failed to parse IP: %s", s)
	}
	return ip
}

func BenchmarkIPCompareStringIPv4(b *testing.B) {
	ip1 := mustParseIP(b, "192.168.1.10").To4()
	ip2 := mustParseIP(b, "192.168.1.10").To4()
	if ip1 == nil || ip2 == nil {
		b.Fatal("expected IPv4")
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ipEqualSink = string(ip1) == string(ip2)
	}
}

func BenchmarkIPCompareEqualIPv4(b *testing.B) {
	ip1 := mustParseIP(b, "192.168.1.10").To4()
	ip2 := mustParseIP(b, "192.168.1.10").To4()
	if ip1 == nil || ip2 == nil {
		b.Fatal("expected IPv4")
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ipEqualSink = ip1.Equal(ip2)
	}
}

func BenchmarkIPCompareStringIPv4Unequal(b *testing.B) {
	ip1 := mustParseIP(b, "192.168.1.10").To4()
	ip2 := mustParseIP(b, "192.168.1.11").To4()
	if ip1 == nil || ip2 == nil {
		b.Fatal("expected IPv4")
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ipEqualSink = string(ip1) == string(ip2)
	}
}

func BenchmarkIPCompareEqualIPv4Unequal(b *testing.B) {
	ip1 := mustParseIP(b, "192.168.1.10").To4()
	ip2 := mustParseIP(b, "192.168.1.11").To4()
	if ip1 == nil || ip2 == nil {
		b.Fatal("expected IPv4")
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ipEqualSink = ip1.Equal(ip2)
	}
}

func BenchmarkIPCompareStringIPv6(b *testing.B) {
	ip1 := mustParseIP(b, "2001:db8::1").To16()
	ip2 := mustParseIP(b, "2001:db8::1").To16()
	if ip1 == nil || ip2 == nil {
		b.Fatal("expected IPv6")
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ipEqualSink = string(ip1) == string(ip2)
	}
}

func BenchmarkIPCompareEqualIPv6(b *testing.B) {
	ip1 := mustParseIP(b, "2001:db8::1").To16()
	ip2 := mustParseIP(b, "2001:db8::1").To16()
	if ip1 == nil || ip2 == nil {
		b.Fatal("expected IPv6")
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ipEqualSink = ip1.Equal(ip2)
	}
}

func BenchmarkIPCompareStringIPv6Unequal(b *testing.B) {
	ip1 := mustParseIP(b, "2001:db8::1").To16()
	ip2 := mustParseIP(b, "2001:db8::2").To16()
	if ip1 == nil || ip2 == nil {
		b.Fatal("expected IPv6")
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ipEqualSink = string(ip1) == string(ip2)
	}
}

func BenchmarkIPCompareEqualIPv6Unequal(b *testing.B) {
	ip1 := mustParseIP(b, "2001:db8::1").To16()
	ip2 := mustParseIP(b, "2001:db8::2").To16()
	if ip1 == nil || ip2 == nil {
		b.Fatal("expected IPv6")
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ipEqualSink = ip1.Equal(ip2)
	}
}

func BenchmarkIPCompareStringIPv4MappedIPv6(b *testing.B) {
	ip1 := mustParseIP(b, "::ffff:192.168.1.10").To16()
	ip2 := mustParseIP(b, "192.168.1.10").To16()
	if ip1 == nil || ip2 == nil {
		b.Fatal("expected IPv6")
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ipEqualSink = string(ip1) == string(ip2)
	}
}

func BenchmarkIPCompareEqualIPv4MappedIPv6(b *testing.B) {
	ip1 := mustParseIP(b, "::ffff:192.168.1.10").To16()
	ip2 := mustParseIP(b, "192.168.1.10").To16()
	if ip1 == nil || ip2 == nil {
		b.Fatal("expected IPv6")
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ipEqualSink = ip1.Equal(ip2)
	}
}

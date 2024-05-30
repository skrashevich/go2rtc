package core

import (
	"fmt"
	"math"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMaxCPUThreads(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		{
			name: "ExpectPositive",
			want: int(math.Round(math.Abs(float64(runtime.NumCPU())))) - 1,
		},
		{
			name: "CompareWithGOMAXPROCS",
			want: runtime.GOMAXPROCS(0) - 1, // This may not always equal NumCPU() if GOMAXPROCS has been set to a specific value.
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaxCPUThreads(1); got != tt.want {
				t.Errorf("NumCPU() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBetween(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		sub1     string
		sub2     string
		expected string
	}{
		{"Basic case", "hello [world]!", "[", "]", "world"},
		{"No sub1", "hello world!", "[", "]", ""},
		{"No sub2", "hello [world!", "[", "]", ""},
		{"Empty string", "", "[", "]", ""},
		{"Sub1 and Sub2 are the same", "hello [world[!", "[", "[", "world"},
		{"Multiple sub1", "hello [world] and [universe]!", "[", "]", "world"},
		{"Sub1 after sub2", "hello ]world[!", "]", "[", "world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Between(tt.s, tt.sub1, tt.sub2)
			if result != tt.expected {
				t.Errorf("Between(%q, %q, %q) = %q; want %q", tt.s, tt.sub1, tt.sub2, result, tt.expected)
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	type args struct {
		v1 string
		v2 string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "equal versions",
			args: args{v1: "1.0.0", v2: "1.0.0"},
			want: 0,
		},
		{
			name: "v1 greater than v2",
			args: args{v1: "1.0.1", v2: "1.0.0"},
			want: 1,
		},
		{
			name: "v1 less than v2",
			args: args{v1: "1.0.0", v2: "1.0.1"},
			want: -1,
		},
		{
			name: "v1 greater with pre-release",
			args: args{v1: "1.0.1-alpha", v2: "1.0.1-beta"},
			want: -1,
		},
		{
			name: "v1 less with different major",
			args: args{v1: "1.2.3", v2: "2.1.1"},
			want: -1,
		},
		{
			name: "v1 greater with different minor",
			args: args{v1: "1.3.0", v2: "1.2.9"},
			want: 1,
		},
		{
			name: "btbn-ffmpeg ebobo version format",
			args: args{v1: "n7.0-7-gd38bf5e08e-20240411", v2: "6.1.1"},
			want: 1,
		},
		{
			name: "btbn-ffmpeg ebobo version format 2",
			args: args{v1: "n7.0-7-gd38bf5e08e-20240411", v2: "7.1"},
			want: -1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CompareVersions(tt.args.v1, tt.args.v2); got != tt.want {
				t.Errorf("CompareVersions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkBetween(b *testing.B) {
	testCases := []struct {
		name string
		s    string
		sub1 string
		sub2 string
	}{
		{"Basic case", "hello [world]!", "[", "]"},
		{"No sub1", "hello world!", "[", "]"},
		{"No sub2", "hello [world!", "[", "]"},
		{"Empty string", "", "[", "]"},
		{"Sub1 and Sub2 are the same", "hello [world[!", "[", "["},
		{"Multiple sub1", "hello [world] and [universe]!", "[", "]"},
		{"Sub1 after sub2", "hello ]world[!", "]", "["},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				Between(tc.s, tc.sub1, tc.sub2)
			}
		})
	}
}

func TestGetRAMUsage(t *testing.T) {
	vMemStat, err := GetRAMUsage()
	require.NoError(t, err)
	require.NotNil(t, vMemStat)
}

func TestGetCPUUsage(t *testing.T) {
	// Short interval to speed up tests; adjust based on your needs
	interval := 100 * time.Millisecond

	avgCPUUsage, err := GetCPUUsage(interval)
	require.NoError(t, err, "GetCPUUsage should not return an error")
	require.GreaterOrEqual(t, avgCPUUsage, 0.0, "Average CPU usage should be >= 0%")
	require.LessOrEqual(t, avgCPUUsage, 100.0, "Average CPU usage should be <= 100%")
}

func TestGetHostInfo(t *testing.T) {
	hostInfo, err := GetHostInfo()

	require.NoError(t, err, "GetHostInfo should not return an error")
	require.NotEmpty(t, hostInfo.Platform, "Platform should not be empty")
	require.NotEmpty(t, hostInfo.Family, "Family should not be empty")
	require.NotEmpty(t, hostInfo.Version, "Version should not be empty")
}

func BenchmarkRandString(b *testing.B) {
	sizes := []int{8, 16, 32, 64, 128}
	bases := []byte{10, 16, 36, 64, 0}

	for _, size := range sizes {
		for _, base := range bases {
			b.Run(fmt.Sprintf("Size%d_Base%d", size, base), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					RandString(size, base)
				}
			})
		}
	}
}

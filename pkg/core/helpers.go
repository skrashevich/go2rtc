package core

import (
	"crypto/rand"
	"fmt"
	"unicode"

	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/unascribed/FlexVer/go/flexver"
)

const (
	BufferSize      = 64 * 1024 // 64K
	ConnDialTimeout = time.Second * 3
	ConnDeadline    = time.Second * 5
	ProbeTimeout    = time.Second * 3
)

// Now90000 - timestamp for Video (clock rate = 90000 samples per second)
func Now90000() uint32 {
	return uint32(time.Duration(time.Now().UnixNano()) * 90000 / time.Second)
}

const symbols = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ-_"

// RandString base10 - numbers, base16 - hex, base36 - digits+letters
// base64 - URL safe symbols, base0 - crypto random
func RandString(size, base byte) string {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	if base == 0 {
		return string(b)
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

	if i = strings.Index(s, sub2); i >= 0 {
		return s[:i]
	}

	return s
}

func Atoi(s string) (i int) {
	if s != "" {
		i, _ = strconv.Atoi(s)
	}
	return
}

func Assert(ok bool) {
	if !ok {
		_, file, line, _ := runtime.Caller(1)
		panic(file + ":" + strconv.Itoa(line))
	}
}

func Caller() string {
	_, file, line, _ := runtime.Caller(1)
	return file + ":" + strconv.Itoa(line)
}

// MaxCPUThreads calculates the maximum number of CPU threads available for use,
// taking into account a specified number of cores to reserve.
//
// The function determines the total number of CPU cores available on the system using runtime.NumCPU()
// and subtracts the number of reservedCores from this total. This calculation is intended to allow
// applications to reserve a certain number of cores for critical tasks, while using the remaining
// cores for other operations.
//
// Parameters:
// - reservedCores: An int specifying the number of CPU cores to reserve.
//
// Returns:
//   - An int representing the maximum number of CPU threads that can be used after reserving the specified
//     number of cores. This function ensures that at least one thread is always available, so it returns
//     a minimum of 1, even if the number of reservedCores equals or exceeds the total number of CPU cores.
//
// Example usage:
//
//	maxThreads := MaxCPUThreads(2)
//	fmt.Printf("Maximum available CPU threads: %d\n", maxThreads)
//
// Note: It's important to consider the workload and performance characteristics of your application
// when deciding how many cores to reserve. Reserving too many cores could lead to underutilization
// of system resources, while reserving too few could impact the performance of critical tasks.
func MaxCPUThreads(reservedCores int) int {
	numCPU := runtime.NumCPU()
	maxThreads := numCPU - reservedCores
	if maxThreads < 1 {
		return 1 // Ensure at least one thread is always available
	}
	return maxThreads
}

// CompareVersions compares two version strings, v1 and v2, after optionally removing a leading letter from each.
// The comparison is performed using the flexver.Compare function. If the first character of either version string
// is a letter, that character is removed before comparison. This function is useful for comparing version strings
// where a leading character might indicate a special version type or pre-release status that should not affect
// the numerical version comparison.
//
// The function returns an integer indicating the relationship between the two versions:
// - 0 if v1 == v2,
// - -1 if v1 < v2,
// - 1 if v1 > v2.
//
// Parameters:
//
//	v1 (string): The first version string to compare.
//	v2 (string): The second version string to compare.
//
// Returns:
//
//	int: An integer indicating the result of the comparison (-1, 0, 1).
//
// Example:
//
//	result := CompareVersions("a1.0", "1.2")
//	// result will be -1 since "1.0" is considered less than "1.2"
func CompareVersions(v1, v2 string) int {
	if len(v1) > 0 && unicode.IsLetter(rune(v1[0])) {
		v1 = v1[1:]
	}
	if len(v2) > 0 && unicode.IsLetter(rune(v2[0])) {
		v2 = v2[1:]
	}
	result, err := flexver.CompareError(v1, v2)
	if err != nil {
		return -1
	}
	if result < 0 {
		return -1
	} else if result > 0 {
		return 1
	}
	return 0
}

// GetCPUUsage calculates the CPU usage percentage over a specified interval.
// It returns the average CPU usage as a float64 and any error encountered.
//
// The function works by first fetching the CPU usage percentage before the sleep interval,
// then calculating the average CPU usage over that interval. This is useful for monitoring
// or logging system performance metrics.
//
// Parameters:
// - interval: A time.Duration value specifying how long to measure CPU usage for.
//
// Returns:
// - A float64 representing the average CPU usage percentage over the interval.
// - An error if there was an issue fetching the CPU usage data.
func GetCPUUsage(interval time.Duration) (float64, error) {
	percentages, err := cpu.Percent(interval, false)
	if err != nil {
		return 0, err
	}

	if len(percentages) == 0 {
		return 0, fmt.Errorf("no CPU usage data available")
	}

	var total float64
	for _, percent := range percentages {
		total += percent
	}
	avgCPUUsage := total / float64(len(percentages))

	return avgCPUUsage, nil
}

// GetRAMUsage fetches the current virtual memory statistics.
// It returns a pointer to a VirtualMemoryStat struct containing detailed memory usage stats
// and any error encountered.
//
// This function is useful for retrieving comprehensive memory usage data, such as total and available RAM,
// used and free amounts, and various other metrics related to system memory performance.
//
// Returns:
// - A pointer to a mem.VirtualMemoryStat struct containing the memory usage statistics.
// - An error if there was an issue fetching the memory usage data.
func GetRAMUsage() (*mem.VirtualMemoryStat, error) {
	return mem.VirtualMemory()
}

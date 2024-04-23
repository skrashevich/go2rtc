/*
 * To the extent possible under law, the author has dedicated all copyright
 * and related and neighboring rights to this software to the public domain
 * worldwide. This software is distributed without any warranty.
 *
 * See <http://creativecommons.org/publicdomain/zero/1.0/>
 */

package core

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

var ENABLED_TESTS = []string{
	// Basic numeric ordering (lexical string sort fails these)
	"10 > 2",
	"100 > 10",

	// Trivial common numerics
	"1.0 < 1.1",
	"1.0 < 1.0.1",
	"1.1 > 1.0.1",

	// SemVer compatibility
	"1.5 > 1.5-pre1",
	"1.5 = 1.5+foobar",

	// SemVer incompatibility
	"1.5 < 1.5-2",
	"1.5-pre10 > 1.5-pre2",

	// Empty strings
	" = ",
	"1 > ",
	" < 1",

	// Check boundary between textual and prerelease
	"a-a < a",

	// Check boundary between textual and appendix
	"a+a = a",

	// Dash is included in prerelease comparison
	"a0-a < a0=a",

	// Pre-releases must contain only non-digit
	"1.16.5-10 > 1.16.5",

	// Pre-releases can have multiple dashes
	"-a- > -a!",

	// Misc
	"b1.7.3 > a1.2.6",
	"b1.2.6 > a1.7.3",
	"a1.1.2 < a1.1.2_01",
	"1.16.5-0.00.5 > 1.14.2-1.3.7",
	"1.0.0 < 1.0.0_01",
	"1.0.1 > 1.0.0_01",
	"1.0.0_01 < 1.0.1",
	"0.17.1-beta.1 < 0.17.1",
	"0.17.1-beta.1 < 0.17.1-beta.2",
	"1.4.5_01 = 1.4.5_01+fabric-1.17",
	"1.4.5_01 = 1.4.5_01+fabric-1.17+ohgod",
	"14w16a < 18w40b",
	"18w40a < 18w40b",
	"1.4.5_01+fabric-1.17 < 18w40b",
	"13w02a < c0.3.0_01",
	"0.6.0-1.18.x < 0.9.beta-1.18.x",

	// removeLeadingZeroes
	"0000.0.0 = 0.0.0",
	"0000.00.0 = 0.00.0",
	"0.0.0 = 0.00.0000",

	// General leading zeroes
	"1.0.01 = 1.0.1",
	"1.0.0001 = 1.0.01",

	// Too large for a 64-bit integer or double
	"36893488147419103232 < 36893488147419103233",
}

const (
	opLT = -1
	opEQ = 0
	opGT = 1
)

func RunCompare(t *testing.T, lefthand string, righthand string, ordering int) {
	res := Compare(lefthand, righthand)
	if (ordering == opLT && !(res < 0)) ||
		(ordering == opEQ && !(res == 0)) ||
		(ordering == opGT && !(res > 0)) {
		t.Errorf("Compare returned %v", res)
	}

	res2, err := CompareError(lefthand, righthand)
	if err != nil {
		t.Fatalf("CompareError returned an unexpected error: %v", err)
	}

	if res != res2 {
		t.Error("CompareError did not give the same result as Compare")
	}
}

func RunLess(t *testing.T, lefthand string, righthand string, ordering int) {
	res := Less(lefthand, righthand)
	if ordering == opLT {
		if !res {
			t.Error("Less incorrectly returned false")
		}
	} else {
		if res {
			t.Error("Less incorrectly returned true")
		}
	}

	res2, err := LessError(lefthand, righthand)
	if err != nil {
		t.Fatalf("LessError returned an unexpected error: %v", err)
	}

	if res != res2 {
		t.Error("LessError did not give the same result as Less")
	}
}

func RunEqual(t *testing.T, lefthand string, righthand string, ordering int) {
	res := Equal(lefthand, righthand)
	if ordering == opEQ {
		if !res {
			t.Error("Equal incorrectly returned false")
		}
	} else {
		if res {
			t.Error("Equal incorrectly returned true")
		}
	}

	res2, err := EqualError(lefthand, righthand)
	if err != nil {
		t.Fatalf("EqualError returned an unexpected error: %v", err)
	}

	if res != res2 {
		t.Error("EqualError did not give the same result as Equal")
	}
}

func TestStandardized(t *testing.T) {
	for _, line := range ENABLED_TESTS {
		if len(line) == 0 {
			continue
		}

		split := strings.Split(line, " ")
		if len(split) != 3 {
			t.Fatal("Line formatted incorrectly, expected 2 spaces: " + line)
		}

		ord := 0
		switch split[1] {
		case "<":
			ord = opLT
		case "=":
			ord = opEQ
		case ">":
			ord = opGT
		}

		lefthand := split[0]
		righthand := split[2]

		t.Run(line, func(t *testing.T) {
			t.Run("Compare", func(t *testing.T) {
				RunCompare(t, lefthand, righthand, ord)
			})

			t.Run("Less", func(t *testing.T) {
				RunLess(t, lefthand, righthand, ord)
			})

			t.Run("Equal", func(t *testing.T) {
				RunEqual(t, lefthand, righthand, ord)
			})
		})

	}
}

func TestInvalid(t *testing.T) {
	// Run through some invalid inputs and fail the test if it didn't return an error
	_, err := CompareError("\xc3\x28", "")
	if err == nil {
		t.Fatal()
	}
	_, err = LessError("\xc3\x28", "")
	if err == nil {
		t.Fatal()
	}
	_, err = EqualError("\xc3\x28", "")
	if err == nil {
		t.Fatal()
	}
	_, err = CompareError("", "\xc3\x28")
	if err == nil {
		t.Fatal()
	}
	_, err = LessError("", "\xc3\x28")
	if err == nil {
		t.Fatal()
	}
	_, err = EqualError("", "\xc3\x28")
	if err == nil {
		t.Fatal()
	}
}

func TestInvalidPanic(t *testing.T) {
	// Ensures that the AssertPanic function is working, as Go doesn't have any built-in way to assert a function panics
	if !DetectPanic(func() { panic("Test") }) || DetectPanic(func() {}) {
		t.Fatal()
	}

	// Run through some invalid inputs and fail the test if it doesn't panic
	if !DetectPanic(func() { Compare("\xc3\x28", "") }) {
		t.Fatal()
	}

	if !DetectPanic(func() { Less("\xc3\x28", "") }) {
		t.Fatal()
	}

	if !DetectPanic(func() { Equal("\xc3\x28", "") }) {
		t.Fatal()
	}

	if !DetectPanic(func() { Compare("", "\xc3\x28") }) {
		t.Fatal()
	}

	if !DetectPanic(func() { Less("", "\xc3\x28") }) {
		t.Fatal()
	}

	if !DetectPanic(func() { Equal("", "\xc3\x28") }) {
		t.Fatal()
	}

}

// Will return true if the given function panics and false if it returned correctly
func DetectPanic(f func()) (ret bool) {
	defer func() {
		if r := recover(); r != nil {
			ret = true
		}
	}()

	f()

	return false // If we reached this statement, we didn't panic
}

func TestBasicSort(t *testing.T) {
	var input = []string{
		"0.17.2", "0.1.0", "1.0.0", "1000", "0.17.2-pre.1", "1.16.5+pre",
	}
	var expect = []string{
		"0.1.0", "0.17.2-pre.1", "0.17.2", "1.0.0", "1.16.5+pre", "1000",
	}
	VersionSlice(input).Sort()
	if !reflect.DeepEqual(input, expect) {
		t.Fatalf("Failed to sort strings: got %v (expected %v)", input, expect)
	}
}

func ExampleCompare() {
	fmt.Println(Compare("1.0.1", "1.0.3"), Compare("10.0.0", "1.0.1"))
	// Output: -2 1
}

func ExampleLess() {
	fmt.Println(Less("10.0.0", "1.0.1"), Less("10.0.0", "10.1.0.2"))
	// Output: false true
}

func ExampleEqual() {
	fmt.Println(Equal("1.0.0+fluffy", "1.0.0"), Equal("1.0.0", "1.0.0-pre.1"))
	// Output: true false
}

func ExampleVersionSlice_Sort() {
	// Type alias, works identically to []string
	versions := VersionSlice{"100", "1.0.2", "0.1.2", "0.3.4-pre"}
	// or VersionSlice([]string{ ... })
	versions.Sort()
	fmt.Println(versions)
	// Output: [0.1.2 0.3.4-pre 1.0.2 100]
}

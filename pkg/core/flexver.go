/*
 * To the extent possible under law, the author has dedicated all copyright
 * and related and neighboring rights to this software to the public domain
 * worldwide. This software is distributed without any warranty.
 *
 * See <http://creativecommons.org/publicdomain/zero/1.0/>
 */

// Package flexver implements FlexVer, a SemVer-compatible intuitive comparator for free-form versioning strings as
// seen in the wild. It's designed to sort versions like people do, rather than attempting to force
// conformance to a rigid and limited standard. As such, it imposes no restrictions. Comparing two
// versions with differing formats will likely produce nonsensical results (garbage in, garbage out),
// but best effort is made to correct for basic structural changes, and versions of differing length
// will be parsed in a logical fashion.
//
// See the specification at https://github.com/unascribed/FlexVer/blob/trunk/SPEC.md
package core

import (
	"errors"
	"sort"
	"unicode/utf8"
)

type segmentType int

const (
	textSegment segmentType = iota
	numberSegment
	preReleaseSegment
	nullSegment
)

var nullComponent = segment{kind: nullSegment, data: nil}

type segment struct {
	kind segmentType
	data []rune
}

func signum(v int) int32 {
	if v > 0 {
		return 1
	} else if v == 0 {
		return 0
	} else {
		return -1
	}
}

// compare determines if this segment is less than, equal to or greater than the given segment.
// A negative, zero or positive integer respectively indicates this result.
func (compA segment) compare(compB segment) int32 {
	// If both components are numeric, compare numerically (codepoint-wise)
	if compA.kind == numberSegment && compB.kind == numberSegment {
		var i, j int
		// Ignore leading zeroes
		for i < len(compA.data) && compA.data[i] == '0' {
			i++
		}
		for j < len(compB.data) && compB.data[j] == '0' {
			j++
		}
		aLen := len(compA.data) - i
		if aLen != len(compB.data)-j {
			// Lengths differ; compare by length - signum used to prevent overflow
			return signum(aLen - (len(compB.data) - j))
		}
		// Compare by digits
		for ; i < aLen; i, j = i+1, j+1 {
			if compA.data[i] != compB.data[j] {
				// Converting to digits is unnecessary (we're subtracting anyway)
				return compA.data[i] - compB.data[j]
			}
		}
	}
	// One or both are null
	if compA.kind == nullSegment {
		if compB.kind == preReleaseSegment {
			return 1
		}
		return -1
	} else if compB.kind == nullSegment {
		if compA.kind == preReleaseSegment {
			return -1
		}
		return 1
	}
	// Textual comparison (differing type, or both textual/pre-release)
	minLen := len(compA.data)
	if len(compB.data) < minLen {
		minLen = len(compB.data)
	}
	for i := 0; i < minLen; i++ {
		a := compA.data[i]
		b := compB.data[i]
		if a != b {
			// Compare by rune
			return a - b
		}
	}
	// Compare by length - signum used to prevent overflow
	return signum(len(compA.data) - len(compB.data))
}

// ErrInvalidUTF8 is returned from the *Error functions when a passed string is invalid UTF-8.
var ErrInvalidUTF8 = errors.New("version string is invalid utf-8")

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func splitComponents(input string) ([]segment, error) {
	var segments []segment
	pos := 0 // Tracks byte position in the input string.

	for pos < len(input) {
		var currentRunes []rune
		currentType := textSegment

		if input[pos] == '+' {
			break // Stop processing on '+' sign.
		} else if input[pos] == '-' {
			currentRunes = append(currentRunes, '-')
			pos++
			if pos < len(input) && !isDigit(rune(input[pos])) {
				currentType = preReleaseSegment
			}
		} else if isDigit(rune(input[pos])) {
			currentType = numberSegment
		}

		for pos < len(input) {
			runeValue, size := utf8.DecodeRuneInString(input[pos:])
			if runeValue == utf8.RuneError {
				return segments, ErrInvalidUTF8
			}

			if (isDigit(runeValue) != (currentType == numberSegment)) ||
				(runeValue == '-' && currentType != preReleaseSegment) ||
				runeValue == '+' {
				break // Do not include this rune in the current segment.
			}

			currentRunes = append(currentRunes, runeValue)
			pos += size // Move forward by the size of the decoded rune.
		}

		if len(currentRunes) > 0 {
			segments = append(segments, segment{kind: currentType, data: currentRunes})
		}
	}

	return segments, nil
}
func CompareError(a, b string) (int32, error) {
	// Decompose input strings, get length of the longest version
	aDecomp, err := splitComponents(a)
	if err != nil {
		return 0, err
	}
	bDecomp, err := splitComponents(b)
	if err != nil {
		return 0, err
	}
	maxLen := len(aDecomp)
	if len(bDecomp) > maxLen {
		maxLen = len(bDecomp)
	}

	// Compare each component; using nullComponent if a string is exhausted
	for i := 0; i < maxLen; i++ {
		var res int32
		if i >= len(aDecomp) {
			res = nullComponent.compare(bDecomp[i])
		} else if i >= len(bDecomp) {
			res = aDecomp[i].compare(nullComponent)
		} else {
			res = aDecomp[i].compare(bDecomp[i])
		}
		if res != 0 {
			return res, nil
		}
	}

	return 0, nil
}

// Compare compares two version numbers according to the FlexVer specification.
// Returns a negative integer, 0 or a positive integer if version a is less than, equal to or greater than version b respectively.
//
// If either input version number is not valid UTF-8, this function panics.
// See [CompareError] for a variant of this function that returns an error instead of panicking.
func Compare(a, b string) int32 {
	r, err := CompareError(a, b)
	if err != nil {
		panic(err)
	}
	return r
}

// Less compares two version numbers according to the FlexVer specification.
// Returns true if version a is less than version b; useful for sorting functions in [sort] and [golang.org/x/exp/slices].
//
// If either input version number is not valid UTF-8, this function panics.
// See [LessError] for a variant of this function that returns an error instead of panicking.
func Less(a, b string) bool {
	return Compare(a, b) < 0
}

// LessError compares two version numbers according to the FlexVer specification.
// Returns true if version a is less than version b; useful for sorting functions in [sort] and [golang.org/x/exp/slices].
//
// If either input version number is not valid UTF-8, [ErrInvalidUTF8] is returned.
// See [Less] for a variant of this function that panics instead of returning an error (more convenient if you know the strings are valid UTF-8).
func LessError(a, b string) (bool, error) {
	r, err := CompareError(a, b)
	if err != nil {
		return false, err
	}
	return r < 0, nil
}

// Equal compares two version numbers according to the FlexVer specification.
// Returns true if version a is semantically equal to version b (only differs in appendix).
//
// If either input version number is not valid UTF-8, this function panics.
// See [Equal] for a variant of this function that returns an error instead of panicking.
func Equal(a, b string) bool {
	return Compare(a, b) == 0
}

// EqualError compares two version numbers according to the FlexVer specification.
// Returns true if version a is semantically equal to version b (only differs in appendix).
//
// If either input version number is not valid UTF-8, [ErrInvalidUTF8] is returned.
// See [EqualError] for a variant of this function that panics instead of returning an error (more convenient if you know the strings are valid UTF-8).
func EqualError(a, b string) (bool, error) {
	r, err := CompareError(a, b)
	if err != nil {
		return false, err
	}
	return r == 0, nil
}

// VersionSlice implements [sort.Interface] as a type alias for a slice of version strings, to sort according to the FlexVer specification.
// This can be obtained from an existing []string slice using a type conversion:
//
//	VersionSlice([]string{"1.0.0", "1.0.1"})
//
// This allows in-place sorting of a []string slice, as follows:
//
//	VersionSlice(vers).Sort()
//
// This implementation will panic if invalid UTF-8 strings are encountered.
type VersionSlice []string

func (s VersionSlice) Len() int           { return len(s) }
func (s VersionSlice) Less(i, j int) bool { return Less(s[i], s[j]) }
func (s VersionSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// Sort is a convenience method which calls [sort.Sort] on this slice.
func (s VersionSlice) Sort() {
	sort.Sort(s)
}

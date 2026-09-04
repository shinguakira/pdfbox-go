package cmap

import "errors"

// ErrCodespaceRangeLengths is what a codespace range whose two bounds are
// different lengths is reported with.
//
// Java throws IllegalArgumentException, which is unchecked; the port returns an
// error, because the constructor is reached straight from a parse of a file and
// every caller has an error path already.
var ErrCodespaceRangeLengths = errors.New(
	"cmap: The start and the end values must not have different lengths.")

// CodespaceRange is one run of the codespacerange operator: which byte
// sequences are codes, and how long each is.
//
// Port of org.apache.fontbox.cmap.CodespaceRange.
type CodespaceRange struct {
	start      []int
	end        []int
	codeLength int
}

// NewCodespaceRange returns the range between the two given bounds.
func NewCodespaceRange(startBytes, endBytes []byte) (*CodespaceRange, error) {
	correctedStartBytes := startBytes
	if len(startBytes) != len(endBytes) && len(startBytes) == 1 && startBytes[0] == 0 {
		correctedStartBytes = make([]byte, len(endBytes))
	} else if len(startBytes) != len(endBytes) {
		return nil, ErrCodespaceRangeLengths
	}

	r := &CodespaceRange{
		start:      make([]int, len(correctedStartBytes)),
		end:        make([]int, len(endBytes)),
		codeLength: len(endBytes),
	}
	for i := range correctedStartBytes {
		r.start[i] = int(correctedStartBytes[i]) & 0xFF
		r.end[i] = int(endBytes[i]) & 0xFF
	}
	return r, nil
}

// CodeLength returns how many bytes a code of this range takes.
func (r *CodespaceRange) CodeLength() int { return r.codeLength }

// Matches reports whether the given bytes are a code of this range.
func (r *CodespaceRange) Matches(code []byte) bool {
	return r.IsFullMatch(code, len(code))
}

// IsFullMatch reports whether the first codeLen bytes of code are a code of
// this range.
func (r *CodespaceRange) IsFullMatch(code []byte, codeLen int) bool {
	// code must be the same length as the bounding codes
	if r.codeLength != codeLen {
		return false
	}
	for i := 0; i < r.codeLength; i++ {
		codeAsInt := int(code[i]) & 0xFF
		if codeAsInt < r.start[i] || codeAsInt > r.end[i] {
			return false
		}
	}
	return true
}

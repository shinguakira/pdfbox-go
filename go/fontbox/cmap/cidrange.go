// Package cmap maps the character codes of a composite font onto CIDs and onto
// Unicode.
//
// Port of org.apache.fontbox.cmap.
package cmap

// cidRange is one run of the cidrange operator: a contiguous span of character
// codes mapping onto a contiguous span of CIDs.
//
// Port of org.apache.fontbox.cmap.CIDRange.
type cidRange struct {
	from       int
	to         int
	unicode    int
	codeLength int
}

// newCIDRange returns the range from..to mapping onto unicode upwards, for
// codes of the given length in bytes.
func newCIDRange(from, to, unicode, codeLength int) *cidRange {
	return &cidRange{from: from, to: to, unicode: unicode, codeLength: codeLength}
}

// CodeLength returns how many bytes a code of this range takes.
func (r *cidRange) CodeLength() int { return r.codeLength }

// MapBytes returns what the given code maps onto, or -1 where the range does
// not cover it.
func (r *cidRange) MapBytes(bytes []byte) int {
	if len(bytes) == r.codeLength {
		ch := ToInt(bytes)
		if r.from <= ch && ch <= r.to {
			return r.unicode + (ch - r.from)
		}
	}
	return -1
}

// Map returns what the given code maps onto, or -1.
func (r *cidRange) Map(code, length int) int {
	if length == r.codeLength && r.from <= code && code <= r.to {
		return r.unicode + (code - r.from)
	}
	return -1
}

// Unmap returns which code maps onto the given value, or -1.
func (r *cidRange) Unmap(code int) int {
	if r.unicode <= code && code <= r.unicode+(r.to-r.from) {
		return r.from + (code - r.unicode)
	}
	return -1
}

// Extend grows the range to take in the one starting where this ends, and
// reports whether it did.
func (r *cidRange) Extend(newFrom, newTo, newCid, length int) bool {
	if r.codeLength == length && newFrom == r.to+1 && newCid == r.unicode+r.to-r.from+1 {
		r.to = newTo
		return true
	}
	return false
}

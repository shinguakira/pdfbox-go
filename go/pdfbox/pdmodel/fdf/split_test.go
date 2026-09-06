package fdf

// Written from java.lang.String.split, which both helpers here stand in for.
//
// The rules the port has to reproduce, from Pattern.split with the default
// limit of zero:
//
//   - a leading empty field is kept, so ",1" splits to ["", "1"];
//   - an interior empty field is kept, so "1,,2" splits to ["1", "", "2"];
//   - *trailing* empty fields are all dropped, so "1,2,,," splits to
//     ["1", "2"];
//   - and where the pattern never matched at all the whole input comes back
//     untrimmed, which is why "" splits to [""].
//
// The middle two matter here for the same reason: parseFloats panics on an
// empty field, reproducing Java's NumberFormatException. Dropping an interior
// empty does not merely lose a field, it makes the port *accept* a coordinate
// list Java rejects, and read the remaining numbers into the wrong positions.

import (
	"reflect"
	"testing"
)

// TestSplitJava covers splitJava, which stands in for String.split(",").
func TestSplitJava(t *testing.T) {
	for _, row := range []struct {
		input string
		want  []string
	}{
		{"1,2,3", []string{"1", "2", "3"}},
		{"1,2,3,", []string{"1", "2", "3"}},
		{"1,2,3,,,", []string{"1", "2", "3"}},
		{",1", []string{"", "1"}},
		{"1,,2", []string{"1", "", "2"}},
		{",,", []string{}},
		{"", []string{""}},
	} {
		t.Run(row.input, func(t *testing.T) {
			got := splitJava(row.input, ",")
			if !equalFields(got, row.want) {
				t.Errorf("splitJava(%q) = %q, want %q", row.input, got, row.want)
			}
		})
	}
}

// TestSplitOnCommaOrSemicolon covers splitOnCommaOrSemicolon, which stands in
// for String.split("[,;]") — the split the ink, polygon and polyline
// annotations read their coordinates with.
func TestSplitOnCommaOrSemicolon(t *testing.T) {
	for _, row := range []struct {
		input string
		want  []string
	}{
		{"1,2;3", []string{"1", "2", "3"}},
		{"1,2;3;", []string{"1", "2", "3"}},
		{"1,2;3,;;", []string{"1", "2", "3"}},
		{";1", []string{"", "1"}},
		{",1", []string{"", "1"}},
		// adjacent separators, of either kind and mixed
		{"1,,2", []string{"1", "", "2"}},
		{"1;;2", []string{"1", "", "2"}},
		{"1,;2", []string{"1", "", "2"}},
		{",;", []string{}},
		{"", []string{""}},
	} {
		t.Run(row.input, func(t *testing.T) {
			got := splitOnCommaOrSemicolon(row.input)
			if !equalFields(got, row.want) {
				t.Errorf("splitOnCommaOrSemicolon(%q) = %q, want %q",
					row.input, got, row.want)
			}
		})
	}
}

// equalFields compares two slices of strings, treating a nil slice and an empty
// one as the same thing.
func equalFields(got, want []string) bool {
	if len(got) == 0 && len(want) == 0 {
		return true
	}
	return reflect.DeepEqual(got, want)
}

package util_test

// Port of org.apache.pdfbox.util.StringUtilTest.

import (
	"reflect"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// TestSplitOnSpace is the three testSplitOnSpace_* cases.
//
// The empty-string and only-spaces rows are the ones worth having: they are
// java.lang.String.split's two surprises, an empty input giving one empty
// element and an all-separator input giving none at all.
func TestSplitOnSpace(t *testing.T) {
	for _, row := range []struct {
		name  string
		input string
		want  []string
	}{
		{"happy path", "a b c", []string{"a", "b", "c"}},
		{"an empty string", "", []string{""}},
		{"only spaces", "   ", []string{}},
	} {
		t.Run(row.name, func(t *testing.T) {
			got := util.SplitOnSpace(row.input)
			if len(got) != len(row.want) {
				t.Fatalf("SplitOnSpace(%q) = %q, want %q", row.input, got, row.want)
			}
			if len(got) > 0 && !reflect.DeepEqual(got, row.want) {
				t.Errorf("SplitOnSpace(%q) = %q, want %q", row.input, got, row.want)
			}
		})
	}
}

// TestTokenizeOnSpace is the four testTokenizeOnSpace_* cases: unlike the split
// above, every space comes back as a token of its own.
func TestTokenizeOnSpace(t *testing.T) {
	for _, row := range []struct {
		name  string
		input string
		want  []string
	}{
		{"happy path", "a b c", []string{"a", " ", "b", " ", "c"}},
		{"an empty string", "", []string{""}},
		{"only spaces", "   ", []string{" ", " ", " "}},
		{"spaces around text", "  a  ", []string{" ", " ", "a", " ", " "}},
	} {
		t.Run(row.name, func(t *testing.T) {
			got := util.TokenizeOnSpace(row.input)
			if !reflect.DeepEqual(got, row.want) {
				t.Errorf("TokenizeOnSpace(%q) = %q, want %q", row.input, got, row.want)
			}
		})
	}
}

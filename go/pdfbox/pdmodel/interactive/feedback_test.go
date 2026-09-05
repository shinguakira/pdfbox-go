package interactive

// The tests below pin a defect the slice 8 review feedback found in this
// package. They are not ports: PDFBox has no test for it. They assert what the
// Java does, read off PlainText's constructor and String.split.

import (
	"testing"
)

// TestValueOfOnlyLineBreaksHasNoParagraphs checks a value made of nothing but
// line breaks.
//
// PlainText's constructor is `textValue.replace('\t',' ').split("\\R")`.
// String.split with a limit of zero drops *every* trailing empty result, not
// all but one: Pattern.split builds ["", ""] for "\n" and then strips both, so
// the array is empty and the text has no paragraphs at all. The port stopped
// stripping at one entry, so it kept an empty paragraph, and the constructor
// turns an empty paragraph into a space -- which draws a space where PDFBox
// draws nothing.
func TestValueOfOnlyLineBreaksHasNoParagraphs(t *testing.T) {
	// The fourth is U+2028 LINE SEPARATOR, which Java's \R matches on its own.
	for _, value := range []string{"\n", "\r\n", "\n\n", " ", "\r\r\r"} {
		if got := NewPlainText(value).Paragraphs(); len(got) != 0 {
			t.Errorf("NewPlainText(%q).Paragraphs() = %d, want none", value, len(got))
		}
	}
}

// TestTrailingLineBreaksAreDropped checks the same stripping after real text,
// and the two cases it must not touch: a value with no line break at all, which
// Java answers whole, and a leading break, whose empty first entry is not
// trailing.
func TestTrailingLineBreaksAreDropped(t *testing.T) {
	for _, c := range []struct {
		value string
		want  []string
	}{
		{"a\n", []string{"a"}},
		{"a\n\n\n", []string{"a"}},
		{"a", []string{"a"}},
		{"\na", []string{" ", "a"}},
		{"a\nb", []string{"a", "b"}},
		// A value with no line break in it is answered whole, even where it is
		// only whitespace: Java returns early with the input before it strips.
		{" ", []string{" "}},
	} {
		paragraphs := NewPlainText(c.value).Paragraphs()
		if len(paragraphs) != len(c.want) {
			t.Errorf("NewPlainText(%q) = %d paragraphs, want %d",
				c.value, len(paragraphs), len(c.want))
			continue
		}
		for i, want := range c.want {
			// Acrobat prints a space for an empty paragraph, so the
			// constructor stores " " for one.
			if got := paragraphs[i].Text(); got != want {
				t.Errorf("NewPlainText(%q)[%d] = %q, want %q", c.value, i, got, want)
			}
		}
	}
}

// TestEmptyValueIsOneEmptyParagraph checks the branch before the split: Java
// guards textValue.isEmpty() and adds a single "" paragraph, which is not the
// same as what the split would answer for it.
func TestEmptyValueIsOneEmptyParagraph(t *testing.T) {
	paragraphs := NewPlainText("").Paragraphs()
	if len(paragraphs) != 1 {
		t.Fatalf(`NewPlainText("") = %d paragraphs, want 1`, len(paragraphs))
	}
	if got := paragraphs[0].Text(); got != "" {
		t.Errorf(`NewPlainText("")[0] = %q, want ""`, got)
	}
}

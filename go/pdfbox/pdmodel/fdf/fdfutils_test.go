package fdf

// Port of org.apache.pdfbox.pdmodel.fdf.FDFUtilsTest.
//
// An internal test, because escapeXML10 is package-private in Java and
// unexported here for the same reason.

import (
	"strings"
	"testing"
)

// TestEscapeXML10 is all thirteen cases of FDFUtilsTest.
func TestEscapeXML10(t *testing.T) {
	for _, row := range []struct {
		name  string
		input string
		want  string
	}{
		{"plain ASCII is unchanged", "Hello World 123", "Hello World 123"},
		{"an empty string returns an empty string", "", ""},
		{"escapes the less-than sign", "<", "&lt;"},
		{"escapes the greater-than sign", ">", "&gt;"},
		{"escapes the ampersand", "&", "&amp;"},
		{"escapes the double quote", `"`, "&quot;"},
		{"escapes the single quote", "'", "&apos;"},
		{
			name:  "escapes all the special characters in one string",
			input: `<tag attr="value" other='x'>&</tag>`,
			want:  "&lt;tag attr=&quot;value&quot; other=&apos;x&apos;&gt;&amp;&lt;/tag&gt;",
		},
		{
			// Tab (0x9), line feed (0xA) and carriage return (0xD) are
			// explicitly legal XML 1.0 characters and are not part of the
			// escaped set.
			name:  "legal whitespace control characters pass through unescaped",
			input: "line1\tline2\nline3\rline4",
			want:  "line1\tline2\nline3\rline4",
		},
		{
			// 'e' with an acute is U+00E9, 233 decimal
			name:  "a non-ASCII BMP character is escaped as a numeric reference",
			input: "café",
			want:  "caf&#233;",
		},
		{
			// U+00E9 = 233, U+00E8 = 232
			name:  "multiple non-ASCII characters are each escaped",
			input: "éè",
			want:  "&#233;&#232;",
		},
		{
			// 0x0B, vertical tab, is not a legal XML 1.0 character and must not
			// appear unescaped in the output
			name:  "an illegal control character is not passed through raw",
			input: "a\vb",
			want:  "a�b",
		},
		{
			// U+1F600 GRINNING FACE is a surrogate pair in Java. It must be
			// escaped as one reference to its code point, not two references to
			// the individual, illegal, surrogate values.
			name:  "a supplementary character produces a single valid reference",
			input: "\U0001F600",
			want:  "&#128512;",
		},
	} {
		t.Run(row.name, func(t *testing.T) {
			got := escapeXML10(row.input)
			if got != row.want {
				t.Errorf("escapeXML10(%q) = %q, want %q", row.input, got, row.want)
			}
			if strings.ContainsRune(got, '\v') {
				t.Error("an illegal control character appears raw in the output")
			}
		})
	}
}

package font_test

// The whole embedded font, checked entry by entry against the running Java.
//
// TestSubsetBytesMatchJava checks the font program that TTFSubsetter produces.
// This checks what this branch wraps around it: the tagged /BaseFont, the /W
// widths array with its run compression, /CIDToGIDMap, /CIDSet and /ToUnicode,
// for both a subsetted font and a whole one.
//
// Every wanted value was printed by the running Java. PDFBox compiles with
// javac once PublicKeySecurityHandler -- which needs a Bouncy Castle jar this
// environment has no copy of and no network to fetch -- is replaced by a
// stand-in that refuses; it has nothing to do with fonts. A driver then wrote
// the same two documents Java's validateCIDFontType2 writes, read them back
// with Loader, and printed each entry.
//
// The /W comparison is over a flattened rendering rather than the COS objects,
// because the port's writer puts each inner array in an indirect object where
// Java writes it inline. That is the writer's choice and not the embedder's;
// the numbers and their nesting are what this case is about.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
)

// TestEmbeddedFontMatchesJava is the branch's D8: the bytes, not the structure.
func TestEmbeddedFontMatchesJava(t *testing.T) {
	const message = "Unicode русский язык Tiếng Việt"

	for _, want := range []struct {
		useSubset bool

		baseFont      string
		programLength int
		programSHA    string

		widthsHead   string
		widthsLength int
		widthsSHA    string

		cidToGIDIdentity bool
		cidToGIDLength   int
		cidToGIDSHA      string

		cidSetLength int
		cidSetSHA    string

		toUnicodeLength int
		toUnicodeSHA    string
	}{
		{
			useSubset:        false,
			baseFont:         "LiberationSans",
			programLength:    410712,
			programSHA:       "76d04c18ea243f42",
			widthsHead:       "[ 0 [ 750 0 ] 2 4 278 5 [ 355 ] 6 7 556 8 [ 889 667 191 ] 11 12 333 13 [ 389 584 278 333 ] 17 18 278 19 28 556 29 30 278 31 33 584 34 [ 556 1015 ] 36 37 667 38 ",
			widthsLength:     10314,
			widthsSHA:        "b23c47581658474f",
			cidToGIDIdentity: true,
			toUnicodeLength:  3238,
			toUnicodeSHA:     "4362bc5b72416c8b",
		},
		{
			useSubset:       true,
			baseFont:        "AALHKC+LiberationSans",
			programLength:   8332,
			programSHA:      "72fcedd104476342",
			widthsHead:      "[ 0 [ 750 ] 3 [ 278 ] 55 [ 611 722 667 ] 70 [ 500 556 556 ] 74 [ 556 ] 76 [ 222 ] 81 [ 556 556 556 ] 87 [ 278 ] 92 [ 500 ] 648 [ 333 ] 1000 [ 458 559 559 438 ] ",
			widthsLength:    278,
			widthsSHA:       "4e342ba605688187",
			cidToGIDLength:  5196,
			cidToGIDSHA:     "33adda0bc2c7a032",
			cidSetLength:    325,
			cidSetSHA:       "4c58ea3e2bc8a0d5",
			toUnicodeLength: 668,
			toUnicodeSHA:    "66e2018313b9db21",
		},
	} {
		name := "wholeFont"
		if want.useSubset {
			name = "subset"
		}
		t.Run(name, func(t *testing.T) {
			document := pdmodel.NewPDDocument()
			page := pdmodel.NewPDPageOfSize(common.A4)
			document.AddPage(page)
			embedded, err := font.LoadPDType0FontSubset(document,
				openFont(t, liberationSans), want.useSubset)
			if err != nil {
				t.Fatalf("LoadPDType0FontSubset: %v", err)
			}
			writeText(t, document, page, embedded, 12, 50, 600, message)
			var out bytes.Buffer
			if err := document.Save(&out); err != nil {
				t.Fatalf("Save: %v", err)
			}
			if err := document.Close(); err != nil {
				t.Fatalf("Close: %v", err)
			}

			reloaded, type0, descendant := reloadDescendant(t, out.Bytes())
			defer reloaded.Close()

			if got := type0.Name(); got != want.baseFont {
				t.Errorf("/BaseFont = %q, want Java's %q", got, want.baseFont)
			}

			// /FontFile2, and the /Length1 that has to agree with it.
			program := type0.DescendantFont().FontDescriptor().FontFile2()
			if program == nil {
				t.Fatal("the descriptor has no /FontFile2")
			}
			programBytes, err := program.ToByteArray()
			if err != nil {
				t.Fatalf("reading /FontFile2: %v", err)
			}
			checkBytes(t, "/FontFile2", programBytes, want.programLength, want.programSHA)
			if got := program.Stream().GetLong(cos.Length1); got != int64(want.programLength) {
				t.Errorf("/Length1 = %d, want %d", got, want.programLength)
			}

			// /W, flattened so that an indirect inner array reads the same as
			// an inline one.
			widths := flattenCOS(descendant.GetDictionaryObject(cos.W))
			if !strings.HasPrefix(widths, want.widthsHead) {
				t.Errorf("/W starts\n  %.180s\nwant Java's\n  %.180s", widths, want.widthsHead)
			}
			checkBytes(t, "/W", []byte(widths), want.widthsLength, want.widthsSHA)

			// /CIDToGIDMap: /Identity for a whole font, a stream for a subset.
			cidToGID := descendant.GetDictionaryObject(cos.CIDToGIDMap)
			if want.cidToGIDIdentity {
				if got, _ := cidToGID.(*cos.Name); got != cos.Identity {
					t.Errorf("/CIDToGIDMap = %v, want /Identity", cidToGID)
				}
			} else {
				stream, ok := cidToGID.(*cos.Stream)
				if !ok {
					t.Fatalf("/CIDToGIDMap = %v, want a stream", cidToGID)
				}
				checkBytes(t, "/CIDToGIDMap", streamBytes(t, stream),
					want.cidToGIDLength, want.cidToGIDSHA)
			}

			// /CIDSet, which a whole font does not get.
			cidSet := type0.DescendantFont().FontDescriptor().CIDSet()
			if want.cidSetLength == 0 {
				if cidSet != nil {
					t.Error("a font that is not subset has a /CIDSet")
				}
			} else {
				if cidSet == nil {
					t.Fatal("the descriptor has no /CIDSet")
				}
				bits, err := cidSet.ToByteArray()
				if err != nil {
					t.Fatalf("reading /CIDSet: %v", err)
				}
				checkBytes(t, "/CIDSet", bits, want.cidSetLength, want.cidSetSHA)
			}

			// /ToUnicode, which buildToUnicodeCMap writes.
			parent, _ := type0.COSObject().(*cos.Dictionary)
			toUnicode, ok := parent.GetDictionaryObject(cos.ToUnicode).(*cos.Stream)
			if !ok {
				t.Fatal("the font has no /ToUnicode stream")
			}
			checkBytes(t, "/ToUnicode", streamBytes(t, toUnicode),
				want.toUnicodeLength, want.toUnicodeSHA)
		})
	}
}

// checkBytes compares a length and the first 16 hex digits of a SHA-256, which
// is what the Java driver printed.
func checkBytes(t *testing.T, what string, got []byte, wantLength int, wantSHA string) {
	t.Helper()
	sum := sha256.Sum256(got)
	gotSHA := hex.EncodeToString(sum[:])[:16]
	if len(got) != wantLength || gotSHA != wantSHA {
		t.Errorf("%s is %d bytes, sha %s; Java's is %d bytes, sha %s",
			what, len(got), gotSHA, wantLength, wantSHA)
	}
}

// streamBytes decodes a stream.
func streamBytes(t *testing.T, stream *cos.Stream) []byte {
	t.Helper()
	reader, err := stream.CreateReader()
	if err != nil {
		t.Fatalf("reading a stream: %v", err)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading a stream: %v", err)
	}
	return data
}

// flattenCOS renders an array of numbers and nested arrays, resolving indirect
// objects, in the shape the Java driver printed.
func flattenCOS(base cos.Base) string {
	var sb strings.Builder
	flattenInto(base, &sb)
	return strings.TrimSpace(sb.String())
}

func flattenInto(base cos.Base, sb *strings.Builder) {
	switch value := base.(type) {
	case *cos.Object:
		flattenInto(value.Object(), sb)
	case *cos.Array:
		sb.WriteString("[ ")
		for i := 0; i < value.Size(); i++ {
			flattenInto(value.Get(i), sb)
		}
		sb.WriteString("] ")
	case *cos.Integer:
		fmt.Fprintf(sb, "%d ", value.LongValue())
	case *cos.Float:
		fmt.Fprintf(sb, "%d ", int64(value.FloatValue()))
	default:
		fmt.Fprintf(sb, "%v ", base)
	}
}

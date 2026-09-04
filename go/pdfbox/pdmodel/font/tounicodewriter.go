package font

import (
	"bufio"
	"io"
	"math"
	"slices"
	"strconv"
	"unicode/utf16"
	"unicode/utf8"
)

// toUnicodeEntry is one CID to text mapping, which stands for Java's
// Map.Entry<Integer, String>: the two allow* helpers take entries, and a Go
// test needs a type to hand them.
type toUnicodeEntry struct {
	cid  int
	text string
}

// maxEntriesPerOperator is the maximum number of entries in a bfrange operator.
//
// Port of ToUnicodeWriter.MAX_ENTRIES_PER_OPERATOR.
const maxEntriesPerOperator = 100

// toUnicodeWriter writes the /ToUnicode CMap of an embedded font.
//
// Port of the package-private org.apache.pdfbox.pdmodel.font.ToUnicodeWriter.
type toUnicodeWriter struct {
	// cidToUnicode is Java's TreeMap: the CIDs are written in order, so the
	// port keeps the map and sorts the keys where it walks them.
	cidToUnicode map[int]string
	wMode        int
}

// newToUnicodeWriter creates a new instance.
func newToUnicodeWriter() *toUnicodeWriter {
	return &toUnicodeWriter{cidToUnicode: map[int]string{}, wMode: 0}
}

// setWMode sets the WMode of the CMap.
func (w *toUnicodeWriter) setWMode(wMode int) {
	w.wMode = wMode
}

// add adds the given CID to Unicode mapping.
//
// Java throws IllegalArgumentException, which is unchecked, so the port panics.
func (w *toUnicodeWriter) add(cid int, text string) {
	if cid < 0 || cid > 0xFFFF {
		panic("CID is not valid")
	}
	if text == "" {
		panic("Text is null or empty")
	}
	w.cidToUnicode[cid] = text
}

// writeTo writes the CMap as a "ToUnicode" CMap file.
func (w *toUnicodeWriter) writeTo(out io.Writer) error {
	// Java wraps the stream in an ASCII writer; everything written here is
	// ASCII, so the bytes of the Go string are the same bytes.
	writer := bufio.NewWriter(out)

	writeLine(writer, "/CIDInit /ProcSet findresource begin")
	writeLine(writer, "12 dict begin\n")

	writeLine(writer, "begincmap")
	writeLine(writer, "/CIDSystemInfo")
	writeLine(writer, "<< /Registry (Adobe)")
	writeLine(writer, "/Ordering (UCS)")
	writeLine(writer, "/Supplement 0")
	writeLine(writer, ">> def\n")

	writeLine(writer, "/CMapName /Adobe-Identity-UCS"+" def")
	writeLine(writer, "/CMapType 2 def\n") // 2 = ToUnicode

	if w.wMode != 0 {
		writeLine(writer, "/WMode /"+strconv.Itoa(w.wMode)+" def")
	}

	// ToUnicode always uses 16-bit CIDs
	writeLine(writer, "1 begincodespacerange")
	writeLine(writer, "<0000> <FFFF>")
	writeLine(writer, "endcodespacerange\n")

	// CID -> Unicode mappings, we use ranges to generate a smaller CMap
	var srcFrom, srcTo []int
	var dstString []string

	var prev *toUnicodeEntry

	cids := make([]int, 0, len(w.cidToUnicode))
	for cid := range w.cidToUnicode {
		cids = append(cids, cid)
	}
	slices.Sort(cids)

	for _, cid := range cids {
		next := &toUnicodeEntry{cid: cid, text: w.cidToUnicode[cid]}
		if allowCIDToUnicodeRange(prev, next) {
			// extend range
			srcTo[len(srcTo)-1] = next.cid
		} else {
			// begin range
			srcFrom = append(srcFrom, next.cid)
			srcTo = append(srcTo, next.cid)
			dstString = append(dstString, next.text)
		}
		prev = next
	}

	// limit entries per operator
	batchCount := int(math.Ceil(float64(len(srcFrom)) / float64(maxEntriesPerOperator)))
	for batch := 0; batch < batchCount; batch++ {
		count := maxEntriesPerOperator
		if batch == batchCount-1 {
			count = len(srcFrom) - maxEntriesPerOperator*batch
		}
		writer.WriteString(strconv.Itoa(count) + " beginbfrange\n")
		for j := 0; j < count; j++ {
			index := batch*maxEntriesPerOperator + j
			writer.WriteByte('<')
			writer.WriteString(hexChars(srcFrom[index]))
			writer.WriteString("> ")

			writer.WriteByte('<')
			writer.WriteString(hexChars(srcTo[index]))
			writer.WriteString("> ")

			writer.WriteByte('<')
			writer.WriteString(hexCharsUTF16BE(dstString[index]))
			writer.WriteString(">\n")
		}
		writeLine(writer, "endbfrange\n")
	}

	// footer
	writeLine(writer, "endcmap")
	writeLine(writer, "CMapName currentdict /CMap defineresource pop")
	writeLine(writer, "end")
	writeLine(writer, "end")

	return writer.Flush()
}

// writeLine writes the text and a newline.
//
// Java's BufferedWriter defers its IOException to the flush; a bufio.Writer
// does the same, holding the first error until Flush reports it, so nothing is
// lost by not checking here.
func writeLine(writer *bufio.Writer, text string) {
	writer.WriteString(text)
	writer.WriteByte('\n')
}

// hexChars is org.apache.pdfbox.util.Hex.getChars(short): the 16-bit value as
// four upper-case hexadecimal digits.
func hexChars(value int) string {
	const digits = "0123456789ABCDEF"
	// Java narrows to a short first; the four digits are the low 16 bits.
	v := uint16(value)
	return string([]byte{
		digits[(v>>12)&0x0F], digits[(v>>8)&0x0F],
		digits[(v>>4)&0x0F], digits[v&0x0F],
	})
}

// hexCharsUTF16BE is Hex.getCharsUTF16BE(String): the UTF-16BE code units of
// the string as upper-case hexadecimal digits.
func hexCharsUTF16BE(text string) string {
	const digits = "0123456789ABCDEF"
	units := utf16.Encode([]rune(text))
	out := make([]byte, 0, 4*len(units))
	for _, unit := range units {
		out = append(out,
			digits[(unit>>12)&0x0F], digits[(unit>>8)&0x0F],
			digits[(unit>>4)&0x0F], digits[unit&0x0F])
	}
	return string(out)
}

// allowCIDToUnicodeRange returns true if the CID and Unicode destination string
// are allowed to follow one another according to the Adobe 1.7 specification as
// described in Section 5.9, Example 5.16.
func allowCIDToUnicodeRange(prev, next *toUnicodeEntry) bool {
	if prev == nil || next == nil {
		return false
	}
	return allowCodeRange(prev.cid, next.cid) &&
		allowDestinationRange(prev.text, next.text)
}

// allowCodeRange returns true if the 16-bit values are sequential and differ
// only in the low-order byte.
func allowCodeRange(prev, next int) bool {
	if (prev + 1) != next {
		return false
	}
	prevH := (prev >> 8) & 0xFF
	prevL := prev & 0xFF
	nextH := (next >> 8) & 0xFF
	nextL := next & 0xFF

	return prevH == nextH && prevL < nextL
}

// allowDestinationRange returns true if the code points represented by the
// strings are sequential and differ only in the low-order byte.
//
// JAVA BUG 33: only prev is checked for being a single code point, next is not,
// so a one-character destination followed by a longer one extends the range and
// everything after the first character of the longer one is lost --- 0x400 to
// "a" and 0x401 to "bc" becomes the range 0x400..0x401 starting at "a", and
// 0x401 then decodes to "b". Ported as written; see migration/JAVA-BUGS.md.
func allowDestinationRange(prev, next string) bool {
	if prev == "" || next == "" {
		return false
	}
	prevCode, _ := utf8.DecodeRuneInString(prev)
	nextCode, _ := utf8.DecodeRuneInString(next)

	// Allow the new destination string if:
	// 1. It is sequential with the previous one and differs only in the low-order byte
	// 2. The previous string does not contain any UTF-16 surrogates
	return allowCodeRange(int(prevCode), int(nextCode)) &&
		utf8.RuneCountInString(prev) == 1
}

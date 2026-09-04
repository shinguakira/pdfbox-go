package cos_test

// The accept() assertions of the Java cos tests, restored now that COSWriter
// exists.
//
// They live in an external test package, and in a file of their own, because
// they drive pdfwriter and pdfwriter imports cos: a test file in package cos
// cannot import it without a cycle. Go allows package cos_test alongside
// package cos in the same directory, which is the whole reason that form
// exists. The four tests slice 1 left as placeholders --- TestBooleanAccept,
// TestIntegerAccept, TestFloatAccept and TestStringObjAccept --- keep asserting
// the double dispatch; these assert the bytes.

import (
	"bytes"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfwriter"
)

// escCharString and escCharStringPDFFormat are ESC_CHAR_STRING and
// ESC_CHAR_STRING_PDF_FORMAT of TestCOSString.
const (
	escCharString          = "( test#some) escaped< \\chars>!~1239857 "
	escCharStringPDFFormat = "\\( test#some\\) escaped< \\\\chars>!~1239857 "
)

// assertBytes is testByteArrays of TestCOSBase.
func assertBytes(t *testing.T, want, got []byte) {
	t.Helper()
	if !bytes.Equal(want, got) {
		t.Errorf("bytes = %q, want %q", got, want)
	}
}

// TestBooleanAcceptWritesBytes is TestCOSBoolean.testAccept.
func TestBooleanAcceptWritesBytes(t *testing.T) {
	var outStream bytes.Buffer
	visitor := pdfwriter.NewCOSWriter(&outStream)
	if err := cos.True.Accept(visitor); err != nil {
		t.Fatalf("True.Accept: %v", err)
	}
	assertBytes(t, []byte(cos.True.String()), outStream.Bytes())
	outStream.Reset()
	if err := cos.False.Accept(visitor); err != nil {
		t.Fatalf("False.Accept: %v", err)
	}
	assertBytes(t, []byte(cos.False.String()), outStream.Bytes())
}

// TestIntegerAcceptWritesBytes is TestCOSInteger.testAccept.
func TestIntegerAcceptWritesBytes(t *testing.T) {
	var outStream bytes.Buffer
	visitor := pdfwriter.NewCOSWriter(&outStream)
	for i := -1000; i < 3000; i += 200 {
		cosInt := cos.GetInteger(int64(i))
		if err := cosInt.Accept(visitor); err != nil {
			t.Fatalf("Failed to write %d: %v", i, err)
		}
		assertBytes(t, []byte(strconv.Itoa(i)), outStream.Bytes())
		outStream.Reset()
	}
}

// floatToString is the helper of the same name in TestCOSFloat: the float's
// shortest round-tripping decimal, rendered without an exponent, with the
// trailing fraction zeroes removed.
//
// Java reaches that through new BigDecimal(String.valueOf(value)).toPlainString().
// String.valueOf(float) gives the shortest round-tripping form, which is what
// strconv's 'g' with precision -1 over a float32 gives; big.Float renders it
// plainly from there.
func floatToString(value float32) string {
	shortest := strconv.FormatFloat(float64(value), 'g', -1, 32)
	var plain string
	if strings.ContainsAny(shortest, "eE") {
		d, _, err := big.ParseFloat(shortest, 10, 200, big.ToNearestEven)
		if err != nil {
			plain = shortest
		} else {
			plain = d.Text('f', -1)
		}
	} else {
		plain = shortest
	}
	if !strings.Contains(plain, ".") {
		// Java always renders a float with a fraction part
		plain += ".0"
	}
	return removeTrailingNull(plain)
}

// removeTrailingNull is the helper of the same name in TestCOSFloat.
func removeTrailingNull(value string) string {
	// remove fraction digit "0" only
	if strings.Contains(value, ".") && !strings.HasSuffix(value, ".0") {
		for strings.HasSuffix(value, "0") && !strings.HasSuffix(value, ".0") {
			value = value[:len(value)-1]
		}
	}
	return value
}

// TestFloatAcceptWritesBytes is the AcceptTester of TestCOSFloat.
//
// Java sweeps i * new Random(seed).nextFloat() for i in [-100000, 300000) step
// 20000, once with a fixed seed and once with the clock. java.util.Random's
// sequence is not reproducible in Go without porting the generator, so the
// sweep is the one slice 1 chose for the rest of float_test.go; the corner case
// of PDFBOX-1778 that TestCOSFloat.testWritePDF adds is here too.
func TestFloatAcceptWritesBytes(t *testing.T) {
	var outStream bytes.Buffer
	visitor := pdfwriter.NewCOSWriter(&outStream)

	runTest := func(num float32) {
		t.Helper()
		cosFloat := cos.NewFloat(num)
		if err := cosFloat.Accept(visitor); err != nil {
			t.Fatalf("Failed to write %v: %v", num, err)
		}
		if got, want := outStream.String(), floatToString(cosFloat.FloatValue()); got != want {
			t.Errorf("Accept(%v) = %q, want %q", num, got, want)
		}
		assertBytes(t, []byte(floatToString(num)), outStream.Bytes())
		outStream.Reset()
	}

	for i := -1000; i < 3000; i += 200 {
		runTest(float32(i))
	}
	// test a corner case as described in PDFBOX-1778
	runTest(0.000000000000000000000000000000001)
}

// TestStringObjAcceptWritesBytes is TestCOSString.testAccept.
func TestStringObjAcceptWritesBytes(t *testing.T) {
	var outStream bytes.Buffer
	visitor := pdfwriter.NewCOSWriter(&outStream)
	testSubj := cos.NewStringObj(escCharString)
	if err := testSubj.Accept(visitor); err != nil {
		t.Fatalf("Accept: %v", err)
	}
	if got, want := outStream.String(), "("+escCharStringPDFFormat+")"; got != want {
		t.Errorf("Accept = %q, want %q", got, want)
	}
	outStream.Reset()
	testSubjHex := cos.NewStringObjHex(escCharString, true)
	if err := testSubjHex.Accept(visitor); err != nil {
		t.Fatalf("Accept of the hex form: %v", err)
	}
	if got, want := outStream.String(), "<"+createHexUnits(escCharString)+">"; got != want {
		t.Errorf("Accept of the hex form = %q, want %q", got, want)
	}
}

// writePDFTests is the helper of the same name in TestCOSString.
func writePDFTests(t *testing.T, expected string, testSubj *cos.StringObj) {
	t.Helper()
	var outStream bytes.Buffer
	if err := pdfwriter.WriteString(testSubj, &outStream); err != nil {
		t.Fatalf("WriteString: %v", err)
	}
	if got := outStream.String(); got != expected {
		t.Errorf("WriteString = %q, want %q", got, expected)
	}
}

// TestStringObjWritePDFViaWriter is TestCOSString.testWritePDF.
func TestStringObjWritePDFViaWriter(t *testing.T) {
	inputString := "Test with a text and a few numbers 1, 2 and 3"
	pdfHex := "<" + createHexUnits(inputString) + ">"
	cosStr := cos.NewStringObjHex(inputString, true)
	writePDFTests(t, pdfHex, cosStr)

	escStr := cos.NewStringObj(escCharString)
	writePDFTests(t, "("+escCharStringPDFFormat+")", escStr)
	escStrHex := cos.NewStringObjHex(escCharString, true)
	// Escape characters not escaped in hex version
	writePDFTests(t, "<"+createHexUnits(escCharString)+">", escStrHex)
}

// TestStringObjFromHexWritePDF is the writePDF half of TestCOSString.testFromHex;
// the parse half is already in string_test.go.
func TestStringObjFromHexWritePDF(t *testing.T) {
	expected := "Quick and simple test"
	hexForm := createHexUnits(expected)
	test1, err := cos.ParseHexString(hexForm)
	if err != nil {
		t.Fatalf("ParseHexString: %v", err)
	}
	writePDFTests(t, "("+expected+")", test1)

	test2, err := cos.ParseHexString(createHexUnits(escCharString))
	if err != nil {
		t.Fatalf("ParseHexString of the escape string: %v", err)
	}
	writePDFTests(t, "("+escCharStringPDFFormat+")", test2)
}

// TestStringObjUnicodeWritePDF is the writeString half of TestCOSString.testUnicode;
// the getString and getBytes halves are already in string_test.go.
func TestStringObjUnicodeWritePDF(t *testing.T) {
	textAscii := "This is some regular text. It should all be visible."
	text8Bit := "En français où les choses sont accentués. En español, así"
	/** をクリックしてく */
	textHighBits := "をクリックしてく"

	stringAscii := cos.NewStringObj(textAscii)
	string8Bit := cos.NewStringObj(text8Bit)
	stringHighBits := cos.NewStringObj(textHighBits)

	var out bytes.Buffer
	if err := pdfwriter.WriteString(stringAscii, &out); err != nil {
		t.Fatalf("WriteString of the ASCII string: %v", err)
	}
	if got, want := out.String(), "("+textAscii+")"; got != want {
		t.Errorf("WriteString = %q, want %q", got, want)
	}

	out.Reset()
	if err := pdfwriter.WriteString(string8Bit, &out); err != nil {
		t.Fatalf("WriteString of the 8 bit string: %v", err)
	}
	var hex strings.Builder
	for _, c := range utf16Units(text8Bit) {
		hex.WriteString(strings.ToUpper(fmt.Sprintf("%x", c)))
	}
	if got, want := out.String(), "<"+hex.String()+">"; got != want {
		t.Errorf("WriteString = %q, want %q", got, want)
	}

	out.Reset()
	if err := pdfwriter.WriteString(stringHighBits, &out); err != nil {
		t.Fatalf("WriteString of the high bit string: %v", err)
	}
	hex.Reset()
	hex.WriteString("FEFF") // Byte Order Mark
	for _, c := range utf16Units(textHighBits) {
		hex.WriteString(strings.ToUpper(fmt.Sprintf("%x", c)))
	}
	if got, want := out.String(), "<"+hex.String()+">"; got != want {
		t.Errorf("WriteString = %q, want %q", got, want)
	}
}

// createHexUnits is createHex of TestCOSString: each UTF-16 code unit rendered
// as upper-case hex, with no zero padding. string_test.go has the same helper;
// this package cannot see it.
func createHexUnits(s string) string {
	var sb strings.Builder
	for _, r := range utf16Units(s) {
		sb.WriteString(strings.ToUpper(fmt.Sprintf("%x", r)))
	}
	return sb.String()
}

// utf16Units returns the UTF-16 code units of s, which is what Java's
// String.toCharArray yields.
func utf16Units(s string) []uint16 {
	return utf16.Encode([]rune(s))
}

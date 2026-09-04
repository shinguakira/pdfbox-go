package filter

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// Tests written from the source for the parts of pdfbox/filter the two Java
// tests do not reach, which slice 6's A4 asks to be named before they are
// written.
//
// TestFilters covers the round trip of every filter that has one -- Flate,
// LZW, ASCIIHex, ASCII85, RunLength, Crypt and Identity -- over twenty rounds
// of adversarial data, and testPDFBOX1977 and testRLE cover two regressions.
// PredictorTest covers getBitSeq and calcSetBitSeq.
//
// What none of them reaches, and what is tested here:
//
//   - The four filters that do not round trip. DCT is exercised by the image
//     tests; CCITTFax by the CCITT factory's; JBIG2 and JPX have nothing to
//     exercise, and what they must do is report the missing reader.
//   - The factory: every name and abbreviation, and the error for an unknown
//     one.
//   - DecodeOptions, whose shared default may not be modified.
//   - The decode parameter lookup, including the two abbreviated keys an inline
//     image uses and the PDFBOX-3932 shape.
//   - The error paths of the ASCII filters, which the round trip never takes
//     because it only ever feeds them their own output.
//   - Crypt, which refuses a crypt filter that is not the identity one.

// TestByNameCoversEveryFilter checks that each name and each abbreviation
// reaches the filter FilterFactory registers it under.
func TestByNameCoversEveryFilter(t *testing.T) {
	cases := []struct {
		names []*cos.Name
		want  Filter
	}{
		{[]*cos.Name{cos.FlateDecode, cos.Fl}, Flate{}},
		{[]*cos.Name{cos.DCTDecode, cos.DCT}, DCT{}},
		{[]*cos.Name{cos.CCITTFaxDecode, cos.CCF}, CCITTFax{}},
		{[]*cos.Name{cos.LZWDecode, cos.LZW}, LZW{}},
		{[]*cos.Name{cos.ASCIIHexDecode, cos.AHx}, ASCIIHex{}},
		{[]*cos.Name{cos.ASCII85Decode, cos.A85}, ASCII85{}},
		{[]*cos.Name{cos.RunLengthDecode, cos.RL}, RunLength{}},
		{[]*cos.Name{cos.Crypt}, Crypt{}},
		{[]*cos.Name{cos.JPXDecode}, JPX{}},
		{[]*cos.Name{cos.JBIG2Decode}, JBIG2{}},
		{[]*cos.Name{cos.Identity}, Identity{}},
	}
	for _, c := range cases {
		for _, name := range c.names {
			got, err := ByName(name)
			if err != nil {
				t.Errorf("ByName(%s): %v", name.Name(), err)
				continue
			}
			if got != c.want {
				t.Errorf("ByName(%s) = %T, want %T", name.Name(), got, c.want)
			}
		}
	}

	if _, err := ByName(cos.GetPDFName("NoSuchFilter")); !errors.Is(err, ErrUnsupportedFilter) {
		t.Errorf("an unknown filter name gave %v", err)
	}
}

// TestAllFiltersIsComplete checks that allFilters and ByName agree: every
// filter the factory holds is reachable by name, and there are no others.
func TestAllFiltersIsComplete(t *testing.T) {
	names := []*cos.Name{
		cos.FlateDecode, cos.DCTDecode, cos.CCITTFaxDecode, cos.LZWDecode,
		cos.ASCIIHexDecode, cos.ASCII85Decode, cos.RunLengthDecode, cos.Crypt,
		cos.JPXDecode, cos.JBIG2Decode,
	}
	if len(allFilters()) != len(names) {
		t.Fatalf("allFilters has %d entries, the names have %d",
			len(allFilters()), len(names))
	}
	byName := map[Filter]bool{}
	for _, name := range names {
		f, err := ByName(name)
		if err != nil {
			t.Fatalf("ByName(%s): %v", name.Name(), err)
		}
		byName[f] = true
	}
	for _, f := range allFilters() {
		if !byName[f] {
			t.Errorf("%T is in allFilters but reachable by no name", f)
		}
	}
}

// TestMissingImageReaders pins what the two filters PDFBox cannot decode on its
// own report. Java reaches them through Filter.findImageReader, which throws
// MissingImageReaderException on a build without the optional jars; the port is
// always such a build.
func TestMissingImageReaders(t *testing.T) {
	cases := []struct {
		filter  Filter
		message string
	}{
		{JBIG2{}, "jbig2-imageio is not installed"},
		{JPX{}, "Java Advanced Imaging (JAI) Image I/O Tools are not installed"},
	}
	for _, c := range cases {
		var out bytes.Buffer
		_, err := c.filter.Decode(&out, bytes.NewReader([]byte{1, 2, 3}),
			cos.NewDictionary(), 0)
		if !errors.Is(err, ErrMissingImageReader) {
			t.Errorf("%T gave %v, want a missing image reader", c.filter, err)
			continue
		}
		if !strings.Contains(err.Error(), c.message) {
			t.Errorf("%T said %q, want it to name %q", c.filter, err.Error(), c.message)
		}
	}
}

// TestEncodingPanics pins the four filters whose encode Java answers with
// UnsupportedOperationException, which the port panics for.
func TestEncodingPanics(t *testing.T) {
	for _, f := range []Filter{DCT{}, JBIG2{}, JPX{}} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%T should refuse to encode", f)
				}
			}()
			_ = f.Encode(&bytes.Buffer{}, bytes.NewReader(nil), cos.NewDictionary())
		}()
	}
}

// TestDecodeOptionsDefaults checks the shared default and that it may not be
// modified, which Java enforces with a subclass whose setters throw.
func TestDecodeOptionsDefaults(t *testing.T) {
	options := NewDecodeOptions()
	if got := options.SubsamplingX(); got != 1 {
		t.Errorf("the default horizontal subsampling is %d, want 1", got)
	}
	if got := options.SubsamplingY(); got != 1 {
		t.Errorf("the default vertical subsampling is %d, want 1", got)
	}
	if options.SourceRegion() != nil {
		t.Error("a new DecodeOptions should have no source region")
	}
	if options.IsFilterSubsampled() {
		t.Error("a new DecodeOptions should not claim the filter subsampled")
	}

	options.SetSubsamplingX(2)
	options.SetSubsamplingY(3)
	options.SetSubsamplingOffsetX(4)
	options.SetSubsamplingOffsetY(5)
	region := &geom.Rectangle{X: 1, Y: 2, Width: 3, Height: 4}
	options.SetSourceRegion(region)
	if options.SubsamplingX() != 2 || options.SubsamplingY() != 3 ||
		options.SubsamplingOffsetX() != 4 || options.SubsamplingOffsetY() != 5 ||
		options.SourceRegion() != region {
		t.Error("the setters did not take")
	}

	if got := NewDecodeOptionsOfSubsampling(4); got.SubsamplingX() != 4 ||
		got.SubsamplingY() != 4 {
		t.Error("NewDecodeOptionsOfSubsampling should set both directions")
	}
	if got := NewDecodeOptionsOfBounds(1, 2, 3, 4).SourceRegion(); got == nil ||
		*got != (geom.Rectangle{X: 1, Y: 2, Width: 3, Height: 4}) {
		t.Errorf("NewDecodeOptionsOfBounds gave %v", got)
	}
}

// TestDefaultDecodeOptionsIsFinal pins Java's FinalDecodeOptions: every setter
// throws, and setFilterSubsampled is ignored rather than throwing, so that the
// shared instance keeps its true.
func TestDefaultDecodeOptionsIsFinal(t *testing.T) {
	if !DefaultDecodeOptions.IsFilterSubsampled() {
		t.Error("DefaultDecodeOptions should claim the filter subsampled")
	}
	setters := []func(){
		func() { DefaultDecodeOptions.SetSourceRegion(nil) },
		func() { DefaultDecodeOptions.SetSubsamplingX(2) },
		func() { DefaultDecodeOptions.SetSubsamplingY(2) },
		func() { DefaultDecodeOptions.SetSubsamplingOffsetX(2) },
		func() { DefaultDecodeOptions.SetSubsamplingOffsetY(2) },
	}
	for i, set := range setters {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("setter %d should refuse to modify the shared default", i)
				}
			}()
			set()
		}()
	}
	// setFilterSubsampled is the one that is ignored rather than refused
	DefaultDecodeOptions.setFilterSubsampled(false)
	if !DefaultDecodeOptions.IsFilterSubsampled() {
		t.Error("the shared default should have kept its true")
	}
}

// The decode parameter lookup itself is already covered case for case by
// TestDecodeParamsFor in flate_test.go, which slice 1 wrote.

// TestDecodeParamsOrEmpty pins the difference between Java's getDecodeParams,
// which never returns null, and the port's decodeParamsFor, which does.
func TestDecodeParamsOrEmpty(t *testing.T) {
	got := decodeParamsOrEmpty(cos.NewDictionary(), 0)
	if got == nil {
		t.Fatal("decodeParamsOrEmpty should never return nothing")
	}
	if got.Size() != 0 {
		t.Errorf("the empty dictionary has %d entries", got.Size())
	}
	if want := 1728; got.GetIntDefault(cos.Columns, 1728) != want {
		t.Error("the empty dictionary should give the caller its defaults")
	}
}

// TestASCIIHexTolerance covers the error paths the round trip never takes: a
// digit that is not one, an odd number of digits, and the whitespace the
// decoder skips.
func TestASCIIHexTolerance(t *testing.T) {
	cases := []struct {
		encoded string
		want    []byte
	}{
		{"48656C6C6F", []byte("Hello")},
		// whitespace between digits is skipped
		{"48 65\n6C\t6C\r6F", []byte("Hello")},
		// the > ends the data
		{"4865>6C6C6F", []byte("He")},
		// an odd digit before the terminator behaves like a trailing zero
		{"486>", []byte{0x48, 0x60}},
		// and so does an odd digit at the end of the stream
		{"486", []byte{0x48, 0x60}},
		// A digit that is not one is logged and its table entry, -1, is added
		// anyway: "4Z" is 4*16 + (-1) = 63, not 64. An invalid FIRST digit is
		// worse -- -1*16 -- and "Z4" comes out as 0xF4. That is what Java does.
		{"4Z65", []byte{63, 0x65}},
		{"Z465", []byte{0xF4, 0x65}},
		{"", nil},
		{">", nil},
	}
	for _, c := range cases {
		var out bytes.Buffer
		if _, err := (ASCIIHex{}).Decode(&out, strings.NewReader(c.encoded),
			cos.NewDictionary(), 0); err != nil {
			t.Errorf("decoding %q: %v", c.encoded, err)
			continue
		}
		if !bytes.Equal(out.Bytes(), c.want) {
			t.Errorf("decoding %q gave %v, want %v", c.encoded, out.Bytes(), c.want)
		}
	}
}

// TestASCII85Invalid covers the one error the ASCII85 decoder reports.
func TestASCII85Invalid(t *testing.T) {
	var out bytes.Buffer
	_, err := (ASCII85{}).Decode(&out, bytes.NewReader([]byte{'!', '!', 0x80, '!', '!'}),
		cos.NewDictionary(), 0)
	if err == nil || !strings.Contains(err.Error(), "Invalid data in Ascii85 stream") {
		t.Errorf("a byte outside the alphabet gave %v", err)
	}
}

// TestASCII85EmptyEncodesToNothing pins the consequence of Java's flushed flag
// starting true: a stream that took no bytes writes nothing at all, terminator
// included.
func TestASCII85EmptyEncodesToNothing(t *testing.T) {
	var encoded bytes.Buffer
	if err := (ASCII85{}).Encode(&encoded, bytes.NewReader(nil), cos.NewDictionary()); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if encoded.Len() != 0 {
		t.Errorf("an empty input encoded to %q, want nothing", encoded.String())
	}
}

// TestASCII85ZeroGroup pins the z shorthand: four zero bytes are one z, and an
// incomplete group of zeroes is expanded back to exclamation marks.
func TestASCII85ZeroGroup(t *testing.T) {
	var encoded bytes.Buffer
	if err := (ASCII85{}).Encode(&encoded, bytes.NewReader([]byte{0, 0, 0, 0}),
		cos.NewDictionary()); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !strings.HasPrefix(encoded.String(), "z") {
		t.Errorf("four zero bytes encoded to %q, want it to start with z", encoded.String())
	}

	var decoded bytes.Buffer
	if _, err := (ASCII85{}).Decode(&decoded, bytes.NewReader(encoded.Bytes()),
		cos.NewDictionary(), 0); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !bytes.Equal(decoded.Bytes(), []byte{0, 0, 0, 0}) {
		t.Errorf("the z round trip gave %v", decoded.Bytes())
	}
}

// TestCryptRefusesANamedFilter pins the one thing CryptFilter does beyond
// passing the data through.
func TestCryptRefusesANamedFilter(t *testing.T) {
	parameters := cos.NewDictionary()
	parameters.SetItem(cos.NameKey, cos.GetPDFName("StdCF"))

	var out bytes.Buffer
	_, err := (Crypt{}).Decode(&out, bytes.NewReader([]byte("data")), parameters, 0)
	if err == nil || !strings.Contains(err.Error(), "Unsupported crypt filter StdCF") {
		t.Errorf("decoding through a named crypt filter gave %v", err)
	}
	if err := (Crypt{}).Encode(&out, bytes.NewReader([]byte("data")), parameters); err == nil ||
		!strings.Contains(err.Error(), "Unsupported crypt filter StdCF") {
		t.Errorf("encoding through a named crypt filter gave %v", err)
	}

	// /Identity and a missing /Name both pass the data through
	for _, parameters := range []*cos.Dictionary{cos.NewDictionary(), identityCryptParams()} {
		var out bytes.Buffer
		if _, err := (Crypt{}).Decode(&out, bytes.NewReader([]byte("data")),
			parameters, 0); err != nil {
			t.Errorf("the identity crypt filter gave %v", err)
		}
		if out.String() != "data" {
			t.Errorf("the identity crypt filter gave %q", out.String())
		}
	}
}

func identityCryptParams() *cos.Dictionary {
	d := cos.NewDictionary()
	d.SetItem(cos.NameKey, cos.Identity)
	return d
}

// TestStaticDecodeChainsFilters checks the static Filter.decode over a chain,
// and the duplicate reduction it does first.
func TestStaticDecodeChainsFilters(t *testing.T) {
	original := []byte("the quick brown fox jumps over the lazy dog")

	// encode ASCIIHex(Flate(data)), which is how a chain is written
	var flated bytes.Buffer
	if err := (Flate{}).Encode(&flated, bytes.NewReader(original), cos.NewDictionary()); err != nil {
		t.Fatal(err)
	}
	var hexed bytes.Buffer
	if err := (ASCIIHex{}).Encode(&hexed, bytes.NewReader(flated.Bytes()),
		cos.NewDictionary()); err != nil {
		t.Fatal(err)
	}

	var results []DecodeResult
	read, err := Decode(bytes.NewReader(hexed.Bytes()), []Filter{ASCIIHex{}, Flate{}},
		cos.NewDictionary(), DefaultDecodeOptions, &results)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	length, err := read.Length()
	if err != nil {
		t.Fatal(err)
	}
	got := make([]byte, length)
	if _, err := read.Read(got); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, original) {
		t.Errorf("the chain gave %q", got)
	}
	if len(results) != 2 {
		t.Errorf("the chain reported %d results, want 2", len(results))
	}

	// a repeated filter is reduced to one, which is what PDFBox does for a
	// stream whose /Filter array names the same filter twice
	var single bytes.Buffer
	if err := (Flate{}).Encode(&single, bytes.NewReader(original),
		cos.NewDictionary()); err != nil {
		t.Fatal(err)
	}
	read, err = Decode(bytes.NewReader(single.Bytes()), []Filter{Flate{}, Flate{}},
		cos.NewDictionary(), DefaultDecodeOptions, nil)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	length, _ = read.Length()
	got = make([]byte, length)
	if _, err := read.Read(got); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, original) {
		t.Errorf("the reduced chain gave %q", got)
	}
}

// TestDamageTolerance is slice 6's D8: every PDFBox filter is written to hand
// back what it decoded when the input is corrupt, rather than failing. Slice 1
// found one place the Go got that wrong for Flate; this checks each filter this
// slice added.
//
// What each must do is not the same, and the Java decides it:
//
//   - LZW catches the EOFException its bit reader raises, logs it and flushes,
//     so a truncated stream yields the codes that did decode.
//   - RunLength breaks out of its loop at the end of the input, on both arms.
//   - ASCIIHex and ASCII85 stop at the end of the data; ASCII85 alone reports an
//     error, and only for a byte outside its alphabet.
//   - CCITTFax fills the rest of the bitmap with zeroes, which is what lets
//     readFromDecoderStream finish a row count the data does not support.
func TestDamageTolerance(t *testing.T) {
	original := bytes.Repeat([]byte("the quick brown fox jumps over the lazy dog. "), 30)

	for _, c := range []struct {
		name   string
		filter Filter
	}{
		{"lzw", LZW{}},
		{"runlength", RunLength{}},
		{"asciihex", ASCIIHex{}},
	} {
		t.Run(c.name, func(t *testing.T) {
			var encoded bytes.Buffer
			if err := c.filter.Encode(&encoded, bytes.NewReader(original),
				cos.NewDictionary()); err != nil {
				t.Fatalf("Encode: %v", err)
			}

			// cut the stream in half and decode what is left
			truncated := encoded.Bytes()[:encoded.Len()/2]
			var decoded bytes.Buffer
			// An error is acceptable -- the point is that whatever decoded
			// before the damage is still handed back rather than discarded.
			_, _ = c.filter.Decode(&decoded, bytes.NewReader(truncated),
				cos.NewDictionary(), 0)

			if decoded.Len() == 0 {
				t.Fatal("a truncated stream yielded nothing; partial data must survive")
			}
			if !bytes.HasPrefix(original, decoded.Bytes()) {
				t.Fatalf("the partial output of %d bytes is not a prefix of the original",
					decoded.Len())
			}
		})
	}
}

// TestCCITTDamageTolerance checks the fax decoder's own kind: it fills the rest
// of the bitmap rather than stopping short, so that a row count the data does
// not support still produces an image.
func TestCCITTDamageTolerance(t *testing.T) {
	const columns, rows = 64, 16
	bitmap := bytes.Repeat([]byte{0x0F}, columns/8*rows)

	dict := cos.NewDictionary()
	dict.SetInt(cos.Columns, columns)
	dict.SetInt(cos.Rows, rows)
	var encoded bytes.Buffer
	if err := (CCITTFax{}).Encode(&encoded, bytes.NewReader(bitmap), dict); err != nil {
		t.Fatalf("Encode: %v", err)
	}

	parms := cos.NewDictionary()
	parms.SetInt(cos.Columns, columns)
	parms.SetInt(cos.Rows, rows)
	parms.SetInt(cos.K, -1)
	stream := cos.NewDictionary()
	stream.SetItem(cos.DecodeParms, parms)
	stream.SetItem(cos.Filter, cos.CCITTFaxDecode)

	var decoded bytes.Buffer
	truncated := encoded.Bytes()[:encoded.Len()/3]
	if _, err := (CCITTFax{}).Decode(&decoded, bytes.NewReader(truncated), stream, 0); err != nil {
		t.Fatalf("a truncated fax stream gave %v, want the rows that decoded", err)
	}
	if want := columns / 8 * rows; decoded.Len() != want {
		t.Errorf("the decoder gave %d bytes, want the full bitmap of %d",
			decoded.Len(), want)
	}
}

// TestASCII85DamageTolerance is ASCII85's own answer, which is not the others'
// and is worth writing out: a truncated stream hands back everything that
// decoded and then repeats its last complete group of four bytes.
//
// That repeat is JAVA-BUGS 32. ASCII85InputStream.read() sets index to 0 before
// it reads a group and returns -1 from inside the loop where the stream ends
// mid-group, leaving n at the previous group's 4; the array read then finds
// index < n and copies that group out a second time. The port does the same,
// so this test asserts the repeat rather than a clean prefix.
func TestASCII85DamageTolerance(t *testing.T) {
	original := bytes.Repeat([]byte("the quick brown fox jumps over the lazy dog. "), 30)

	var encoded bytes.Buffer
	if err := (ASCII85{}).Encode(&encoded, bytes.NewReader(original),
		cos.NewDictionary()); err != nil {
		t.Fatalf("Encode: %v", err)
	}

	truncated := encoded.Bytes()[:encoded.Len()/2]
	var decoded bytes.Buffer
	_, _ = (ASCII85{}).Decode(&decoded, bytes.NewReader(truncated), cos.NewDictionary(), 0)

	got := decoded.Bytes()
	if len(got) == 0 {
		t.Fatal("a truncated stream yielded nothing; partial data must survive")
	}
	if len(got)%4 != 0 {
		t.Fatalf("the output is %d bytes, which is not whole groups", len(got))
	}
	// everything but the last group is the original
	body := got[:len(got)-4]
	if !bytes.HasPrefix(original, body) {
		t.Fatal("the decoded body is not a prefix of the original")
	}
	// and the last group repeats the one before it
	if !bytes.Equal(got[len(got)-4:], body[len(body)-4:]) {
		t.Errorf("the last group is %q, want a repeat of %q",
			got[len(got)-4:], body[len(body)-4:])
	}
}

package filter

import (
	"bytes"
	"math/rand"
	"os"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// The rest of pdfbox/src/test/java/org/apache/pdfbox/filter/TestFilters.java.
//
// Slice 1 ported the random data generator of testFilters and ran it over the
// two filters it had. This is the whole test as the Java writes it -- every
// filter the factory holds, minus the four that do not round trip -- together
// with testPDFBOX1977, testRLE and testEmptyFilterList, which slice 1 left for
// the filters they exercise.
//
// testPDFBOX4517 is not ported. It reads target/pdfs/PDFBOX-4517-cryptfilter.pdf,
// which the Java build downloads and this repository does not carry; the same
// reason two of slice 5's tests are absent.

// filterFixture is where the Java test resources of this package live,
// relative to it.
const filterFixture = "../../../pdfbox/src/test/resources/org/apache/pdfbox/filter/"

// TestFilters will test all of the filters in the system. There will be COUNT
// of deterministic tests and COUNT of non-deterministic tests, see also the
// discussion in PDFBOX-1977.
func TestFilters(t *testing.T) {
	const count = 10
	// Java seeds a Random with 123456 and draws ten seeds from it, then runs
	// ten more from a non-deterministic seed. Go's generator is not Java's, so
	// the seeds cannot be the same numbers; what the test needs is data of that
	// shape, and a failing run that can be repeated.
	rd := rand.New(rand.NewSource(123456))
	for iter := 0; iter < count*2; iter++ {
		original := randomFilterData(rd)
		for _, f := range allFilters() {
			// Skip filters that don't currently support roundtripping
			switch f.(type) {
			case DCT, CCITTFax, JPX, JBIG2:
				continue
			}
			checkEncodeDecode(t, f, original)
		}
	}
}

// TestPDFBOX1977 will test the LZW filter with the sequence that failed in
// PDFBOX-1977. To check that the test itself is legit, revert LZWFilter.java to
// rev 1571801, which should fail this test.
func TestPDFBOX1977(t *testing.T) {
	byteArray, err := os.ReadFile(filterFixture + "PDFBOX-1977.bin")
	if err != nil {
		t.Fatalf("reading PDFBOX-1977.bin: %v", err)
	}
	lzwFilter, err := ByName(cos.LZWDecode)
	if err != nil {
		t.Fatalf("ByName(LZWDecode): %v", err)
	}
	checkEncodeDecode(t, lzwFilter, byteArray)
}

// TestRLE tests simple and corner cases (128 identical, 128 identical at the
// end) of the RLE implementation. 128 non identical bytes likely to be caught
// in random testing.
func TestRLE(t *testing.T) {
	rleFilter, err := ByName(cos.RunLengthDecode)
	if err != nil {
		t.Fatalf("ByName(RunLengthDecode): %v", err)
	}
	input0 := []byte{}
	checkEncodeDecode(t, rleFilter, input0)
	input1 := []byte{1, 2, 3, 4, 5, 128, 140, 180, 0xFF}
	checkEncodeDecode(t, rleFilter, input1)
	input2 := make([]byte, 10)
	checkEncodeDecode(t, rleFilter, input2)
	input3 := make([]byte, 128)
	checkEncodeDecode(t, rleFilter, input3)
	input4 := make([]byte, 129)
	checkEncodeDecode(t, rleFilter, input4)
	input5 := make([]byte, 128+128)
	checkEncodeDecode(t, rleFilter, input5)
	input6 := make([]byte, 1)
	checkEncodeDecode(t, rleFilter, input6)
	input7 := []byte{1, 2}
	checkEncodeDecode(t, rleFilter, input7)
	input8 := make([]byte, 2)
	checkEncodeDecode(t, rleFilter, input8)
}

// TestEmptyFilterList checks that decoding through no filter at all is refused.
// Java throws IllegalArgumentException, which is unchecked, so the port panics.
func TestEmptyFilterList(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("an empty filter list should panic")
		}
	}()
	_, _ = Decode(bytes.NewReader(nil), nil, cos.NewDictionary(), nil, nil)
}

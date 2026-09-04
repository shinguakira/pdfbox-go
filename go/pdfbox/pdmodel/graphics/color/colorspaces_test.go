package color

import (
	"errors"
	goimage "image"
	"strings"
	"testing"

	awtimage "github.com/shinguakira/pdfbox-go/go/awt/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// The colour space tests of slice 6.
//
// Ported from Java: PDLabTest whole, PDIndexedTest's parameter checks and the
// first half of its factory test.
//
// Not ported: PDDeviceCMYKTest, both of whose tests are about loading the ICC
// profile through java.awt.color -- the machinery this port does not have, see
// the note on PDDeviceCMYK -- and the second half of PDIndexedTest.testFactory,
// which saves a document and so needs the writer of slice 7.
//
// The rest is written from the source, which slice 6's A4 asks to be named
// first: the create dispatch and every branch of it, the device spaces'
// component counts and decode arrays, the Indexed lookup, the CalRGB and
// CalGray identity cases, and DeviceN's tint transform path.

// Port of PDLabTest.testLAB.
func TestLAB(t *testing.T) {
	pdLab := NewPDLab()
	cosArray := pdLab.COSObject().(*cos.Array)
	dict := cosArray.GetObject(1).(*cos.Dictionary)

	// test with default values
	if got := pdLab.Name(); got != "Lab" {
		t.Errorf("Name = %q", got)
	}
	if got := pdLab.NumberOfComponents(); got != 3 {
		t.Errorf("NumberOfComponents = %d", got)
	}
	if pdLab.InitialColor() == nil {
		t.Fatal("InitialColor is nil")
	}
	assertComponents(t, pdLab.InitialColor().Components(), []float32{0, 0, 0})
	assertFloat(t, "BlackPoint.X", pdLab.BlackPoint().X(), 0)
	assertFloat(t, "BlackPoint.Y", pdLab.BlackPoint().Y(), 0)
	assertFloat(t, "BlackPoint.Z", pdLab.BlackPoint().Z(), 0)
	assertFloat(t, "Whitepoint.X", pdLab.Whitepoint().X(), 1)
	assertFloat(t, "Whitepoint.Y", pdLab.Whitepoint().Y(), 1)
	assertFloat(t, "Whitepoint.Z", pdLab.Whitepoint().Z(), 1)
	assertFloat(t, "ARange.Min", pdLab.ARange().Min(), -100)
	assertFloat(t, "ARange.Max", pdLab.ARange().Max(), 100)
	assertFloat(t, "BRange.Min", pdLab.BRange().Min(), -100)
	assertFloat(t, "BRange.Max", pdLab.BRange().Max(), 100)
	if got := dict.Size(); got != 0 {
		t.Errorf("read operations should not change the size of /Lab objects: size = %d", got)
	}
	_ = dict.String() // rev 1571125 did a stack overflow here

	// test setting specific values
	pdRange := common.NewPDRange()
	pdRange.SetMin(-1)
	pdRange.SetMax(2)
	pdLab.SetARange(pdRange)
	pdRange = common.NewPDRange()
	pdRange.SetMin(3)
	pdRange.SetMax(4)
	pdLab.SetBRange(pdRange)
	assertFloat(t, "ARange.Min", pdLab.ARange().Min(), -1)
	assertFloat(t, "ARange.Max", pdLab.ARange().Max(), 2)
	assertFloat(t, "BRange.Min", pdLab.BRange().Min(), 3)
	assertFloat(t, "BRange.Max", pdLab.BRange().Max(), 4)

	pdTristimulus := NewPDTristimulus()
	pdTristimulus.SetX(5)
	pdTristimulus.SetY(6)
	pdTristimulus.SetZ(7)
	pdLab.SetWhitePoint(pdTristimulus)
	pdTristimulus = NewPDTristimulus()
	pdTristimulus.SetX(8)
	pdTristimulus.SetY(9)
	pdTristimulus.SetZ(10)
	pdLab.SetBlackPoint(pdTristimulus)
	assertFloat(t, "Whitepoint.X", pdLab.Whitepoint().X(), 5)
	assertFloat(t, "Whitepoint.Y", pdLab.Whitepoint().Y(), 6)
	assertFloat(t, "Whitepoint.Z", pdLab.Whitepoint().Z(), 7)
	assertFloat(t, "BlackPoint.X", pdLab.BlackPoint().X(), 8)
	assertFloat(t, "BlackPoint.Y", pdLab.BlackPoint().Y(), 9)
	assertFloat(t, "BlackPoint.Z", pdLab.BlackPoint().Z(), 10)
	assertComponents(t, pdLab.InitialColor().Components(), []float32{0, 0, 3})
}

func assertFloat(t *testing.T, what string, got, want float32) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", what, got, want)
	}
}

func assertComponents(t *testing.T, got, want []float32) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("components = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("components = %v, want %v", got, want)
			return
		}
	}
}

// TestIndexedFactory is the first half of PDIndexedTest.testFactory: the array
// the factory builds. Its second half writes the document out and reads the
// bytes back, which needs the writer of slice 7.
func TestIndexedFactory(t *testing.T) {
	baseColorspace := DeviceRGB

	// define 6 color values
	const hival = 5
	// create s string containing 6 RGB values. Spaces are added for a better readability
	stringLookupData := strings.ReplaceAll("AA1166 112233 000000 FEDC01 4561FE DC34DA", " ", "")

	lookupData := mustParseHex(t, stringLookupData)
	pdIndexed, err := NewPDIndexed(baseColorspace, hival, lookupData)
	if err != nil {
		t.Fatalf("NewPDIndexed: %v", err)
	}
	indexedCOSArray := pdIndexed.COSObject().(*cos.Array)

	if got := int(indexedCOSArray.GetObject(2).(cos.Number).IntValue()); got != hival {
		t.Errorf("unexpected value for hival: %d", got)
	}
	if got := pdIndexed.Name(); got != cos.Indexed.Name() {
		t.Errorf("unexpected value for name: %q", got)
	}
	if got := pdIndexed.BaseColorSpace(); got != PDColorSpace(baseColorspace) {
		t.Errorf("unexpected value for base colorspace: %v", got)
	}
	lookupDataString := toHexString(indexedCOSArray.GetObject(3).(*cos.StringObj).Bytes())
	if lookupDataString != stringLookupData {
		t.Errorf("unexpected value for lookup data: %q", lookupDataString)
	}
}

func mustParseHex(t *testing.T, s string) []byte {
	t.Helper()
	out := make([]byte, len(s)/2)
	for i := range out {
		out[i] = hexValue(t, s[i*2])<<4 | hexValue(t, s[i*2+1])
	}
	return out
}

func hexValue(t *testing.T, c byte) byte {
	t.Helper()
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	}
	t.Fatalf("not a hex digit: %q", string(rune(c)))
	return 0
}

func toHexString(b []byte) string {
	const digits = "0123456789ABCDEF"
	out := make([]byte, 0, len(b)*2)
	for _, v := range b {
		out = append(out, digits[v>>4], digits[v&0x0F])
	}
	return string(out)
}

// TestIndexedFactoryParameterChecks is PDIndexedTest.testFactoryParameterChecks.
// Java throws IllegalArgumentException for each, which is unchecked, so the
// port panics.
func TestIndexedFactoryParameterChecks(t *testing.T) {
	baseColorspace := DeviceRGB
	// empty lookupData as placeholder
	lookupDataEmpty := make([]byte, 5)
	// define 6 color values
	const hival = 5
	stringLookupData := strings.ReplaceAll("AA1166 112233 000000 FEDC01 4561FE DC34DA", " ", "")
	lookupData := mustParseHex(t, stringLookupData)

	// check lookupData not null
	mustPanic(t, "lookupData must not be null", func() {
		_, _ = NewPDIndexed(baseColorspace, 0, nil)
	})
	// check base colorspace not null
	mustPanic(t, "base must not be null", func() {
		_, _ = NewPDIndexed(nil, 0, lookupDataEmpty)
	})
	// check hival not negative
	mustPanic(t, "hival not negative", func() {
		_, _ = NewPDIndexed(baseColorspace, -1, lookupDataEmpty)
	})
	// check hival <= 255
	mustPanic(t, "hival <= 255", func() {
		_, _ = NewPDIndexed(baseColorspace, 256, lookupDataEmpty)
	})
	// check minimum size of lookupData array:
	// (hival + 1) * numberOfComponents of base colorspace
	mustPanic(t, "lookupData too short", func() {
		_, _ = NewPDIndexed(baseColorspace, hival, lookupDataEmpty)
	})
	// everything is fine
	if _, err := NewPDIndexed(baseColorspace, hival, lookupData); err != nil {
		t.Errorf("NewPDIndexed with valid parameters: %v", err)
	}
}

func mustPanic(t *testing.T, what string, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s: expected a panic", what)
		}
	}()
	f()
}

// TestICCBasedConstructor is PDICCBasedTest.testConstructor, whose PDDocument
// the port replaces with the stream itself: pdmodel is above this package.
func TestICCBasedConstructor(t *testing.T) {
	iccBased := NewPDICCBasedOfStream(cos.NewStream(nil))
	if got := iccBased.Name(); got != "ICCBased" {
		t.Errorf("Name = %q", got)
	}
	if iccBased.PDStream() == nil {
		t.Error("PDStream is nil")
	}
}

// From here on the tests are written from the source.

// TestDeviceColorSpaces pins what each device space says about itself.
func TestDeviceColorSpaces(t *testing.T) {
	cases := []struct {
		space      PDColorSpace
		name       string
		components int
		decode     []float32
		initial    []float32
	}{
		{DeviceGray, "DeviceGray", 1, []float32{0, 1}, []float32{0}},
		{DeviceRGB, "DeviceRGB", 3, []float32{0, 1, 0, 1, 0, 1}, []float32{0, 0, 0}},
		{DeviceCMYK, "DeviceCMYK", 4, []float32{0, 1, 0, 1, 0, 1, 0, 1}, []float32{0, 0, 0, 1}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.space.Name(); got != c.name {
				t.Errorf("Name = %q", got)
			}
			if got := c.space.NumberOfComponents(); got != c.components {
				t.Errorf("NumberOfComponents = %d, want %d", got, c.components)
			}
			assertComponents(t, c.space.DefaultDecode(8), c.decode)
			assertComponents(t, c.space.InitialColor().Components(), c.initial)
		})
	}
}

// TestDeviceGrayAndRGBToRGB checks the two conversions that are identities.
func TestDeviceGrayAndRGBToRGB(t *testing.T) {
	gray, err := DeviceGray.ToRGB([]float32{0.25})
	if err != nil {
		t.Fatal(err)
	}
	assertComponents(t, gray, []float32{0.25, 0.25, 0.25})

	rgb, err := DeviceRGB.ToRGB([]float32{0.1, 0.2, 0.3})
	if err != nil {
		t.Fatal(err)
	}
	assertComponents(t, rgb, []float32{0.1, 0.2, 0.3})
}

// TestDeviceCMYKToRGB pins the naive conversion the port makes in place of
// Java's ICC transform. The values are the arithmetic, not a reading of the
// Java: R = (1-C)(1-K) and the same for the other two.
func TestDeviceCMYKToRGB(t *testing.T) {
	cases := []struct {
		cmyk []float32
		rgb  []float32
	}{
		{[]float32{0, 0, 0, 0}, []float32{1, 1, 1}}, // white
		{[]float32{0, 0, 0, 1}, []float32{0, 0, 0}}, // black
		{[]float32{1, 0, 0, 0}, []float32{0, 1, 1}}, // cyan
		{[]float32{0, 1, 0, 0}, []float32{1, 0, 1}}, // magenta
		{[]float32{0, 0, 1, 0}, []float32{1, 1, 0}}, // yellow
		{[]float32{0.5, 0, 0, 0.5}, []float32{0.25, 0.5, 0.5}},
	}
	for _, c := range cases {
		got, err := DeviceCMYK.ToRGB(c.cmyk)
		if err != nil {
			t.Fatal(err)
		}
		assertComponents(t, got, c.rgb)
	}
}

// TestDeviceRGBToRGBImage checks that a raster survives the conversion.
func TestDeviceRGBToRGBImage(t *testing.T) {
	raster := awtimage.NewInterleavedRaster(awtimage.TypeByte, 2, 1, 3)
	raster.SetPixel(0, 0, []int{10, 20, 30})
	raster.SetPixel(1, 0, []int{200, 210, 220})

	img, err := DeviceRGB.ToRGBImage(raster)
	if err != nil {
		t.Fatal(err)
	}
	assertRGBAt(t, img, 0, 0, 10, 20, 30)
	assertRGBAt(t, img, 1, 0, 200, 210, 220)
}

// TestCreateNames checks the dispatch for a bare colour space name.
func TestCreateNames(t *testing.T) {
	cases := []struct {
		name *cos.Name
		want PDColorSpace
	}{
		{cos.DeviceGray, DeviceGray},
		{cos.DeviceRGB, DeviceRGB},
		{cos.DeviceCMYK, DeviceCMYK},
	}
	for _, c := range cases {
		got, err := Create(c.name)
		if err != nil {
			t.Fatalf("Create(%v): %v", c.name, err)
		}
		if got != c.want {
			t.Errorf("Create(%v) = %v, want %v", c.name, got, c.want)
		}
	}

	// a name with no resources to look it up in
	if _, err := Create(cos.GetPDFName("CS0")); !errors.Is(err, ErrMissingResource) {
		t.Errorf("an unknown name gave %v, want a missing resource", err)
	}
	// /Pattern is slice 9
	if _, err := Create(cos.Pattern); !errors.Is(err, ErrColorSpaceNotPorted) {
		t.Errorf("/Pattern gave %v", err)
	}
}

// TestCreateArrays checks the dispatch for a colour space array.
func TestCreateArrays(t *testing.T) {
	// [/CalRGB <<>>]
	calRGB := cos.NewArray()
	calRGB.Add(cos.CalRGB)
	calRGB.Add(cos.NewDictionary())
	if space, err := Create(calRGB); err != nil {
		t.Errorf("CalRGB: %v", err)
	} else if _, ok := space.(*PDCalRGB); !ok {
		t.Errorf("CalRGB gave %T", space)
	}

	// [/CalGray <<>>]
	calGray := cos.NewArray()
	calGray.Add(cos.CalGray)
	calGray.Add(cos.NewDictionary())
	if space, err := Create(calGray); err != nil {
		t.Errorf("CalGray: %v", err)
	} else if _, ok := space.(*PDCalGray); !ok {
		t.Errorf("CalGray gave %T", space)
	}

	// [/Lab <<>>]
	lab := cos.NewArray()
	lab.Add(cos.Lab)
	lab.Add(cos.NewDictionary())
	if space, err := Create(lab); err != nil {
		t.Errorf("Lab: %v", err)
	} else if _, ok := space.(*PDLab); !ok {
		t.Errorf("Lab gave %T", space)
	}

	// [/DeviceRGB], which the specification does not allow but PDFBox accepts
	deviceRGB := cos.NewArray()
	deviceRGB.Add(cos.DeviceRGB)
	if space, err := Create(deviceRGB); err != nil {
		t.Errorf("[/DeviceRGB]: %v", err)
	} else if space != PDColorSpace(DeviceRGB) {
		t.Errorf("[/DeviceRGB] gave %v", space)
	}

	// an empty array
	if _, err := Create(cos.NewArray()); err == nil {
		t.Error("an empty array should be refused")
	}
	// an array whose first entry is not a name
	notAName := cos.NewArray()
	notAName.Add(cos.IntegerOne)
	if _, err := Create(notAName); err == nil {
		t.Error("an array not starting with a name should be refused")
	}
	// an unknown kind
	unknown := cos.NewArray()
	unknown.Add(cos.GetPDFName("NoSuchSpace"))
	if _, err := Create(unknown); err == nil {
		t.Error("an unknown colour space kind should be refused")
	}
}

// TestCreateRecursiveDictionary pins PDFBOX-5315: a dictionary whose
// /ColorSpace entry points at itself.
func TestCreateRecursiveDictionary(t *testing.T) {
	d := cos.NewDictionary()
	d.SetItem(cos.ColorSpace, d)
	_, err := Create(d)
	if err == nil || !strings.Contains(err.Error(), "Recursion in colorspace") {
		t.Errorf("a self-referencing dictionary gave %v", err)
	}
}

// TestCreateDictionaryWithColorSpace pins PDFBOX-4833: a dictionary that
// carries a /ColorSpace entry is read through it.
func TestCreateDictionaryWithColorSpace(t *testing.T) {
	d := cos.NewDictionary()
	d.SetItem(cos.ColorSpace, cos.DeviceRGB)
	space, err := Create(d)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if space != PDColorSpace(DeviceRGB) {
		t.Errorf("Create gave %v", space)
	}
}

// TestIndexedLookup checks the table lookup, both one value at a time and over
// a raster.
func TestIndexedLookup(t *testing.T) {
	// three entries: red, green, blue
	lookup := []byte{255, 0, 0, 0, 255, 0, 0, 0, 255}
	indexed, err := NewPDIndexed(DeviceRGB, 2, lookup)
	if err != nil {
		t.Fatalf("NewPDIndexed: %v", err)
	}
	if got := indexed.NumberOfComponents(); got != 1 {
		t.Errorf("NumberOfComponents = %d", got)
	}
	assertComponents(t, indexed.DefaultDecode(8), []float32{0, 255})

	for index, want := range [][]float32{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}} {
		got, err := indexed.ToRGB([]float32{float32(index)})
		if err != nil {
			t.Fatal(err)
		}
		assertComponents(t, got, want)
	}
	// an index past the end is clamped
	got, err := indexed.ToRGB([]float32{99})
	if err != nil {
		t.Fatal(err)
	}
	assertComponents(t, got, []float32{0, 0, 1})
}

// TestCalRGBIdentity checks that a CalRGB with the default white point, gamma
// and matrix reaches the CIE conversion rather than the passthrough, and that
// one with another white point takes the passthrough PDFBOX-2553 describes.
func TestCalRGBIdentity(t *testing.T) {
	array := cos.NewArray()
	array.Add(cos.CalRGB)
	array.Add(cos.NewDictionary())
	calRGB := NewPDCalRGBOfArray(array)

	if got := calRGB.Name(); got != "CalRGB" {
		t.Errorf("Name = %q", got)
	}
	if got := calRGB.NumberOfComponents(); got != 3 {
		t.Errorf("NumberOfComponents = %d", got)
	}
	// the default white point is 1 1 1, so the CIE path runs; black stays black
	black, err := calRGB.ToRGB([]float32{0, 0, 0})
	if err != nil {
		t.Fatal(err)
	}
	assertComponents(t, black, []float32{0, 0, 0})

	// a white point that is not 1 1 1 takes the passthrough
	dict := cos.NewDictionary()
	wp := cos.NewArray()
	wp.Add(cos.NewFloat(0.9505))
	wp.Add(cos.FloatOne)
	wp.Add(cos.NewFloat(1.089))
	dict.SetItem(cos.WhitePoint, wp)
	other := cos.NewArray()
	other.Add(cos.CalRGB)
	other.Add(dict)
	passthrough := NewPDCalRGBOfArray(other)
	got, err := passthrough.ToRGB([]float32{0.1, 0.2, 0.3})
	if err != nil {
		t.Fatal(err)
	}
	assertComponents(t, got, []float32{0.1, 0.2, 0.3})
}

// TestCalGrayIdentity checks the same two paths for CalGray.
func TestCalGrayIdentity(t *testing.T) {
	array := cos.NewArray()
	array.Add(cos.CalGray)
	array.Add(cos.NewDictionary())
	calGray := NewPDCalGrayOfArray(array)

	if got := calGray.Name(); got != "CalGray" {
		t.Errorf("Name = %q", got)
	}
	if got := calGray.NumberOfComponents(); got != 1 {
		t.Errorf("NumberOfComponents = %d", got)
	}
	if got := calGray.Gamma(); got != 1 {
		t.Errorf("the default gamma is %v, want 1", got)
	}
	black, err := calGray.ToRGB([]float32{0})
	if err != nil {
		t.Fatal(err)
	}
	assertComponents(t, black, []float32{0, 0, 0})

	// the cache returns a copy, so a caller may modify what it got
	first, _ := calGray.ToRGB([]float32{0.5})
	first[0] = 99
	second, _ := calGray.ToRGB([]float32{0.5})
	if second[0] == 99 {
		t.Error("the cached value was handed out rather than a copy of it")
	}
}

func assertRGBAt(t *testing.T, img goimage.Image, x, y, wantR, wantG, wantB int) {
	t.Helper()
	got := pixelOfImage(img, x, y)
	if got[0] != wantR || got[1] != wantG || got[2] != wantB {
		t.Errorf("pixel (%d,%d) = %v, want [%d %d %d]", x, y, got, wantR, wantG, wantB)
	}
}

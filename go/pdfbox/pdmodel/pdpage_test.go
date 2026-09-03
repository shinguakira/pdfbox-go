package pdmodel

import (
	"io"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/filter"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// Written from org.apache.pdfbox.pdmodel.PDPage. The Java suite's TestPDPage
// loads PDFs from the test corpus through PDDocument, which this port has not
// reached; these cover the same behaviour against dictionaries built by hand.

// contentStream returns a stream holding the given content, unfiltered.
func contentStream(t *testing.T, content string) *cos.Stream {
	t.Helper()
	s := cos.NewStream(filter.Provider{})
	w, err := s.CreateWriter()
	if err != nil {
		t.Fatalf("CreateWriter: %v", err)
	}
	if _, err := w.Write([]byte(content)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	return s
}

func readAll(t *testing.T, r pdfio.RandomAccessRead) string {
	t.Helper()
	b, err := io.ReadAll(pdfio.NewReader(r))
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	return string(b)
}

func TestPDPageConstruction(t *testing.T) {
	page := NewPDPage()
	if got := page.Dictionary().GetCOSName(cos.Type); got != cos.Page {
		t.Errorf("/Type = %v, want /Page", got)
	}
	if got := page.MediaBox().Width(); got != common.Letter.Width() {
		t.Errorf("a new page is %v wide, want U.S. Letter", got)
	}

	sized := NewPDPageOfSize(common.A4)
	if got := sized.MediaBox().Width(); got != common.A4.Width() {
		t.Errorf("MediaBox = %v, want A4", sized.MediaBox())
	}
}

func TestPDPageContents(t *testing.T) {
	page := NewPDPage()
	if page.HasContents() {
		t.Error("a new page has contents")
	}
	if got := readAll(t, mustContents(t, page)); got != "" {
		t.Errorf("a page with no contents read %q", got)
	}

	page.Dictionary().SetItem(cos.Contents, contentStream(t, "BT ET"))
	if !page.HasContents() {
		t.Error("the page has no contents after one was set")
	}
	if got := readAll(t, mustContents(t, page)); got != "BT ET" {
		t.Errorf("contents = %q, want %q", got, "BT ET")
	}
}

// TestPDPageContentsArray pins that several content streams are read as one,
// separated by a newline so that a token cannot run across the join.
func TestPDPageContentsArray(t *testing.T) {
	array := cos.NewArray()
	array.Add(contentStream(t, "BT"))
	array.Add(cos.NewObject(contentStream(t, "ET")))
	page := NewPDPage()
	page.Dictionary().SetItem(cos.Contents, array)

	if !page.HasContents() {
		t.Error("a page with an array of contents has none")
	}
	if got, want := readAll(t, mustContents(t, page)), "BT\nET\n"; got != want {
		t.Errorf("contents = %q, want %q", got, want)
	}
}

// TestPDPageContentsArraySkipsNonStreams pins that an entry that is not a
// stream is stepped over rather than ending the read.
func TestPDPageContentsArraySkipsNonStreams(t *testing.T) {
	array := cos.NewArray()
	array.Add(cos.NewDictionary())
	array.Add(contentStream(t, "BT"))
	page := NewPDPage()
	page.Dictionary().SetItem(cos.Contents, array)

	if got, want := readAll(t, mustContents(t, page)), "BT\n"; got != want {
		t.Errorf("contents = %q, want %q", got, want)
	}
}

func TestPDPageResources(t *testing.T) {
	page := NewPDPage()
	if page.Resources() != nil {
		t.Error("a new page has resources")
	}

	resources := NewPDResources()
	page.SetResources(resources)
	if page.Resources() != resources {
		t.Error("the page did not keep the resources it was given")
	}
	if page.Dictionary().GetCOSDictionary(cos.Resources) == nil {
		t.Error("the resources were not written into the page dictionary")
	}

	page.SetResources(nil)
	if page.Dictionary().ContainsKey(cos.Resources) {
		t.Error("/Resources was left behind after being cleared")
	}
}

// TestPDPageResourcesInherited pins that a page with no resources of its own
// takes those of the tree node above it.
func TestPDPageResourcesInherited(t *testing.T) {
	pageDict := pageNode(0)
	root := pagesNode(pageDict)
	shared := cos.NewDictionary()
	root.SetItem(cos.Resources, shared)

	page := NewPDPageOf(pageDict)
	if got := page.Resources(); got == nil || got.Dictionary() != shared {
		t.Error("the page did not inherit the resources above it")
	}
}

func TestPDPageBoxes(t *testing.T) {
	page := NewPDPageOfSize(common.NewPDRectangleOf(0, 0, 100, 100))

	// Each box defaults to the crop box, which defaults to the media box.
	for name, box := range map[string]*common.PDRectangle{
		"CropBox":  page.CropBox(),
		"BleedBox": page.BleedBox(),
		"TrimBox":  page.TrimBox(),
		"ArtBox":   page.ArtBox(),
		"BBox":     page.BBox(),
	} {
		if box.Width() != 100 || box.Height() != 100 {
			t.Errorf("%s = %v, want the media box", name, box)
		}
	}

	page.SetCropBox(common.NewPDRectangleOf(10, 10, 50, 50))
	if got := page.CropBox(); got.LowerLeftX() != 10 || got.UpperRightX() != 60 {
		t.Errorf("CropBox = %v, want [10,10,60,60]", got)
	}
	// The other boxes now follow the crop box.
	if got := page.TrimBox(); got.LowerLeftX() != 10 {
		t.Errorf("TrimBox = %v, want the crop box", got)
	}
}

// TestPDPageBoxesClipToMediaBox pins that a box reaching outside the medium is
// pulled back to it.
func TestPDPageBoxesClipToMediaBox(t *testing.T) {
	page := NewPDPageOfSize(common.NewPDRectangleOf(0, 0, 100, 100))
	page.SetCropBox(common.NewPDRectangleOf(-50, -50, 400, 400))

	got := page.CropBox()
	if got.LowerLeftX() != 0 || got.LowerLeftY() != 0 ||
		got.UpperRightX() != 100 || got.UpperRightY() != 100 {
		t.Errorf("CropBox = %v, want the media box", got)
	}
}

func TestPDPageSetBoxes(t *testing.T) {
	page := NewPDPageOfSize(common.NewPDRectangleOf(0, 0, 200, 200))
	box := common.NewPDRectangleOf(10, 10, 50, 50)

	cases := []struct {
		key  *cos.Name
		set  func(*common.PDRectangle)
		get  func() *common.PDRectangle
		name string
	}{
		{cos.BleedBox, page.SetBleedBox, page.BleedBox, "BleedBox"},
		{cos.TrimBox, page.SetTrimBox, page.TrimBox, "TrimBox"},
		{cos.ArtBox, page.SetArtBox, page.ArtBox, "ArtBox"},
	}
	for _, c := range cases {
		c.set(box)
		if got := c.get(); got.LowerLeftX() != 10 || got.UpperRightX() != 60 {
			t.Errorf("%s = %v, want [10,10,60,60]", c.name, got)
		}
		c.set(nil)
		if page.Dictionary().ContainsKey(c.key) {
			t.Errorf("/%s was left behind after being cleared", c.name)
		}
	}
}

// TestPDPageRotation pins that the angle is normalized, and that anything that
// is not a multiple of 90 is ignored rather than passed on.
func TestPDPageRotation(t *testing.T) {
	cases := []struct {
		set  int
		want int
	}{
		{0, 0},
		{90, 90},
		{450, 90},
		{-90, 270},
		{-450, 270},
		{45, 0},
	}
	for _, c := range cases {
		page := NewPDPage()
		page.SetRotation(c.set)
		if got := page.Rotation(); got != c.want {
			t.Errorf("Rotation after SetRotation(%d) = %d, want %d", c.set, got, c.want)
		}
	}

	// The angle is inherited like the boxes are.
	pageDict := pageNode(0)
	root := pagesNode(pageDict)
	root.SetInt(cos.Rotate, 180)
	if got := NewPDPageOf(pageDict).Rotation(); got != 180 {
		t.Errorf("inherited Rotation = %d, want 180", got)
	}
}

func TestPDPageMatrixIsIdentity(t *testing.T) {
	if !NewPDPage().Matrix().Equals(NewPDPage().Matrix()) {
		t.Error("two page matrices differ")
	}
	if got := NewPDPage().Matrix().ScaleX(); got != 1 {
		t.Errorf("the page matrix scales x by %v, want 1", got)
	}
}

func TestPDPageStructParents(t *testing.T) {
	page := NewPDPage()
	page.SetStructParents(4)
	if got := page.StructParents(); got != 4 {
		t.Errorf("StructParents = %d, want 4", got)
	}
}

func TestPDPageEquals(t *testing.T) {
	dict := pageNode(0)
	if !NewPDPageOf(dict).Equals(NewPDPageOf(dict)) {
		t.Error("two pages over the same dictionary are unequal")
	}
	if NewPDPage().Equals(NewPDPage()) {
		t.Error("two pages over different dictionaries are equal")
	}
	if NewPDPage().Equals(nil) {
		t.Error("Equals = true against nil")
	}
}

// mustContents reads the page's content, failing the test if it cannot.
func mustContents(t *testing.T, page *PDPage) pdfio.RandomAccessRead {
	t.Helper()
	r, err := page.ContentsForRandomAccess()
	if err != nil {
		t.Fatalf("ContentsForRandomAccess: %v", err)
	}
	return r
}

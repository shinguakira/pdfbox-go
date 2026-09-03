package pdmodel

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// Written from org.apache.pdfbox.pdmodel.PDPageTree. The Java suite's
// TestPDPageTree loads PDFs from the test corpus through PDDocument, which this
// port has not reached; these cover the same tree walking against dictionaries
// built by hand.

// pagesNode builds an intermediate node holding the given kids, with the count
// the file would carry.
func pagesNode(kids ...*cos.Dictionary) *cos.Dictionary {
	node := cos.NewDictionary()
	node.SetItem(cos.Type, cos.Pages)
	array := cos.NewArray()
	count := 0
	for _, kid := range kids {
		array.Add(kid)
		kid.SetItem(cos.Parent, node)
		if cos.Pages == kid.GetCOSName(cos.Type) {
			count += kid.GetIntDefault(cos.Count, 0)
		} else {
			count++
		}
	}
	node.SetItem(cos.Kids, array)
	node.SetInt(cos.Count, count)
	return node
}

// pageNode builds a leaf page carrying a marker so that tests can tell the
// pages apart.
func pageNode(marker int) *cos.Dictionary {
	page := cos.NewDictionary()
	page.SetItem(cos.Type, cos.Page)
	page.SetInt(cos.StructParents, marker)
	return page
}

func TestPDPageTreeConstruction(t *testing.T) {
	tree := NewPDPageTree()
	if got := tree.Count(); got != 0 {
		t.Errorf("Count = %d, want 0", got)
	}
	if got := tree.Dictionary().GetCOSName(cos.Type); got != cos.Pages {
		t.Errorf("/Type = %v, want /Pages", got)
	}
	if tree.Dictionary().GetCOSArray(cos.Kids) == nil {
		t.Error("a new tree has no /Kids")
	}
}

// TestPDPageTreeRepairsPageAsRoot covers PDFBOX-3154: a file whose page tree
// root is a page rather than a tree gets a tree wrapped around it.
func TestPDPageTreeRepairsPageAsRoot(t *testing.T) {
	tree := NewPDPageTreeOf(pageNode(7))

	if got := tree.Count(); got != 1 {
		t.Errorf("Count = %d, want 1", got)
	}
	if got := tree.Get(0).StructParents(); got != 7 {
		t.Errorf("the page held marker %d, want 7", got)
	}
}

func TestPDPageTreeNilRoot(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("no panic on a nil root, want one")
		}
	}()
	NewPDPageTreeOf(nil)
}

func TestPDPageTreeGetAndWalk(t *testing.T) {
	// A two-level tree: root -> [branch -> [0, 1], 2]
	branch := pagesNode(pageNode(0), pageNode(1))
	root := pagesNode(branch, pageNode(2))
	tree := NewPDPageTreeOf(root)

	if got := tree.Count(); got != 3 {
		t.Fatalf("Count = %d, want 3", got)
	}
	for want := 0; want < 3; want++ {
		if got := tree.Get(want).StructParents(); got != want {
			t.Errorf("Get(%d) held marker %d", want, got)
		}
	}

	var walked []int
	for page := range tree.All {
		walked = append(walked, page.StructParents())
	}
	if len(walked) != 3 || walked[0] != 0 || walked[1] != 1 || walked[2] != 2 {
		t.Errorf("the walk saw %v, want [0 1 2]", walked)
	}
}

// TestPDPageTreeAllStopsEarly pins that the walk can be broken out of, which is
// what makes it a range-over-func rather than a slice.
func TestPDPageTreeAllStopsEarly(t *testing.T) {
	tree := NewPDPageTreeOf(pagesNode(pageNode(0), pageNode(1), pageNode(2)))

	seen := 0
	for range tree.All {
		seen++
		break
	}
	if seen != 1 {
		t.Errorf("the walk yielded %d pages after a break, want 1", seen)
	}
}

func TestPDPageTreeGetOutOfBounds(t *testing.T) {
	tree := NewPDPageTreeOf(pagesNode(pageNode(0)))
	for _, index := range []int{-1, 1} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("Get(%d) did not panic", index)
				}
			}()
			tree.Get(index)
		}()
	}
}

// TestPDPageTreeRecursion covers PDFBOX-5009 and PDFBOX-3953: a tree whose node
// lists itself as its own kid must not be walked forever.
//
// The root is not put in the visited set before the walk descends from it, so a
// root listing itself is descended into a second time and its pages come back
// twice before the guard stops the third. The guard is against the stack
// overflow, not against a duplicate; a corrupt tree gets a corrupt answer.
func TestPDPageTreeRecursion(t *testing.T) {
	root := pagesNode(pageNode(0))
	root.GetCOSArray(cos.Kids).Add(root)

	tree := NewPDPageTreeOf(root)
	var walked []int
	for page := range tree.All {
		walked = append(walked, page.StructParents())
		if len(walked) > 10 {
			t.Fatal("the walk did not stop")
		}
	}
	if len(walked) != 2 {
		t.Errorf("the walk saw %v, want the one page twice", walked)
	}
}

// TestPDPageTreeMissingKids pins that a node with no /Kids is treated as having
// none rather than failing, since a malformed file may omit it.
func TestPDPageTreeMissingKids(t *testing.T) {
	root := cos.NewDictionary()
	root.SetItem(cos.Type, cos.Pages)

	var walked int
	for range NewPDPageTreeOf(root).All {
		walked++
	}
	if walked != 0 {
		t.Errorf("the walk saw %d pages, want none", walked)
	}
}

// TestPDPageTreeNullKidBecomesEmptyPage pins that a null entry in /Kids is
// replaced with an empty page rather than dropped, so the page count still
// lines up with the file.
func TestPDPageTreeNullKidBecomesEmptyPage(t *testing.T) {
	root := pagesNode(pageNode(0))
	root.GetCOSArray(cos.Kids).Add(nil)
	root.SetInt(cos.Count, 2)

	var walked int
	for range NewPDPageTreeOf(root).All {
		walked++
	}
	if walked != 2 {
		t.Errorf("the walk saw %d pages, want 2", walked)
	}
}

// TestPDPageTreeSanitizeType pins that a page with no /Type gets one, and that a
// page with the wrong /Type is refused rather than read as a page.
func TestPDPageTreeSanitizeType(t *testing.T) {
	untyped := cos.NewDictionary()
	untyped.SetInt(cos.StructParents, 5)
	root := pagesNode(untyped)

	page := NewPDPageTreeOf(root).Get(0)
	if got := page.Dictionary().GetCOSName(cos.Type); got != cos.Page {
		t.Errorf("/Type = %v, want /Page", got)
	}

	wrong := cos.NewDictionary()
	wrong.SetItem(cos.Type, cos.Font)
	wrongTree := NewPDPageTreeOf(pagesNode(wrong))
	defer func() {
		if recover() == nil {
			t.Error("a page typed /Font was accepted, want a panic")
		}
	}()
	wrongTree.Get(0)
}

func TestPDPageTreeIndexOf(t *testing.T) {
	first, second := pageNode(0), pageNode(1)
	tree := NewPDPageTreeOf(pagesNode(pagesNode(first), second))

	if got := tree.IndexOf(NewPDPageOf(second)); got != 1 {
		t.Errorf("IndexOf = %d, want 1", got)
	}
	if got := tree.IndexOf(NewPDPageOf(pageNode(9))); got != -1 {
		t.Errorf("IndexOf of a page not in the tree = %d, want -1", got)
	}
}

func TestPDPageTreeAddAndRemove(t *testing.T) {
	tree := NewPDPageTree()
	page := NewPDPage()
	tree.Add(page)

	if got := tree.Count(); got != 1 {
		t.Fatalf("Count after Add = %d, want 1", got)
	}
	if got := tree.IndexOf(page); got != 0 {
		t.Errorf("IndexOf after Add = %d, want 0", got)
	}

	tree.Remove(page)
	if got := tree.Count(); got != 0 {
		t.Errorf("Count after Remove = %d, want 0", got)
	}
}

func TestPDPageTreeInsert(t *testing.T) {
	tree := NewPDPageTree()
	first := NewPDPageOf(pageNode(1))
	tree.Add(first)

	before := NewPDPageOf(pageNode(0))
	tree.InsertBefore(before, first)
	after := NewPDPageOf(pageNode(2))
	tree.InsertAfter(after, first)

	if got := tree.Count(); got != 3 {
		t.Fatalf("Count = %d, want 3", got)
	}
	for want := 0; want < 3; want++ {
		if got := tree.Get(want).StructParents(); got != want {
			t.Errorf("Get(%d) held marker %d", want, got)
		}
	}
}

func TestPDPageTreeInsertOrphan(t *testing.T) {
	tree := NewPDPageTree()
	tree.Add(NewPDPage())

	orphan := NewPDPageOf(pageNode(9))
	orphan.Dictionary().SetItem(cos.Parent, tree.Dictionary())

	defer func() {
		if recover() == nil {
			t.Error("inserting next to an orphan was accepted, want a panic")
		}
	}()
	tree.InsertBefore(NewPDPage(), orphan)
}

func TestGetInheritableAttribute(t *testing.T) {
	page := pageNode(0)
	root := pagesNode(page)
	root.SetItem(cos.MediaBox, cos.ArrayOfFloats([]float32{0, 0, 100, 200}))

	// The page has no media box of its own, so it takes the one above it.
	if got := GetInheritableAttribute(page, cos.MediaBox); got == nil {
		t.Error("the inherited MediaBox was not found")
	}
	// A key nothing carries is simply absent.
	if got := GetInheritableAttribute(page, cos.CropBox); got != nil {
		t.Errorf("GetInheritableAttribute = %v, want nil", got)
	}
}

// TestGetInheritableAttributeStopsAtNonTree pins that inheritance climbs only
// through page tree nodes, so a page whose parent is another page does not take
// that page's attributes.
func TestGetInheritableAttributeStopsAtNonTree(t *testing.T) {
	parent := pageNode(0)
	parent.SetItem(cos.MediaBox, cos.ArrayOfFloats([]float32{0, 0, 100, 200}))
	page := pageNode(1)
	page.SetItem(cos.Parent, parent)

	if got := GetInheritableAttribute(page, cos.MediaBox); got != nil {
		t.Errorf("GetInheritableAttribute = %v, want nil — the parent is a page", got)
	}
}

// TestGetInheritableAttributeCycle pins that a parent chain pointing back at
// itself ends the climb instead of looping.
func TestGetInheritableAttributeCycle(t *testing.T) {
	node := cos.NewDictionary()
	node.SetItem(cos.Type, cos.Pages)
	node.SetItem(cos.Parent, node)

	if got := GetInheritableAttribute(node, cos.MediaBox); got != nil {
		t.Errorf("GetInheritableAttribute = %v, want nil", got)
	}
}

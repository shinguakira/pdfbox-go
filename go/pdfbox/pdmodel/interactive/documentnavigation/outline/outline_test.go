package outline

// Ported from
// pdfbox/src/test/java/org/apache/pdfbox/pdmodel/interactive/documentnavigation/outline/PDOutlineItemIteratorTest.java,
// PDDocumentOutlineTest.java and PDOutlineItemTest.java.

import "testing"

// sameItem is the assertEquals of Java over two outline items: PDDictionaryWrapper
// overrides equals to compare the dictionaries, so two wrappers over one
// dictionary are equal even though each accessor builds a new wrapper.
func sameItem(a, b *PDOutlineItem) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Equals(&b.PDDictionaryWrapper)
}

// TestOutlineItemIteratorSingleItem is PDOutlineItemIteratorTest.singleItem.
func TestOutlineItemIteratorSingleItem(t *testing.T) {
	first := NewPDOutlineItem()
	iterator := NewPDOutlineItemIterator(first)
	if !iterator.HasNext() {
		t.Fatal("HasNext() = false, want true")
	}
	if got := iterator.Next(); !sameItem(got, first) {
		t.Errorf("Next() = %v, want %v", got, first)
	}
	if iterator.HasNext() {
		t.Error("HasNext() = true, want false")
	}
}

// TestOutlineItemIteratorMultipleItem is PDOutlineItemIteratorTest.multipleItem.
func TestOutlineItemIteratorMultipleItem(t *testing.T) {
	first := NewPDOutlineItem()
	second := NewPDOutlineItem()
	first.SetNextSibling(second)
	iterator := NewPDOutlineItemIterator(first)
	if !iterator.HasNext() {
		t.Fatal("HasNext() = false, want true")
	}
	if got := iterator.Next(); !sameItem(got, first) {
		t.Errorf("Next() = %v, want %v", got, first)
	}
	if !iterator.HasNext() {
		t.Fatal("HasNext() = false, want true")
	}
	if got := iterator.Next(); !sameItem(got, second) {
		t.Errorf("Next() = %v, want %v", got, second)
	}
	if iterator.HasNext() {
		t.Error("HasNext() = true, want false")
	}
}

// TestOutlineItemIteratorRemoveUnsupported is
// PDOutlineItemIteratorTest.removeUnsupported. Java asserts
// UnsupportedOperationException, which is unchecked, so the port panics.
func TestOutlineItemIteratorRemoveUnsupported(t *testing.T) {
	pdOutlineItemIterator := NewPDOutlineItemIterator(NewPDOutlineItem())
	defer func() {
		if recover() == nil {
			t.Error("Remove() did not panic")
		}
	}()
	pdOutlineItemIterator.Remove()
}

// TestOutlineItemIteratorNoChildren is PDOutlineItemIteratorTest.noChildren.
func TestOutlineItemIteratorNoChildren(t *testing.T) {
	iterator := NewPDOutlineItemIterator(nil)
	if iterator.HasNext() {
		t.Error("HasNext() = true, want false")
	}
}

// TestOutlinesCountShouldNotBeNegative is
// PDDocumentOutlineTest.outlinesCountShouldNotBeNegative; see PDF 32000-1:2008
// table 152.
func TestOutlinesCountShouldNotBeNegative(t *testing.T) {
	outline := NewPDDocumentOutline()
	firstLevelChild := NewPDOutlineItem()
	outline.AddLast(firstLevelChild)
	secondLevelChild := NewPDOutlineItem()
	firstLevelChild.AddLast(secondLevelChild)
	if got := secondLevelChild.OpenCount(); got != 0 {
		t.Errorf("secondLevelChild.OpenCount() = %d, want 0", got)
	}
	if got := firstLevelChild.OpenCount(); got != -1 {
		t.Errorf("firstLevelChild.OpenCount() = %d, want -1", got)
	}
	if got := outline.OpenCount(); got < 0 {
		t.Errorf("Outlines count cannot be %d", got)
	}
}

// TestOutlinesCount is PDDocumentOutlineTest.outlinesCount.
func TestOutlinesCount(t *testing.T) {
	outline := NewPDDocumentOutline()
	root := NewPDOutlineItem()
	outline.AddLast(root)
	if got := outline.OpenCount(); got != 1 {
		t.Errorf("outline.OpenCount() = %d, want 1", got)
	}
	root.AddLast(NewPDOutlineItem())
	if got := root.OpenCount(); got != -1 {
		t.Errorf("root.OpenCount() = %d, want -1", got)
	}
	if got := outline.OpenCount(); got != 1 {
		t.Errorf("outline.OpenCount() = %d, want 1", got)
	}
	root.AddLast(NewPDOutlineItem())
	if got := root.OpenCount(); got != -2 {
		t.Errorf("root.OpenCount() = %d, want -2", got)
	}
	if got := outline.OpenCount(); got != 1 {
		t.Errorf("outline.OpenCount() = %d, want 1", got)
	}
	root.OpenNode()
	if got := root.OpenCount(); got != 2 {
		t.Errorf("root.OpenCount() = %d, want 2", got)
	}
	if got := outline.OpenCount(); got != 3 {
		t.Errorf("outline.OpenCount() = %d, want 3", got)
	}
}

// outlineItemFixture is the setUp of PDOutlineItemTest.
type outlineItemFixture struct {
	root       *PDOutlineItem
	first      *PDOutlineItem
	second     *PDOutlineItem
	newSibling *PDOutlineItem
}

// newOutlineItemFixture is PDOutlineItemTest.setUp.
func newOutlineItemFixture() *outlineItemFixture {
	f := &outlineItemFixture{
		root:       NewPDOutlineItem(),
		first:      NewPDOutlineItem(),
		second:     NewPDOutlineItem(),
		newSibling: NewPDOutlineItem(),
	}
	f.root.AddLast(f.first)
	f.root.AddLast(f.second)
	f.newSibling.AddLast(NewPDOutlineItem())
	f.newSibling.AddLast(NewPDOutlineItem())
	return f
}

// assertSiblingsAround checks that newSibling sits between first and second.
func (f *outlineItemFixture) assertSiblingsAround(t *testing.T) {
	t.Helper()
	if got := f.first.NextSibling(); !sameItem(got, f.newSibling) {
		t.Errorf("first.NextSibling() = %v, want %v", got, f.newSibling)
	}
	if got := f.second.PreviousSibling(); !sameItem(got, f.newSibling) {
		t.Errorf("second.PreviousSibling() = %v, want %v", got, f.newSibling)
	}
}

// TestInsertSiblingAfterOpenChildToOpenParent is
// PDOutlineItemTest.insertSiblingAfter_OpenChildToOpenParent.
func TestInsertSiblingAfterOpenChildToOpenParent(t *testing.T) {
	f := newOutlineItemFixture()
	f.newSibling.OpenNode()
	f.root.OpenNode()
	if got := f.root.OpenCount(); got != 2 {
		t.Errorf("root.OpenCount() = %d, want 2", got)
	}
	f.first.InsertSiblingAfter(f.newSibling)
	f.assertSiblingsAround(t)
	if got := f.root.OpenCount(); got != 5 {
		t.Errorf("root.OpenCount() = %d, want 5", got)
	}
}

// TestInsertSiblingBeforeOpenChildToOpenParent is
// PDOutlineItemTest.insertSiblingBefore_OpenChildToOpenParent.
func TestInsertSiblingBeforeOpenChildToOpenParent(t *testing.T) {
	f := newOutlineItemFixture()
	f.newSibling.OpenNode()
	f.root.OpenNode()
	if got := f.root.OpenCount(); got != 2 {
		t.Errorf("root.OpenCount() = %d, want 2", got)
	}
	f.second.InsertSiblingBefore(f.newSibling)
	f.assertSiblingsAround(t)
	if got := f.root.OpenCount(); got != 5 {
		t.Errorf("root.OpenCount() = %d, want 5", got)
	}
}

// TestInsertSiblingAfterOpenChildToClosedParent is
// PDOutlineItemTest.insertSiblingAfter_OpenChildToClosedParent.
func TestInsertSiblingAfterOpenChildToClosedParent(t *testing.T) {
	f := newOutlineItemFixture()
	f.newSibling.OpenNode()
	if got := f.root.OpenCount(); got != -2 {
		t.Errorf("root.OpenCount() = %d, want -2", got)
	}
	f.first.InsertSiblingAfter(f.newSibling)
	f.assertSiblingsAround(t)
	if got := f.root.OpenCount(); got != -5 {
		t.Errorf("root.OpenCount() = %d, want -5", got)
	}
}

// TestInsertSiblingBeforeOpenChildToClosedParent is
// PDOutlineItemTest.insertSiblingBefore_OpenChildToClosedParent.
func TestInsertSiblingBeforeOpenChildToClosedParent(t *testing.T) {
	f := newOutlineItemFixture()
	f.newSibling.OpenNode()
	if got := f.root.OpenCount(); got != -2 {
		t.Errorf("root.OpenCount() = %d, want -2", got)
	}
	f.second.InsertSiblingBefore(f.newSibling)
	f.assertSiblingsAround(t)
	if got := f.root.OpenCount(); got != -5 {
		t.Errorf("root.OpenCount() = %d, want -5", got)
	}
}

// TestInsertSiblingAfterClosedChildToOpenParent is
// PDOutlineItemTest.insertSiblingAfter_ClosedChildToOpenParent.
func TestInsertSiblingAfterClosedChildToOpenParent(t *testing.T) {
	f := newOutlineItemFixture()
	f.root.OpenNode()
	if got := f.root.OpenCount(); got != 2 {
		t.Errorf("root.OpenCount() = %d, want 2", got)
	}
	f.first.InsertSiblingAfter(f.newSibling)
	f.assertSiblingsAround(t)
	if got := f.root.OpenCount(); got != 3 {
		t.Errorf("root.OpenCount() = %d, want 3", got)
	}
}

// TestInsertSiblingBeforeClosedChildToOpenParent is
// PDOutlineItemTest.insertSiblingBefore_ClosedChildToOpenParent.
func TestInsertSiblingBeforeClosedChildToOpenParent(t *testing.T) {
	f := newOutlineItemFixture()
	f.root.OpenNode()
	if got := f.root.OpenCount(); got != 2 {
		t.Errorf("root.OpenCount() = %d, want 2", got)
	}
	f.second.InsertSiblingBefore(f.newSibling)
	f.assertSiblingsAround(t)
	if got := f.root.OpenCount(); got != 3 {
		t.Errorf("root.OpenCount() = %d, want 3", got)
	}
}

// TestInsertSiblingAfterClosedChildToClosedParent is
// PDOutlineItemTest.insertSiblingAfter_ClosedChildToClosedParent.
func TestInsertSiblingAfterClosedChildToClosedParent(t *testing.T) {
	f := newOutlineItemFixture()
	if got := f.root.OpenCount(); got != -2 {
		t.Errorf("root.OpenCount() = %d, want -2", got)
	}
	f.first.InsertSiblingAfter(f.newSibling)
	f.assertSiblingsAround(t)
	if got := f.root.OpenCount(); got != -3 {
		t.Errorf("root.OpenCount() = %d, want -3", got)
	}
}

// TestInsertSiblingBeforeClosedChildToClosedParent is
// PDOutlineItemTest.insertSiblingBefore_ClosedChildToClosedParent.
func TestInsertSiblingBeforeClosedChildToClosedParent(t *testing.T) {
	f := newOutlineItemFixture()
	if got := f.root.OpenCount(); got != -2 {
		t.Errorf("root.OpenCount() = %d, want -2", got)
	}
	f.second.InsertSiblingBefore(f.newSibling)
	f.assertSiblingsAround(t)
	if got := f.root.OpenCount(); got != -3 {
		t.Errorf("root.OpenCount() = %d, want -3", got)
	}
}

// TestInsertSiblingTop is PDOutlineItemTest.insertSiblingTop.
func TestInsertSiblingTop(t *testing.T) {
	f := newOutlineItemFixture()
	if got := f.root.FirstChild(); !sameItem(got, f.first) {
		t.Errorf("root.FirstChild() = %v, want %v", got, f.first)
	}
	newSibling := NewPDOutlineItem()
	f.first.InsertSiblingBefore(newSibling)
	if got := f.first.PreviousSibling(); !sameItem(got, newSibling) {
		t.Errorf("first.PreviousSibling() = %v, want %v", got, newSibling)
	}
	if got := f.root.FirstChild(); !sameItem(got, newSibling) {
		t.Errorf("root.FirstChild() = %v, want %v", got, newSibling)
	}
}

// TestInsertSiblingTopNoParent is PDOutlineItemTest.insertSiblingTopNoParent.
func TestInsertSiblingTopNoParent(t *testing.T) {
	f := newOutlineItemFixture()
	if got := f.root.FirstChild(); !sameItem(got, f.first) {
		t.Errorf("root.FirstChild() = %v, want %v", got, f.first)
	}
	newSibling := NewPDOutlineItem()
	f.root.InsertSiblingBefore(newSibling)
	if got := f.root.PreviousSibling(); !sameItem(got, newSibling) {
		t.Errorf("root.PreviousSibling() = %v, want %v", got, newSibling)
	}
}

// TestInsertSiblingBottom is PDOutlineItemTest.insertSiblingBottom.
func TestInsertSiblingBottom(t *testing.T) {
	f := newOutlineItemFixture()
	if got := f.root.LastChild(); !sameItem(got, f.second) {
		t.Errorf("root.LastChild() = %v, want %v", got, f.second)
	}
	newSibling := NewPDOutlineItem()
	f.second.InsertSiblingAfter(newSibling)
	if got := f.second.NextSibling(); !sameItem(got, newSibling) {
		t.Errorf("second.NextSibling() = %v, want %v", got, newSibling)
	}
	if got := f.root.LastChild(); !sameItem(got, newSibling) {
		t.Errorf("root.LastChild() = %v, want %v", got, newSibling)
	}
}

// TestInsertSiblingBottomNoParent is
// PDOutlineItemTest.insertSiblingBottomNoParent.
func TestInsertSiblingBottomNoParent(t *testing.T) {
	f := newOutlineItemFixture()
	if got := f.root.LastChild(); !sameItem(got, f.second) {
		t.Errorf("root.LastChild() = %v, want %v", got, f.second)
	}
	newSibling := NewPDOutlineItem()
	f.root.InsertSiblingAfter(newSibling)
	if got := f.root.NextSibling(); !sameItem(got, newSibling) {
		t.Errorf("root.NextSibling() = %v, want %v", got, newSibling)
	}
}

// TestCannotInsertSiblingBeforeAList is
// PDOutlineItemTest.cannotInsertSiblingBeforeAList. Java asserts
// IllegalArgumentException, which is unchecked, so the port panics.
func TestCannotInsertSiblingBeforeAList(t *testing.T) {
	f := newOutlineItemFixture()
	child := NewPDOutlineItem()
	child.InsertSiblingAfter(NewPDOutlineItem())
	child.InsertSiblingAfter(NewPDOutlineItem())
	defer func() {
		if recover() == nil {
			t.Error("InsertSiblingBefore() did not panic")
		}
	}()
	f.root.InsertSiblingBefore(child)
}

// TestCannotInsertSiblingAfterAList is
// PDOutlineItemTest.cannotInsertSiblingAfterAList.
func TestCannotInsertSiblingAfterAList(t *testing.T) {
	f := newOutlineItemFixture()
	child := NewPDOutlineItem()
	child.InsertSiblingAfter(NewPDOutlineItem())
	child.InsertSiblingAfter(NewPDOutlineItem())
	defer func() {
		if recover() == nil {
			t.Error("InsertSiblingAfter() did not panic")
		}
	}()
	f.root.InsertSiblingAfter(child)
}

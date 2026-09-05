package outline

// Ported from
// pdfbox/src/test/java/org/apache/pdfbox/pdmodel/interactive/documentnavigation/outline/PDOutlineNodeTest.java.

import "testing"

// assertOpenCounts checks the open count of a node and of its child, which the
// Java test asserts as a pair over and over.
func assertOpenCounts(t *testing.T, root, child *PDOutlineItem, wantRoot, wantChild int) {
	t.Helper()
	if got := root.OpenCount(); got != wantRoot {
		t.Errorf("root.OpenCount() = %d, want %d", got, wantRoot)
	}
	if got := child.OpenCount(); got != wantChild {
		t.Errorf("child.OpenCount() = %d, want %d", got, wantChild)
	}
}

// TestNodeGetParent is PDOutlineNodeTest.getParent.
func TestNodeGetParent(t *testing.T) {
	root := NewPDOutlineItem()
	child := NewPDOutlineItem()
	root.AddLast(child)
	outline := NewPDDocumentOutline()
	outline.AddLast(root)
	if got := outline.Parent(); got != nil {
		t.Errorf("outline.Parent() = %v, want nil", got)
	}
	if got := root.Parent(); got == nil ||
		got.Dictionary() != outline.Dictionary() {
		t.Errorf("root.Parent() = %v, want %v", got, outline)
	}
	if got := child.Parent(); got == nil || got.Dictionary() != root.Dictionary() {
		t.Errorf("child.Parent() = %v, want %v", got, root)
	}
}

// TestNodeNullLastChild is PDOutlineNodeTest.nullLastChild.
func TestNodeNullLastChild(t *testing.T) {
	if got := NewPDOutlineItem().LastChild(); got != nil {
		t.Errorf("LastChild() = %v, want nil", got)
	}
}

// TestNodeNullFirstChild is PDOutlineNodeTest.nullFirstChild.
func TestNodeNullFirstChild(t *testing.T) {
	if got := NewPDOutlineItem().FirstChild(); got != nil {
		t.Errorf("FirstChild() = %v, want nil", got)
	}
}

// TestOpenAlreadyOpenedRootNode is PDOutlineNodeTest.openAlreadyOpenedRootNode.
func TestOpenAlreadyOpenedRootNode(t *testing.T) {
	root := NewPDOutlineItem()
	child := NewPDOutlineItem()
	if got := root.OpenCount(); got != 0 {
		t.Errorf("root.OpenCount() = %d, want 0", got)
	}
	root.AddLast(child)
	root.OpenNode()
	if !root.IsNodeOpen() {
		t.Error("IsNodeOpen() = false, want true")
	}
	if got := root.OpenCount(); got != 1 {
		t.Errorf("root.OpenCount() = %d, want 1", got)
	}
	root.OpenNode()
	if !root.IsNodeOpen() {
		t.Error("IsNodeOpen() = false, want true")
	}
	if got := root.OpenCount(); got != 1 {
		t.Errorf("root.OpenCount() = %d, want 1", got)
	}
}

// TestCloseAlreadyClosedRootNode is
// PDOutlineNodeTest.closeAlreadyClosedRootNode.
func TestCloseAlreadyClosedRootNode(t *testing.T) {
	root := NewPDOutlineItem()
	child := NewPDOutlineItem()
	if got := root.OpenCount(); got != 0 {
		t.Errorf("root.OpenCount() = %d, want 0", got)
	}
	root.AddLast(child)
	root.OpenNode()
	root.CloseNode()
	if root.IsNodeOpen() {
		t.Error("IsNodeOpen() = true, want false")
	}
	if got := root.OpenCount(); got != -1 {
		t.Errorf("root.OpenCount() = %d, want -1", got)
	}
	root.CloseNode()
	if root.IsNodeOpen() {
		t.Error("IsNodeOpen() = true, want false")
	}
	if got := root.OpenCount(); got != -1 {
		t.Errorf("root.OpenCount() = %d, want -1", got)
	}
}

// TestOpenLeaf is PDOutlineNodeTest.openLeaf.
func TestOpenLeaf(t *testing.T) {
	root := NewPDOutlineItem()
	child := NewPDOutlineItem()
	root.AddLast(child)
	child.OpenNode()
	if child.IsNodeOpen() {
		t.Error("IsNodeOpen() = true, want false")
	}
}

// TestNodeClosedByDefault is PDOutlineNodeTest.nodeClosedByDefault.
func TestNodeClosedByDefault(t *testing.T) {
	root := NewPDOutlineItem()
	child := NewPDOutlineItem()
	root.AddLast(child)
	if root.IsNodeOpen() {
		t.Error("IsNodeOpen() = true, want false")
	}
	if got := root.OpenCount(); got != -1 {
		t.Errorf("root.OpenCount() = %d, want -1", got)
	}
}

// TestCloseNodeWithOpendParent is PDOutlineNodeTest.closeNodeWithOpendParent.
func TestCloseNodeWithOpendParent(t *testing.T) {
	root := NewPDOutlineItem()
	child := NewPDOutlineItem()
	child.AddLast(NewPDOutlineItem())
	child.AddLast(NewPDOutlineItem())
	child.OpenNode()
	root.AddLast(child)
	root.OpenNode()
	assertOpenCounts(t, root, child, 3, 2)
	child.CloseNode()
	assertOpenCounts(t, root, child, 1, -2)
}

// TestCloseNodeWithClosedParent is PDOutlineNodeTest.closeNodeWithClosedParent.
func TestCloseNodeWithClosedParent(t *testing.T) {
	root := NewPDOutlineItem()
	child := NewPDOutlineItem()
	child.AddLast(NewPDOutlineItem())
	child.AddLast(NewPDOutlineItem())
	child.OpenNode()
	root.AddLast(child)
	assertOpenCounts(t, root, child, -3, 2)
	child.CloseNode()
	assertOpenCounts(t, root, child, -1, -2)
}

// TestOpenNodeWithOpendParent is PDOutlineNodeTest.openNodeWithOpendParent.
func TestOpenNodeWithOpendParent(t *testing.T) {
	root := NewPDOutlineItem()
	child := NewPDOutlineItem()
	child.AddLast(NewPDOutlineItem())
	child.AddLast(NewPDOutlineItem())
	root.AddLast(child)
	root.OpenNode()
	assertOpenCounts(t, root, child, 1, -2)
	child.OpenNode()
	assertOpenCounts(t, root, child, 3, 2)
}

// TestOpenNodeWithClosedParent is PDOutlineNodeTest.openNodeWithClosedParent.
func TestOpenNodeWithClosedParent(t *testing.T) {
	root := NewPDOutlineItem()
	child := NewPDOutlineItem()
	child.AddLast(NewPDOutlineItem())
	child.AddLast(NewPDOutlineItem())
	root.AddLast(child)
	assertOpenCounts(t, root, child, -1, -2)
	child.OpenNode()
	assertOpenCounts(t, root, child, -3, 2)
}

// TestAddLastSingleChild is PDOutlineNodeTest.addLastSingleChild.
func TestAddLastSingleChild(t *testing.T) {
	root := NewPDOutlineItem()
	child := NewPDOutlineItem()
	root.AddLast(child)
	if got := root.FirstChild(); !sameItem(child, got) {
		t.Errorf("FirstChild() = %v, want %v", got, child)
	}
	if got := root.LastChild(); !sameItem(child, got) {
		t.Errorf("LastChild() = %v, want %v", got, child)
	}
}

// TestAddFirstSingleChild is PDOutlineNodeTest.addFirstSingleChild.
func TestAddFirstSingleChild(t *testing.T) {
	root := NewPDOutlineItem()
	child := NewPDOutlineItem()
	root.AddFirst(child)
	if got := root.FirstChild(); !sameItem(child, got) {
		t.Errorf("FirstChild() = %v, want %v", got, child)
	}
	if got := root.LastChild(); !sameItem(child, got) {
		t.Errorf("LastChild() = %v, want %v", got, child)
	}
}

// addChildCase is one of the eight addLast/addFirst tests, which differ only in
// which end the child goes on, whether either node is open, and the counts.
type addChildCase struct {
	name string

	// first says whether the child is added with addFirst rather than addLast.
	first bool

	// openChild and openParent say which nodes are opened before the add.
	openChild  bool
	openParent bool

	// beforeRoot, beforeChild and afterRoot are the three open counts asserted.
	beforeRoot  int
	beforeChild int
	afterRoot   int
}

// TestAddChild is the eight addLast/addFirst tests of PDOutlineNodeTest.
func TestAddChild(t *testing.T) {
	for _, c := range []addChildCase{
		{"addLastOpenChildToOpenParent", false, true, true, 1, 2, 4},
		{"addFirstOpenChildToOpenParent", true, true, true, 1, 2, 4},
		{"addLastOpenChildToClosedParent", false, true, false, -1, 2, -4},
		{"addFirstOpenChildToClosedParent", true, true, false, -1, 2, -4},
		{"addLastClosedChildToOpenParent", false, false, true, 1, -2, 2},
		{"addFirstClosedChildToOpenParent", true, false, true, 1, -2, 2},
		{"addLastClosedChildToClosedParent", false, false, false, -1, -2, -2},
		{"addFirstClosedChildToClosedParent", true, false, false, -1, -2, -2},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := NewPDOutlineItem()
			child := NewPDOutlineItem()
			add := root.AddLast
			addToChild := child.AddLast
			if c.first {
				add = root.AddFirst
				addToChild = child.AddFirst
			}
			addToChild(NewPDOutlineItem())
			addToChild(NewPDOutlineItem())
			if c.openChild {
				child.OpenNode()
			}
			add(NewPDOutlineItem())
			if c.openParent {
				root.OpenNode()
			}
			assertOpenCounts(t, root, child, c.beforeRoot, c.beforeChild)
			add(child)
			// addLast asserts the child is the last and not the first; addFirst
			// asserts the other way round.
			near, far := root.FirstChild(), root.LastChild()
			nearName, farName := "FirstChild", "LastChild"
			if !c.first {
				near, far = far, near
				nearName, farName = farName, nearName
			}
			if !sameItem(child, near) {
				t.Errorf("root.%s() = %v, want %v", nearName, near, child)
			}
			if sameItem(child, far) {
				t.Errorf("root.%s() = %v, want anything else", farName, far)
			}
			if got := root.OpenCount(); got != c.afterRoot {
				t.Errorf("root.OpenCount() = %d, want %d", got, c.afterRoot)
			}
		})
	}
}

// TestCannotAddLastAList is PDOutlineNodeTest.cannotAddLastAList. Java asserts
// IllegalArgumentException, which is unchecked, so the port panics.
func TestCannotAddLastAList(t *testing.T) {
	root := NewPDOutlineItem()
	child := NewPDOutlineItem()
	child.InsertSiblingAfter(NewPDOutlineItem())
	child.InsertSiblingAfter(NewPDOutlineItem())
	defer func() {
		if recover() == nil {
			t.Error("AddLast() did not panic")
		}
	}()
	root.AddLast(child)
}

// TestCannotAddFirstAList is PDOutlineNodeTest.cannotAddFirstAList.
func TestCannotAddFirstAList(t *testing.T) {
	root := NewPDOutlineItem()
	child := NewPDOutlineItem()
	child.InsertSiblingAfter(NewPDOutlineItem())
	child.InsertSiblingAfter(NewPDOutlineItem())
	defer func() {
		if recover() == nil {
			t.Error("AddFirst() did not panic")
		}
	}()
	root.AddFirst(child)
}

// TestEqualsNode is PDOutlineNodeTest.equalsNode.
func TestEqualsNode(t *testing.T) {
	root := NewPDOutlineItem()
	root.AddFirst(NewPDOutlineItem())
	if !sameItem(root.FirstChild(), root.LastChild()) {
		t.Errorf("FirstChild() = %v, LastChild() = %v, want equal",
			root.FirstChild(), root.LastChild())
	}
}

// TestNodeIterator is PDOutlineNodeTest.iterator.
func TestNodeIterator(t *testing.T) {
	root := NewPDOutlineItem()
	first := NewPDOutlineItem()
	root.AddFirst(first)
	root.AddLast(NewPDOutlineItem())
	second := NewPDOutlineItem()
	first.InsertSiblingAfter(second)
	counter := 0
	for range root.Children() {
		counter++
	}
	if counter != 3 {
		t.Errorf("counter = %d, want 3", counter)
	}
}

// TestNodeIteratorNoChildren is PDOutlineNodeTest.iteratorNoChildre.
func TestNodeIteratorNoChildren(t *testing.T) {
	counter := 0
	for range NewPDOutlineItem().Children() {
		counter++
	}
	if counter != 0 {
		t.Errorf("counter = %d, want 0", counter)
	}
}

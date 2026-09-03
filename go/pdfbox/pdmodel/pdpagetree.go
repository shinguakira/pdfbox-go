package pdmodel

import (
	"fmt"
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// PDPageTree is the page tree, which defines the ordering of pages in the
// document in an efficient manner.
//
// Port of org.apache.pdfbox.pdmodel.PDPageTree.
//
// Java's reading constructor also takes the PDDocument, only so that a page can
// be handed the document's ResourceCache. Neither is ported yet, so neither is
// the parameter. See migration/STATUS.md.
type PDPageTree struct {
	root *cos.Dictionary

	// pageSet collects the nodes a search has been through, so that a tree
	// pointing back at itself is caught instead of overflowing the stack.
	pageSet map[*cos.Dictionary]bool
}

var _ common.COSObjectable = (*PDPageTree)(nil)

// NewPDPageTree returns an empty page tree, for embedding.
func NewPDPageTree() *PDPageTree {
	root := cos.NewDictionary()
	root.SetItem(cos.Type, cos.Pages)
	root.SetItem(cos.Kids, cos.NewArray())
	root.SetItem(cos.Count, cos.IntegerZero)
	return &PDPageTree{root: root, pageSet: map[*cos.Dictionary]bool{}}
}

// NewPDPageTreeOf returns the page tree under the given root, for reading.
func NewPDPageTreeOf(root *cos.Dictionary) *PDPageTree {
	if root == nil {
		panic("pdmodel: page tree root cannot be null")
	}
	tree := &PDPageTree{pageSet: map[*cos.Dictionary]bool{}}
	// repair bad PDFs which contain a Page dict instead of a page tree, see PDFBOX-3154
	if cos.Page == root.GetCOSName(cos.Type) {
		kids := cos.NewArray()
		kids.Add(root)
		tree.root = cos.NewDictionary()
		tree.root.SetItem(cos.Kids, kids)
		tree.root.SetInt(cos.Count, 1)
	} else {
		tree.root = root
	}
	return tree
}

// GetInheritableAttribute returns the given attribute, inheriting from parent
// tree nodes if necessary.
func GetInheritableAttribute(node *cos.Dictionary, key *cos.Name) cos.Base {
	return getInheritableAttribute(node, key, map[*cos.Dictionary]bool{})
}

func getInheritableAttribute(node *cos.Dictionary, key *cos.Name, visited map[*cos.Dictionary]bool) cos.Base {
	if visited[node] {
		return nil
	}
	visited[node] = true

	if value := node.GetDictionaryObject(key); value != nil {
		return value
	}
	parent := node.GetCOSDictionary2(cos.Parent, cos.P)
	if parent != nil && cos.Pages == parent.GetCOSName(cos.Type) {
		return getInheritableAttribute(parent, key, visited)
	}
	return nil
}

// COSObject returns the dictionary at the root of the tree.
func (t *PDPageTree) COSObject() cos.Base { return t.root }

// Dictionary returns the dictionary at the root of the tree.
func (t *PDPageTree) Dictionary() *cos.Dictionary { return t.root }

// kids returns the kids of a node, working around malformed PDFs.
func (t *PDPageTree) kids(node *cos.Dictionary) []*cos.Dictionary {
	kids := node.GetCOSArray(cos.Kids)
	if kids == nil {
		// probably a malformed PDF
		return nil
	}

	size := kids.Size()
	result := make([]*cos.Dictionary, 0, size)
	for i := 0; i < size; i++ {
		base := kids.GetObject(i)
		if dict, ok := base.(*cos.Dictionary); ok {
			result = append(result, dict)
			continue
		}
		if base == nil {
			slog.Warn("replaced null entry with an empty page")
			emptyPage := cos.NewDictionary()
			emptyPage.SetItem(cos.Type, cos.Page)
			kids.Set(i, emptyPage)
			result = append(result, emptyPage)
			continue
		}
		slog.Warn("COSDictionary expected", "got", fmt.Sprintf("%T", base))
	}
	return result
}

// All walks every page in the tree, in order.
//
// Port of the iterator PDPageTree hands out as an Iterable. As in Java the
// whole tree is walked up front, so the pages are settled before the first one
// is handed back.
func (t *PDPageTree) All(yield func(*PDPage) bool) {
	var queue []*cos.Dictionary
	visited := map[*cos.Dictionary]bool{}

	var enqueueKids func(node *cos.Dictionary)
	enqueueKids = func(node *cos.Dictionary) {
		if t.isPageTreeNode(node) {
			for _, kid := range t.kids(node) {
				if visited[kid] {
					// PDFBOX-5009, PDFBOX-3953: prevent stack overflow with malformed PDFs
					slog.Error("this page tree node has already been visited")
					continue
				} else if kid.ContainsKey(cos.Kids) {
					visited[kid] = true
				}
				enqueueKids(kid)
			}
			return
		}
		if node != nil && cos.Page == node.GetCOSName(cos.Type) {
			queue = append(queue, node)
			return
		}
		if node == nil {
			slog.Error("page skipped due to an invalid or missing type", "type", "(null)")
		} else {
			slog.Error("page skipped due to an invalid or missing type",
				"type", node.GetCOSName(cos.Type))
		}
	}
	enqueueKids(t.root)

	for _, next := range queue {
		sanitizeType(next)
		if !yield(NewPDPageOf(next)) {
			return
		}
	}
}

// Get returns the page at the given zero-based index.
//
// An index past the end, or one that lands on something that is not a page, is
// a caller's or a file's error that Java reports with an unchecked exception;
// the port panics rather than putting an error on every page lookup.
func (t *PDPageTree) Get(index int) *PDPage {
	dict := t.get(index+1, t.root, 0)
	sanitizeType(dict)
	return NewPDPageOf(dict)
}

// sanitizeType fills in a missing type and rejects one that is not Page.
func sanitizeType(dictionary *cos.Dictionary) {
	typ := dictionary.GetCOSName(cos.Type)
	if typ == nil {
		dictionary.SetItem(cos.Type, cos.Page)
		return
	}
	if cos.Page != typ {
		panic(fmt.Sprintf("pdmodel: expected 'Page' but found %v", typ))
	}
}

// get returns the COS page with the given 1-based number, using a depth-first
// search from node, having already passed the given number of pages.
func (t *PDPageTree) get(pageNum int, node *cos.Dictionary, encountered int) *cos.Dictionary {
	if pageNum < 1 {
		panic(fmt.Sprintf("pdmodel: index out of bounds: %d", pageNum))
	}
	if t.pageSet[node] {
		clear(t.pageSet)
		panic(fmt.Sprintf("pdmodel: possible recursion found when searching for page %d", pageNum))
	}
	// collect already processed pages to detect possible recursions
	// to avoid a stack overflow
	t.pageSet[node] = true

	if !t.isPageTreeNode(node) {
		if encountered == pageNum {
			clear(t.pageSet)
			return node
		}
		panic(fmt.Sprintf("pdmodel: 1-based index not found: %d", pageNum))
	}

	count := node.GetIntDefault(cos.Count, 0)
	if pageNum > encountered+count {
		panic(fmt.Sprintf("pdmodel: 1-based index out of bounds: %d", pageNum))
	}
	// it's a kid of this node
	for _, kid := range t.kids(node) {
		// which kid?
		if t.isPageTreeNode(kid) {
			kidCount := kid.GetIntDefault(cos.Count, 0)
			if pageNum <= encountered+kidCount {
				// it's this kid
				return t.get(pageNum, kid, encountered)
			}
			encountered += kidCount
			continue
		}
		// single page
		encountered++
		if pageNum == encountered {
			// it's this page
			return t.get(pageNum, kid, encountered)
		}
	}
	panic(fmt.Sprintf("pdmodel: 1-based index not found: %d", pageNum))
}

// isPageTreeNode reports whether the node is an intermediate node of the tree.
func (t *PDPageTree) isPageTreeNode(node *cos.Dictionary) bool {
	// some files such as PDFBOX-2250-229205.pdf don't have Pages set as the
	// Type, so we have to check for the presence of Kids too
	return node != nil &&
		(cos.Pages == node.GetCOSName(cos.Type) || node.ContainsKey(cos.Kids))
}

// IndexOf returns the zero-based index of the given page, or -1 if the page is
// not found.
func (t *PDPageTree) IndexOf(page *PDPage) int {
	context := &searchContext{searched: page.Dictionary(), index: -1}
	if t.findPage(context, t.root) {
		return context.index
	}
	return -1
}

func (t *PDPageTree) findPage(context *searchContext, node *cos.Dictionary) bool {
	for _, kid := range t.kids(node) {
		if context.found {
			break
		}
		if t.isPageTreeNode(kid) {
			t.findPage(context, kid)
		} else {
			context.visitPage(kid)
		}
	}
	return context.found
}

type searchContext struct {
	searched *cos.Dictionary
	index    int
	found    bool
}

func (c *searchContext) visitPage(current *cos.Dictionary) {
	c.index++
	c.found = c.searched == current
}

// Count returns the number of leaf nodes (page objects) that are descendants of
// this root within the page tree, or 0 if not present.
func (t *PDPageTree) Count() int {
	return t.root.GetIntDefault(cos.Count, 0)
}

// RemoveAt removes the page with the given zero-based index from the page tree.
func (t *PDPageTree) RemoveAt(index int) {
	t.removeNode(t.get(index+1, t.root, 0))
}

// Remove removes the given page from the page tree.
func (t *PDPageTree) Remove(page *PDPage) {
	t.removeNode(page.Dictionary())
}

// removeNode removes the given COS page.
func (t *PDPageTree) removeNode(node *cos.Dictionary) {
	// remove from parent's kids
	parent := node.GetCOSDictionary2(cos.Parent, cos.P)
	kids := parent.GetCOSArray(cos.Kids)
	if kids.RemoveObject(node) {
		// update ancestor counts
		for {
			node = node.GetCOSDictionary2(cos.Parent, cos.P)
			if node == nil {
				break
			}
			node.SetInt(cos.Count, node.GetInt(cos.Count)-1)
		}
	}
}

// Add adds the given page to this page tree.
func (t *PDPageTree) Add(page *PDPage) {
	// set parent
	node := page.Dictionary()
	node.SetItem(cos.Parent, t.root)

	// todo: re-balance tree? (or at least group new pages into tree nodes of e.g. 20)

	// add to parent's kids
	t.root.GetCOSArray(cos.Kids).Add(node)

	// update ancestor counts
	for {
		node = node.GetCOSDictionary2(cos.Parent, cos.P)
		if node == nil {
			break
		}
		node.SetInt(cos.Count, node.GetInt(cos.Count)+1)
	}
}

// InsertBefore inserts a page before another page within a page tree.
//
// Inserting next to a page that is not in a tree is a caller's mistake, which
// Java reports with IllegalArgumentException.
func (t *PDPageTree) InsertBefore(newPage, nextPage *PDPage) {
	t.insert(newPage, nextPage, 0)
}

// InsertAfter inserts a page after another page within a page tree.
func (t *PDPageTree) InsertAfter(newPage, prevPage *PDPage) {
	t.insert(newPage, prevPage, 1)
}

// insert puts newPage next to neighbour, at the given offset from it. Java
// writes the two directions out separately; they differ only in that offset.
func (t *PDPageTree) insert(newPage, neighbour *PDPage, offset int) {
	parentDict := neighbour.Dictionary().GetCOSDictionary2(cos.Parent, cos.P)
	kids := parentDict.GetCOSArray(cos.Kids)
	found := false
	for i := 0; i < kids.Size(); i++ {
		// Java casts here, so a kid that is not a dictionary ends the walk with
		// a ClassCastException rather than being stepped over.
		pageDict := kids.GetObject(i).(*cos.Dictionary)
		if pageDict == neighbour.Dictionary() {
			kids.AddAt(i+offset, newPage.Dictionary())
			newPage.Dictionary().SetItem(cos.Parent, parentDict)
			found = true
			break
		}
	}
	if !found {
		panic("pdmodel: attempted to insert before orphan page")
	}
	t.increaseParents(parentDict)
}

func (t *PDPageTree) increaseParents(parentDict *cos.Dictionary) {
	for parentDict != nil {
		cnt := parentDict.GetInt(cos.Count)
		parentDict.SetInt(cos.Count, cnt+1)
		parentDict = parentDict.GetCOSDictionary2(cos.Parent, cos.P)
	}
}

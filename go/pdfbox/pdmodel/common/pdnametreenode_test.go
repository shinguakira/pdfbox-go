package common

// Ported from
// pdfbox/src/test/java/org/apache/pdfbox/pdmodel/common/TestPDNameTreeNode.java,
// together with the PDIntegerNameTreeNode it drives, which is a test class in
// the same Java package.

import (
	"fmt"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// pdIntegerNameTreeNode is
// pdfbox/src/test/java/org/apache/pdfbox/pdmodel/common/PDIntegerNameTreeNode.java.
type pdIntegerNameTreeNode struct {
	PDNameTreeNode[*cos.Integer]
}

var _ NameTreeNode[*cos.Integer] = (*pdIntegerNameTreeNode)(nil)

func newPDIntegerNameTreeNode() *pdIntegerNameTreeNode {
	n := &pdIntegerNameTreeNode{}
	n.InitNameTreeNode(n)
	return n
}

func newPDIntegerNameTreeNodeOf(dic *cos.Dictionary) *pdIntegerNameTreeNode {
	n := &pdIntegerNameTreeNode{}
	n.InitNameTreeNodeOf(n, dic)
	return n
}

func (n *pdIntegerNameTreeNode) ConvertCOSToPD(base cos.Base) (*cos.Integer, error) {
	if base == nil {
		return nil, nil
	}
	value, ok := base.(*cos.Integer)
	if !ok {
		return nil, fmt.Errorf("integer expected here, but got %v", base)
	}
	return value, nil
}

func (n *pdIntegerNameTreeNode) CreateChildNode(dic *cos.Dictionary) NameTreeNode[*cos.Integer] {
	return newPDIntegerNameTreeNodeOf(dic)
}

// nameTreeFixture is the @BeforeEach of TestPDNameTreeNode.
type nameTreeFixture struct {
	node1, node2, node4, node5, node24 *pdIntegerNameTreeNode
}

func setUpNameTree(t *testing.T) nameTreeFixture {
	t.Helper()
	var f nameTreeFixture

	f.node5 = newPDIntegerNameTreeNode()
	names := map[string]*cos.Integer{
		"Actinium":  cos.GetInteger(89),
		"Aluminum":  cos.GetInteger(13),
		"Americium": cos.GetInteger(95),
		"Antimony":  cos.GetInteger(51),
		"Argon":     cos.GetInteger(18),
		"Arsenic":   cos.GetInteger(33),
		"Astatine":  cos.GetInteger(85),
	}
	f.node5.SetNames(names)

	f.node24 = newPDIntegerNameTreeNode()
	names = map[string]*cos.Integer{
		"Xenon":     cos.GetInteger(54),
		"Ytterbium": cos.GetInteger(70),
		"Yttrium":   cos.GetInteger(39),
		"Zinc":      cos.GetInteger(30),
		"Zirconium": cos.GetInteger(40),
	}
	f.node24.SetNames(names)

	f.node2 = newPDIntegerNameTreeNode()
	kids := kidsOrEmpty(f.node2)
	kids = append(kids, f.node5)
	f.node2.SetKids(kids)

	f.node4 = newPDIntegerNameTreeNode()
	kids = kidsOrEmpty(f.node4)
	kids = append(kids, f.node24)
	f.node4.SetKids(kids)

	f.node1 = newPDIntegerNameTreeNode()
	kids = kidsOrEmpty(f.node1)
	kids = append(kids, f.node2, f.node4)
	f.node1.SetKids(kids)

	return f
}

// kidsOrEmpty is the `if (kids == null) kids = new COSArrayList<>();` the Java
// test repeats before every add.
func kidsOrEmpty(n *pdIntegerNameTreeNode) []NameTreeNode[*cos.Integer] {
	if kids := n.Kids(); kids != nil {
		return kids.ToSlice()
	}
	return nil
}

func TestNameTreeUpperLimit(t *testing.T) {
	f := setUpNameTree(t)
	assertLimit(t, "Astatine", f.node5.UpperLimit(), "node5")
	assertLimit(t, "Astatine", f.node2.UpperLimit(), "node2")

	assertLimit(t, "Zirconium", f.node24.UpperLimit(), "node24")
	assertLimit(t, "Zirconium", f.node4.UpperLimit(), "node4")

	// Java asserts null; the port's absent limit is the empty string.
	assertLimit(t, "", f.node1.UpperLimit(), "node1")
}

func TestNameTreeLowerLimit(t *testing.T) {
	f := setUpNameTree(t)
	assertLimit(t, "Actinium", f.node5.LowerLimit(), "node5")
	assertLimit(t, "Actinium", f.node2.LowerLimit(), "node2")

	assertLimit(t, "Xenon", f.node24.LowerLimit(), "node24")
	assertLimit(t, "Xenon", f.node4.LowerLimit(), "node4")

	assertLimit(t, "", f.node1.LowerLimit(), "node1")
}

func assertLimit(t *testing.T, want, got, which string) {
	t.Helper()
	if got != want {
		t.Errorf("%s limit = %q, want %q", which, got, want)
	}
}

package common

// Ported from
// pdfbox/src/test/java/org/apache/pdfbox/pdmodel/common/TestPDNumberTreeNode.java.

import (
	"fmt"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// pdTest is the PDTest inner class of the Java test: a COSObjectable over an
// int, with value equality.
type pdTest struct {
	value int
}

var _ COSObjectable = (*pdTest)(nil)

func newPDTest(value int) *pdTest { return &pdTest{value: value} }

// newPDTestOfCOS is Java's PDTest(COSInteger) constructor, which the reflective
// value lookup finds. The port passes it in as the node's ValueConverter, which
// is what a reflection factory becomes; see
// migration/conventions/java-to-go.md.
func newPDTestOfCOS(base cos.Base) (COSObjectable, error) {
	cosInt, ok := base.(*cos.Integer)
	if !ok {
		return nil, fmt.Errorf("integer expected here, but got %v", base)
	}
	return &pdTest{value: int(cosInt.IntValue())}, nil
}

func (p *pdTest) COSObject() cos.Base { return cos.GetInteger(int64(p.value)) }

// equals is PDTest.equals, which compares the value.
func (p *pdTest) equals(other COSObjectable) bool {
	if p == other {
		return true
	}
	if other == nil {
		return false
	}
	o, ok := other.(*pdTest)
	if !ok {
		return false
	}
	return p.value == o.value
}

type numberTreeFixture struct {
	node1, node2, node4, node5, node24 *PDNumberTreeNode
}

func setUpNumberTree(t *testing.T) numberTreeFixture {
	t.Helper()
	var f numberTreeFixture

	f.node5 = NewPDNumberTreeNode(newPDTestOfCOS)
	numbers := map[int]COSObjectable{
		1: newPDTest(89),
		2: newPDTest(13),
		3: newPDTest(95),
		4: newPDTest(51),
		5: newPDTest(18),
		6: newPDTest(33),
		7: newPDTest(85),
	}
	f.node5.SetNumbers(numbers)

	f.node24 = NewPDNumberTreeNode(newPDTestOfCOS)
	numbers = map[int]COSObjectable{
		8:  newPDTest(54),
		9:  newPDTest(70),
		10: newPDTest(39),
		11: newPDTest(30),
		12: newPDTest(40),
	}
	f.node24.SetNumbers(numbers)

	f.node2 = NewPDNumberTreeNode(newPDTestOfCOS)
	kids := numberKidsOrEmpty(f.node2)
	kids = append(kids, f.node5)
	f.node2.SetKids(kids)

	f.node4 = NewPDNumberTreeNode(newPDTestOfCOS)
	kids = numberKidsOrEmpty(f.node4)
	kids = append(kids, f.node24)
	f.node4.SetKids(kids)

	f.node1 = NewPDNumberTreeNode(newPDTestOfCOS)
	kids = numberKidsOrEmpty(f.node1)
	kids = append(kids, f.node2, f.node4)
	f.node1.SetKids(kids)

	return f
}

func numberKidsOrEmpty(n *PDNumberTreeNode) []*PDNumberTreeNode {
	if kids := n.Kids(); kids != nil {
		return kids.ToSlice()
	}
	return nil
}

func TestNumberTreeGetValue(t *testing.T) {
	f := setUpNumberTree(t)

	value, err := f.node5.Value(4)
	if err != nil {
		t.Fatalf("node5.Value(4): %v", err)
	}
	if !newPDTest(51).equals(value) {
		t.Errorf("node5.Value(4) = %v, want PDTest{51}", value)
	}

	value, err = f.node1.Value(9)
	if err != nil {
		t.Fatalf("node1.Value(9): %v", err)
	}
	if !newPDTest(70).equals(value) {
		t.Errorf("node1.Value(9) = %v, want PDTest{70}", value)
	}

	f.node1.SetKids(nil)
	f.node1.SetNumbers(nil)
	value, err = f.node1.Value(0)
	if err != nil {
		t.Fatalf("node1.Value(0): %v", err)
	}
	if value != nil {
		t.Errorf("node1.Value(0) = %v, want nil", value)
	}
}

func TestNumberTreeUpperLimit(t *testing.T) {
	f := setUpNumberTree(t)
	assertIntLimit(t, 7, f.node5.UpperLimit(), "node5")
	assertIntLimit(t, 7, f.node2.UpperLimit(), "node2")

	assertIntLimit(t, 12, f.node24.UpperLimit(), "node24")
	assertIntLimit(t, 12, f.node4.UpperLimit(), "node4")

	assertIntLimit(t, 12, f.node1.UpperLimit(), "node1")

	f.node24.SetNumbers(map[int]COSObjectable{})
	assertNilLimit(t, f.node24.UpperLimit(), "node24 after an empty map")

	f.node5.SetNumbers(nil)
	assertNilLimit(t, f.node5.UpperLimit(), "node5 after nil")

	f.node1.SetKids(nil)
	assertNilLimit(t, f.node1.UpperLimit(), "node1 after nil kids")
}

func TestNumberTreeLowerLimit(t *testing.T) {
	f := setUpNumberTree(t)
	assertIntLimit(t, 1, f.node5.LowerLimit(), "node5")
	assertIntLimit(t, 1, f.node2.LowerLimit(), "node2")

	assertIntLimit(t, 8, f.node24.LowerLimit(), "node24")
	assertIntLimit(t, 8, f.node4.LowerLimit(), "node4")

	assertIntLimit(t, 1, f.node1.LowerLimit(), "node1")

	f.node24.SetNumbers(map[int]COSObjectable{})
	assertNilLimit(t, f.node24.LowerLimit(), "node24 after an empty map")

	f.node5.SetNumbers(nil)
	assertNilLimit(t, f.node5.LowerLimit(), "node5 after nil")

	f.node1.SetKids(nil)
	assertNilLimit(t, f.node1.LowerLimit(), "node1 after nil kids")
}

func assertIntLimit(t *testing.T, want int, got *int, which string) {
	t.Helper()
	if got == nil {
		t.Errorf("%s limit = nil, want %d", which, want)
		return
	}
	if *got != want {
		t.Errorf("%s limit = %d, want %d", which, *got, want)
	}
}

func assertNilLimit(t *testing.T, got *int, which string) {
	t.Helper()
	if got != nil {
		t.Errorf("%s limit = %d, want nil", which, *got)
	}
}

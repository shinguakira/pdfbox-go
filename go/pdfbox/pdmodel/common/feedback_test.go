package common

// The test below pins a defect the slice 8 review feedback found in this
// package. It is not a port: PDFBox has no test for it. It asserts what the
// Java does, read off PDNameTreeNode.getValue.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// TestValueSkipsAChildWhoseLowerLimitIsTheEmptyName checks that a child whose
// /Limits really are ["" "a"] is skipped for a name outside that range.
//
// Java's getUpperLimit and getLowerLimit answer null where there is no /Limits
// array, and getValue treats a null limit as "this child may hold anything" --
// it descends and returns whatever that child answers, without looking at the
// children after it. The port answered "" for both the absent limit and the
// empty-string limit, so a child whose lower limit is the empty name looked
// unlimited and swallowed the search.
//
// The empty name is a legal PDF name and sorts before every other, so
// ["" "a"] is a legal first range.
func TestValueSkipsAChildWhoseLowerLimitIsTheEmptyName(t *testing.T) {
	root := newTestNameTreeNode(cos.NewDictionary())
	root.node.SetItem(cos.Kids, arrayOfNameTreeNodes([]NameTreeNode[*testValue]{
		limitedChild(t, "", "a", map[string]string{"": "empty", "a": "first"}),
		limitedChild(t, "b", "b", map[string]string{"b": "second"}),
	}))

	// The first child's range does not hold "b", so the search must go on to
	// the second child.
	got, err := root.Value("b")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal(`Value("b") = nil, want the value of the second child`)
	}
	if got.name != "second" {
		t.Errorf(`Value("b") = %q, want "second"`, got.name)
	}

	// And a name the first child does hold is still found there.
	got, err = root.Value("a")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.name != "first" {
		t.Errorf(`Value("a") = %v, want "first"`, got)
	}
}

// TestValueDescendsIntoAChildWithNoLimits checks the other half: a child with
// no /Limits at all is still treated as unlimited, which is what Java's null
// limit means.
func TestValueDescendsIntoAChildWithNoLimits(t *testing.T) {
	unlimited := newTestNameTreeNode(cos.NewDictionary())
	unlimited.SetNames(map[string]*testValue{"b": {name: "found"}})

	root := newTestNameTreeNode(cos.NewDictionary())
	root.node.SetItem(cos.Kids, arrayOfNameTreeNodes(
		[]NameTreeNode[*testValue]{unlimited}))

	got, err := root.Value("b")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.name != "found" {
		t.Errorf(`Value("b") = %v, want "found"`, got)
	}
}

// limitedChild returns a node holding the given names, with its /Limits set to
// the given pair rather than computed, so that the empty name can be a limit.
func limitedChild(t *testing.T, lower, upper string,
	names map[string]string) *testNameTreeNode {
	t.Helper()
	child := newTestNameTreeNode(cos.NewDictionary())
	values := map[string]*testValue{}
	for name, value := range names {
		values[name] = &testValue{name: value}
	}
	child.SetNames(values)
	limits := cos.NewArray()
	limits.Add(cos.NewStringObj(lower))
	limits.Add(cos.NewStringObj(upper))
	child.node.SetItem(cos.Limits, limits)
	return child
}

// testValue is the value type the nodes above hold.
type testValue struct{ name string }

func (v *testValue) COSObject() cos.Base { return cos.NewStringObj(v.name) }

// testNameTreeNode is a concrete node over testValue.
type testNameTreeNode struct {
	PDNameTreeNode[*testValue]
}

var _ NameTreeNode[*testValue] = (*testNameTreeNode)(nil)

func newTestNameTreeNode(dict *cos.Dictionary) *testNameTreeNode {
	n := &testNameTreeNode{}
	n.InitNameTreeNodeOf(n, dict)
	return n
}

func (n *testNameTreeNode) ConvertCOSToPD(base cos.Base) (*testValue, error) {
	str, isString := base.(*cos.StringObj)
	if !isString {
		return nil, nil
	}
	return &testValue{name: str.Value()}, nil
}

func (n *testNameTreeNode) CreateChildNode(dic *cos.Dictionary) NameTreeNode[*testValue] {
	return newTestNameTreeNode(dic)
}

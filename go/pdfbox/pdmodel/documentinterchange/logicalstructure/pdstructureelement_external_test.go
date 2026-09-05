package logicalstructure_test

// Ported from
// pdfbox/src/test/java/org/apache/pdfbox/pdmodel/documentinterchange/logicalstructure/PDStructureElementTest.java.
//
// The package is logicalstructure_test rather than logicalstructure: testClassMap
// opens a document, and pdmodel sits above this package.
//
// testPDFBox4197 is not here: it reads a PDF out of target/pdfs, which the Maven
// build downloads from the issue tracker. See migration/STATUS.md.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/logicalstructure"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/markedcontent"
	_ "github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/taggedpdf"
)

// structureFixture is the directory the logical structure test PDF lives in.
const structureFixture = "../../../../../pdfbox/src/test/resources/org/apache/pdfbox/" +
	"pdmodel/documentinterchange/logicalstructure/"

// TestClassMap is PDStructureElementTest.testClassMap.
func TestClassMap(t *testing.T) {
	// Java collects the attributes into a HashSet of Revisions, which has no
	// equals of its own, so the set is one entry per object; the port collects
	// them into a slice, which is the same thing.
	attributeSet := []*logicalstructure.Revisions[logicalstructure.PDAttributeObject]{}
	classSet := map[string]bool{}

	doc, err := pdfbox.LoadPDF(structureFixture + "PDFBOX-2725-878725.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Close()
	structureTreeRoot := structureTreeRootOf(t, doc.DocumentCatalog().Dictionary())
	checkElement(t, structureTreeRoot.K(), &attributeSet, structureTreeRoot.ClassMap(), classSet)

	for _, r := range attributeSet {
		// check a few that we know
		if r.Size() < 2 {
			continue
		}
		// e.g. in Root/StructTreeRoot/K/[2]/K/[14]/K/[5]/K/[0]/K/[2]/A
		// and     Root/StructTreeRoot/K/[2]/K/[14]/K/[5]/K/[2]/K/[0]/A
		// and     Root/StructTreeRoot/K/[2]/K/[14]/K/[5]/K/[2]/K/[2]/A
		obj0, isTable := r.Object(0).(tableAttributeObject)
		if !isTable {
			t.Fatalf("Object(0) = %T, want a table attribute object", r.Object(0))
		}
		if got, want := obj0.Owner(), "Table"; got != want {
			t.Errorf("Owner() = %q, want %q", got, want)
		}
		if got := obj0.ColSpan(); got != 2 {
			t.Errorf("ColSpan() = %d, want 2", got)
		}
		obj1, isLayout := r.Object(1).(layoutAttributeObject)
		if !isLayout {
			t.Fatalf("Object(1) = %T, want a layout attribute object", r.Object(1))
		}
		if got, want := obj1.Owner(), "Layout"; got != want {
			t.Errorf("Owner() = %q, want %q", got, want)
		}
		// Java casts the Object each answers to Float before comparing.
		if got := obj1.Width().(float32); got != 166.375 && got != 246.75 {
			t.Errorf("Width() = %v, want 166.375 or 246.75", got)
		}
		if got := obj1.Height().(float32); got != 14 && got != 17 {
			t.Errorf("Height() = %v, want 14 or 17", got)
		}
		if got, want := obj1.InlineAlign(), "Start"; got != want {
			t.Errorf("InlineAlign() = %q, want %q", got, want)
		}
		if got := obj1.BlockAlign(); got != "After" && got != "Before" {
			t.Errorf("BlockAlign() = %q, want After or Before", got)
		}
		if got := r.RevisionNumber(0); got != 0 {
			t.Errorf("RevisionNumber(0) = %d, want 0", got)
		}
		if got := r.RevisionNumber(1); got != 0 {
			t.Errorf("RevisionNumber(1) = %d, want 0", got)
		}
	}

	// collect attributes and check their count.
	if got := len(attributeSet); got != 72 {
		t.Errorf("attributeSet size = %d, want 72", got)
	}
	cnt := 0
	for _, r := range attributeSet {
		cnt += r.Size()
	}
	if cnt != 45 {
		t.Errorf("revisions total = %d, want 45", cnt)
	}
	if got := len(classSet); got != 10 {
		t.Errorf("classSet size = %d, want 10", got)
	}
}

// tableAttributeObject is the half of PDTableAttributeObject the test reads.
type tableAttributeObject interface {
	Owner() string
	ColSpan() int
}

// layoutAttributeObject is the half of PDLayoutAttributeObject the test reads.
type layoutAttributeObject interface {
	Owner() string
	// Width and Height answer any: Java answers Object, because the entry is
	// either a number or the name Auto.
	Width() any
	Height() any
	InlineAlign() string
	BlockAlign() string
}

// structureTreeRootOf answers the structure tree root of the given catalogue
// dictionary.
//
// Java reads it through PDDocumentCatalog.getStructureTreeRoot; that accessor
// is not ported, because the catalogue cannot name PDStructureTreeRoot -- this
// package imports pdmodel through its injected constructors. See
// migration/STATUS.md.
func structureTreeRootOf(t *testing.T, catalog *cos.Dictionary) *logicalstructure.PDStructureTreeRoot {
	t.Helper()
	root := catalog.GetCOSDictionary(cos.StructTreeRoot)
	if root == nil {
		t.Fatal("/StructTreeRoot = nil, want the structure tree root")
	}
	return logicalstructure.NewPDStructureTreeRootOf(root)
}

// checkElement is the private checkElement of the Java test. Each element can be
// an array, a dictionary or a number; see PDF specification Table 323 - Entries
// in a structure element dictionary.
func checkElement(t *testing.T, base cos.Base,
	attributeSet *[]*logicalstructure.Revisions[logicalstructure.PDAttributeObject],
	classMap map[string]any, classSet map[string]bool) {
	t.Helper()
	switch typed := base.(type) {
	case *cos.Array:
		for _, base2 := range typed.ToList() {
			if object, isObject := base2.(*cos.Object); isObject {
				base2 = object.Object()
			}
			checkElement(t, base2, attributeSet, classMap, classSet)
		}
	case *cos.Dictionary:
		if typed.ContainsKey(cos.Pg) {
			structureElement := logicalstructure.NewPDStructureElementOf(typed)
			*attributeSet = append(*attributeSet, structureElement.Attributes())
			classNames := structureElement.ClassNames()
			// "If both the A and C entries are present and a given attribute is
			// specified by both, the one specified by the A entry shall take
			// precedence."
			if typed.ContainsKey(cos.C) && !typed.ContainsKey(cos.A) {
				for i := 0; i < classNames.Size(); i++ {
					className := classNames.Object(i)
					classSet[className] = true
					if _, inClassMap := classMap[className]; !inClassMap {
						t.Errorf("'%s' not in ClassMap %v", className, classMap)
					}
				}
			}
		}
		if typed.ContainsKey(cos.K) {
			checkElement(t, typed.GetDictionaryObject(cos.K), attributeSet, classMap, classSet)
		}
	}
}

// TestSimple is PDStructureElementTest.testSimple.
func TestSimple(t *testing.T) {
	structureElement := logicalstructure.NewPDStructureElement("S", nil)
	if got, want := structureElement.Type(), logicalstructure.TypeStructElem; got != want {
		t.Errorf("Type() = %q, want %q", got, want)
	}
	if got, want := structureElement.StructureType(), "S"; got != want {
		t.Errorf("StructureType() = %q, want %q", got, want)
	}
	if got := structureElement.Parent(); got != nil {
		t.Errorf("Parent() = %v, want nil", got)
	}
	structureElement.SetStructureType("T")
	if got, want := structureElement.StructureType(), "T"; got != want {
		t.Errorf("StructureType() = %q, want %q", got, want)
	}
	structureElement.SetElementIdentifier("Ident")
	if got, want := structureElement.ElementIdentifier(), "Ident"; got != want {
		t.Errorf("ElementIdentifier() = %q, want %q", got, want)
	}
	structureElement.SetRevisionNumber(33)
	if got := structureElement.RevisionNumber(); got != 33 {
		t.Errorf("RevisionNumber() = %d, want 33", got)
	}
	structureElement.IncrementRevisionNumber()
	if got := structureElement.RevisionNumber(); got != 34 {
		t.Errorf("RevisionNumber() = %d, want 34", got)
	}
	assertPanics(t, "SetRevisionNumber(-1)", func() { structureElement.SetRevisionNumber(-1) })

	structureElement.SetTitle("Title")
	if got, want := structureElement.Title(), "Title"; got != want {
		t.Errorf("Title() = %q, want %q", got, want)
	}
	structureElement.SetLanguage("Klingon")
	if got, want := structureElement.Language(), "Klingon"; got != want {
		t.Errorf("Language() = %q, want %q", got, want)
	}
	structureElement.SetAlternateDescription("Alto")
	if got, want := structureElement.AlternateDescription(), "Alto"; got != want {
		t.Errorf("AlternateDescription() = %q, want %q", got, want)
	}
	structureElement.SetActualText("Actual")
	if got, want := structureElement.ActualText(), "Actual"; got != want {
		t.Errorf("ActualText() = %q, want %q", got, want)
	}
	structureElement.SetExpandedForm("ExpF")
	if got, want := structureElement.ExpandedForm(), "ExpF"; got != want {
		t.Errorf("ExpandedForm() = %q, want %q", got, want)
	}

	assertPanics(t, "AppendKidMCID(-1)", func() { structureElement.AppendKidMCID(-1) })
	structureElement.AppendKidMCID(0)

	mcr1 := logicalstructure.NewPDMarkedContentReference()
	mcr1.SetMCID(1)
	structureElement.AppendKidMarkedContentReference(mcr1)

	mcr2 := logicalstructure.NewPDMarkedContentReference()
	mcr2.SetMCID(2)
	mc2 := markedcontent.Create(cos.S, mcr2.Dictionary())
	structureElement.AppendKidMarkedContent(mc2)

	mcrSubZero := logicalstructure.NewPDMarkedContentReference()
	assertPanics(t, "SetMCID(-1)", func() { mcrSubZero.SetMCID(-1) })
	mcrSubZero.Dictionary().SetInt(cos.MCID, -1)
	mcSubZero := markedcontent.Create(cos.S, mcrSubZero.Dictionary())
	assertPanics(t, "AppendKidMarkedContent(mcSubZero)",
		func() { structureElement.AppendKidMarkedContent(mcSubZero) })

	kids := structureElement.Kids()
	if got := len(kids); got != 3 {
		t.Fatalf("Kids() size = %d, want 3", got)
	}
	if got := kids[0]; got != 0 {
		t.Errorf("Kids()[0] = %v, want 0", got)
	}
	firstReference, isReference := kids[1].(*logicalstructure.PDMarkedContentReference)
	if !isReference {
		t.Fatalf("Kids()[1] = %T, want *PDMarkedContentReference", kids[1])
	}
	if got, want := firstReference.Dictionary().GetNameAsString(cos.Type, ""),
		logicalstructure.TypeMarkedContentReference; got != want {
		t.Errorf("/Type = %q, want %q", got, want)
	}
	if got := firstReference.MCID(); got != 1 {
		t.Errorf("MCID() = %d, want 1", got)
	}
	if got := kids[2]; got != 2 {
		t.Errorf("Kids()[2] = %v, want 2", got)
	}
}

// assertPanics is the assertThrows(IllegalArgumentException.class) of Java,
// which the port raises as a panic.
func assertPanics(t *testing.T, what string, call func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s did not panic", what)
		}
	}()
	call()
}

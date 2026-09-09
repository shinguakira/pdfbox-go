package schema_test

// JAVA-BUGS 53: `XMPSchema.mergeComplexProperty` returns true from the first
// value the two schemas share, and `merge` takes that as a reason to stop
// merging the whole schema.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/xmpbox"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/schema"
)

// TestMergeCarriesOnPastASharedValue is the defect.
//
// The expected values are everything both schemas hold: a value that is
// already there is skipped, not a reason to stop. Java returns from `merge` at
// the first duplicate, so the rest of that array, every later array and every
// later property of the other schema are silently dropped -- merging two
// Dublin Core schemas that share one creator loses the rest.
func TestMergeCarriesOnPastASharedValue(t *testing.T) {
	parent := xmpbox.CreateXMPMetadata()
	const namespace = "http://www.test.org/schem/"

	first, err := schema.NewXMPSchemaOfNamespace(parent, namespace, "test")
	noError(t, "NewXMPSchemaOfNamespace", err)
	noError(t, "AddQualifiedBagValue", first.AddQualifiedBagValue("bagName", "shared"))
	noError(t, "AddQualifiedBagValue", first.AddQualifiedBagValue("bagName", "onlyFirst"))
	noError(t, "AddUnqualifiedSequenceValue",
		first.AddUnqualifiedSequenceValue("seqName", "seqFirst"))

	second, err := schema.NewXMPSchemaOfNamespace(parent, namespace, "test")
	noError(t, "NewXMPSchemaOfNamespace", err)
	// the shared value comes first, so everything after it is what Java drops
	noError(t, "AddQualifiedBagValue", second.AddQualifiedBagValue("bagName", "shared"))
	noError(t, "AddQualifiedBagValue", second.AddQualifiedBagValue("bagName", "onlySecond"))
	noError(t, "AddUnqualifiedSequenceValue",
		second.AddUnqualifiedSequenceValue("seqName", "seqSecond"))

	noError(t, "Merge", first.Merge(second))

	holdsAll(t, "UnqualifiedBagValueList", first.UnqualifiedBagValueList("bagName"),
		[]string{"shared", "onlyFirst", "onlySecond"})
	holdsAll(t, "UnqualifiedSequenceValueList", first.UnqualifiedSequenceValueList("seqName"),
		[]string{"seqFirst", "seqSecond"})
}

// TestMergeDoesNotDuplicateASharedValue keeps the thing the early return was
// standing in for: the value both schemas hold is there once, not twice.
func TestMergeDoesNotDuplicateASharedValue(t *testing.T) {
	parent := xmpbox.CreateXMPMetadata()
	const namespace = "http://www.test.org/schem/"

	first, err := schema.NewXMPSchemaOfNamespace(parent, namespace, "test")
	noError(t, "NewXMPSchemaOfNamespace", err)
	noError(t, "AddQualifiedBagValue", first.AddQualifiedBagValue("bagName", "shared"))

	second, err := schema.NewXMPSchemaOfNamespace(parent, namespace, "test")
	noError(t, "NewXMPSchemaOfNamespace", err)
	noError(t, "AddQualifiedBagValue", second.AddQualifiedBagValue("bagName", "shared"))

	noError(t, "Merge", first.Merge(second))

	values := first.UnqualifiedBagValueList("bagName")
	if len(values) != 1 {
		t.Errorf("the bag holds %v, want the shared value once", values)
	}
}

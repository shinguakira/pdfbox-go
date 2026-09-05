package form

import (
	"iter"
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// PDFieldTree walks every field of a form, parents before children.
//
// Port of PDFieldTree, which implements Iterable<PDField>.
type PDFieldTree struct {
	acroForm *PDAcroForm
}

// NewPDFieldTree returns a tree over the given form.
//
// Java throws IllegalArgumentException for a nil form, which is unchecked, so
// the port panics.
func NewPDFieldTree(acroForm *PDAcroForm) *PDFieldTree {
	if acroForm == nil {
		panic("root cannot be null")
	}
	return &PDFieldTree{acroForm: acroForm}
}

// All walks the fields of the form.
//
// Java answers an Iterator over a queue the constructor fills breadth first;
// the port answers the range-over-function sequence Go reads the same way, over
// the same queue.
func (t *PDFieldTree) All() iter.Seq[PDField] {
	return func(yield func(PDField) bool) {
		for _, field := range t.queue() {
			if !yield(field) {
				return
			}
		}
	}
}

// queue builds the walk, which is what Java's FieldIterator constructor does.
func (t *PDFieldTree) queue() []PDField {
	queue := []PDField{}

	// PDFBOX-5044: to prevent recursion
	// must be COSDictionary and not PDField, because PDField is newly created each time
	set := map[*cos.Dictionary]bool{}

	var enqueueKids func(node PDField)
	enqueueKids = func(node PDField) {
		queue = append(queue, node)
		set[node.FieldDictionary()] = true
		nonTerminal, isNonTerminal := node.(*PDNonTerminalField)
		if !isNonTerminal {
			return
		}
		for _, kid := range nonTerminal.Children() {
			if set[kid.FieldDictionary()] {
				slog.Error("form: child of field already exists elsewhere, "+
					"ignored to avoid recursion",
					slog.String("field", node.FullyQualifiedName()))
				continue
			}
			enqueueKids(kid)
		}
	}

	for _, field := range t.acroForm.Fields() {
		enqueueKids(field)
	}
	return queue
}

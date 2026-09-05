// Package form holds the interactive form of a document: the AcroForm and the
// fields that fill it in.
//
// Port of org.apache.pdfbox.pdmodel.interactive.form.
package form

import (
	"sort"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/action"
)

// ScriptingHandler runs the JavaScript actions a field carries.
//
// Port of the interface ScriptingHandler.
type ScriptingHandler interface {
	// Keyboard runs the keystroke action.
	Keyboard(javaScriptAction *action.PDActionJavaScript, value string) string

	// Format runs the format action.
	Format(javaScriptAction *action.PDActionJavaScript, value string) string

	// Validate runs the validate action.
	Validate(javaScriptAction *action.PDActionJavaScript, value string) bool

	// Calculate runs the calculate action.
	Calculate(javaScriptAction *action.PDActionJavaScript, value string) string
}

// keyValue is one export value and the display value beside it.
//
// Port of the package-private FieldUtils.KeyValue.
type keyValue struct {
	key   string
	value string
}

// Key returns the export value.
func (kv keyValue) Key() string { return kv.key }

// Value returns the display value.
func (kv keyValue) Value() string { return kv.value }

// String renders the pair the way Java writes it.
func (kv keyValue) String() string { return "(" + kv.key + ", " + kv.value + ")" }

// toKeyValueList pairs the keys with the values.
//
// Port of the package-private FieldUtils.toKeyValueList.
func toKeyValueList(key, value []string) []keyValue {
	list := make([]keyValue, 0, len(key))
	for i := 0; i < len(key); i++ {
		list = append(list, keyValue{key: key[i], value: value[i]})
	}
	return list
}

// sortByValue sorts the pairs on their display value.
//
// Java sorts with List.sort, which is stable, so the port sorts stably too.
func sortByValue(pairs []keyValue) {
	sort.SliceStable(pairs, func(i, j int) bool { return pairs[i].value < pairs[j].value })
}

// sortByKey sorts the pairs on their export value.
func sortByKey(pairs []keyValue) {
	sort.SliceStable(pairs, func(i, j int) bool { return pairs[i].key < pairs[j].key })
}

// getPairableItems reads the strings of an /Opt entry, taking the given half of
// each two element array.
//
// Port of the package-private FieldUtils.getPairableItems. Java throws
// IllegalArgumentException for an index outside the pair, which is unchecked,
// so the port panics.
func getPairableItems(items cos.Base, pairIdx int) []string {
	if pairIdx < 0 || pairIdx > 1 {
		panic("Only 0 and 1 are allowed as an index into two-element arrays")
	}
	switch value := items.(type) {
	case *cos.StringObj:
		return []string{value.Value()}
	case *cos.Array:
		entryList := []string{}
		for i := 0; i < value.Size(); i++ {
			switch entry := value.Get(i).(type) {
			case *cos.StringObj:
				entryList = append(entryList, entry.Value())
			case *cos.Array:
				if entry.Size() >= pairIdx+1 {
					if pair, isString := entry.Get(pairIdx).(*cos.StringObj); isString {
						entryList = append(entryList, pair.Value())
					}
				}
			}
		}
		return entryList
	}
	return []string{}
}

// stringList flattens what COSArray.toCOSStringStringList answers.
//
// Java leaves a null in place of an entry that is not a string, and the fields
// carry those nulls into their value lists; the port leaves the empty string
// there instead, since a Go string cannot be null. Only a malformed file has
// such an entry.
func stringList(a *cos.Array) []string {
	entries := a.ToStringStringList()
	out := make([]string, len(entries))
	for i, entry := range entries {
		if entry != nil {
			out[i] = *entry
		}
	}
	return out
}

// intList flattens what COSArray.toCOSNumberIntegerList answers, leaving zero
// where Java leaves null.
func intList(a *cos.Array) []int {
	entries := a.ToNumberIntegerList()
	out := make([]int, len(entries))
	for i, entry := range entries {
		if entry != nil {
			out[i] = *entry
		}
	}
	return out
}

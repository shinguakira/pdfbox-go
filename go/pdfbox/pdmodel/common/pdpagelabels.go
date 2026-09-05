package common

import (
	"slices"
	"strconv"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// PageCounter is what PDPageLabels needs of the document it labels.
//
// Java takes a PDDocument, which imports this package; the port names what is
// used, so that the dependency runs one way.
type PageCounter interface {
	// NumberOfPages returns how many pages the document has.
	NumberOfPages() int
}

// PDPageLabels represents the page label dictionary of a document.
//
// Port of org.apache.pdfbox.pdmodel.common.PDPageLabels. Java's TreeMap keeps
// the ranges in page order, which every walk here depends on; the port keeps a
// map and sorts the keys where it walks them.
type PDPageLabels struct {
	labels map[int]*PDPageLabelRange
	doc    PageCounter
}

var _ COSObjectable = (*PDPageLabels)(nil)

// NewPDPageLabels creates a new page label dictionary with a default decimal
// range starting at page 0.
func NewPDPageLabels(document PageCounter) *PDPageLabels {
	l := &PDPageLabels{labels: map[int]*PDPageLabelRange{}, doc: document}
	defaultRange := NewPDPageLabelRange()
	defaultRange.SetStyle(StyleDecimal)
	l.labels[0] = defaultRange
	return l
}

// NewPDPageLabelsOf reads the page labels out of the given dictionary.
func NewPDPageLabelsOf(document PageCounter, dict *cos.Dictionary) (*PDPageLabels, error) {
	l := NewPDPageLabels(document)
	if dict == nil {
		return l, nil
	}
	root := NewPDNumberTreeNodeOf(dict, pageLabelRangeOfCOS)
	if err := l.findLabels(root); err != nil {
		return nil, err
	}
	return l, nil
}

// pageLabelRangeOfCOS is the PDPageLabelRange(COSDictionary) constructor Java
// finds reflectively through the Class it hands PDNumberTreeNode.
func pageLabelRangeOfCOS(base cos.Base) (COSObjectable, error) {
	dictionary, ok := asNodeDictionary(base)
	if !ok {
		// Java's reflective lookup fails with an IOException wrapping the
		// NoSuchMethodException; the port reports the same shape.
		return nil, errNoPageLabelRangeConstructor
	}
	return NewPDPageLabelRangeOf(dictionary), nil
}

// errNoPageLabelRangeConstructor stands for the IOException Java's reflective
// constructor lookup throws when the value is not a dictionary.
var errNoPageLabelRangeConstructor = errPageLabelRange("Error while trying to create value in number tree")

type errPageLabelRange string

func (e errPageLabelRange) Error() string { return string(e) }

func (l *PDPageLabels) findLabels(node *PDNumberTreeNode) error {
	kids := node.Kids()
	if kids != nil {
		for i := 0; i < kids.Size(); i++ {
			if err := l.findLabels(kids.Get(i)); err != nil {
				return err
			}
		}
		return nil
	}
	numbers, err := node.Numbers()
	if err != nil {
		return err
	}
	for _, key := range sortedKeys(numbers) {
		if key >= 0 {
			// Java casts to PDPageLabelRange and throws ClassCastException
			// otherwise; the port asserts the same way.
			l.labels[key] = numbers[key].(*PDPageLabelRange)
		}
	}
	return nil
}

// PageRangeCount returns how many page label ranges there are.
func (l *PDPageLabels) PageRangeCount() int { return len(l.labels) }

// PageLabelRange returns the range starting at the given page index, or nil.
func (l *PDPageLabels) PageLabelRange(startPage int) *PDPageLabelRange {
	return l.labels[startPage]
}

// SetLabelItem sets the range starting at the given page index.
//
// Java throws IllegalArgumentException for a negative page, which is unchecked,
// so the port panics.
func (l *PDPageLabels) SetLabelItem(startPage int, item *PDPageLabelRange) {
	if startPage < 0 {
		panic("startPage parameter of setLabelItem may not be < 0")
	}
	l.labels[startPage] = item
}

// COSObject builds the /PageLabels number tree dictionary.
func (l *PDPageLabels) COSObject() cos.Base {
	arr := cos.NewArray()
	for _, key := range sortedKeys(l.labels) {
		arr.Add(cos.GetInteger(int64(key)))
		arr.Add(l.labels[key].COSObject())
	}
	dict := cos.NewDictionary()
	dict.SetItem(cos.Nums, arr)
	return dict
}

// PageIndicesByLabels returns a map from each page label to the index of the
// page it names.
func (l *PDPageLabels) PageIndicesByLabels() map[string]int {
	numberOfPages := l.doc.NumberOfPages()
	labelMap := map[string]int{}
	l.computeLabels(func(pageIndex int, label string) {
		labelMap[label] = pageIndex
	}, numberOfPages)
	return labelMap
}

// LabelsByPageIndices returns the label of each page, indexed by page.
func (l *PDPageLabels) LabelsByPageIndices() []string {
	numberOfPages := l.doc.NumberOfPages()
	m := make([]string, numberOfPages)
	l.computeLabels(func(pageIndex int, label string) {
		if pageIndex < numberOfPages {
			m[pageIndex] = label
		}
	}, numberOfPages)
	return m
}

// PageIndices returns the page indices the ranges start at, in order.
//
// Java returns a NavigableSet; the port returns the sorted slice, which is what
// every caller walks it as.
func (l *PDPageLabels) PageIndices() []int {
	keys := sortedKeys(l.labels)
	return keys
}

// labelHandler is the private LabelHandler interface, which Go writes as a
// function.
type labelHandler func(pageIndex int, label string)

func (l *PDPageLabels) computeLabels(handler labelHandler, numberOfPages int) {
	keys := sortedKeys(l.labels)
	if len(keys) == 0 {
		return
	}
	pageIndex := 0
	lastKey := keys[0]
	for i := 1; i < len(keys); i++ {
		key := keys[i]
		numPages := key - lastKey
		gen := newLabelGenerator(l.labels[lastKey], numPages)
		for gen.hasNext() {
			handler(pageIndex, gen.next())
			pageIndex++
		}
		lastKey = key
	}
	gen := newLabelGenerator(l.labels[lastKey], numberOfPages-lastKey)
	for gen.hasNext() {
		handler(pageIndex, gen.next())
		pageIndex++
	}
}

// labelGenerator yields the labels of one range.
//
// Port of the private static LabelGenerator.
type labelGenerator struct {
	labelInfo   *PDPageLabelRange
	numPages    int
	currentPage int
}

func newLabelGenerator(label *PDPageLabelRange, pages int) *labelGenerator {
	return &labelGenerator{labelInfo: label, numPages: pages, currentPage: 0}
}

func (g *labelGenerator) hasNext() bool { return g.currentPage < g.numPages }

// next returns the next label.
//
// Java throws NoSuchElementException when there is none, which is unchecked, so
// the port panics.
func (g *labelGenerator) next() string {
	if !g.hasNext() {
		panic("no such element")
	}
	var buf strings.Builder
	label := g.labelInfo.Prefix()
	if label != "" {
		// there may be some labels with some null bytes at the end
		// which will lead to an incomplete output, see PDFBOX-1047
		if index := strings.IndexByte(label, 0); index > -1 {
			label = label[:index]
		}
		buf.WriteString(label)
	}
	style := g.labelInfo.Style()
	if style != "" {
		buf.WriteString(getNumber(g.labelInfo.Start()+g.currentPage, style))
	}
	g.currentPage++
	return buf.String()
}

func getNumber(pageIndex int, style string) string {
	switch style {
	case StyleDecimal:
		return strconv.Itoa(pageIndex)
	case StyleLettersLower:
		return makeLetterLabel(pageIndex)
	case StyleLettersUpper:
		return strings.ToUpper(makeLetterLabel(pageIndex))
	case StyleRomanLower:
		return makeRomanLabel(pageIndex)
	case StyleRomanUpper:
		return strings.ToUpper(makeRomanLabel(pageIndex))
	}
	// Fall back to decimals.
	return strconv.Itoa(pageIndex)
}

var romans = [3][10]string{
	{"", "i", "ii", "iii", "iv", "v", "vi", "vii", "viii", "ix"},
	{"", "x", "xx", "xxx", "xl", "l", "lx", "lxx", "lxxx", "xc"},
	{"", "c", "cc", "ccc", "cd", "d", "dc", "dcc", "dccc", "cm"},
}

func makeRomanLabel(pageIndex int) string {
	var parts []string
	power := 0
	for power < 3 && pageIndex > 0 {
		parts = append(parts, romans[power][pageIndex%10])
		pageIndex /= 10
		power++
	}
	slices.Reverse(parts)
	// Prepend as many m as there are thousands (which is
	// incorrect by the roman numeral rules for numbers > 3999,
	// but is unbounded and Adobe Acrobat does it this way).
	// This code is somewhat inefficient for really big numbers,
	// but those don't occur too often (and the numbers in those cases
	// would be incomprehensible even if we and Adobe
	// used strict Roman rules).
	return strings.Repeat("m", max(pageIndex, 0)) + strings.Join(parts, "")
}

func makeLetterLabel(num int) string {
	numLetters := num/26 + signum(num%26)
	letter := num%26 + 26*(1-signum(num%26)) + 'a' - 1
	var buf strings.Builder
	for i := 0; i < numLetters; i++ {
		buf.WriteRune(rune(letter))
	}
	return buf.String()
}

// signum is Integer.signum.
func signum(i int) int {
	switch {
	case i > 0:
		return 1
	case i < 0:
		return -1
	default:
		return 0
	}
}

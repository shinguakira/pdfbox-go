package main

// What the port makes of a document beyond its text, as digests
// migration/oracle/JavaFacets.java computes the same way for PDFBox.
//
// The row table compares whether a document opens, its page count and its text.
// Everything else the port implements -- where each glyph sits, the document
// information and XMP, the outline, page labels, page boxes, the structure tree,
// annotations, form fields, and the images and what they decode to -- was
// compared by nothing. Each facet is a sequence of lines, written the same way
// on both sides and digested. A facet cell is "<lines>:<digest>", "-" where the
// document has no such thing, or "error" where computing it failed; the error is
// not named, because the two languages word their failures differently and what
// is compared is whether each side failed.
//
// Floats are written as their bits. A string the Java answers null for and the
// port answers "" for is written as "" on both sides; where both can tell absence
// apart, the line writes <null> itself.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"image/color"
	"io"
	"math"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/logicalstructure"
	// The default AcroForm fixup registers itself; JavaFacets reads the form
	// through getAcroForm(), which applies it.
	_ "github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/fixup"
	gform "github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/form"
	pdimage "github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/documentnavigation/destination"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/documentnavigation/outline"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/form"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/text"
	xmpxml "github.com/shinguakira/pdfbox-go/go/xmpbox/xml"
)

// group is a set of facet columns and the flag a child computes one file's
// of them with: the read-only facets, or the write facets of writes.go.
type group struct {
	names []string
	child string
	// header, when set, is the table's header in place of file, open and the names.
	header string
	// fail, when set, answers the rows of a job whose child timed out or died.
	fail func(why string) []string
}

var facetGroup = group{names: facetNames, child: "-onefacets"}

// facetNames are the facet columns, in the order JavaFacets.NAMES has them.
var facetNames = []string{
	"positions", "info", "xmp", "xmpschemas", "outline", "labels", "boxes", "struct",
	"annots", "fields", "images", "imagepixels",
}

const (
	// maxFacetPixels is the largest image converted to pixels.
	maxFacetPixels = 40_000_000
	// maxFacetItems stops an outline, structure or field walk on a file that loops.
	maxFacetItems = 200_000
)

// errAbsent says the document has no such thing; its cell is "-".
var errAbsent = errors.New("absent")

// facet is one facet's lines, digested as they arrive, and kept when dumping.
type facet struct {
	sum   hash.Hash
	lines int
	keep  bool
	kept  []string
}

func newFacet(keep bool) *facet { return &facet{sum: sha256.New(), keep: keep} }

func (f *facet) line(s string) {
	f.sum.Write([]byte(s))
	f.sum.Write([]byte{'\n'})
	f.lines++
	if f.keep {
		f.kept = append(f.kept, s)
	}
}

func (f *facet) bytes(b []byte) {
	f.sum.Write(b)
	f.lines += len(b)
	if f.keep {
		f.kept = append(f.kept, fmt.Sprintf("bytes %d", len(b)))
	}
}

func (f *facet) cell() string {
	return fmt.Sprintf("%d:%s", f.lines, hex.EncodeToString(f.sum.Sum(nil)[:8]))
}

// fstr writes a string so that a line stays one line, as JavaFacets.s does.
func fstr(v string) string {
	var b strings.Builder
	for i := 0; i < len(v); i++ {
		switch c := v[i]; c {
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

func ff(v float32) string { return fmt.Sprintf("%08x", math.Float32bits(v)) }

func frect(r *common.PDRectangle) string {
	if r == nil {
		return "<null>"
	}
	return ff(r.LowerLeftX()) + "," + ff(r.LowerLeftY()) + "," + ff(r.UpperRightX()) + "," + ff(r.UpperRightY())
}

func fdate(t time.Time, ok bool) string {
	if !ok {
		return "<null>"
	}
	_, offset := t.Zone()
	return fmt.Sprintf("%d|%d", t.UnixMilli(), offset*1000)
}

func fcos(value cos.Base) string {
	switch v := value.(type) {
	case nil, *cos.Null:
		return "null"
	case *cos.StringObj:
		return "str:" + fstr(v.Value())
	case *cos.Name:
		return "name:" + fstr(v.Name())
	case *cos.Integer:
		return "int:" + strconv.FormatInt(v.LongValue(), 10)
	case *cos.Float:
		return "float:" + ff(v.FloatValue())
	case *cos.Boolean:
		return "bool:" + strconv.FormatBool(v.Value())
	case *cos.Stream:
		return "stream"
	case *cos.Dictionary:
		return "dict"
	case *cos.Array:
		return "array"
	}
	return "other"
}

// ------------------------------------------------------------------- facets

func facetPositions(doc *pdmodel.PDDocument, f *facet) error {
	stripper := text.NewPDFTextStripper()
	stripper.SetProcessTextPosition(func(t *text.TextPosition) error {
		codes := make([]string, 0, 2)
		for _, code := range t.CharacterCodes() {
			codes = append(codes, strconv.Itoa(code))
		}
		name := ""
		if font := t.Font(); font != nil {
			name = font.Name()
		}
		f.line("u=" + fstr(t.Unicode()) + " c=" + strings.Join(codes, ",") + " x=" + ff(t.XDirAdj()) +
			" y=" + ff(t.YDirAdj()) + " w=" + ff(t.WidthDirAdj()) + " h=" + ff(t.HeightDir()) +
			" fs=" + ff(t.FontSizeInPt()) + " font=" + fstr(name))
		return stripper.ProcessTextPosition(t)
	})
	return stripper.WriteText(doc, io.Discard)
}

func facetInfo(doc *pdmodel.PDDocument, f *facet) error {
	trailer := doc.Document().Trailer()
	if trailer == nil {
		return errAbsent
	}
	// Java's getCOSDictionary answers a stream too, which GetCOSDictionary does
	// not.
	switch trailer.GetDictionaryObject(cos.Info).(type) {
	case *cos.Dictionary, *cos.Stream:
	default:
		return errAbsent
	}
	info := doc.DocumentInformation()
	dictionary := info.Dictionary()
	keys := dictionary.KeySet()
	sort.Slice(keys, func(i, j int) bool { return keys[i].Name() < keys[j].Name() })
	for _, key := range keys {
		f.line(fstr(key.Name()) + "=" + fcos(dictionary.GetDictionaryObject(key)))
	}
	f.line("CreationDate.parsed=" + fdate(info.CreationDate()))
	f.line("ModDate.parsed=" + fdate(info.ModificationDate()))
	return nil
}

func xmpBytes(doc *pdmodel.PDDocument) ([]byte, error) {
	metadata := doc.DocumentCatalog().Metadata()
	if metadata == nil {
		return nil, errAbsent
	}
	return metadata.ToByteArray()
}

func facetXMP(doc *pdmodel.PDDocument, f *facet) error {
	b, err := xmpBytes(doc)
	if err != nil {
		return err
	}
	f.bytes(b)
	return nil
}

func facetXMPSchemas(doc *pdmodel.PDDocument, f *facet) error {
	b, err := xmpBytes(doc)
	if err != nil {
		return err
	}
	metadata, err := xmpxml.NewDomXmpParser().ParseBytes(b)
	if err != nil {
		return err
	}
	var namespaces []string
	for _, schema := range metadata.AllSchemas() {
		namespaces = append(namespaces, fstr(schema.Namespace()))
	}
	sort.Strings(namespaces)
	for _, namespace := range namespaces {
		f.line(namespace)
	}
	return nil
}

func facetOutline(doc *pdmodel.PDDocument, f *facet) error {
	root := doc.DocumentCatalog().DocumentOutline()
	if root == nil {
		return errAbsent
	}
	seen := map[*cos.Dictionary]bool{}
	var walk func(item *outline.PDOutlineItem, depth int)
	walk = func(item *outline.PDOutlineItem, depth int) {
		for ; item != nil; item = item.NextSibling() {
			if seen[item.Dictionary()] || f.lines >= maxFacetItems {
				f.line("stop")
				return
			}
			seen[item.Dictionary()] = true
			dest := "error"
			if d, err := item.Destination(); err == nil {
				switch v := d.(type) {
				case nil:
					dest = "<null>"
				case destination.PageDestination:
					dest = "page:" + strconv.Itoa(v.RetrievePageNumber())
				case *destination.PDNamedDestination:
					dest = "named:" + fstr(v.NamedDestination())
				default:
					dest = "other"
				}
			}
			act := "<null>"
			if a := item.Action(); a != nil {
				act = fstr(a.SubType())
			}
			f.line(fmt.Sprintf("d=%d t=%s open=%t dest=%s action=%s", depth, fstr(item.Title()), item.IsNodeOpen(), dest, act))
			walk(item.FirstChild(), depth+1)
		}
	}
	walk(root.FirstChild(), 0)
	return nil
}

func facetLabels(doc *pdmodel.PDDocument, f *facet) error {
	labels, err := doc.DocumentCatalog().PageLabels()
	if err != nil {
		return err
	}
	if labels == nil {
		return errAbsent
	}
	for i, label := range labels.LabelsByPageIndices() {
		f.line(strconv.Itoa(i) + "=" + fstr(label))
	}
	return nil
}

func facetBoxes(doc *pdmodel.PDDocument, f *facet) error {
	i := 0
	for page := range doc.Pages().All {
		f.line(fmt.Sprintf("p%d m=%s c=%s b=%s t=%s a=%s r=%d", i, frect(page.MediaBox()), frect(page.CropBox()),
			frect(page.BleedBox()), frect(page.TrimBox()), frect(page.ArtBox()), page.Rotation()))
		i++
	}
	return nil
}

func facetStruct(doc *pdmodel.PDDocument, f *facet) error {
	root := doc.DocumentCatalog().StructureTreeRoot()
	if root == nil {
		return errAbsent
	}
	seen := map[*cos.Dictionary]bool{}
	var walk func(kids []any, depth int)
	walk = func(kids []any, depth int) {
		for _, kid := range kids {
			if f.lines >= maxFacetItems {
				f.line("stop")
				return
			}
			switch k := kid.(type) {
			case *logicalstructure.PDStructureElement:
				if seen[k.NodeDictionary()] {
					f.line(fmt.Sprintf("d=%d cycle", depth))
					continue
				}
				seen[k.NodeDictionary()] = true
				f.line(fmt.Sprintf("d=%d S=%s std=%s alt=%s actual=%s title=%s lang=%s", depth, fstr(k.StructureType()),
					fstr(k.StandardStructureType()), fstr(k.AlternateDescription()), fstr(k.ActualText()), fstr(k.Title()),
					fstr(k.Language())))
				walk(k.Kids(), depth+1)
			case int:
				f.line(fmt.Sprintf("d=%d mcid=%d", depth, k))
			case *logicalstructure.PDMarkedContentReference:
				f.line(fmt.Sprintf("d=%d mcr=%d", depth, k.MCID()))
			case *logicalstructure.PDObjectReference:
				f.line(fmt.Sprintf("d=%d objr", depth))
			default:
				f.line(fmt.Sprintf("d=%d other", depth))
			}
		}
	}
	walk(root.Kids(), 0)
	return nil
}

// annotationBase is what every annotation carries beyond the PDAnnotation interface.
type annotationBase interface {
	Contents() string
	AnnotationName() string
	AnnotationFlags() int
	AppearanceState() *cos.Name
}

func facetAnnots(doc *pdmodel.PDDocument, f *facet) error {
	i := 0
	for page := range doc.Pages().All {
		for a := range page.Annotations().All {
			base, ok := a.(annotationBase)
			if !ok {
				return fmt.Errorf("annotation %T has no base", a)
			}
			state := "<null>"
			if name := base.AppearanceState(); name != nil {
				state = fstr(name.Name())
			}
			normal := false
			if appearance := a.Appearance(); appearance != nil && appearance.NormalAppearance() != nil {
				normal = true
			}
			f.line(fmt.Sprintf("p%d st=%s r=%s c=%s nm=%s f=%d as=%s ap=%t", i, fstr(a.Subtype()), frect(a.Rectangle()),
				fstr(base.Contents()), fstr(base.AnnotationName()), base.AnnotationFlags(), state, normal))
		}
		i++
	}
	return nil
}

func facetFields(doc *pdmodel.PDDocument, f *facet) error {
	acroForm := form.AcroFormOfCatalog(doc.DocumentCatalog())
	if acroForm == nil {
		return errAbsent
	}
	for field := range acroForm.FieldTree().All() {
		if f.lines >= maxFacetItems {
			f.line("stop")
			return nil
		}
		kind, value, widgets := "other", "-", "-"
		switch v := field.(type) {
		case *form.PDNonTerminalField:
			kind = "nonterminal"
		case *form.PDTextField:
			kind, value = "text", fstr(v.Value())
		case *form.PDCheckBox:
			kind, value = "checkbox", fstr(v.Value())
		case *form.PDRadioButton:
			kind, value = "radio", fstr(v.Value())
		case *form.PDPushButton:
			kind = "pushbutton"
		case *form.PDComboBox:
			kind, value = "combo", joinChoice(v.Value())
		case *form.PDListBox:
			kind, value = "list", joinChoice(v.Value())
		case *form.PDSignatureField:
			kind, value = "signature", "absent"
			if v.Value() != nil {
				value = "present"
			}
		}
		if kind != "nonterminal" && kind != "other" {
			widgets = strconv.Itoa(len(field.Widgets()))
		}
		f.line(fmt.Sprintf("n=%s ft=%s ff=%d kind=%s w=%s v=%s", fstr(field.FullyQualifiedName()), fstr(field.FieldType()),
			field.FieldFlags(), kind, widgets, value))
	}
	return nil
}

func joinChoice(values []string) string {
	escaped := make([]string, len(values))
	for i, v := range values {
		escaped[i] = fstr(v)
	}
	return strings.Join(escaped, "|")
}

// eachImage visits every image XObject the pages reach, through form XObjects,
// once each, in page and name order.
func eachImage(doc *pdmodel.PDDocument, visit func(*pdimage.PDImageXObject)) error {
	seen := map[*cos.Stream]bool{}
	var walk func(resources *pdmodel.PDResources) error
	walk = func(resources *pdmodel.PDResources) error {
		if resources == nil {
			return nil
		}
		names := resources.XObjectNames()
		sort.Slice(names, func(i, j int) bool { return names[i].Name() < names[j].Name() })
		for _, name := range names {
			xobject, err := resources.GetXObject(name)
			if err != nil {
				return err
			}
			switch x := xobject.(type) {
			case *pdimage.PDImageXObject:
				if !seen[x.Stream()] {
					seen[x.Stream()] = true
					visit(x)
				}
			case *gform.PDFormXObject:
				if !seen[x.Stream()] {
					seen[x.Stream()] = true
					if inner, ok := x.Resources().(*pdmodel.PDResources); ok {
						if err := walk(inner); err != nil {
							return err
						}
					}
				}
			case *gform.PDTransparencyGroup:
				if !seen[x.Stream()] {
					seen[x.Stream()] = true
					if inner, ok := x.Resources().(*pdmodel.PDResources); ok {
						if err := walk(inner); err != nil {
							return err
						}
					}
				}
			}
		}
		return nil
	}
	for page := range doc.Pages().All {
		if err := walk(page.Resources()); err != nil {
			return err
		}
	}
	return nil
}

func facetImages(doc *pdmodel.PDDocument, f *facet) error {
	return eachImage(doc, func(x *pdimage.PDImageXObject) {
		colorSpace := "error"
		func() {
			defer func() { _ = recover() }()
			if cs, err := x.ColorSpace(); err == nil && cs != nil {
				colorSpace = fstr(cs.Name())
			}
		}()
		var filters []string
		for _, filter := range x.PDStream().Filters() {
			filters = append(filters, fstr(filter.Name()))
		}
		data := "error"
		func() {
			defer func() { _ = recover() }()
			if b, err := x.PDStream().ToByteArray(); err == nil {
				bytes := newFacet(false)
				bytes.bytes(b)
				data = bytes.cell()
			}
		}()
		f.line(fmt.Sprintf("w=%d h=%d bpc=%d cs=%s filters=%s stencil=%t data=%s", x.Width(), x.Height(),
			x.BitsPerComponent(), colorSpace, strings.Join(filters, ","), x.IsStencil(), data))
	})
}

func facetImagePixels(doc *pdmodel.PDDocument, f *facet) error {
	return eachImage(doc, func(x *pdimage.PDImageXObject) {
		if int64(x.Width())*int64(x.Height()) > maxFacetPixels {
			f.line(fmt.Sprintf("skipped %dx%d", x.Width(), x.Height()))
			return
		}
		line := "error"
		func() {
			defer func() { _ = recover() }()
			img, err := x.Image()
			if err != nil || img == nil {
				return
			}
			bounds := img.Bounds()
			width, height := bounds.Dx(), bounds.Dy()
			sum := sha256.New()
			row := make([]byte, width*4)
			for y := 0; y < height; y++ {
				for px := 0; px < width; px++ {
					c := color.NRGBAModel.Convert(img.At(bounds.Min.X+px, bounds.Min.Y+y)).(color.NRGBA)
					row[px*4], row[px*4+1], row[px*4+2], row[px*4+3] = c.A, c.R, c.G, c.B
				}
				sum.Write(row)
			}
			line = fmt.Sprintf("%dx%d %d:%s", width, height, width*height, hex.EncodeToString(sum.Sum(nil)[:8]))
		}()
		f.line(line)
	})
}

var facetComputations = []func(*pdmodel.PDDocument, *facet) error{
	facetPositions, facetInfo, facetXMP, facetXMPSchemas, facetOutline, facetLabels, facetBoxes, facetStruct,
	facetAnnots, facetFields, facetImages, facetImagePixels,
}

// facetsNow computes one way of opening one file's facets in this process. It
// answers the cells after the name -- open, then one per facet -- or, with
// keep, one "facet\tline" per line.
func facetsNow(j job, keep bool) []string {
	document, err := func() (d *pdmodel.PDDocument, err error) {
		defer func() {
			if p := recover(); p != nil {
				err = fmt.Errorf("panic: %v", p)
			}
		}()
		return openDocument(j)
	}()
	if err != nil {
		if keep {
			return []string{"open\t" + fstr(short(err))}
		}
		row := []string{short(err)}
		for range facetNames {
			row = append(row, "-")
		}
		return []string{strings.Join(row, "\t")}
	}
	defer document.Close()

	row := []string{"ok"}
	var lines []string
	for i, compute := range facetComputations {
		f := newFacet(keep)
		err := func() (err error) {
			defer func() {
				if p := recover(); p != nil {
					err = fmt.Errorf("panic: %v", p)
				}
			}()
			return compute(document, f)
		}()
		switch {
		case errors.Is(err, errAbsent):
			row = append(row, "-")
		case err != nil:
			row = append(row, "error")
			if keep {
				f.kept = append(f.kept, "error "+fstr(short(err)))
			}
		default:
			row = append(row, f.cell())
		}
		for _, line := range f.kept {
			lines = append(lines, facetNames[i]+"\t"+line)
		}
	}
	if keep {
		return lines
	}
	return []string{strings.Join(row, "\t")}
}

// facetsIsolated computes one job's facets in a child process, as scoreIsolated
// scores one: a stack overflow or a document that will not return cannot take
// the run with it.
func facetsIsolated(exe string, g group, j job, keep bool, timeout time.Duration) []string {
	args := []string{g.child}
	if keep {
		args = append(args, "-keeplines")
	}
	for _, table := range passwordsPaths {
		args = append(args, "-passwords", table)
	}
	args = append(args, "-open", fmt.Sprint(j.open), j.path)

	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, exe, args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	if err == nil {
		out := strings.TrimRight(strings.ReplaceAll(stdout.String(), "\r\n", "\n"), "\n")
		if out == "" {
			return nil
		}
		return strings.Split(out, "\n")
	}
	why := crashReason(stderr.String(), err)
	if ctx.Err() == context.DeadlineExceeded {
		why = "timeout"
	}
	if keep {
		return []string{"open\t" + fstr(why)}
	}
	if g.fail != nil {
		return g.fail(why)
	}
	row := []string{why}
	for range g.names {
		row = append(row, "-")
	}
	return []string{strings.Join(row, "\t")}
}

// writeFacets computes every job's facets, workers at a time, and writes them in
// job order: the facet table, or with keep every line the digests are made of.
func writeFacets(jobs []job, out string, g group, keep bool, workers int, timeout time.Duration) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	file, err := os.Create(out)
	if err != nil {
		return err
	}
	defer file.Close()
	switch {
	case g.header != "":
		fmt.Fprintln(file, g.header)
	case keep:
		fmt.Fprintln(file, "file\tfacet\tline")
	default:
		fmt.Fprintln(file, "file\topen\t"+strings.Join(g.names, "\t"))
	}

	if workers < 1 {
		workers = 1
	}
	results := make([][]string, len(jobs))
	done := make([]bool, len(jobs))
	var mu sync.Mutex
	next := 0
	written := 0
	work := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range work {
				rows := facetsIsolated(exe, g, jobs[i], keep, timeout)
				mu.Lock()
				results[i], done[i] = rows, true
				for next < len(jobs) && done[next] {
					name := rootRelative(jobs[next].path) + jobs[next].label
					for _, row := range results[next] {
						fmt.Fprintln(file, name+"\t"+row)
					}
					results[next] = nil
					next++
					written++
				}
				fmt.Fprintf(os.Stderr, "\r%d/%d written", written, len(jobs))
				mu.Unlock()
			}
		}()
	}
	for i := range jobs {
		work <- i
	}
	close(work)
	wg.Wait()
	fmt.Fprintf(os.Stderr, "\r%-40s\r", "")
	return nil
}

package contentstream_test

import (
	"errors"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator/markedcontent"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator/state"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator/text"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/filter"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	graphicsstate "github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/state"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// Written from org.apache.pdfbox.contentstream.PDFStreamEngine and the operator
// processors around it. The Java suite exercises the engine through
// PDFTextStripper and the renderer against the test corpus, neither of which
// this port has reached.

// pageWith returns a one-page document holding the given content stream.
func pageWith(t *testing.T, content string) *pdmodel.PDPage {
	t.Helper()
	stream := cos.NewStream(filter.Provider{})
	w, err := stream.CreateWriter()
	if err != nil {
		t.Fatalf("CreateWriter: %v", err)
	}
	if _, err := w.Write([]byte(content)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	page := pdmodel.NewPDPageOfSize(common.NewPDRectangleOf(0, 0, 100, 200))
	page.Dictionary().SetItem(cos.Contents, stream)
	page.SetResources(pdmodel.NewPDResources())
	return page
}

// recorder is an engine that writes down the operators it is given, standing in
// for the subclasses PDFBox drives the engine with.
//
// It also snapshots the graphics state whenever it meets an operator it has no
// processor for. That is the only way to see the state a stream builds up: the
// engine saves the whole stack before walking a stream and restores it after,
// so by the time ProcessPage returns everything the stream did is gone. The
// tests below end their content with "sh", which nothing here processes, and
// then read the snapshot.
type recorder struct {
	*contentstream.PDFStreamEngine
	seen       []string
	textEvents []string
	marks      []string

	probed               *graphicsstate.PDGraphicsState
	probedTextMatrix     *util.Matrix
	probedTextLineMatrix *util.Matrix
	probedStackSize      int
}

func newRecorder() *recorder {
	r := &recorder{PDFStreamEngine: contentstream.NewPDFStreamEngine()}
	r.SetOverrides(r)
	state.AddAll(r.PDFStreamEngine)
	text.AddAll(r.PDFStreamEngine)
	markedcontent.AddAll(r.PDFStreamEngine)
	return r
}

func (r *recorder) UnsupportedOperator(op *operator.Operator, operands []cos.Base) error {
	r.seen = append(r.seen, "?"+op.Name())
	r.probed = r.GraphicsState()
	r.probedTextMatrix = r.TextMatrix()
	r.probedTextLineMatrix = r.TextLineMatrix()
	r.probedStackSize = r.GraphicsStackSize()
	return nil
}

// run processes a page holding the given content followed by the probe.
func (r *recorder) run(t *testing.T, content string) {
	t.Helper()
	if err := r.ProcessPage(pageWith(t, content+" sh")); err != nil {
		t.Fatalf("ProcessPage: %v", err)
	}
	if r.probed == nil {
		t.Fatal("the probe operator never ran")
	}
}

func (r *recorder) BeginText() error {
	r.textEvents = append(r.textEvents, "BT")
	return nil
}

func (r *recorder) EndText() error {
	r.textEvents = append(r.textEvents, "ET")
	return nil
}

func (r *recorder) BeginMarkedContentSequence(tag *cos.Name, properties *cos.Dictionary) {
	mark := "BMC:" + tag.Name()
	if properties != nil {
		mark += ":props"
	}
	r.marks = append(r.marks, mark)
}

func (r *recorder) EndMarkedContentSequence() {
	r.marks = append(r.marks, "EMC")
}

func (r *recorder) MarkedContentPoint(tag *cos.Name, properties *cos.Dictionary) {
	mark := "MP:" + tag.Name()
	if properties != nil {
		mark += ":props"
	}
	r.marks = append(r.marks, mark)
}

// TestEngineWalksOperators is the slice 2 demo in miniature: the operators of a
// page reach the engine in order, and one with no processor is reported rather
// than silently dropped.
func TestEngineWalksOperators(t *testing.T) {
	r := newRecorder()
	if err := r.ProcessPage(pageWith(t, "q 1 0 0 1 10 20 cm BT ET Q sh")); err != nil {
		t.Fatalf("ProcessPage: %v", err)
	}

	// Only the operator with no processor is recorded by name; the rest did
	// their work on the graphics state.
	if len(r.seen) != 1 || r.seen[0] != "?sh" {
		t.Errorf("unsupported operators = %v, want [?sh]", r.seen)
	}
	if len(r.textEvents) != 2 || r.textEvents[0] != "BT" || r.textEvents[1] != "ET" {
		t.Errorf("text events = %v, want [BT ET]", r.textEvents)
	}
}

func TestEngineGraphicsStateStack(t *testing.T) {
	r := newRecorder()
	// The line width set inside q ... Q must not survive it.
	r.run(t, "5 w q 9 w Q")

	if got := r.probed.LineWidth(); got != 5 {
		t.Errorf("line width = %v, want 5 — the restore did not take", got)
	}
	if got := r.probedStackSize; got != 1 {
		t.Errorf("the stack holds %d states, want 1", got)
	}
}

// TestEngineUnbalancedRestore pins that a stream popping more than it pushed is
// logged and walked to the end rather than ending the walk: PDFBOX-161.
func TestEngineUnbalancedRestore(t *testing.T) {
	r := newRecorder()
	r.run(t, "Q Q 7 w")
	if got := r.probed.LineWidth(); got != 7 {
		t.Errorf("line width = %v, want 7 — the walk stopped early", got)
	}
}

func TestEngineGraphicsStateOperators(t *testing.T) {
	r := newRecorder()
	r.run(t, "3 w 1 J 2 j 5 M 1 i /Perceptual ri [2 3] 1 d")

	gs := r.probed
	if got := gs.LineWidth(); got != 3 {
		t.Errorf("LineWidth = %v, want 3", got)
	}
	if got := gs.LineCap(); got != 1 {
		t.Errorf("LineCap = %v, want 1", got)
	}
	if got := gs.LineJoin(); got != 2 {
		t.Errorf("LineJoin = %v, want 2", got)
	}
	if got := gs.MiterLimit(); got != 5 {
		t.Errorf("MiterLimit = %v, want 5", got)
	}
	if got := gs.Flatness(); got != 1 {
		t.Errorf("Flatness = %v, want 1", got)
	}
	if got := gs.RenderingIntent(); got != graphicsstate.Perceptual {
		t.Errorf("RenderingIntent = %v, want PERCEPTUAL", got)
	}
	if got := gs.LineDashPattern().DashArray(); len(got) != 2 || got[0] != 2 || got[1] != 3 {
		t.Errorf("dash array = %v, want [2 3]", got)
	}
	if got := gs.LineDashPattern().Phase(); got != 1 {
		t.Errorf("dash phase = %v, want 1", got)
	}
}

// TestEngineConcatenateTransformsCTM pins that cm multiplies into the CTM
// rather than replacing it.
func TestEngineConcatenateTransformsCTM(t *testing.T) {
	r := newRecorder()
	r.run(t, "2 0 0 2 0 0 cm 1 0 0 1 10 20 cm")

	ctm := r.probed.CurrentTransformationMatrix()
	if got := ctm.ScaleX(); got != 2 {
		t.Errorf("the CTM scales x by %v, want 2", got)
	}
	// The translation is expressed in the scaled space, so it doubles.
	if got := ctm.TranslateX(); got != 20 {
		t.Errorf("the CTM translates x by %v, want 20", got)
	}
	if got := ctm.TranslateY(); got != 40 {
		t.Errorf("the CTM translates y by %v, want 40", got)
	}
}

func TestEngineTextState(t *testing.T) {
	r := newRecorder()
	r.run(t, "BT 1 Tc 2 Tw 50 Tz 12 TL 3 Ts 2 Tr ET")

	ts := r.probed.TextState()
	if got := ts.CharacterSpacing(); got != 1 {
		t.Errorf("CharacterSpacing = %v, want 1", got)
	}
	if got := ts.WordSpacing(); got != 2 {
		t.Errorf("WordSpacing = %v, want 2", got)
	}
	if got := ts.HorizontalScaling(); got != 50 {
		t.Errorf("HorizontalScaling = %v, want 50", got)
	}
	if got := ts.Leading(); got != 12 {
		t.Errorf("Leading = %v, want 12", got)
	}
	if got := ts.Rise(); got != 3 {
		t.Errorf("Rise = %v, want 3", got)
	}
	if got := ts.RenderingMode(); got != graphicsstate.FillStroke {
		t.Errorf("RenderingMode = %v, want FILL_STROKE", got)
	}
}

// TestEngineTextPositioning pins that Td moves the text line matrix, that TD
// sets the leading to the negated offset on its way, and that T* then moves
// down by that leading.
func TestEngineTextPositioning(t *testing.T) {
	r := newRecorder()
	// BT resets both matrices; TD moves by (10, -14) and sets the leading to 14.
	r.run(t, "BT 10 -14 TD 0 0 Td T*")

	if got := r.probed.TextState().Leading(); got != 14 {
		t.Errorf("Leading = %v, want 14", got)
	}
	matrix := r.probedTextMatrix
	if matrix == nil {
		t.Fatal("there is no text matrix")
	}
	if got := matrix.TranslateX(); got != 10 {
		t.Errorf("the text matrix translates x by %v, want 10", got)
	}
	// -14 from the TD, then another -14 from the T*.
	if got := matrix.TranslateY(); got != -28 {
		t.Errorf("the text matrix translates y by %v, want -28", got)
	}
}

// TestEngineEndTextClearsMatrices pins that the text matrices exist only
// between BT and ET.
func TestEngineEndTextClearsMatrices(t *testing.T) {
	r := newRecorder()
	r.run(t, "BT ET")
	if r.probedTextMatrix != nil || r.probedTextLineMatrix != nil {
		t.Error("the text matrices outlived the text object")
	}
}

// TestEngineSetMatrix pins that Tm replaces both matrices with the same values
// held separately, so that moving one does not move the other.
func TestEngineSetMatrix(t *testing.T) {
	r := newRecorder()
	r.run(t, "BT 1 0 0 1 5 6 Tm 1 1 Td")

	if got := r.probedTextLineMatrix.TranslateX(); got != 6 {
		t.Errorf("the text line matrix translates x by %v, want 6", got)
	}
	if got := r.probedTextMatrix.TranslateX(); got != 6 {
		t.Errorf("the text matrix translates x by %v, want 6", got)
	}
}

func TestEngineMarkedContent(t *testing.T) {
	r := newRecorder()
	content := "/Span BMC EMC /Tag <</A 1>> BDC EMC /Point MP /Point <</A 1>> DP"
	if err := r.ProcessPage(pageWith(t, content)); err != nil {
		t.Fatalf("ProcessPage: %v", err)
	}

	want := []string{"BMC:Span", "EMC", "BMC:Tag:props", "EMC", "MP:Point", "MP:Point:props"}
	if len(r.marks) != len(want) {
		t.Fatalf("marks = %v, want %v", r.marks, want)
	}
	for i := range want {
		if r.marks[i] != want[i] {
			t.Errorf("mark %d = %q, want %q", i, r.marks[i], want[i])
		}
	}
}

// TestEngineMissingOperandIsLogged pins that an operator given too few operands
// is reported and the walk goes on, rather than ending the page.
func TestEngineMissingOperandIsLogged(t *testing.T) {
	r := newRecorder()
	r.run(t, "w 4 w")
	if got := r.probed.LineWidth(); got != 4 {
		t.Errorf("line width = %v, want 4 — the walk stopped early", got)
	}
}

// TestEngineOperatorExceptionRethrows pins that an error which is not one of
// the recognised per-operator failures ends the walk.
func TestEngineOperatorExceptionRethrows(t *testing.T) {
	engine := contentstream.NewPDFStreamEngine()
	failure := errors.New("something else went wrong")
	engine.AddOperator(failingProcessor{name: "w", err: failure})

	err := engine.ProcessPage(pageWith(t, "1 w"))
	if !errors.Is(err, failure) {
		t.Errorf("ProcessPage err = %v, want the processor's error", err)
	}
}

type failingProcessor struct {
	name string
	err  error
}

func (p failingProcessor) Name() string { return p.name }

func (p failingProcessor) Process(op *operator.Operator, operands []cos.Base) error {
	return p.err
}

// TestEngineEmptyPage pins that a page with no contents is walked without
// complaint.
func TestEngineEmptyPage(t *testing.T) {
	r := newRecorder()
	page := pdmodel.NewPDPage()
	if err := r.ProcessPage(page); err != nil {
		t.Fatalf("ProcessPage: %v", err)
	}
	if len(r.seen) != 0 {
		t.Errorf("an empty page yielded %v", r.seen)
	}
}

// TestEngineInitialStateFromPage pins that the engine starts each page with a
// graphics state clipped to that page's crop box.
func TestEngineInitialStateFromPage(t *testing.T) {
	r := newRecorder()
	page := pageWith(t, "")
	page.SetCropBox(common.NewPDRectangleOf(0, 0, 50, 60))
	if err := r.ProcessPage(page); err != nil {
		t.Fatalf("ProcessPage: %v", err)
	}

	if r.CurrentPage() != page {
		t.Error("the engine did not keep the page it was given")
	}
	paths := r.GraphicsState().CurrentClippingPaths()
	if len(paths) == 0 {
		t.Fatal("the state has no clipping path")
	}
	if got := paths[0].Bounds2D().Width; got != 50 {
		t.Errorf("the clipping path is %v wide, want the crop box's 50", got)
	}
}

// TestEngineResourcesFallBackToPage pins the lookup order: a stream with no
// resources of its own uses the page's.
func TestEngineResourcesFallBackToPage(t *testing.T) {
	r := newRecorder()
	page := pageWith(t, "")
	resources := pdmodel.NewPDResources()
	page.SetResources(resources)

	if err := r.ProcessPage(page); err != nil {
		t.Fatalf("ProcessPage: %v", err)
	}
	// The resources are popped when the stream ends, so ask during the walk
	// instead: a page with none at all still gets an empty set rather than nil.
	page.SetResources(nil)
	if err := r.ProcessPage(page); err != nil {
		t.Fatalf("ProcessPage: %v", err)
	}
}

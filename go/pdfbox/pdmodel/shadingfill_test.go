package pdmodel_test

// The `sh` operator, which a content stream could not write.
//
// `pdabstractcontentstream.go` says: "shadingFill is not here: it names
// PDShading, which belongs to the rendering this port has not reached, and
// PDResources cannot add one either." Slice 9 ported `PDShading` and all seven
// shading types; the reason stopped being true and nothing went back to look.
//
// What is missing is the writing side, and it is two methods: `PDResources.add
// (PDShading)` and `PDAbstractContentStream.shadingFill`. Neither has anything
// to do with a rasteriser -- `sh` paints a shading into the current clip, and
// what a reader does with it later is the renderer's business, not the writer's.

import (
	"bytes"
	"strings"
	"testing"

	pdfbox "github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/shading"
)

// axialShading builds the simplest shading a content stream can name: a type 2
// axial shading over device grey, from one corner to another.
func axialShading() *shading.PDShadingType2 {
	dictionary := cos.NewDictionary()
	dictionary.SetInt(cos.ShadingType, 2)
	dictionary.SetItem(cos.ColorSpace, cos.GetPDFName("DeviceGray"))
	coordinates := cos.NewArray()
	for _, value := range []float64{0, 0, 100, 100} {
		coordinates.Add(cos.NewFloat(float32(value)))
	}
	dictionary.SetItem(cos.Coords, coordinates)

	// A type 2 exponential function from black to white, which is the least a
	// shading can carry and still be one.
	function := cos.NewDictionary()
	function.SetInt(cos.FunctionType, 2)
	domain := cos.NewArray()
	domain.Add(cos.NewFloat(0))
	domain.Add(cos.NewFloat(1))
	function.SetItem(cos.Domain, domain)
	c0 := cos.NewArray()
	c0.Add(cos.NewFloat(0))
	c1 := cos.NewArray()
	c1.Add(cos.NewFloat(1))
	function.SetItem(cos.GetPDFName("C0"), c0)
	function.SetItem(cos.GetPDFName("C1"), c1)
	function.SetItem(cos.N, cos.NewFloat(1))
	dictionary.SetItem(cos.Function, function)

	return shading.NewPDShadingType2(dictionary)
}

// TestShadingFillWritesTheOperator is what the missing method is for: a shading
// added to the resources, and `sh` naming it in the content stream.
func TestShadingFillWritesTheOperator(t *testing.T) {
	document := pdmodel.NewPDDocument()
	defer document.Close()
	page := pdmodel.NewPDPageOfSize(common.A4)
	document.AddPage(page)

	stream, err := pdmodel.NewPDPageContentStream(document, page)
	if err != nil {
		t.Fatalf("NewPDPageContentStream: %v", err)
	}
	if err := stream.ShadingFill(axialShading()); err != nil {
		t.Fatalf("ShadingFill: %v", err)
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	var saved bytes.Buffer
	if err := document.Save(&saved); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reloaded, err := pdfbox.LoadPDFBytes(saved.Bytes())
	if err != nil {
		t.Fatalf("LoadPDFBytes: %v", err)
	}
	defer reloaded.Close()

	// The resource is named, and the name Java gives it starts with "sh".
	names := reloaded.Page(0).Resources().ShadingNames()
	if len(names) != 1 {
		t.Fatalf("the page has %d shading resources, want 1", len(names))
	}
	if got := names[0].Name(); !strings.HasPrefix(got, "sh") {
		t.Errorf("the shading resource is named %q; Java names it with the "+
			"prefix \"sh\"", got)
	}

	// And the operator names it.
	contents := contentsOfFirstPage(t, reloaded)
	want := "/" + names[0].Name() + " sh"
	if !strings.Contains(contents, want) {
		t.Errorf("the content stream is %q, want it to carry %q", contents, want)
	}
}

// TestShadingFillIsRefusedInATextBlock is the Java's guard, which throws
// IllegalStateException with this message.
func TestShadingFillIsRefusedInATextBlock(t *testing.T) {
	document := pdmodel.NewPDDocument()
	defer document.Close()
	page := pdmodel.NewPDPageOfSize(common.A4)
	document.AddPage(page)

	stream, err := pdmodel.NewPDPageContentStream(document, page)
	if err != nil {
		t.Fatalf("NewPDPageContentStream: %v", err)
	}
	if err := stream.BeginText(); err != nil {
		t.Fatal(err)
	}
	// Java throws IllegalStateException, which is unchecked; this port renders
	// an unchecked exception as a panic, which is what every other guard in
	// PDAbstractContentStream does -- stroke, fill, closeAndStroke.
	const want = "Error: shadingFill is not allowed within a text block."
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Error("shadingFill inside a text block was accepted")
			return
		}
		if got, isString := recovered.(string); !isString || got != want {
			t.Errorf("the panic is %v, want %q", recovered, want)
		}
	}()
	_ = stream.ShadingFill(axialShading())
}

// contentsOfFirstPage reads a document's first page content stream.
func contentsOfFirstPage(t *testing.T, document *pdmodel.PDDocument) string {
	t.Helper()
	var out bytes.Buffer
	for _, stream := range document.Page(0).ContentStreams() {
		reader, err := stream.CreateInputStream()
		if err != nil {
			t.Fatalf("reading the contents: %v", err)
		}
		if _, err := out.ReadFrom(reader); err != nil {
			t.Fatalf("reading the contents: %v", err)
		}
	}
	return out.String()
}

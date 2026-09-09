package rendering

// JAVA-BUGS 49: `PageDrawer.drawTilingPattern` swaps six fields of the drawer,
// runs the pattern's content stream, and swaps them back with plain
// statements rather than a `finally`, so an error out of the stream leaves the
// drawer pointing at the tile.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/filter"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/pattern"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// brokenTilingPattern is a pattern whose content stream cannot be read: it
// declares a filter no decoder answers to, so the engine fails where the
// entry says it fails, part way through drawTilingPattern.
func brokenTilingPattern(t *testing.T) *pattern.PDTilingPattern {
	t.Helper()
	tiling := pattern.NewPDTilingPattern(filter.Provider{})
	tiling.SetPaintType(1)
	tiling.SetTilingType(1)
	tiling.SetBBox(common.NewPDRectangleOfSize(16, 16))
	tiling.SetXStep(16)
	tiling.SetYStep(16)

	stream := tiling.ContentStream()
	output, err := stream.CreateOutputStream()
	if err != nil {
		t.Fatalf("CreateOutputStream: %v", err)
	}
	if _, err := output.Write([]byte("q Q\n")); err != nil {
		t.Fatalf("writing the pattern stream: %v", err)
	}
	if err := output.Close(); err != nil {
		t.Fatalf("closing the pattern stream: %v", err)
	}
	tiling.Dictionary().SetItem(cos.Filter, cos.GetPDFName("NoSuchFilter"))
	return tiling
}

// TestDrawTilingPatternRestoresAfterAFailure is the defect.
//
// The expected behaviour is the three siblings' in the same class -- the
// transparency group constructor, ProcessTransparencyGroup and
// ProcessTilingPatternMatrix itself -- all of which restore in a `finally`.
// The drawer is a field of the renderer and the rest of the page is drawn
// after this returns, so leaving it pointed at the tile's backend is not
// recoverable.
func TestDrawTilingPatternRestoresAfterAFailure(t *testing.T) {
	document, page := pageWithContent(t, 100, 100, "")
	drawer, err := NewPageDrawer(newPageDrawerParameters(
		NewPDFRenderer(document), page, false, Export, RenderingHints{}, 0))
	if err != nil {
		t.Fatalf("NewPageDrawer: %v", err)
	}

	// draw the (empty) page first, so the engine is initialised the way it is
	// when a real tiling pattern reaches the drawer
	pageBackend := newRecordingBackend()
	if err := drawer.DrawPage(pageBackend, common.NewPDRectangleOfSize(100, 100)); err != nil {
		t.Fatalf("DrawPage: %v", err)
	}
	// TilingPaint reaches the drawer while the page is still being drawn, so put
	// the page's backend back where DrawPage's own cleanup left nil
	drawer.backend = pageBackend
	pagePath := drawer.linePath
	drawer.clipWindingRule = 7

	tileBackend := newRecordingBackend()
	err = drawer.DrawTilingPattern(tileBackend, brokenTilingPattern(t), nil, nil,
		util.NewMatrix())
	if err == nil {
		t.Fatal("the broken pattern drew without an error, so the test proves nothing")
	}

	if drawer.backend != pageBackend {
		t.Error("the drawer is still pointed at the tile's backend")
	}
	if drawer.linePath != pagePath {
		t.Error("the drawer is still holding the tile's line path")
	}
	if drawer.flipTG {
		t.Error("flipTG is still the tile's true")
	}
	if drawer.clipWindingRule != 7 {
		t.Errorf("the clip winding rule is %d, want the page's 7", drawer.clipWindingRule)
	}
	if drawer.hasLastClips {
		t.Error("the tile's clip bookkeeping outlived it")
	}
}

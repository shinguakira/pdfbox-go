// Package glyphlayout lays text out into positioned glyphs.
//
// It is the Go answer to the two Java modules `pdfbox-layout-awt` and
// `pdfbox-layout-fop`, and it is **not a port of either**. Both of those are a
// thin shell around one call into a library that shapes text —
// `java.awt.Font.layoutGlyphVector` in one, Apache FOP's `GlyphMapping` in the
// other — and Go has neither library. There is no third Java implementation in
// PDFBox to translate: the two modules exist precisely because PDFBox has no
// shaper of its own.
//
// So this package is the shaper, assembled from the two halves the port does
// have:
//
//   - **GSUB**, which slice 4 ported from PDFBox's own reader, decides which
//     glyph to draw — an `f` and an `i` become an `fi`, an Arabic letter takes
//     its initial or medial form.
//   - **GPOS**, written for this branch from the OpenType specification because
//     neither PDFBox nor Go has one, decides where each glyph goes — the kern
//     that pulls a `V` under an `A`, the vowel mark that sits over the right
//     letter.
//
// The bidirectional splitting above it is `pdmodel.AbstractGlyphLayoutProcessor`,
// which *is* a port, over `javatext/bidi`.
//
// Java's package names say which backend they are — `glyphlayout.awt`,
// `glyphlayout.fop`. This one is neither, so it is not called either; naming it
// `awt` would claim a fidelity it does not have. See migration/STATUS.md.
package glyphlayout

import (
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf"
	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf/gsub"
	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf/model"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
)

// fontScale is the thousandth of an em that a PDF text space unit is.
const fontScale = 1000

// Processor lays text out and writes it to a content stream.
//
// It fills the same place as `GlyphLayoutProcessorAwt`: it satisfies
// pdmodel.GlyphLayoutProcessor, and PDPageContentStream.SetGlyphLayoutProcessor
// takes it.
type Processor struct {
	pdmodel.AbstractGlyphLayoutProcessor

	// features are the OpenType features to turn on. Java carries these as
	// java.awt.font.TextAttribute values on the font -- KERNING_ON,
	// LIGATURES_ON -- and defaults to neither; this port names them by their
	// OpenType tags and defaults to the same nothing.
	features Features

	gsubFactory *gsub.Factory
	gsubWorkers map[*font.PDType0Font]gsub.GsubWorker
}

// Features says which OpenType features the layout turns on.
//
// Port of the meaning of GlyphLayoutFontLoaderAwt.FontOptions, which is two
// booleans; the tags are what those two mean in a font.
type Features struct {
	// Kerning turns on "kern", which is Java's TextAttribute.KERNING_ON.
	Kerning bool

	// Ligatures turns on "liga", which is TextAttribute.LIGATURES_ON.
	Ligatures bool

	// Marks turns on "mark" and "mkmk", which place a combining mark against
	// the letter it belongs to. Java has no option for these because
	// layoutGlyphVector always does them; this port defaults them on for the
	// same reason, and the field is here so a caller can say otherwise.
	NoMarks bool
}

var _ pdmodel.GlyphLayoutProcessor = (*Processor)(nil)

// NewProcessor returns a processor with no optional feature turned on, which is
// what `new GlyphLayoutProcessorAwt()` gives.
func NewProcessor() *Processor { return NewProcessorWithFeatures(Features{}) }

// NewProcessorWithFeatures returns a processor with the given features.
func NewProcessorWithFeatures(features Features) *Processor {
	p := &Processor{
		features:    features,
		gsubFactory: gsub.NewFactory(),
		gsubWorkers: map[*font.PDType0Font]gsub.GsubWorker{},
	}
	p.StringWidthUni = p.stringWidthUni
	p.ShowTextUniFunc = p.showTextUni
	return p
}

// SupportsFont reports whether this processor can lay text out in the font.
//
// Port of supportsFont, which answers true for a PDType0Font whose program is a
// TrueType font with glyf outlines. A CFF-based OpenType font is refused for
// the same reason Java refuses it: PDFBox does not support one here.
func (p *Processor) SupportsFont(f font.PDFont) bool {
	type0, isType0 := f.(*font.PDType0Font)
	if !isType0 {
		return false
	}
	return type0.TrueTypeFont() != nil
}

// positionedGlyph is one glyph of a laid-out run: which glyph, where it sits
// relative to the pen, and how far the pen moves after it.
//
// It is what `java.awt.font.GlyphVector` carries, reduced to what showTextUni
// reads off one.
type positionedGlyph struct {
	code    int
	xOffset int
	yOffset int

	// advance is how far the pen moves after the glyph, with whatever GPOS
	// added to it. It is what `GlyphVector.getGlyphPosition` accumulates.
	advance int

	// natural is the advance the font declares in hmtx, before positioning.
	// It is what `GlyphVector.getGlyphMetrics(i).getAdvanceX()` answers, and
	// showTextUni needs both numbers to work out what to write.
	natural int
}

// layout shapes a run of text that goes one way.
//
// Port of computeGlyphVector in shape rather than in code: Java hands the
// characters to the platform and reads a GlyphVector back, and this does the
// two steps that call is made of.
func (p *Processor) layout(f *font.PDType0Font, text string,
	bidiLevel int) ([]positionedGlyph, error) {
	program := f.TrueTypeFont()
	if program == nil {
		return nil, fmt.Errorf("glyphlayout: the font carries no TrueType program")
	}

	cmap, err := program.UnicodeCmapLookup(true)
	if err != nil {
		return nil, err
	}

	// The characters in the order they are written, which is the order to shape
	// them in whichever way they read: a letter takes its joining form from the
	// letters around it in the text, not on the page. The run is turned round
	// at the end, where it goes right to left.
	glyphs := make([]int, 0, len(text))
	for _, r := range text {
		gid := cmap.GetGlyphID(int(r))
		if gid == 0 {
			return nil, missingGlyph(program, r)
		}
		glyphs = append(glyphs, gid)
	}

	glyphs, err = p.substitute(f, program, glyphs)
	if err != nil {
		return nil, err
	}

	hmtx, err := program.HorizontalMetrics()
	if err != nil {
		return nil, err
	}

	laid := make([]positionedGlyph, len(glyphs))
	for i, gid := range glyphs {
		width := hmtx.AdvanceWidth(gid)
		laid[i] = positionedGlyph{code: gid, advance: width, natural: width}
	}

	if err := p.position(program, glyphs, laid, bidiLevel); err != nil {
		return nil, err
	}

	// A right-to-left run is drawn from its last character to its first. Java
	// asks for this with Font.LAYOUT_RIGHT_TO_LEFT and gets a GlyphVector back
	// that is already in that order; the caller's bidi split only put the run
	// itself in the right place on the line, not the letters inside it.
	if bidiLevel%2 != 0 {
		for i, j := 0, len(laid)-1; i < j; i, j = i+1, j-1 {
			laid[i], laid[j] = laid[j], laid[i]
		}
	}
	return laid, nil
}

// substitute runs GSUB, which is the half of the shaper the port already had.
func (p *Processor) substitute(f *font.PDType0Font, program *ttf.TrueTypeFont,
	glyphs []int) ([]int, error) {
	gsubData, err := program.GsubData()
	if err != nil {
		return nil, err
	}
	if gsubData == nil {
		return glyphs, nil
	}
	if !p.features.Ligatures && !isScriptShaping(gsubData.Language()) {
		// Java turns ligatures on only when the font was loaded with
		// LIGATURES_ON, and measuring the reference PDF says it means it: with
		// no options the Latin lines carry no ligature at all. The
		// script-specific workers are a different matter -- see
		// isScriptShaping.
		return glyphs, nil
	}
	worker, known := p.gsubWorkers[f]
	if !known {
		worker = p.gsubFactory.GetGsubWorker(f.CmapLookup(), gsubData)
		p.gsubWorkers[f] = worker
	}
	if worker == nil {
		return glyphs, nil
	}
	return worker.ApplyTransforms(glyphs), nil
}

// isScriptShaping reports whether a language's worker does more than apply
// optional typographic features.
//
// The Indic workers reorder the glyphs and build conjuncts: without them the
// text is not merely plainer, it is in the wrong order and unreadable. Java
// gets that from the platform whether or not ligatures were asked for -- the
// reference PDF shows the pre-base vowel moved ahead of its consonant on the
// line written with no options at all -- and PDAbstractContentStream.showText
// applies them for every Type 0 font that has GSUB data, with no option to
// turn them off. So they run here regardless, and only the ligature features
// of the Latin and default workers wait to be asked for.
func isScriptShaping(language model.Language) bool {
	switch language {
	case model.Bengali, model.Devanagari, model.Gujarati, model.Tamil:
		return true
	default:
		return false
	}
}

// position runs GPOS, which is the half written for this branch.
func (p *Processor) position(program *ttf.TrueTypeFont, glyphs []int,
	laid []positionedGlyph, bidiLevel int) error {
	tags := p.featureTags()
	if len(tags) == 0 {
		return nil
	}
	gpos, err := program.GPOS()
	if err != nil || gpos == nil {
		// A font with no GPOS positions its glyphs by their advances alone,
		// which is what `laid` already holds.
		return nil
	}

	scripts := p.scriptTagsFor(program, bidiLevel)
	positions := gpos.Position(glyphs, scripts, tags)
	for i := range laid {
		laid[i].advance += positions[i].XAdvance
	}
	resolveAttachments(laid, positions, bidiLevel%2 != 0)
	return nil
}

// resolveAttachments turns each adjustment into a distance from the pen.
//
// A placement GPOS gives outright is already that. An attachment is not: it is
// a distance from the origin of the glyph the mark hangs off, and where that
// origin sits depends on the order the run is drawn in. In a left-to-right run
// the pen reaches the letter first and the mark second; in a right-to-left one
// it is the other way round, and a mark whose letter has not been drawn yet
// has to reach forwards for it.
//
// A mark can hang off another mark, and that one may have been moved as well,
// so the second is measured from where the first ended up. Java gets all of
// this from layoutGlyphVector, which answers positions rather than offsets.
func resolveAttachments(laid []positionedGlyph, positions []ttf.GlyphPosition,
	rightToLeft bool) {
	// Where the pen is when each glyph is drawn. The run is written in the
	// order it is drawn, so a right-to-left run counts from the other end.
	pen := make([]int, len(laid))
	running := 0
	for i := range laid {
		at := i
		if rightToLeft {
			at = len(laid) - 1 - i
		}
		pen[at] = running
		running += laid[at].advance
	}

	// In the order the marks were found, so a mark that hangs off a mark sees
	// where that one ended up. A subtable only ever attaches a glyph to one
	// before it in the text, whichever way the text runs.
	for i := range laid {
		if positions[i].AttachedTo == ttf.NotAttached {
			laid[i].xOffset += positions[i].XPlacement
			laid[i].yOffset += positions[i].YPlacement
			continue
		}
		base := positions[i].AttachedTo
		laid[i].xOffset += pen[base] + laid[base].xOffset + positions[i].XPlacement - pen[i]
		laid[i].yOffset += laid[base].yOffset + positions[i].YPlacement
	}
}

// featureTags answers which OpenType features to turn on.
//
// The mark features are four, not two. "mark" and "mkmk" are what a Latin,
// Greek or Arabic font puts its mark anchors under; an Indic font puts them
// under "abvm" and "blwm" instead -- above-base and below-base mark
// positioning -- and asking only for the first two leaves a Bengali vowel sign
// sitting on the baseline. All four say the same thing, which is where a mark
// goes, so all four are asked for together.
func (p *Processor) featureTags() []string {
	var tags []string
	if p.features.Kerning {
		tags = append(tags, "kern")
	}
	if !p.features.NoMarks {
		tags = append(tags, "mark", "mkmk", "abvm", "blwm")
	}
	return tags
}

// scriptTagsFor answers which OpenType scripts to look the features up under.
//
// The font's own substitution data names a script, and that is the best answer
// there is: it is what chose the GSUB worker, so positioning and substitution
// agree about what is being written. Without it a Bengali font gets its
// features looked up under "latn", finds none, and its vowel marks are never
// placed.
//
// A run's direction is what is left, and it is what the two Java backends pass
// down -- `Font.LAYOUT_LEFT_TO_RIGHT` or `LAYOUT_RIGHT_TO_LEFT`. "DFLT" is the
// fallback every font is meant to carry, and GlyphPositioningTable falls back
// to it on its own where none of these is present.
func (p *Processor) scriptTagsFor(program *ttf.TrueTypeFont, bidiLevel int) []string {
	var tags []string
	if gsubData, err := program.GsubData(); err == nil && gsubData != nil {
		tags = append(tags, gsubData.Language().ScriptNames()...)
	}
	if bidiLevel%2 == 0 {
		return append(tags, "latn", "DFLT")
	}
	return append(tags, "arab", "hebr", "DFLT")
}

// missingGlyph is the failure checkMissingGlyphs throws.
//
// Port of it, message and all. Two details of the Java are carried rather than
// improved on:
//
//   - The name is `awtFont.getName()`, which for a font read from a stream is
//     the full font name in its name table -- "Lohit Bengali", not the
//     PostScript name "Lohit-Bengali" that fontbox answers.
//   - `'%c'` is fed a `char`, which is one UTF-16 code unit, while the code
//     point beside it is the whole character. For anything outside the basic
//     plane Java prints half a surrogate pair there, and so does this. See
//     migration/JAVA-BUGS.md.
func missingGlyph(program *ttf.TrueTypeFont, r rune) error {
	unit := r
	if r > 0xFFFF {
		unit = 0xD800 + (r-0x10000)>>10
	}
	return fmt.Errorf(
		"Missing glyph in font '%s' for the character '%c', codePoint: %d (U+%04x).",
		fontName(program), unit, r, r)
}

// fontName answers the font's name for a message, or a placeholder.
//
// Java's `Font.getName()` answers the font's full name -- name ID 4 -- looked
// up in the same order of platforms fontbox uses for the family name. fontbox
// itself reads only the family, sub-family and PostScript names, so the lookup
// is spelled out here.
func fontName(program *ttf.TrueTypeFont) string {
	naming, err := program.Naming()
	if err == nil && naming != nil {
		if name, ok := fullFontName(naming); ok {
			return name
		}
	}
	name, err := program.Name()
	if err != nil || name == "" {
		return "unknown"
	}
	return name
}

// fullFontName answers name ID 4, by the same best effort fontbox makes for
// the names it does read.
func fullFontName(naming *ttf.NamingTable) (string, bool) {
	// Unicode, from the newest encoding to the oldest.
	for encoding := 4; encoding >= 0; encoding-- {
		if name, ok := naming.GetName(ttf.NameFullFontName, ttf.PlatformUnicode,
			encoding, ttf.LanguageUnicode); ok {
			return name, true
		}
	}
	if name, ok := naming.GetName(ttf.NameFullFontName, ttf.PlatformWindows,
		ttf.EncodingWindowsUnicodeBMP, ttf.LanguageWindowsENUS); ok {
		return name, true
	}
	return naming.GetName(ttf.NameFullFontName, ttf.PlatformMacintosh,
		ttf.EncodingMacintoshRoman, ttf.LanguageMacintoshEnglish)
}

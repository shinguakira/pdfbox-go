package state

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/blend"
)

// CacheLike is the resource cache an extended graphics state reads its soft
// mask through.
//
// Java names ResourceCache, which lives in pdmodel; pdmodel imports this
// package through the graphics state, so the dependency cannot run both ways.
// The port names what is used, which is nothing yet: the soft mask is the one
// entry this class does not read, since PDSoftMask is not ported.
type CacheLike any

// PDExtendedGraphicsState is an /ExtGState dictionary: a set of graphics state
// parameters a content stream can switch to in one operator.
//
// Port of PDExtendedGraphicsState.
type PDExtendedGraphicsState struct {
	dict  *cos.Dictionary
	cache CacheLike
}

var _ common.COSObjectable = (*PDExtendedGraphicsState)(nil)

// NewPDExtendedGraphicsState builds an empty extended graphics state.
func NewPDExtendedGraphicsState() *PDExtendedGraphicsState {
	dict := cos.NewDictionary()
	dict.SetItem(cos.Type, cos.ExtGState)
	return &PDExtendedGraphicsState{dict: dict}
}

// NewPDExtendedGraphicsStateOf builds one over the given dictionary.
func NewPDExtendedGraphicsStateOf(dictionary *cos.Dictionary) *PDExtendedGraphicsState {
	return NewPDExtendedGraphicsStateOfCache(dictionary, nil)
}

// NewPDExtendedGraphicsStateOfCache builds one that reads through the given
// cache.
func NewPDExtendedGraphicsStateOfCache(dictionary *cos.Dictionary,
	resourceCache CacheLike) *PDExtendedGraphicsState {
	return &PDExtendedGraphicsState{dict: dictionary, cache: resourceCache}
}

// CopyIntoGraphicsState applies every parameter this dictionary holds to the
// given graphics state.
//
// The soft mask the /SMask entry names is handed the current transformation
// matrix at the moment it is installed, which is what it is later painted
// through.
func (e *PDExtendedGraphicsState) CopyIntoGraphicsState(gs *PDGraphicsState) error {
	for _, key := range e.dict.KeySet() {
		switch key {
		case cos.LW:
			gs.SetLineWidth(defaultIfNil(e.LineWidth(), 1))
		case cos.LC:
			gs.SetLineCap(e.LineCapStyle())
		case cos.LJ:
			gs.SetLineJoin(e.LineJoinStyle())
		case cos.ML:
			gs.SetMiterLimit(defaultIfNil(e.MiterLimit(), 10))
		case cos.D:
			gs.SetLineDashPattern(e.LineDashPattern())
		case cos.RI:
			gs.SetRenderingIntentOrNil(e.RenderingIntent())
		case cos.OPM:
			overprintMode := e.OverprintMode()
			if overprintMode != nil {
				gs.SetOverprintMode(*overprintMode)
			} else {
				gs.SetOverprintMode(0)
			}
		case cos.OP:
			gs.SetOverprint(e.StrokingOverprintControl())
		case cos.Op:
			gs.SetNonStrokingOverprint(e.NonStrokingOverprintControl())
		case cos.Font:
			setting := e.FontSetting()
			if setting != nil {
				settingFont, err := setting.Font()
				if err != nil {
					return err
				}
				gs.TextState().SetFont(settingFont)
				gs.TextState().SetFontSize(setting.FontSize())
			}
		case cos.FL:
			gs.SetFlatness(float64(defaultIfNil(e.FlatnessTolerance(), 1)))
		case cos.SM:
			gs.SetSmoothness(float64(defaultIfNil(e.SmoothnessTolerance(), 0)))
		case cos.SA:
			gs.SetStrokeAdjustment(e.AutomaticStrokeAdjustment())
		case cos.CA:
			gs.SetAlphaConstant(float64(defaultIfNil(e.StrokingAlphaConstant(), 1)))
		case cos.Ca:
			gs.SetNonStrokeAlphaConstant(float64(defaultIfNil(e.NonStrokingAlphaConstant(), 1)))
		case cos.AIS:
			gs.SetAlphaSource(e.AlphaSourceFlag())
		case cos.TK:
			gs.TextState().SetKnockoutFlag(e.TextKnockoutFlag())
		case cos.SMask:
			softmask := e.SoftMask()
			if softmask != nil {
				// Softmask must know the CTM at the time the ExtGState is
				// activated. Read
				// https://bugs.ghostscript.com/show_bug.cgi?id=691157#c7 for a
				// good explanation.
				softmask.SetInitialTransformationMatrix(
					gs.CurrentTransformationMatrix().Clone())
			}
			gs.SetSoftMask(softmask)
		case cos.BM:
			gs.SetBlendMode(e.BlendMode())
		case cos.TR:
			if e.dict.ContainsKey(cos.TR2) {
				// "If both TR and TR2 are present in the same graphics state
				// parameter dictionary, TR2 shall take precedence."
				continue
			}
			gs.SetTransfer(e.Transfer())
		case cos.TR2:
			gs.SetTransfer(e.Transfer2())
		}
	}
	return nil
}

// defaultIfNil answers the value, or the default where Java has null.
func defaultIfNil(standardValue *float32, defaultValue float32) float32 {
	if standardValue != nil {
		return *standardValue
	}
	return defaultValue
}

// COSObject returns the dictionary.
func (e *PDExtendedGraphicsState) COSObject() cos.Base { return e.dict }

// Dictionary returns the dictionary, typed.
func (e *PDExtendedGraphicsState) Dictionary() *cos.Dictionary { return e.dict }

// LineWidth returns the /LW line width, or nil.
func (e *PDExtendedGraphicsState) LineWidth() *float32 {
	return e.floatItem(cos.LW)
}

// SetLineWidth sets the /LW line width, and removes it where the value is nil.
func (e *PDExtendedGraphicsState) SetLineWidth(width *float32) {
	e.setFloatItem(cos.LW, width)
}

// LineCapStyle returns the /LC line cap style.
func (e *PDExtendedGraphicsState) LineCapStyle() int {
	return e.dict.GetInt(cos.LC)
}

// SetLineCapStyle sets the /LC line cap style.
func (e *PDExtendedGraphicsState) SetLineCapStyle(style int) {
	e.dict.SetInt(cos.LC, style)
}

// LineJoinStyle returns the /LJ line join style.
func (e *PDExtendedGraphicsState) LineJoinStyle() int {
	return e.dict.GetInt(cos.LJ)
}

// SetLineJoinStyle sets the /LJ line join style.
func (e *PDExtendedGraphicsState) SetLineJoinStyle(style int) {
	e.dict.SetInt(cos.LJ, style)
}

// MiterLimit returns the /ML miter limit, or nil.
func (e *PDExtendedGraphicsState) MiterLimit() *float32 {
	return e.floatItem(cos.ML)
}

// SetMiterLimit sets the /ML miter limit, and removes it where the value is
// nil.
func (e *PDExtendedGraphicsState) SetMiterLimit(miterLimit *float32) {
	e.setFloatItem(cos.ML, miterLimit)
}

// LineDashPattern returns the /D dash pattern, or nil.
func (e *PDExtendedGraphicsState) LineDashPattern() *graphics.PDLineDashPattern {
	dp := e.dict.GetCOSArray(cos.D)
	if dp == nil || dp.Size() != 2 {
		return nil
	}
	dashArray, isArray := dp.GetObject(0).(*cos.Array)
	phase, isNumber := dp.GetObject(1).(cos.Number)
	if isArray && isNumber {
		return graphics.NewPDLineDashPatternOf(dashArray, phase.IntValue())
	}
	return nil
}

// SetLineDashPattern sets the /D dash pattern.
func (e *PDExtendedGraphicsState) SetLineDashPattern(dashPattern *graphics.PDLineDashPattern) {
	e.dict.SetItem(cos.D, dashPattern.COSObject())
}

// RenderingIntent returns the /RI rendering intent, or nil.
func (e *PDExtendedGraphicsState) RenderingIntent() *RenderingIntent {
	ri := e.dict.GetNameAsString(cos.RI, "")
	if ri == "" {
		return nil
	}
	intent := RenderingIntentFromString(ri)
	return &intent
}

// SetRenderingIntent sets the /RI rendering intent.
func (e *PDExtendedGraphicsState) SetRenderingIntent(ri string) {
	e.dict.SetName(cos.RI, ri)
}

// StrokingOverprintControl returns the /OP overprint flag.
func (e *PDExtendedGraphicsState) StrokingOverprintControl() bool {
	return e.dict.GetBoolean(cos.OP, false)
}

// SetStrokingOverprintControl sets the /OP overprint flag.
func (e *PDExtendedGraphicsState) SetStrokingOverprintControl(op bool) {
	e.dict.SetBoolean(cos.OP, op)
}

// NonStrokingOverprintControl returns the /op overprint flag, which defaults to
// the stroking one.
func (e *PDExtendedGraphicsState) NonStrokingOverprintControl() bool {
	return e.dict.GetBoolean(cos.Op, e.StrokingOverprintControl())
}

// SetNonStrokingOverprintControl sets the /op overprint flag.
func (e *PDExtendedGraphicsState) SetNonStrokingOverprintControl(op bool) {
	e.dict.SetBoolean(cos.Op, op)
}

// OverprintMode returns the /OPM overprint mode, or nil.
func (e *PDExtendedGraphicsState) OverprintMode() *int {
	if value, isNumber := e.dict.GetDictionaryObject(cos.OPM).(cos.Number); isNumber {
		mode := value.IntValue()
		return &mode
	}
	return nil
}

// SetOverprintMode sets the /OPM overprint mode, and removes it where the value
// is nil.
func (e *PDExtendedGraphicsState) SetOverprintMode(overprintMode *int) {
	if overprintMode == nil {
		e.dict.RemoveItem(cos.OPM)
		return
	}
	e.dict.SetInt(cos.OPM, *overprintMode)
}

// FontSetting returns the /Font setting, or nil.
func (e *PDExtendedGraphicsState) FontSetting() *graphics.PDFontSetting {
	if fontSetting := e.dict.GetCOSArray(cos.Font); fontSetting != nil {
		return graphics.NewPDFontSettingOf(fontSetting)
	}
	return nil
}

// SetFontSetting sets the /Font setting.
func (e *PDExtendedGraphicsState) SetFontSetting(fs *graphics.PDFontSetting) {
	if fs == nil {
		e.dict.SetItem(cos.Font, nil)
		return
	}
	e.dict.SetItem(cos.Font, fs.COSObject())
}

// FlatnessTolerance returns the /FL flatness tolerance, or nil.
func (e *PDExtendedGraphicsState) FlatnessTolerance() *float32 {
	return e.floatItem(cos.FL)
}

// SetFlatnessTolerance sets the /FL flatness tolerance.
func (e *PDExtendedGraphicsState) SetFlatnessTolerance(flatness *float32) {
	e.setFloatItem(cos.FL, flatness)
}

// SmoothnessTolerance returns the /SM smoothness tolerance, or nil.
func (e *PDExtendedGraphicsState) SmoothnessTolerance() *float32 {
	return e.floatItem(cos.SM)
}

// SetSmoothnessTolerance sets the /SM smoothness tolerance.
func (e *PDExtendedGraphicsState) SetSmoothnessTolerance(smoothness *float32) {
	e.setFloatItem(cos.SM, smoothness)
}

// AutomaticStrokeAdjustment returns the /SA stroke adjustment flag.
func (e *PDExtendedGraphicsState) AutomaticStrokeAdjustment() bool {
	return e.dict.GetBoolean(cos.SA, false)
}

// SetAutomaticStrokeAdjustment sets the /SA stroke adjustment flag.
func (e *PDExtendedGraphicsState) SetAutomaticStrokeAdjustment(sa bool) {
	e.dict.SetBoolean(cos.SA, sa)
}

// StrokingAlphaConstant returns the /CA stroking alpha, or nil.
func (e *PDExtendedGraphicsState) StrokingAlphaConstant() *float32 {
	return e.floatItem(cos.CA)
}

// SetStrokingAlphaConstant sets the /CA stroking alpha.
func (e *PDExtendedGraphicsState) SetStrokingAlphaConstant(alpha *float32) {
	e.setFloatItem(cos.CA, alpha)
}

// NonStrokingAlphaConstant returns the /ca non-stroking alpha, or nil.
func (e *PDExtendedGraphicsState) NonStrokingAlphaConstant() *float32 {
	return e.floatItem(cos.Ca)
}

// SetNonStrokingAlphaConstant sets the /ca non-stroking alpha.
func (e *PDExtendedGraphicsState) SetNonStrokingAlphaConstant(alpha *float32) {
	e.setFloatItem(cos.Ca, alpha)
}

// AlphaSourceFlag returns the /AIS alpha source flag.
func (e *PDExtendedGraphicsState) AlphaSourceFlag() bool {
	return e.dict.GetBoolean(cos.AIS, false)
}

// SetAlphaSourceFlag sets the /AIS alpha source flag.
func (e *PDExtendedGraphicsState) SetAlphaSourceFlag(alpha bool) {
	e.dict.SetBoolean(cos.AIS, alpha)
}

// BlendMode returns the /BM blend mode.
func (e *PDExtendedGraphicsState) BlendMode() *blend.BlendMode {
	return blend.GetInstance(e.dict.GetDictionaryObject(cos.BM))
}

// SetBlendMode sets the /BM blend mode.
func (e *PDExtendedGraphicsState) SetBlendMode(bm *blend.BlendMode) {
	e.dict.SetItem(cos.BM, bm.COSName())
}

// TextKnockoutFlag returns the /TK text knockout flag, which defaults to true.
func (e *PDExtendedGraphicsState) TextKnockoutFlag() bool {
	return e.dict.GetBoolean(cos.TK, true)
}

// SetTextKnockoutFlag sets the /TK text knockout flag.
func (e *PDExtendedGraphicsState) SetTextKnockoutFlag(tk bool) {
	e.dict.SetBoolean(cos.TK, tk)
}

// floatItem returns a number entry, or nil where it is missing or is not a
// number. Java declares it private.
func (e *PDExtendedGraphicsState) floatItem(key *cos.Name) *float32 {
	if value, isNumber := e.dict.GetDictionaryObject(key).(cos.Number); isNumber {
		retval := value.FloatValue()
		return &retval
	}
	return nil
}

// setFloatItem writes a number entry, and removes it where the value is nil.
// Java declares it private.
func (e *PDExtendedGraphicsState) setFloatItem(key *cos.Name, value *float32) {
	if value == nil {
		e.dict.RemoveItem(key)
		return
	}
	e.dict.SetItem(key, cos.NewFloat(*value))
}

// Transfer returns the /TR transfer function, or nil where it is an array of
// any length but four.
func (e *PDExtendedGraphicsState) Transfer() cos.Base {
	base := e.dict.GetDictionaryObject(cos.TR)
	if array, isArray := base.(*cos.Array); isArray && array.Size() != 4 {
		return nil
	}
	return base
}

// SetTransfer sets the /TR transfer function.
func (e *PDExtendedGraphicsState) SetTransfer(transfer cos.Base) {
	e.dict.SetItem(cos.TR, transfer)
}

// Transfer2 returns the /TR2 transfer function, or nil where it is an array of
// any length but four.
func (e *PDExtendedGraphicsState) Transfer2() cos.Base {
	base := e.dict.GetDictionaryObject(cos.TR2)
	if array, isArray := base.(*cos.Array); isArray && array.Size() != 4 {
		return nil
	}
	return base
}

// SetTransfer2 sets the /TR2 transfer function.
func (e *PDExtendedGraphicsState) SetTransfer2(transfer2 cos.Base) {
	e.dict.SetItem(cos.TR2, transfer2)
}

// SoftMask returns the /SMask entry, or nil where there is none.
//
// Port of getSoftMask.
func (e *PDExtendedGraphicsState) SoftMask() *PDSoftMask {
	smask := e.dict.GetDictionaryObject(cos.SMask)
	if smask == nil {
		return nil
	}
	return NewPDSoftMaskOfCache(smask, e.cache)
}

package filter

import "github.com/shinguakira/pdfbox-go/go/awt/geom"

// DecodeOptions says which part of an image a filter should decode, and how
// coarsely.
//
// Port of org.apache.pdfbox.filter.DecodeOptions. Not every filter supports
// every option, so a caller checks FilterSubsampled afterwards to see whether
// the subsampling was applied by the filter or is still its own to do.
type DecodeOptions struct {
	sourceRegion       *geom.Rectangle
	subsamplingX       int
	subsamplingY       int
	subsamplingOffsetX int
	subsamplingOffsetY int
	filterSubsampled   bool

	// final marks the shared DefaultDecodeOptions, which Java models with a
	// private subclass whose setters throw UnsupportedOperationException.
	final bool
}

// DefaultDecodeOptions is the options every caller that has none of its own
// passes. It may not be modified.
//
// Port of DecodeOptions.DEFAULT, which is a FinalDecodeOptions(true).
var DefaultDecodeOptions = &DecodeOptions{
	subsamplingX:     1,
	subsamplingY:     1,
	filterSubsampled: true,
	final:            true,
}

// NewDecodeOptions returns options that decode the whole image at full size.
func NewDecodeOptions() *DecodeOptions {
	return &DecodeOptions{subsamplingX: 1, subsamplingY: 1}
}

// NewDecodeOptionsOfRegion returns options that decode only the given region.
func NewDecodeOptionsOfRegion(sourceRegion *geom.Rectangle) *DecodeOptions {
	options := NewDecodeOptions()
	options.sourceRegion = sourceRegion
	return options
}

// NewDecodeOptionsOfBounds returns options that decode only the given region.
func NewDecodeOptionsOfBounds(x, y, width, height int) *DecodeOptions {
	return NewDecodeOptionsOfRegion(&geom.Rectangle{X: x, Y: y, Width: width, Height: height})
}

// NewDecodeOptionsOfSubsampling returns options that decode every nth row and
// column.
func NewDecodeOptionsOfSubsampling(subsampling int) *DecodeOptions {
	options := NewDecodeOptions()
	options.subsamplingX = subsampling
	options.subsamplingY = subsampling
	return options
}

// checkModifiable panics where Java's FinalDecodeOptions throws
// UnsupportedOperationException, which is unchecked.
func (o *DecodeOptions) checkModifiable() {
	if o.final {
		panic("This instance may not be modified.")
	}
}

// SourceRegion returns the region to decode, or nil for the whole image.
func (o *DecodeOptions) SourceRegion() *geom.Rectangle { return o.sourceRegion }

// SetSourceRegion sets the region to decode.
func (o *DecodeOptions) SetSourceRegion(sourceRegion *geom.Rectangle) {
	o.checkModifiable()
	o.sourceRegion = sourceRegion
}

// SubsamplingX returns the horizontal subsampling.
func (o *DecodeOptions) SubsamplingX() int { return o.subsamplingX }

// SetSubsamplingX sets the horizontal subsampling.
func (o *DecodeOptions) SetSubsamplingX(ssX int) {
	o.checkModifiable()
	o.subsamplingX = ssX
}

// SubsamplingY returns the vertical subsampling.
func (o *DecodeOptions) SubsamplingY() int { return o.subsamplingY }

// SetSubsamplingY sets the vertical subsampling.
func (o *DecodeOptions) SetSubsamplingY(ssY int) {
	o.checkModifiable()
	o.subsamplingY = ssY
}

// SubsamplingOffsetX returns the horizontal subsampling offset.
func (o *DecodeOptions) SubsamplingOffsetX() int { return o.subsamplingOffsetX }

// SetSubsamplingOffsetX sets the horizontal subsampling offset.
func (o *DecodeOptions) SetSubsamplingOffsetX(ssOffsetX int) {
	o.checkModifiable()
	o.subsamplingOffsetX = ssOffsetX
}

// SubsamplingOffsetY returns the vertical subsampling offset.
func (o *DecodeOptions) SubsamplingOffsetY() int { return o.subsamplingOffsetY }

// SetSubsamplingOffsetY sets the vertical subsampling offset.
func (o *DecodeOptions) SetSubsamplingOffsetY(ssOffsetY int) {
	o.checkModifiable()
	o.subsamplingOffsetY = ssOffsetY
}

// IsFilterSubsampled reports whether the filter applied the subsampling itself.
func (o *DecodeOptions) IsFilterSubsampled() bool { return o.filterSubsampled }

// setFilterSubsampled records that the filter applied the subsampling itself.
//
// Java's FinalDecodeOptions overrides this one to ignore the call rather than
// to throw, so that the shared DEFAULT keeps its true.
func (o *DecodeOptions) setFilterSubsampled(filterSubsampled bool) {
	if o.final {
		// Silently ignore the request.
		return
	}
	o.filterSubsampled = filterSubsampled
}

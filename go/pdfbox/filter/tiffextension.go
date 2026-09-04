package filter

// The TIFF constants the CCITT filter reads its options with.
//
// Port of org.apache.pdfbox.filter.TIFFExtension, an interface of constants;
// only the ones the CCITT code uses are here, and the file says which of the
// others exist so that a later slice can tell an omission from an oversight.
//
// Not carried, because nothing in the port reads them: the other compression
// kinds (LZW, old JPEG, JPEG, Deflate, ZLib), the photometric interpretations,
// the planar configuration, the predictors, the sample formats, the YCbCr
// positioning, the JPEG process kinds, the ink sets and the orientations.
const (
	compressionCCITTModifiedHuffmanRLE = 2
	compressionCCITTT4                 = 3
	compressionCCITTT6                 = 4

	group3Opt2DEncoding   = 1
	group3OptUncompressed = 2
	group3OptFillBits     = 4
	group4OptUncompressed = 2

	fillLeftToRight = 1 // Default
)

package image

import (
	"bytes"
	"log/slog"
	"math"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/filter"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
)

// The converter that turns a PNG into an image XObject without recoding it.
//
// Port of org.apache.pdfbox.pdmodel.graphics.image.PNGConverter. A PNG and a
// PDF image compress the same way -- deflate over PNG predicted rows -- so the
// IDAT chunks can go into the stream as they are, with a /DecodeParms that says
// so. Where anything about the PNG does not fit, every method here returns
// nothing rather than an error, and PDImageXObject.createFromByteArray falls
// back to LosslessFactory; the port keeps that, because it is how PDFBox
// handles a PNG it cannot shortcut.

// The chunk types a PNG carries, as the four bytes of their name.
const (
	chunkIHDR = 0x49484452 // IHDR: 73 72 68 82
	chunkIDAT = 0x49444154 // IDAT: 73 68 65 84
	chunkPLTE = 0x504C5445 // PLTE: 80 76 84 69
	chunkIEND = 0x49454E44 // IEND: 73 69 78 68
	chunkTRNS = 0x74524E53 // tRNS: 116 82 78 83
	chunkCHRM = 0x6348524D // cHRM: 99 72 82 77
	chunkGAMA = 0x67414D41 // gAMA: 103 65 77 65
	chunkICCP = 0x69434350 // iCCP: 105 67 67 80
	chunkSBIT = 0x73424954 // sBIT: 115 66 73 84
	chunkSRGB = 0x73524742 // sRGB: 115 82 71 66
	chunkTEXT = 0x74455874 // tEXt: 116 69 88 116
	chunkZTXT = 0x7A545874 // zTXt: 122 84 88 116
	chunkITXT = 0x69545874 // iTXt: 105 84 88 116
	chunkKBKG = 0x6B424B47 // kBKG: 107 66 75 71
	chunkHIST = 0x68495354 // hIST: 104 73 83 84
	chunkPHYS = 0x70485973 // pHYs: 112 72 89 115
	chunkSPLT = 0x73504C54 // sPLT: 115 80 76 84
	chunkTIME = 0x74494D45 // tIME: 116 73 77 69
)

// ConvertPNGImage turns a PNG into an image XObject, or returns nothing where
// it cannot.
//
// Port of the package-private PNGConverter.convertPNGImage.
func ConvertPNGImage(doc DocumentLike, imageData []byte) (*PDImageXObject, error) {
	state := parsePNGChunks(imageData)
	if !checkConverterState(state) {
		// There is something wrong, we can't convert this PNG
		return nil, nil
	}
	return convertPng(doc, state)
}

func convertPng(doc DocumentLike, state *pngConverterState) (*PDImageXObject, error) {
	ihdr := state.ihdr
	ihdrStart := ihdr.start
	width := readIntAt(ihdr.bytes, ihdrStart)
	height := readIntAt(ihdr.bytes, ihdrStart+4)
	bitDepth := int(ihdr.bytes[ihdrStart+8])
	colorType := int(ihdr.bytes[ihdrStart+9])
	compressionMethod := int(ihdr.bytes[ihdrStart+10])
	filterMethod := int(ihdr.bytes[ihdrStart+11])
	interlaceMethod := int(ihdr.bytes[ihdrStart+12])

	if bitDepth != 1 && bitDepth != 2 && bitDepth != 4 && bitDepth != 8 && bitDepth != 16 {
		slog.Error("image: invalid bit depth", "depth", bitDepth)
		return nil, nil
	}
	if width <= 0 || height <= 0 {
		slog.Error("image: invalid image size", "width", width, "height", height)
		return nil, nil
	}
	if compressionMethod != 0 {
		slog.Error("image: unknown PNG compression method", "method", compressionMethod)
		return nil, nil
	}
	if filterMethod != 0 {
		// Java logs the compression method here rather than the filter one,
		// which is a slip in the message and not in the check.
		slog.Error("image: unknown PNG filtering method", "method", compressionMethod)
		return nil, nil
	}
	if interlaceMethod != 0 {
		slog.Debug("image: can't handle interlace method", "method", interlaceMethod)
		return nil, nil
	}

	state.width = width
	state.height = height
	state.bitsPerComponent = bitDepth

	switch colorType {
	case 0:
		// Grayscale
		slog.Debug("image: can't handle grayscale yet.")
		return nil, nil
	case 2:
		// Truecolor
		if state.tRNS != nil {
			slog.Debug("image: can't handle images with transparent colors.")
			return nil, nil
		}
		return buildImageObject(doc, state)
	case 3:
		// Indexed image
		return buildIndexImage(doc, state)
	case 4:
		// Grayscale with alpha.
		slog.Debug("image: can't handle grayscale with alpha, " +
			"would need to separate alpha from image data")
		return nil, nil
	case 6:
		// Truecolor with alpha.
		slog.Debug("image: can't handle truecolor with alpha, " +
			"would need to separate alpha from image data")
		return nil, nil
	default:
		slog.Error("image: unknown PNG color type", "type", colorType)
		return nil, nil
	}
}

func buildIndexImage(doc DocumentLike, state *pngConverterState) (*PDImageXObject, error) {
	plte := state.plte
	if plte == nil {
		slog.Error("image: indexed image without PLTE chunk.")
		return nil, nil
	}
	if plte.length%3 != 0 {
		slog.Error("image: PLTE table corrupted, last (r,g,b) tuple is not complete.")
		return nil, nil
	}
	if state.bitsPerComponent > 8 {
		slog.Debug("image: can only convert indexed images with bit depth <= 8",
			"depth", state.bitsPerComponent)
		return nil, nil
	}

	img, err := buildImageObject(doc, state)
	if err != nil || img == nil {
		return img, err
	}

	highVal := (plte.length / 3) - 1
	if highVal > 255 {
		slog.Error("image: too many colors in PLTE, only 256 allowed", "found", highVal+1)
		return nil, nil
	}
	if err := setupIndexedColorSpace(doc, plte, img, highVal); err != nil {
		return nil, err
	}

	if state.tRNS != nil {
		mask, err := buildTransparencyMaskFromIndexedData(doc, img, state)
		if err != nil {
			return nil, err
		}
		img.COSDictionary().SetItem(cos.SMask, mask.COSObject())
	}
	return img, nil
}

func buildTransparencyMaskFromIndexedData(doc DocumentLike, img *PDImageXObject,
	state *pngConverterState) (*PDImageXObject, error) {
	var decoded bytes.Buffer
	decodeParams := buildDecodeParams(state, color.DeviceGray)
	imageDict := cos.NewDictionary()
	imageDict.SetItem(cos.Filter, cos.FlateDecode)
	imageDict.SetItem(cos.DecodeParms, decodeParams)
	if _, err := (filter.Flate{}).Decode(&decoded, idatReader(state), imageDict, 0); err != nil {
		return nil, err
	}

	length := img.Width() * img.Height()
	out := make([]byte, length)
	transparencyTable := state.tRNS.data()

	iis := newImageBitReader(bytes.NewReader(decoded.Bytes()))
	bitsPerComponent := state.bitsPerComponent
	w := 0
	neededBits := bitsPerComponent * state.width
	bitPadding := neededBits % 8
	for i := 0; i < len(out); i++ {
		idx := int(iis.readBits(bitsPerComponent))
		if idx < len(transparencyTable) {
			// Inside the table, use the transparency value
			out[i] = transparencyTable[idx]
		} else {
			// Outside the table -> transparent value is 0xFF here.
			out[i] = 0xFF
		}
		w++
		if w == state.width {
			w = 0
			iis.readBits(bitPadding)
		}
	}

	return prepareImageXObject(doc, out, img.Width(), img.Height(), 8, color.DeviceGray)
}

func setupIndexedColorSpace(doc DocumentLike, lookupTable *pngChunk, img *PDImageXObject,
	highVal int) error {
	baseColorSpace, err := img.ColorSpace()
	if err != nil {
		return err
	}

	indexedArray := cos.NewArray()
	indexedArray.Add(cos.Indexed)
	indexedArray.Add(baseColorSpace.COSObject())

	// Java casts the /DecodeParms entry without a check; buildImageObject put a
	// dictionary there, so it is one.
	img.COSDictionary().GetCOSDictionary(cos.DecodeParms).SetItem(cos.Colors, cos.IntegerOne)
	indexedArray.Add(cos.GetInteger(int64(highVal)))

	colorTable := common.NewPDStream(doc.CreateStream())
	colorTableStream, err := colorTable.Stream().CreateWriterWithFilters(cos.FlateDecode)
	if err != nil {
		return err
	}
	if _, err := colorTableStream.Write(
		lookupTable.bytes[lookupTable.start : lookupTable.start+lookupTable.length]); err != nil {
		colorTableStream.Close()
		return err
	}
	if err := colorTableStream.Close(); err != nil {
		return err
	}
	indexedArray.Add(colorTable.Stream())

	indexed, err := color.NewPDIndexedOfArray(indexedArray, nil)
	if err != nil {
		return err
	}
	img.SetColorSpace(indexed)
	return nil
}

func buildImageObject(document DocumentLike, state *pngConverterState) (*PDImageXObject, error) {
	encoded, err := readAllIDAT(state)
	if err != nil {
		return nil, err
	}
	colorSpace := color.PDColorSpace(color.DeviceRGB)
	imageXObject, err := newPDImageXObjectOfEncoded(document, encoded, cos.FlateDecode,
		state.width, state.height, state.bitsPerComponent, colorSpace)
	if err != nil {
		return nil, err
	}

	decodeParams := buildDecodeParams(state, colorSpace)
	imageXObject.COSDictionary().SetItem(cos.DecodeParms, decodeParams)

	// We ignore gAMA and cHRM chunks if we have a ICC profile, as the ICC
	// profile takes preference
	hasICCColorProfile := state.sRGB != nil || state.iCCP != nil

	if state.gAMA != nil && !hasICCColorProfile {
		if state.gAMA.length != 4 {
			slog.Error("image: invalid gAMA chunk length", "length", state.gAMA.length)
			return nil, nil
		}
		gamma := readPNGFloat(state.gAMA.bytes, state.gAMA.start)
		// If the gamma is 2.2 for sRGB everything is fine. Otherwise bail out.
		// The gamma is stored as 1 / gamma.
		if math.Abs(float64(gamma)-(1/2.2)) > 0.00001 {
			slog.Debug("image: we can't handle this gamma yet", "gamma", gamma)
			return nil, nil
		}
	}

	if state.sRGB != nil {
		if state.sRGB.length != 1 {
			slog.Error("image: sRGB chunk has an invalid length", "length", state.sRGB.length)
			return nil, nil
		}
		// Store the specified rendering intent
		renderIntent := int(int8(state.sRGB.bytes[state.sRGB.start]))
		if value := MapPNGRenderIntent(renderIntent); value != nil {
			imageXObject.COSDictionary().SetItem(cos.Intent, value)
		}
	}

	if state.cHRM != nil && !hasICCColorProfile {
		if state.cHRM.length != 32 {
			slog.Error("image: invalid cHRM chunk length", "length", state.cHRM.length)
			return nil, nil
		}
		slog.Debug("image: we can not handle cHRM chunks yet.")
		return nil, nil
	}

	// If possible we prefer a ICCBased color profile, just because its way
	// faster to decode ...
	//
	// The port attaches one only where the PNG carries it. Java's other arm
	// tags the image with the JRE's own sRGB profile, which Go has no copy of
	// -- see the note on PDDeviceCMYK -- so an sRGB chunk leaves the colour
	// space DeviceRGB, which is what that profile says anyway.
	if state.iCCP != nil {
		cosStream, err := createCOSStreamWithICCProfile(document, colorSpace, state)
		if err != nil {
			return nil, err
		}
		if cosStream == nil {
			return nil, nil
		}
		array := cos.NewArray()
		array.Add(cos.ICCBased)
		array.Add(cosStream)
		profile, err := color.NewPDICCBased(array, nil)
		if err != nil {
			return nil, err
		}
		imageXObject.SetColorSpace(profile)
	}
	return imageXObject, nil
}

func createCOSStreamWithICCProfile(document DocumentLike, colorSpace color.PDColorSpace,
	state *pngConverterState) (*cos.Stream, error) {
	numberOfComponents := colorSpace.NumberOfComponents()
	cosStream := document.CreateStream()
	cosStream.SetInt(cos.N, numberOfComponents)
	if numberOfComponents == 1 {
		cosStream.SetItem(cos.Alternate, cos.DeviceGray)
	} else {
		cosStream.SetItem(cos.Alternate, cos.DeviceRGB)
	}
	cosStream.SetItem(cos.Filter, cos.FlateDecode)

	// We need to skip over the name
	iccProfileDataStart := 0
	for iccProfileDataStart < 80 && iccProfileDataStart < state.iCCP.length {
		if state.iCCP.bytes[state.iCCP.start+iccProfileDataStart] == 0 {
			break
		}
		iccProfileDataStart++
	}
	iccProfileDataStart++
	if iccProfileDataStart >= state.iCCP.length {
		slog.Error("image: invalid iCCP chunk, too few bytes")
		return nil, nil
	}
	compressionMethod := state.iCCP.bytes[state.iCCP.start+iccProfileDataStart]
	if compressionMethod != 0 {
		slog.Error("image: iCCP chunk has an invalid compression method",
			"method", compressionMethod)
		return nil, nil
	}
	// Skip over the compression method
	iccProfileDataStart++

	rawOutputStream, err := cosStream.CreateRawWriter()
	if err != nil {
		return nil, err
	}
	if _, err := rawOutputStream.Write(state.iCCP.bytes[state.iCCP.start+iccProfileDataStart : state.iCCP.start+state.iCCP.length]); err != nil {
		rawOutputStream.Close()
		return nil, err
	}
	if err := rawOutputStream.Close(); err != nil {
		return nil, err
	}
	return cosStream, nil
}

func buildDecodeParams(state *pngConverterState, colorSpace color.PDColorSpace) *cos.Dictionary {
	decodeParms := cos.NewDictionary()
	decodeParms.SetItem(cos.BitsPerComponent, cos.GetInteger(int64(state.bitsPerComponent)))
	decodeParms.SetItem(cos.Predictor, cos.GetInteger(15))
	decodeParms.SetItem(cos.Columns, cos.GetInteger(int64(state.width)))
	decodeParms.SetItem(cos.Colors, cos.GetInteger(int64(colorSpace.NumberOfComponents())))
	return decodeParms
}

// idatReader reads the IDAT chunks one after the other.
//
// Port of getIDATInputStream together with the private MultipleInputStream,
// which is io.MultiReader here.
func idatReader(state *pngConverterState) *bytes.Reader {
	data, _ := readAllIDAT(state)
	return bytes.NewReader(data)
}

func readAllIDAT(state *pngConverterState) ([]byte, error) {
	var out bytes.Buffer
	for _, idat := range state.idats {
		out.Write(idat.bytes[idat.start : idat.start+idat.length])
	}
	return out.Bytes(), nil
}

// MapPNGRenderIntent turns a PNG rendering intent into the PDF name for it, or
// nil where there is none.
//
// Port of the package-private PNGConverter.mapPNGRenderIntent.
func MapPNGRenderIntent(renderIntent int) *cos.Name {
	switch renderIntent {
	case 0:
		return cos.Perceptual
	case 1:
		return cos.RelativeColorimetric
	case 2:
		return cos.Saturation
	case 3:
		return cos.AbsoluteColorimetric
	default:
		return nil
	}
}

func checkConverterState(state *pngConverterState) bool {
	if state == nil {
		return false
	}
	if state.ihdr == nil || !checkChunkSane(state.ihdr) {
		slog.Error("image: invalid IHDR chunk.")
		return false
	}
	if !checkChunkSane(state.plte) {
		slog.Error("image: invalid PLTE chunk.")
		return false
	}
	if !checkChunkSane(state.iCCP) {
		slog.Error("image: invalid iCCP chunk.")
		return false
	}
	if !checkChunkSane(state.tRNS) {
		slog.Error("image: invalid tRNS chunk.")
		return false
	}
	if !checkChunkSane(state.sRGB) {
		slog.Error("image: invalid sRGB chunk.")
		return false
	}
	if !checkChunkSane(state.cHRM) {
		slog.Error("image: invalid cHRM chunk.")
		return false
	}
	if !checkChunkSane(state.gAMA) {
		slog.Error("image: invalid gAMA chunk.")
		return false
	}

	// Check the IDATs
	if len(state.idats) == 0 {
		slog.Error("image: no IDAT chunks.")
		return false
	}
	for _, idat := range state.idats {
		if !checkChunkSane(idat) {
			slog.Error("image: invalid IDAT chunk.")
			return false
		}
	}
	return true
}

func checkChunkSane(chunk *pngChunk) bool {
	if chunk == nil {
		// If the chunk does not exist, it can not be wrong...
		return true
	}
	if chunk.start+chunk.length > len(chunk.bytes) {
		return false
	}
	if chunk.start < 4 {
		return false
	}
	// We must include the chunk type in the CRC calculation
	ourCRC := pngCRC(chunk.bytes, chunk.start-4, chunk.length+4)
	if ourCRC != chunk.crc {
		slog.Error("image: invalid CRC on chunk",
			"crc", ourCRC, "chunkType", chunk.chunkType, "expected", chunk.crc)
		return false
	}
	return true
}

// pngChunk is one chunk of a PNG.
//
// Port of the package-private PNGConverter.Chunk.
type pngChunk struct {
	bytes     []byte
	chunkType int32
	crc       int32
	start     int
	length    int
}

func (c *pngChunk) data() []byte {
	return append([]byte(nil), c.bytes[c.start:c.start+c.length]...)
}

// pngConverterState is what the chunk walk collected.
//
// Port of the package-private PNGConverter.PNGConverterState.
type pngConverterState struct {
	idats []*pngChunk
	ihdr  *pngChunk
	plte  *pngChunk
	iCCP  *pngChunk
	tRNS  *pngChunk
	sRGB  *pngChunk
	gAMA  *pngChunk
	cHRM  *pngChunk

	// Parsed header fields
	width            int
	height           int
	bitsPerComponent int
}

// readIntAt reads a big endian 32 bit value.
//
// Java combines four masked bytes in an int, so the result is signed; the port
// keeps int32 for the chunk type and the CRC, which are compared, and widens to
// int for the lengths and sizes, which are used as counts.
func readIntAt(data []byte, offset int) int {
	return int(readInt32At(data, offset))
}

func readInt32At(data []byte, offset int) int32 {
	b1 := int32(data[offset]) << 24
	b2 := int32(data[offset+1]) << 16
	b3 := int32(data[offset+2]) << 8
	b4 := int32(data[offset+3])
	return b1 | b2 | b3 | b4
}

func readPNGFloat(b []byte, offset int) float32 {
	v := readIntAt(b, offset)
	return float32(v) / 100000
}

func parsePNGChunks(imageData []byte) *pngConverterState {
	if len(imageData) < 20 {
		slog.Error("image: byte array far too small", "length", len(imageData))
		return nil
	}

	state := &pngConverterState{}
	ptr := 8
	firstChunkType := readInt32At(imageData, ptr+4)
	if firstChunkType != chunkIHDR {
		slog.Error("image: first chunk type was not IHDR", "type", firstChunkType)
		return nil
	}

	for ptr+12 <= len(imageData) {
		chunkLength := readIntAt(imageData, ptr)
		chunkType := readInt32At(imageData, ptr+4)
		ptr += 8
		if chunkLength < 0 || ptr+chunkLength+4 > len(imageData) {
			slog.Error("image: not enough bytes for a PNG chunk",
				"offset", ptr, "expected", chunkLength, "length", len(imageData))
			return nil
		}
		chunk := &pngChunk{
			chunkType: chunkType,
			bytes:     imageData,
			start:     ptr,
			length:    chunkLength,
		}
		switch chunkType {
		case chunkIHDR:
			if state.ihdr != nil {
				slog.Error("image: two IHDR chunks? There is something wrong.")
				return nil
			}
			state.ihdr = chunk
		case chunkIDAT:
			// The image data itself
			state.idats = append(state.idats, chunk)
		case chunkPLTE:
			// For indexed images the palette table
			if state.plte != nil {
				slog.Error("image: two PLTE chunks? There is something wrong.")
				return nil
			}
			state.plte = chunk
		case chunkIEND:
			// We are done, return the state
			return state
		case chunkTRNS:
			// For indexed images the alpha transparency table
			if state.tRNS != nil {
				slog.Error("image: two tRNS chunks? There is something wrong.")
				return nil
			}
			state.tRNS = chunk
		case chunkGAMA:
			// Gama
			state.gAMA = chunk
		case chunkCHRM:
			// Chroma
			state.cHRM = chunk
		case chunkICCP:
			// ICC Profile
			state.iCCP = chunk
		case chunkSBIT:
			slog.Debug("image: can't convert PNGs with sBIT chunk.")
		case chunkSRGB:
			// We use the rendering intent from the chunk
			state.sRGB = chunk
		case chunkTEXT, chunkZTXT, chunkITXT:
			// We don't care about this text infos / metadata
		case chunkKBKG:
			// As we can handle transparency we don't need the background colour
		case chunkHIST:
			// We don't need the color histogram
		case chunkPHYS:
			// The PDImageXObject will be placed by the user however he wants,
			// so we can not enforce the physical dpi information stored here.
			// We just ignore it.
		case chunkSPLT:
			// This palette stuff seems editor related, we don't need it.
		case chunkTIME:
			// We don't need the last image change time either
		default:
			slog.Debug("image: unknown PNG chunk type, skipping", "type", chunkType)
		}

		ptr += chunkLength
		// Read the CRC
		chunk.crc = readInt32At(imageData, ptr)
		ptr += 4
	}
	slog.Error("image: no IEND chunk found.")
	return nil
}

// pngCRCTable is the table Java builds in its static initialiser.
var pngCRCTable = func() [256]uint32 {
	var table [256]uint32
	for n := 0; n < 256; n++ {
		c := uint32(n)
		for k := 0; k < 8; k++ {
			if c&1 != 0 {
				c = 0xEDB88320 ^ (c >> 1)
			} else {
				c = c >> 1
			}
		}
		table[n] = c
	}
	return table
}()

func updatePNGCRC(buf []byte, offset, length int) uint32 {
	c := uint32(0xFFFFFFFF)
	end := offset + length
	for n := offset; n < end; n++ {
		// Java writes buf[n], a signed byte, and masks the XOR with 0xff; the
		// mask makes the sign extension irrelevant, so the port XORs the byte.
		c = pngCRCTable[(c^uint32(buf[n]))&0xff] ^ (c >> 8)
	}
	return c
}

// pngCRC is the CRC-32 a PNG chunk carries.
//
// Port of the package-private PNGConverter.crc.
func pngCRC(buf []byte, offset, length int) int32 {
	return int32(^updatePNGCRC(buf, offset, length))
}

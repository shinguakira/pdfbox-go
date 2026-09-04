package function

import (
	"errors"
	"log/slog"
	"math"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// float32Bits is java.lang.Float.floatToIntBits, which floatCompare needs.
func float32Bits(f float32) int32 { return int32(math.Float32bits(f)) }

// PDFunctionType0 is a sampled function: a table of samples, interpolated
// between.
//
// Port of org.apache.pdfbox.pdmodel.common.function.PDFunctionType0.
type PDFunctionType0 struct {
	pdFunctionBase

	encode  *cos.Array
	decode  *cos.Array
	size    *cos.Array
	samples [][]int
}

var _ PDFunction = (*PDFunctionType0)(nil)

// NewPDFunctionType0 creates a sampled function from its stream.
func NewPDFunctionType0(function cos.Base) *PDFunctionType0 {
	return &PDFunctionType0{pdFunctionBase: newPDFunctionBase(function)}
}

// FunctionType returns 0.
func (f *PDFunctionType0) FunctionType() int { return 0 }

// Size returns the /Size array, the number of samples along each input.
func (f *PDFunctionType0) Size() *cos.Array {
	if f.size == nil {
		f.size = f.COSDictionary().GetCOSArray(cos.Size)
	}
	return f.size
}

// BitsPerSample returns how many bits one sample takes.
func (f *PDFunctionType0) BitsPerSample() int {
	return f.COSDictionary().GetInt(cos.BitsPerSample)
}

// Order returns the interpolation order, 1 for linear.
func (f *PDFunctionType0) Order() int {
	return f.COSDictionary().GetIntDefault(cos.Order, 1)
}

// SetBitsPerSample sets how many bits one sample takes.
func (f *PDFunctionType0) SetBitsPerSample(bps int) {
	f.COSDictionary().SetInt(cos.BitsPerSample, bps)
}

func (f *PDFunctionType0) encodeValues() *cos.Array {
	if f.encode == nil {
		f.encode = f.COSDictionary().GetCOSArray(cos.Encode)
		// the default value is [0 (size[0]-1) 0 (size[1]-1) ...]
		if f.encode == nil {
			f.encode = cos.NewArray()
			sizeValues := f.Size()
			sizeValuesSize := sizeValues.Size()
			for i := 0; i < sizeValuesSize; i++ {
				f.encode.Add(cos.IntegerZero)
				f.encode.Add(cos.GetInteger(int64(sizeValues.GetInt(i)) - 1))
			}
		}
	}
	return f.encode
}

func (f *PDFunctionType0) decodeValues() *cos.Array {
	if f.decode == nil {
		f.decode = f.COSDictionary().GetCOSArray(cos.Decode)
		// if decode is null, the default values are the range values
		if f.decode == nil {
			f.decode = f.rangeValues()
		}
	}
	return f.decode
}

// EncodeForParameter returns the /Encode range of input paramNum, or nil where
// the array is too short.
func (f *PDFunctionType0) EncodeForParameter(paramNum int) *common.PDRange {
	encodeValues := f.encodeValues()
	if encodeValues != nil && encodeValues.Size() >= paramNum*2+1 {
		return common.NewPDRangeOfIndex(encodeValues, paramNum)
	}
	return nil
}

// SetEncodeValues sets the /Encode array.
func (f *PDFunctionType0) SetEncodeValues(encodeValues *cos.Array) {
	f.encode = encodeValues
	f.COSDictionary().SetItem(cos.Encode, encodeValues)
}

// DecodeForParameter returns the /Decode range of output paramNum, or nil where
// the array is too short.
func (f *PDFunctionType0) DecodeForParameter(paramNum int) *common.PDRange {
	decodeValues := f.decodeValues()
	if decodeValues != nil && decodeValues.Size() >= paramNum*2+1 {
		return common.NewPDRangeOfIndex(decodeValues, paramNum)
	}
	return nil
}

// SetDecodeValues sets the /Decode array.
func (f *PDFunctionType0) SetDecodeValues(decodeValues *cos.Array) {
	f.decode = decodeValues
	f.COSDictionary().SetItem(cos.Decode, decodeValues)
}

// rinterpol is the multilinear interpolation over the sample table.
//
// Port of the inner class PDFunctionType0.Rinterpol.
type rinterpol struct {
	f *PDFunctionType0
	// coordinate that is to be interpolated
	in []float32
	// coordinate of the "ceil" point
	inPrev []int
	// coordinate of the "floor" point
	inNext               []int
	numberOfInputValues  int
	numberOfOutputValues int
}

func newRinterpol(f *PDFunctionType0, input []float32, inputPrev, inputNext []int) *rinterpol {
	return &rinterpol{
		f:                    f,
		in:                   input,
		inPrev:               inputPrev,
		inNext:               inputNext,
		numberOfInputValues:  len(input),
		numberOfOutputValues: f.NumberOfOutputParameters(),
	}
}

func (r *rinterpol) rinterpolate() []float32 {
	return r.rinterpol(make([]int, r.numberOfInputValues), 0)
}

func (r *rinterpol) rinterpol(coord []int, step int) []float32 {
	resultSample := make([]float32, r.numberOfOutputValues)
	if step == len(r.in)-1 {
		// leaf
		if r.inPrev[step] == r.inNext[step] {
			coord[step] = r.inPrev[step]
			tmpSample := r.f.getSamples()[r.calcSampleIndex(coord)]
			for i := 0; i < r.numberOfOutputValues; i++ {
				resultSample[i] = float32(tmpSample[i])
			}
			return resultSample
		}
		coord[step] = r.inPrev[step]
		sample1 := r.f.getSamples()[r.calcSampleIndex(coord)]
		coord[step] = r.inNext[step]
		sample2 := r.f.getSamples()[r.calcSampleIndex(coord)]
		for i := 0; i < r.numberOfOutputValues; i++ {
			resultSample[i] = interpolate(r.in[step], float32(r.inPrev[step]),
				float32(r.inNext[step]), float32(sample1[i]), float32(sample2[i]))
		}
		return resultSample
	}
	// branch
	if r.inPrev[step] == r.inNext[step] {
		coord[step] = r.inPrev[step]
		return r.rinterpol(coord, step+1)
	}
	coord[step] = r.inPrev[step]
	sample1 := r.rinterpol(coord, step+1)
	coord[step] = r.inNext[step]
	sample2 := r.rinterpol(coord, step+1)
	for i := 0; i < r.numberOfOutputValues; i++ {
		resultSample[i] = interpolate(r.in[step], float32(r.inPrev[step]),
			float32(r.inNext[step]), sample1[i], sample2[i])
	}
	return resultSample
}

func (r *rinterpol) calcSampleIndex(vector []int) int {
	// inspiration: http://stackoverflow.com/a/12113479/535646
	// but used in reverse
	sizeValues := r.f.Size().ToFloatArray()
	index := 0
	sizeProduct := 1
	dimension := len(vector)
	for i := dimension - 2; i >= 0; i-- {
		sizeProduct *= int(sizeValues[i])
	}
	for i := dimension - 1; i >= 0; i-- {
		index += sizeProduct * vector[i]
		if i-1 >= 0 {
			sizeProduct /= int(sizeValues[i-1])
		}
	}
	return index
}

// getSamples reads the sample table out of the stream, once.
//
// Java holds it on the outer class and the inner class fills it, so the port
// puts it on the function too.
func (f *PDFunctionType0) getSamples() [][]int {
	if f.samples != nil {
		return f.samples
	}
	arraySize := 1
	nIn := f.NumberOfInputParameters()
	nOut := f.NumberOfOutputParameters()
	sizes := f.Size()
	for i := 0; i < nIn; i++ {
		arraySize *= sizes.GetInt(i)
	}
	f.samples = make([][]int, arraySize)
	for i := range f.samples {
		f.samples[i] = make([]int, nOut)
	}
	bitsPerSample := f.BitsPerSample()
	index := 0
	// Java logs an IOException here and returns the samples it read; the port
	// does the same, so a truncated stream leaves the rest of the table zero.
	if err := func() error {
		is, err := f.PDStream().CreateInputStream()
		if err != nil {
			return err
		}
		// PDF spec 1.7 p.171:
		// Each sample value is represented as a sequence of BitsPerSample bits.
		// Successive values are adjacent in the bit stream; there is no padding
		// at byte boundaries.
		mciis := newSampleBitReader(is)
		for i := 0; i < arraySize; i++ {
			for k := 0; k < nOut; k++ {
				// TODO will this cast work properly for 32 bitsPerSample or
				// should we use long[]?
				bits, err := mciis.readBits(bitsPerSample)
				if err != nil {
					return err
				}
				f.samples[index][k] = int(int32(bits))
			}
			index++
		}
		return nil
	}(); err != nil {
		slog.Error("function: IOException while reading the sample values of this function.",
			"err", err)
	}
	return f.samples
}

// Eval interpolates between the samples.
func (f *PDFunctionType0) Eval(input []float32) ([]float32, error) {
	// This involves linear interpolation based on a set of sample points.
	// Theoretically it's not that difficult ... see section 3.9.1 of the PDF
	// Reference.

	sizeValues := f.Size().ToFloatArray()
	bitsPerSample := f.BitsPerSample()
	maxSample := float32(math.Pow(2, float64(bitsPerSample)) - 1.0)
	numberOfInputValues := len(input)
	numberOfOutputValues := f.NumberOfOutputParameters()

	inputPrev := make([]int, numberOfInputValues)
	inputNext := make([]int, numberOfInputValues)
	input = append([]float32(nil), input...) // PDFBOX-4461

	for i := 0; i < numberOfInputValues; i++ {
		domain := f.DomainForInput(i)
		encodeValues := f.EncodeForParameter(i)
		min := domain.Min()
		max := domain.Max()
		input[i] = clipToRange(input[i], min, max)
		input[i] = interpolate(input[i], min, max, encodeValues.Min(), encodeValues.Max())
		input[i] = clipToRange(input[i], 0, sizeValues[i]-1)
		inputPrev[i] = int(math.Floor(float64(input[i])))
		inputNext[i] = int(math.Ceil(float64(input[i])))
	}

	outputValues := newRinterpol(f, input, inputPrev, inputNext).rinterpolate()

	for i := 0; i < numberOfOutputValues; i++ {
		rangeForOutput := f.RangeForOutput(i)
		decodeValues := f.DecodeForParameter(i)
		if decodeValues == nil {
			return nil, errors.New("Range missing in function /Decode entry")
		}
		outputValues[i] = interpolate(outputValues[i], 0, maxSample,
			decodeValues.Min(), decodeValues.Max())
		outputValues[i] = clipToRange(outputValues[i], rangeForOutput.Min(), rangeForOutput.Max())
	}

	return outputValues, nil
}

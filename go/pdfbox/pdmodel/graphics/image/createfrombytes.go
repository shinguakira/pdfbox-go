package image

import (
	"bytes"
	"fmt"
	"image/gif"
	"image/png"
	"os"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/util/filetypedetector"
)

// The createFrom* factory methods of PDImageXObject, which pick a factory by
// the kind of image they are handed.
//
// Port of the static createFromFile, createFromFileByExtension,
// createFromFileByContent and createFromByteArray of
// org.apache.pdfbox.pdmodel.graphics.image.PDImageXObject. They are in a file
// of their own because they name every factory in the package, where the rest
// of PDImageXObject names none.

// CustomFactory builds an image XObject from bytes.
//
// Port of the functional interface
// org.apache.pdfbox.pdmodel.graphics.image.CustomFactory.
type CustomFactory func(document DocumentLike, byteArray []byte) (*PDImageXObject, error)

// CreateFromFileByExtension creates an image XObject from a file, choosing the
// factory by the file's extension.
//
// Port of createFromFileByExtension. Java throws IllegalArgumentException for
// an unsupported name, which is unchecked, so the port panics.
func CreateFromFileByExtension(document DocumentLike, path string) (*PDImageXObject, error) {
	name := path
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	dot := strings.LastIndex(name, ".")
	if dot == -1 {
		panic(fmt.Sprintf("Image type not supported: %s", name))
	}
	ext := strings.ToLower(name[dot+1:])

	if ext == "jpg" || ext == "jpeg" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		return CreateFromJPEGByteArray(document, data)
	}
	if ext == "tif" || ext == "tiff" {
		img, err := CreateCCITTFromFile(document, path)
		if err == nil && img != nil {
			return img, nil
		}
		// Reading as TIFF failed, setting fileType to PNG.
		// Plan B: try reading with the image decoders
		// common exception:
		// First image in tiff is not CCITT T4 or T6 compressed
		ext = "png"
	}
	if ext == "gif" || ext == "bmp" || ext == "png" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		return createLosslessFromBytes(document, data, name)
	}
	panic(fmt.Sprintf("Image type not supported: %s", name))
}

// CreateFromFile creates an image XObject from a file at the given path.
//
// Port of createFromFile(String, PDDocument), which is
// createFromFileByExtension of the same file.
func CreateFromFile(document DocumentLike, imagePath string) (*PDImageXObject, error) {
	return CreateFromFileByExtension(document, imagePath)
}

// CreateFromFileByContent creates an image XObject from a file, choosing the
// factory by what the file starts with rather than by its name.
//
// Port of createFromFileByContent.
func CreateFromFileByContent(document DocumentLike, path string) (*PDImageXObject, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Could not determine file type: %s: %w", path, err)
	}
	return CreateFromByteArrayNamed(document, data, path, nil)
}

// CreateFromByteArray creates an image XObject from bytes, choosing the factory
// by what they start with.
//
// Port of createFromByteArray(PDDocument, byte[], String).
func CreateFromByteArray(document DocumentLike, byteArray []byte,
	name string) (*PDImageXObject, error) {
	return CreateFromByteArrayNamed(document, byteArray, name, nil)
}

// CreateFromByteArrayNamed creates an image XObject from bytes, letting a
// custom factory take the formats the port would otherwise decode itself.
//
// Port of createFromByteArray(PDDocument, byte[], String, CustomFactory).
func CreateFromByteArrayNamed(document DocumentLike, byteArray []byte, name string,
	customFactory CustomFactory) (*PDImageXObject, error) {
	fileType := filetypedetector.DetectFileTypeOfBytes(byteArray)

	if fileType == filetypedetector.JPEG {
		return CreateFromJPEGByteArray(document, byteArray)
	}
	if fileType == filetypedetector.PNG {
		// Try to directly convert the image without recoding it.
		img, err := ConvertPNGImage(document, byteArray)
		if err != nil {
			return nil, err
		}
		if img != nil {
			return img, nil
		}
	}
	if fileType == filetypedetector.TIFF {
		img, err := CreateCCITTFromByteArray(document, byteArray)
		if err == nil && img != nil {
			return img, nil
		}
		// Reading as TIFF failed, setting fileType to PNG.
		fileType = filetypedetector.PNG
	}
	if fileType == filetypedetector.BMP || fileType == filetypedetector.GIF ||
		fileType == filetypedetector.PNG {
		if customFactory != nil {
			return customFactory(document, byteArray)
		}
		return createLosslessFromBytes(document, byteArray, name)
	}
	// Java throws IllegalArgumentException, which is unchecked.
	panic(fmt.Sprintf("Image type %v not supported: %s", fileType, name))
}

// createLosslessFromBytes decodes a picture and hands it to LosslessFactory.
//
// Java decodes with ImageIO.read, which reads GIF, BMP and PNG. Go's standard
// library reads GIF and PNG; BMP is not in it, and the port reports that rather
// than pretending, because a BMP is exactly the case Java's caller has already
// decided belongs here. See migration/STATUS.md.
func createLosslessFromBytes(document DocumentLike, data []byte,
	name string) (*PDImageXObject, error) {
	switch filetypedetector.DetectFileTypeOfBytes(data) {
	case filetypedetector.PNG:
		img, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		return CreateFromImage(document, img)
	case filetypedetector.GIF:
		img, err := gif.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		return CreateFromImage(document, img)
	}
	return nil, fmt.Errorf("image: no decoder for %s in this port", name)
}

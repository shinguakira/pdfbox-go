package encryption

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// defaultKeyLength is the key length a handler starts with.
const defaultKeyLength int16 = 40

// aesSalt is appended to the object key when the crypt filter is AES.
//
// See 7.6.2, page 58, PDF 32000-1:2008.
var aesSalt = []byte{0x73, 0x41, 0x6c, 0x54}

// SecurityHandler is a security handler as described in the PDF
// specifications. A security handler is responsible for documents protection.
//
// Port of the abstract generic class
// org.apache.pdfbox.pdmodel.encryption.SecurityHandler<TPOLICY>. Java's type
// parameter says which protection policy the handler takes; Go has no covariant
// override, so the concrete handlers narrow the policy themselves.
type SecurityHandler interface {
	// PrepareDocumentForEncryption prepares everything to decrypt the document.
	PrepareDocumentForEncryption(doc PDDocumentLike) error

	// PrepareForDecryption prepares everything to decrypt the document.
	PrepareForDecryption(encryption *PDEncryption, documentIDArray *cos.Array,
		decryptionMaterial DecryptionMaterial) error

	// Decrypt returns the given object with every string and stream below it
	// decrypted.
	Decrypt(obj cos.Base, objNum, genNum int64) (cos.Base, error)

	// DecryptStream decrypts a stream in place.
	DecryptStream(stream *cos.Stream, objNum, genNum int64) error

	// EncryptStream encrypts a stream in place.
	EncryptStream(stream *cos.Stream, objNum int64, genNum int) error

	// EncryptString returns the encrypted form of the given string.
	EncryptString(str *cos.StringObj, objNum int64, genNum int) (cos.Base, error)

	// KeyLength returns the length of the secret key used to encrypt the
	// document.
	KeyLength() int

	// SetKeyLength sets the length of the secret key.
	SetKeyLength(keyLen int)

	// CurrentAccessPermission returns the access permissions granted to the
	// user of the document.
	CurrentAccessPermission() *AccessPermission

	// SetCurrentAccessPermission sets the access permissions granted to the
	// user of the document.
	SetCurrentAccessPermission(currentAccessPermission *AccessPermission)

	// IsAES reports whether the security handler uses AES.
	IsAES() bool

	// SetAES sets whether the security handler uses AES.
	SetAES(aesValue bool)

	// HasProtectionPolicy reports whether a protection policy has been set.
	HasProtectionPolicy() bool

	// EncryptionKey returns the current encryption key data.
	EncryptionKey() []byte

	// SetEncryptionKey sets the current encryption key data.
	SetEncryptionKey(encryptionKey []byte)

	// IsDecryptMetadata reports whether the metadata is decrypted.
	IsDecryptMetadata() bool

	// SetCustomSecureRandom sets a custom source of randomness, which is what a
	// test uses to make a run repeatable.
	SetCustomSecureRandom(customSecureRandom io.Reader)
}

// PDDocumentLike is what a security handler needs of the document it prepares
// for encryption: the encryption dictionary it reads and the two places it puts
// the one it builds.
//
// Java takes a PDDocument, which lives in pdmodel and imports this package; the
// port declares what is used, so that the dependency runs one way.
type PDDocumentLike interface {
	// Encryption returns the encryption dictionary of the document, or nil
	// where it has none.
	Encryption() *PDEncryption

	// SetEncryptionDictionary stores the encryption dictionary on the document.
	SetEncryptionDictionary(encryption *PDEncryption)

	// COSDocument returns the COS document below the PD one.
	COSDocument() COSDocumentLike
}

// COSDocumentLike is what a security handler needs of the COS document.
type COSDocumentLike interface {
	// DocumentID returns the /ID array of the trailer, or nil.
	DocumentID() *cos.Array

	// SetDocumentID sets the /ID array of the trailer.
	SetDocumentID(id *cos.Array)

	// SetEncryptionDictionary sets the encryption dictionary.
	SetEncryptionDictionary(dictionary *cos.Dictionary)

	// String is Java Object.toString, which the revision 2 to 4 document ID
	// digest feeds on.
	String() string
}

// securityHandlerBase is the state and the concrete methods every security
// handler shares.
//
// Port of the non-abstract half of SecurityHandler. The two abstract methods
// reach the concrete handler through self, since Go embedding does not
// dispatch.
type securityHandlerBase struct {
	self SecurityHandler

	keyLength     int16
	encryptionKey []byte
	rc4           *rc4Cipher

	decryptMetadata    bool
	customSecureRandom io.Reader

	// PDFBOX-4453, PDFBOX-4477: Originally this was just a Set. This failed in
	// rare cases when a decrypted string was identical to an encrypted string.
	// Because COSString.equals() checks the contents, decryption was then
	// skipped. This solution keeps all different "equal" objects.
	// IdentityHashMap solves this problem and is also faster than a HashMap.
	//
	// A Go map keyed on an interface holding a pointer already compares by
	// identity, which is what the IdentityHashMap is there for.
	objects map[cos.Base]bool

	useAES                  bool
	hasPolicy               bool
	currentAccessPermission *AccessPermission
	streamFilterName        *cos.Name
	stringFilterName        *cos.Name
}

// newSecurityHandlerBase is Java's protected no-argument constructor.
func newSecurityHandlerBase() securityHandlerBase {
	return securityHandlerBase{
		keyLength: defaultKeyLength,
		rc4:       newRC4Cipher(),
		objects:   map[cos.Base]bool{},
	}
}

// setDecryptMetadata sets whether the metadata is decrypted.
func (h *securityHandlerBase) setDecryptMetadata(decryptMetadata bool) {
	h.decryptMetadata = decryptMetadata
}

// IsDecryptMetadata reports whether the metadata is decrypted.
func (h *securityHandlerBase) IsDecryptMetadata() bool { return h.decryptMetadata }

// setStringFilterName sets the string filter name.
func (h *securityHandlerBase) setStringFilterName(stringFilterName *cos.Name) {
	h.stringFilterName = stringFilterName
}

// setStreamFilterName sets the stream filter name.
func (h *securityHandlerBase) setStreamFilterName(streamFilterName *cos.Name) {
	h.streamFilterName = streamFilterName
}

// SetCustomSecureRandom sets a custom source of randomness.
func (h *securityHandlerBase) SetCustomSecureRandom(customSecureRandom io.Reader) {
	h.customSecureRandom = customSecureRandom
}

// encryptData encrypts or decrypts the data read from the given reader.
func (h *securityHandlerBase) encryptData(objectNumber, genNumber int64, data io.Reader,
	output io.Writer, decrypt bool) error {
	// Determine whether we're using Algorithm 1 (for RC4 and AES-128), or 1.A (for AES-256)
	if h.useAES && len(h.encryptionKey) == 32 {
		return h.encryptDataAES256(data, output, decrypt)
	}
	finalKey := h.calcFinalKey(objectNumber, genNumber)
	if h.useAES {
		return h.encryptDataAESother(finalKey, data, output, decrypt)
	}
	return h.encryptDataRC4Stream(finalKey, data, output)
}

// calcFinalKey computes the key to be used for the given object.
func (h *securityHandlerBase) calcFinalKey(objectNumber, genNumber int64) []byte {
	newKey := make([]byte, len(h.encryptionKey)+5)
	copy(newKey, h.encryptionKey)
	// PDF 1.4 reference pg 73
	// step 1
	// we have the reference
	// step 2
	newKey[len(newKey)-5] = byte(objectNumber & 0xff)
	newKey[len(newKey)-4] = byte(objectNumber >> 8 & 0xff)
	newKey[len(newKey)-3] = byte(objectNumber >> 16 & 0xff)
	newKey[len(newKey)-2] = byte(genNumber & 0xff)
	newKey[len(newKey)-1] = byte(genNumber >> 8 & 0xff)
	// step 3
	md := md5.New()
	md.Write(newKey)
	if h.useAES {
		md.Write(aesSalt)
	}
	digestedKey := md.Sum(nil)
	// step 4
	length := min(len(newKey), 16)
	finalKey := make([]byte, length)
	copy(finalKey, digestedKey[:length])
	return finalKey
}

// encryptDataRC4Stream encrypts or decrypts data with RC4.
func (h *securityHandlerBase) encryptDataRC4Stream(finalKey []byte, input io.Reader,
	output io.Writer) error {
	if err := h.rc4.setKey(finalKey); err != nil {
		return err
	}
	return h.rc4.writeStream(input, output)
}

// encryptDataRC4Bytes encrypts or decrypts data with RC4.
func (h *securityHandlerBase) encryptDataRC4Bytes(finalKey, input []byte,
	output io.Writer) error {
	if err := h.rc4.setKey(finalKey); err != nil {
		return err
	}
	return h.rc4.writeBytes(input, output)
}

// encryptDataAESother encrypts or decrypts data with AES-128 or AES-192.
func (h *securityHandlerBase) encryptDataAESother(finalKey []byte, data io.Reader,
	output io.Writer, decrypt bool) error {
	iv := make([]byte, 16)
	ready, err := h.prepareAESInitializationVector(decrypt, iv, data, output)
	if err != nil {
		return err
	}
	if !ready {
		return nil
	}
	rest, err := io.ReadAll(data)
	if err != nil {
		return err
	}
	// Java feeds the stream through Cipher.update in 256-byte pieces and then
	// calls doFinal; for CBC that is the same as running the whole input at
	// once, and a GeneralSecurityException from either becomes an IOException.
	result, err := aesCBC(finalKey, iv, rest, decrypt)
	if err != nil {
		return err
	}
	_, err = output.Write(result)
	return err
}

// encryptDataAES256 encrypts or decrypts data with AES-256.
func (h *securityHandlerBase) encryptDataAES256(data io.Reader, output io.Writer,
	decrypt bool) error {
	iv := make([]byte, 16)
	ready, err := h.prepareAESInitializationVector(decrypt, iv, data, output)
	if err != nil {
		return err
	}
	if !ready {
		return nil
	}
	rest, err := io.ReadAll(data)
	if err != nil {
		return err
	}
	result, err := aesCBC(h.encryptionKey, iv, rest, decrypt)
	if err != nil {
		// starting with java 8 the JVM wraps an IOException around a
		// GeneralSecurityException; it should be safe to swallow a
		// GeneralSecurityException.
		//
		// Java reads through a CipherInputStream, which hands over every block
		// it has already decrypted before the failing final one; the partial
		// result below is that same output.
		var partial *badPaddingError
		if !errors.As(err, &partial) {
			return err
		}
		slog.Debug("A GeneralSecurityException occurred when decrypting some stream data",
			"err", err)
		_, writeErr := output.Write(partial.decrypted)
		return writeErr
	}
	_, err = output.Write(result)
	return err
}

// badPaddingError is javax.crypto.BadPaddingException, which the AES-256 path
// swallows and the AES-128 path turns into an IOException. It carries the
// plaintext of every block before the one that failed, which is what Java's
// CipherInputStream has already written by then.
type badPaddingError struct {
	decrypted []byte
	reason    string
}

func (e *badPaddingError) Error() string { return e.reason }

// aesCBC runs AES in CBC mode with the PKCS#5 padding the PDF specification
// asks for.
//
// Port of SecurityHandler.createCipher and the update/doFinal calls around it.
// Java asks the JCE for "AES/CBC/PKCS5Padding"; Go's crypto/cipher has the mode
// but not the padding, so the padding is written out.
func aesCBC(key, iv, data []byte, decrypt bool) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if decrypt {
		if len(data)%aes.BlockSize != 0 {
			// javax.crypto.IllegalBlockSizeException
			return nil, fmt.Errorf(
				"encryption: input length not a multiple of the block size: %d", len(data))
		}
		if len(data) == 0 {
			return nil, &badPaddingError{reason: "no data to decrypt"}
		}
		plain := make([]byte, len(data))
		cipher.NewCBCDecrypter(block, iv).CryptBlocks(plain, data)
		unpadded, err := pkcs5Unpad(plain)
		if err != nil {
			return nil, &badPaddingError{
				decrypted: plain[:len(plain)-aes.BlockSize],
				reason:    err.Error(),
			}
		}
		return unpadded, nil
	}
	padded := pkcs5Pad(data)
	encrypted := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(encrypted, padded)
	return encrypted, nil
}

// pkcs5Pad appends the padding PKCS#5 calls for, which is always at least one
// byte.
func pkcs5Pad(data []byte) []byte {
	padding := aes.BlockSize - len(data)%aes.BlockSize
	padded := make([]byte, len(data)+padding)
	copy(padded, data)
	for i := len(data); i < len(padded); i++ {
		padded[i] = byte(padding)
	}
	return padded
}

// pkcs5Unpad strips the padding, reporting what BadPaddingException reports.
func pkcs5Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("Given final block not properly padded")
	}
	padding := int(data[len(data)-1])
	if padding < 1 || padding > aes.BlockSize || padding > len(data) {
		return nil, errors.New("Given final block not properly padded")
	}
	for i := len(data) - padding; i < len(data); i++ {
		if int(data[i]) != padding {
			return nil, errors.New("Given final block not properly padded")
		}
	}
	return data[:len(data)-padding], nil
}

// prepareAESInitializationVector reads the initialization vector from the
// stream when decrypting, and generates and writes one when encrypting. The
// first result is false where there was nothing to read.
func (h *securityHandlerBase) prepareAESInitializationVector(decrypt bool, iv []byte,
	data io.Reader, output io.Writer) (bool, error) {
	if decrypt {
		// read IV from stream
		ivSize, err := io.ReadFull(data, iv)
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
			return false, err
		}
		if ivSize == 0 {
			return false, nil
		}
		if ivSize != len(iv) {
			return false, fmt.Errorf(
				"AES initialization vector not fully read: only %d bytes read instead of %d",
				ivSize, len(iv))
		}
	} else {
		// generate random IV and write to stream
		rnd := h.secureRandom()
		if _, err := io.ReadFull(rnd, iv); err != nil {
			return false, err
		}
		if _, err := output.Write(iv); err != nil {
			return false, err
		}
	}
	return true, nil
}

func (h *securityHandlerBase) secureRandom() io.Reader {
	if h.customSecureRandom != nil {
		return h.customSecureRandom
	}
	return rand.Reader
}

// decryptStringIfAbsent decrypts a string, unless it has been decrypted
// already.
func (h *securityHandlerBase) decryptStringIfAbsent(str *cos.StringObj,
	objNum, genNum int64) cos.Base {
	// PDFBOX-4477: only cache strings and streams, this improves speed and
	// memory footprint
	if h.objects[cos.Base(str)] {
		return str
	}
	// replace the given COSString object with the encrypted/decrypted version
	decryptedString := h.decryptString(str, objNum, genNum)
	h.objects[decryptedString] = true
	return decryptedString
}

// Decrypt returns the given object with every string and stream below it
// decrypted.
func (h *securityHandlerBase) Decrypt(obj cos.Base, objNum, genNum int64) (cos.Base, error) {
	// PDFBOX-4477: only cache strings and streams, this improves speed and
	// memory footprint
	switch value := obj.(type) {
	case *cos.StringObj:
		return h.decryptStringIfAbsent(value, objNum, genNum), nil
	case *cos.Stream:
		return h.decryptStreamIfAbsent(value, objNum, genNum)
	case *cos.Dictionary:
		return h.decryptDictionary(value, objNum, genNum)
	case *cos.Array:
		return h.decryptArray(value, objNum, genNum)
	}
	return obj, nil
}

// decryptStreamIfAbsent decrypts a stream, unless it has been decrypted
// already.
func (h *securityHandlerBase) decryptStreamIfAbsent(stream *cos.Stream,
	objNum, genNum int64) (cos.Base, error) {
	if !h.objects[cos.Base(stream)] {
		h.objects[cos.Base(stream)] = true
		if err := h.DecryptStream(stream, objNum, genNum); err != nil {
			return nil, err
		}
	}
	return stream, nil
}

// DecryptStream decrypts a stream in place.
func (h *securityHandlerBase) DecryptStream(stream *cos.Stream, objNum, genNum int64) error {
	// Stream encrypted with identity filter
	if cos.Identity.Equals(h.streamFilterName) {
		return nil
	}

	streamType := stream.GetCOSName(cos.Type)
	if !h.decryptMetadata && cos.Metadata.Equals(streamType) {
		return nil
	}
	// "The cross-reference stream shall not be encrypted"
	if cos.XRef.Equals(streamType) {
		return nil
	}
	if cos.Metadata.Equals(streamType) {
		// PDFBOX-3229 check case where metadata is not encrypted despite
		// /EncryptMetadata missing
		raw, err := stream.CreateRawReader()
		if err != nil {
			return err
		}
		const nBytes = 10
		buf := make([]byte, nBytes)
		read, err := io.ReadFull(raw, buf)
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
			return err
		}
		buf = buf[:read]
		if read != nBytes {
			slog.Debug("Tried reading bytes but only fewer read", "wanted", nBytes, "read", read)
		}
		if bytes.Equal(buf, []byte("<?xpacket ")) {
			slog.Warn("Metadata is not encrypted, but was expected to be")
			slog.Warn("Read PDF specification about EncryptMetadata (default value: true)")
			return nil
		}
	}

	if _, err := h.decryptDictionary(&stream.Dictionary, objNum, genNum); err != nil {
		return err
	}

	// the input and the output stream of a still encrypted COSStream aren't no
	// longer based on the same object so that it is safe to omit the
	// intermediate ByteArrayStream
	encryptedStream, err := stream.CreateRawReader()
	if err != nil {
		return err
	}
	output, err := stream.CreateRawWriter()
	if err != nil {
		return err
	}
	if err := h.encryptData(objNum, genNum, encryptedStream, output, true /* decrypt */); err != nil {
		output.Close()
		slog.Error("error thrown when decrypting object", "objNum", objNum, "genNum", genNum,
			"err", err)
		return err
	}
	return output.Close()
}

// EncryptStream encrypts a stream in place.
func (h *securityHandlerBase) EncryptStream(stream *cos.Stream, objNum int64, genNum int) error {
	// empty streams don't need to be encrypted
	if !stream.HasData() {
		return nil
	}
	in, err := stream.CreateRawReader()
	if err != nil {
		return err
	}
	rawData, err := io.ReadAll(in)
	if err != nil {
		return err
	}
	encryptedStream := bytes.NewReader(rawData)
	output, err := stream.CreateRawWriter()
	if err != nil {
		return err
	}
	if err := h.encryptData(objNum, int64(genNum), encryptedStream, output,
		false /* encrypt */); err != nil {
		output.Close()
		return err
	}
	return output.Close()
}

// decryptDictionary decrypts a dictionary in place.
func (h *securityHandlerBase) decryptDictionary(dictionary *cos.Dictionary,
	objNum, genNum int64) (cos.Base, error) {
	if dictionary.GetItem(cos.CF) != nil {
		// PDFBOX-2936: avoid orphan /CF dictionaries found in US govt "I-" files
		return dictionary, nil
	}
	dictionaryType := dictionary.GetCOSName(cos.Type)
	_, contentsIsString := dictionary.GetDictionaryObject(cos.Contents).(*cos.StringObj)
	_, byteRangeIsArray := dictionary.GetDictionaryObject(cos.ByteRange).(*cos.Array)
	isSignature := cos.Sig.Equals(dictionaryType) || cos.DocTimeStamp.Equals(dictionaryType) ||
		// PDFBOX-4466: /Type is optional, see
		// https://ec.europa.eu/cefdigital/tracker/browse/DSS-1538
		(contentsIsString && byteRangeIsArray)

	for _, key := range dictionary.KeySet() {
		if isSignature && cos.Contents.Equals(key) {
			// do not decrypt the signature contents string
			continue
		}
		value := dictionary.GetItem(key)
		// within a dictionary only the following kind of COS objects have to be
		// decrypted
		switch typed := value.(type) {
		case *cos.StringObj:
			dictionary.SetItem(key, h.decryptStringIfAbsent(typed, objNum, genNum))
		case *cos.Array:
			decrypted, err := h.decryptArray(typed, objNum, genNum)
			if err != nil {
				return nil, err
			}
			dictionary.SetItem(key, decrypted)
		case *cos.Stream:
			decrypted, err := h.decryptStreamIfAbsent(typed, objNum, genNum)
			if err != nil {
				return nil, err
			}
			dictionary.SetItem(key, decrypted)
		case *cos.Dictionary:
			decrypted, err := h.decryptDictionary(typed, objNum, genNum)
			if err != nil {
				return nil, err
			}
			dictionary.SetItem(key, decrypted)
		}
	}
	return dictionary, nil
}

// decryptString decrypts a string.
func (h *securityHandlerBase) decryptString(str *cos.StringObj, objNum, genNum int64) cos.Base {
	// String encrypted with identity filter
	if cos.Identity.Equals(h.stringFilterName) {
		return str
	}
	data := bytes.NewReader(str.Bytes())
	var outputStream bytes.Buffer
	if err := h.encryptData(objNum, genNum, data, &outputStream, true /* decrypt */); err != nil {
		slog.Error("Failed to decrypt COSString", "length", len(str.Bytes()),
			"object", objNum, "err", err)
		return str
	}
	return cos.NewStringObjBytes(outputStream.Bytes())
}

// EncryptString returns the encrypted form of the given string.
func (h *securityHandlerBase) EncryptString(str *cos.StringObj, objNum int64,
	genNum int) (cos.Base, error) {
	data := bytes.NewReader(str.Bytes())
	var buffer bytes.Buffer
	if err := h.encryptData(objNum, int64(genNum), data, &buffer, false /* encrypt */); err != nil {
		return nil, err
	}
	return cos.NewStringObjBytes(buffer.Bytes()), nil
}

// decryptArray decrypts an array in place.
func (h *securityHandlerBase) decryptArray(array *cos.Array, objNum,
	genNum int64) (cos.Base, error) {
	for i := 0; i < array.Size(); i++ {
		decrypted, err := h.Decrypt(array.Get(i), objNum, genNum)
		if err != nil {
			return nil, err
		}
		array.Set(i, decrypted)
	}
	return array, nil
}

// KeyLength returns the length of the secret key used to encrypt the document.
func (h *securityHandlerBase) KeyLength() int { return int(h.keyLength) }

// SetKeyLength sets the length of the secret key used to encrypt the document.
func (h *securityHandlerBase) SetKeyLength(keyLen int) { h.keyLength = int16(keyLen) }

// SetCurrentAccessPermission sets the access permissions granted to the user of
// the document.
func (h *securityHandlerBase) SetCurrentAccessPermission(
	currentAccessPermission *AccessPermission) {
	h.currentAccessPermission = currentAccessPermission
}

// CurrentAccessPermission returns the access permissions granted to the user of
// the document.
func (h *securityHandlerBase) CurrentAccessPermission() *AccessPermission {
	return h.currentAccessPermission
}

// IsAES reports whether the security handler uses AES.
func (h *securityHandlerBase) IsAES() bool { return h.useAES }

// SetAES sets whether the security handler uses AES.
func (h *securityHandlerBase) SetAES(aesValue bool) { h.useAES = aesValue }

// HasProtectionPolicy reports whether a protection policy has been set.
func (h *securityHandlerBase) HasProtectionPolicy() bool { return h.hasPolicy }

// EncryptionKey returns the current encryption key data.
func (h *securityHandlerBase) EncryptionKey() []byte { return h.encryptionKey }

// SetEncryptionKey sets the current encryption key data.
func (h *securityHandlerBase) SetEncryptionKey(encryptionKey []byte) {
	h.encryptionKey = encryptionKey
}

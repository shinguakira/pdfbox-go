package encryption

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"errors"
	"fmt"
	"hash"
	"io"
	"log/slog"
	"math/big"
	"time"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// The revisions of the standard security handler.
const (
	revision2 = 2
	revision3 = 3
	revision4 = 4
	revision5 = 5
	revision6 = 6
)

// StandardSecurityHandlerFilter is the /Filter name of this handler.
//
// Port of StandardSecurityHandler.FILTER.
const StandardSecurityHandlerFilter = "Standard"

// encryptPadding is the 32-byte pad every revision 2 to 4 password is padded
// with or truncated to.
var encryptPadding = []byte{
	0x28, 0xBF, 0x4E, 0x5E, 0x4E,
	0x75, 0x8A, 0x41, 0x64, 0x00,
	0x4E, 0x56, 0xFF, 0xFA, 0x01,
	0x08, 0x2E, 0x2E, 0x00, 0xB6,
	0xD0, 0x68, 0x3E, 0x80, 0x2F,
	0x0C, 0xA9, 0xFE, 0x64, 0x53,
	0x69, 0x7A,
}

// hashes2B are the hashes used for Algorithm 2.B, depending on remainder from E
// modulo 3.
var hashes2B = []func() hash.Hash{sha256.New, sha512.New384, sha512.New}

// StandardSecurityHandler is the standard security handler. This security
// handler protects document with password.
//
// Port of org.apache.pdfbox.pdmodel.encryption.StandardSecurityHandler.
type StandardSecurityHandler struct {
	securityHandlerBase

	protectionPolicy *StandardProtectionPolicy
}

var _ SecurityHandler = (*StandardSecurityHandler)(nil)

// NewStandardSecurityHandler creates a handler for reading an encrypted
// document.
func NewStandardSecurityHandler() *StandardSecurityHandler {
	h := &StandardSecurityHandler{securityHandlerBase: newSecurityHandlerBase()}
	h.self = h
	return h
}

// NewStandardSecurityHandlerOfPolicy creates a handler for encrypting a
// document with the given policy.
func NewStandardSecurityHandlerOfPolicy(
	standardProtectionPolicy *StandardProtectionPolicy) *StandardSecurityHandler {
	h := NewStandardSecurityHandler()
	h.protectionPolicy = standardProtectionPolicy
	h.hasPolicy = true
	h.keyLength = int16(standardProtectionPolicy.EncryptionKeyLength())
	return h
}

// computeVersionNumber computes the version number of the standard security
// handler based on the encryption key length.
//
// Port of SecurityHandler.computeVersionNumber, which reads the protection
// policy and so cannot live on the shared base in Go.
func (h *StandardSecurityHandler) computeVersionNumber() int {
	if h.keyLength == 40 {
		return 1
	} else if h.keyLength == 128 && h.protectionPolicy.IsPreferAES() {
		return 4
	} else if h.keyLength == 256 {
		return 5
	}
	return 2
}

// computeRevisionNumber computes the revision version of the
// StandardSecurityHandler to use regarding the version number and the
// permissions bits set.
func (h *StandardSecurityHandler) computeRevisionNumber(version int) int {
	protectionPolicy := h.protectionPolicy
	permissions := protectionPolicy.Permissions()
	if version < revision2 && !permissions.hasAnyRevision3PermissionSet() {
		return revision2
	}
	if version == revision5 {
		// note about revision 5: "Shall not be used. This value was used by a
		// deprecated Adobe extension."
		return revision6
	}
	if version == revision4 {
		return revision4
	}
	if version == revision2 || version == revision3 ||
		permissions.hasAnyRevision3PermissionSet() {
		return revision3
	}
	return revision4
}

// PrepareForDecryption prepares everything to decrypt the document.
//
// Only if decryption of single objects is needed this should be called.
func (h *StandardSecurityHandler) PrepareForDecryption(encryption *PDEncryption,
	documentIDArray *cos.Array, decryptionMaterial DecryptionMaterial) error {
	material, ok := decryptionMaterial.(*StandardDecryptionMaterial)
	if !ok {
		return errors.New("Decryption material is not compatible with the document")
	}

	encryptionVersion := encryption.Version()
	// This is only used with security version 4 and 5.
	if encryptionVersion >= revision4 {
		h.setStreamFilterName(encryption.StreamFilterName())
		h.setStringFilterName(encryption.StringFilterName())
	}
	h.setDecryptMetadata(encryption.IsEncryptMetaData())

	password := material.Password()

	dicPermissions := encryption.Permissions()
	dicRevision := encryption.Revision()
	dicLength := encryption.Length() / 8
	if encryptionVersion == 1 {
		dicLength = 5
	}

	if encryptionVersion == revision4 || encryptionVersion == revision5 {
		// detect whether AES encryption is used. This assumes that the
		// encryption algo is stored in the PDCryptFilterDictionary
		// However, crypt filters are used only when V is 4 or 5.
		stdCryptFilterDictionary := encryption.StdCryptFilterDictionary()
		if stdCryptFilterDictionary != nil {
			cryptFilterMethod := stdCryptFilterDictionary.CryptFilterMethod()
			if cos.AESV2.Equals(cryptFilterMethod) {
				dicLength = 128 / 8
				h.SetAES(true)
				if encryption.COSObject().ContainsKey(cos.Length) {
					// PDFBOX-5345
					newLength := encryption.Length() / 8
					if newLength < dicLength {
						slog.Warn("Using fewer bytes key length in AESV2 encryption?!",
							"used", newLength, "expected", dicLength)
						dicLength = newLength
					}
				}
			}
			if cos.AESV3.Equals(cryptFilterMethod) {
				dicLength = 256 / 8
				h.SetAES(true)
				if encryption.COSObject().ContainsKey(cos.Length) {
					// PDFBOX-5345
					newLength := encryption.Length() / 8
					if newLength < dicLength {
						slog.Warn("Using fewer bytes key length in AESV3 encryption?!",
							"used", newLength, "expected", dicLength)
						dicLength = newLength
					}
				}
			}
		}
	}

	documentIDBytes := getDocumentIDBytes(documentIDArray)

	// we need to know whether the meta data was encrypted for password calculation
	encryptMetadata := encryption.IsEncryptMetaData()

	userKey := encryption.UserKey()
	ownerKey := encryption.OwnerKey()
	var ue, oe []byte

	// Java's ISO_8859_1 charset maps each char to its low byte; UTF_8 encodes.
	passwordIsUTF8 := false
	if dicRevision == revision5 || dicRevision == revision6 {
		passwordIsUTF8 = true
		ue = encryption.UserEncryptionKey()
		oe = encryption.OwnerEncryptionKey()
	}

	if dicRevision == revision6 {
		prepared, err := SaslPrepQuery(password) // PDFBOX-4155
		if err != nil {
			return err
		}
		password = prepared
	}

	var currentAccessPermission *AccessPermission

	passwordBytes := encodeISO88591(password)
	if passwordIsUTF8 {
		passwordBytes = []byte(password)
	}
	var isOwnerPassword bool

	ownerMatches, err := h.IsOwnerPasswordBytes(passwordBytes, userKey, ownerKey,
		dicPermissions, documentIDBytes, dicRevision, dicLength, encryptMetadata)
	if err != nil {
		return err
	}
	if ownerMatches {
		currentAccessPermission = OwnerAccessPermission()
		h.SetCurrentAccessPermission(currentAccessPermission)
		if dicRevision != revision5 && dicRevision != revision6 {
			passwordBytes, err = h.getUserPassword234(passwordBytes, ownerKey, dicRevision,
				dicLength)
			if err != nil {
				return err
			}
		}
		isOwnerPassword = true
	} else {
		userMatches, err := h.IsUserPasswordBytes(passwordBytes, userKey, ownerKey,
			dicPermissions, documentIDBytes, dicRevision, dicLength, encryptMetadata)
		if err != nil {
			return err
		}
		if userMatches {
			currentAccessPermission = NewAccessPermissionOf(int32(dicPermissions))
			currentAccessPermission.SetReadOnly()
			h.SetCurrentAccessPermission(currentAccessPermission)
			isOwnerPassword = false
		} else {
			return newInvalidPasswordError("Cannot decrypt PDF, the password is incorrect")
		}
	}

	encryptedKey, err := h.ComputeEncryptedKey(passwordBytes, ownerKey, userKey, oe, ue,
		dicPermissions, documentIDBytes, dicRevision, dicLength, encryptMetadata, isOwnerPassword)
	if err != nil {
		return err
	}
	if dicRevision == revision4 && len(encryptedKey) < 16 {
		slog.Info("PDFBOX-5955: padding RC4 key to length 16")
		encryptedKey = copyOf(encryptedKey, 16)
	}
	h.SetEncryptionKey(encryptedKey)

	if dicRevision == revision5 || dicRevision == revision6 {
		return h.validatePerms(encryption, dicPermissions, encryptMetadata)
	}
	return nil
}

// getDocumentIDBytes returns the first element of the document ID array, or an
// empty slice.
//
// Some documents may not have a document id, see
// test/encryption/encrypted_doc_no_id.pdf.
func getDocumentIDBytes(documentIDArray *cos.Array) []byte {
	if documentIDArray != nil && documentIDArray.Size() != 0 {
		// Java casts the first element to COSString without checking.
		id := documentIDArray.GetObject(0).(*cos.StringObj)
		return id.Bytes()
	}
	return []byte{}
}

// validatePerms is Algorithm 13: validate permissions ("Perms" field). Relaxed
// to accommodate buggy encoders.
//
// https://www.adobe.com/content/dam/Adobe/en/devnet/acrobat/pdfs/adobe_supplement_iso32000.pdf
func (h *StandardSecurityHandler) validatePerms(encryption *PDEncryption, dicPermissions int,
	encryptMetadata bool) error {
	// "Decrypt the 16-byte Perms string using AES-256 in ECB mode with an
	// initialization vector of zero and the file encryption key as the key."
	block, err := aes.NewCipher(h.EncryptionKey())
	if err != nil {
		logIfStrongEncryptionMissing()
		return err
	}
	encryptedPerms := encryption.Perms()
	if len(encryptedPerms)%aes.BlockSize != 0 || len(encryptedPerms) < 12 {
		logIfStrongEncryptionMissing()
		return fmt.Errorf("encryption: /Perms is %d bytes", len(encryptedPerms))
	}
	perms := make([]byte, len(encryptedPerms))
	for offset := 0; offset < len(encryptedPerms); offset += aes.BlockSize {
		block.Decrypt(perms[offset:offset+aes.BlockSize], encryptedPerms[offset:])
	}

	// "Verify that bytes 9-11 of the result are the characters 'a', 'd', 'b'."
	if perms[9] != 'a' || perms[10] != 'd' || perms[11] != 'b' {
		slog.Warn("Verification of permissions failed (constant)")
	}

	// "Bytes 0-3 of the decrypted Perms entry, treated as a little-endian
	// integer, are the user permissions. They should match the value in the P
	// key."
	permsP := int32(perms[0]) | int32(perms[1])<<8 | int32(perms[2])<<16 | int32(perms[3])<<24
	if permsP != int32(dicPermissions) {
		slog.Warn("Verification of permissions failed",
			"perms", fmt.Sprintf("%08X", uint32(permsP)),
			"dictionary", fmt.Sprintf("%08X", uint32(dicPermissions)))
	}
	if encryptMetadata && perms[8] != 'T' || !encryptMetadata && perms[8] != 'F' {
		slog.Warn("Verification of permissions failed (EncryptMetadata)")
	}
	return nil
}

// PrepareDocumentForEncryption prepares everything to encrypt the document.
//
// If you calling this method directly, it is recommended to also call
// document.setAllSecurityToBeRemoved(false) afterwards.
func (h *StandardSecurityHandler) PrepareDocumentForEncryption(document PDDocumentLike) error {
	encryptionDictionary := document.Encryption()
	if encryptionDictionary == nil {
		encryptionDictionary = NewPDEncryption()
	}
	version := h.computeVersionNumber()
	revision := h.computeRevisionNumber(version)
	encryptionDictionary.SetFilter(StandardSecurityHandlerFilter)
	encryptionDictionary.SetVersion(version)
	if version != revision4 && version != revision5 {
		// remove CF, StmF, and StrF entries that may be left from a previous
		// encryption
		encryptionDictionary.RemoveV45filters()
	}
	encryptionDictionary.SetRevision(revision)
	encryptionDictionary.SetLength(h.KeyLength())

	protectionPolicy := h.protectionPolicy
	ownerPassword := protectionPolicy.OwnerPassword()
	userPassword := protectionPolicy.UserPassword()

	// If no owner password is set, use the user password instead.
	if ownerPassword == "" {
		ownerPassword = userPassword
	}

	permissionInt := protectionPolicy.Permissions().PermissionBytes()

	encryptionDictionary.SetPermissions(int(permissionInt))

	length := h.KeyLength() / 8

	if revision == revision6 {
		// PDFBOX-4155
		preparedOwner, err := SaslPrepStored(ownerPassword)
		if err != nil {
			return err
		}
		preparedUser, err := SaslPrepStored(userPassword)
		if err != nil {
			return err
		}
		if err := h.prepareEncryptionDictRev6(preparedOwner, preparedUser, encryptionDictionary,
			permissionInt); err != nil {
			return err
		}
	} else {
		if err := h.prepareEncryptionDictRev234(ownerPassword, userPassword, encryptionDictionary,
			permissionInt, document, revision, length); err != nil {
			return err
		}
	}

	document.SetEncryptionDictionary(encryptionDictionary)
	document.COSDocument().SetEncryptionDictionary(encryptionDictionary.COSObject())
	return nil
}

func (h *StandardSecurityHandler) prepareEncryptionDictRev6(ownerPassword, userPassword string,
	encryptionDictionary *PDEncryption, permissionInt int32) error {
	// make a random 256-bit file encryption key
	h.SetEncryptionKey(make([]byte, 32))
	if _, err := io.ReadFull(h.secureRandom(), h.EncryptionKey()); err != nil {
		return err
	}

	// Algorithm 8a: Compute U
	userPasswordBytes := truncate127([]byte(userPassword))
	userValidationSalt := make([]byte, 8)
	userKeySalt := make([]byte, 8)
	if _, err := io.ReadFull(h.secureRandom(), userValidationSalt); err != nil {
		return err
	}
	if _, err := io.ReadFull(h.secureRandom(), userKeySalt); err != nil {
		return err
	}
	hashU, err := computeHash2B(concat(userPasswordBytes, userValidationSalt),
		userPasswordBytes, nil)
	if err != nil {
		return err
	}
	u := concat3(hashU, userValidationSalt, userKeySalt)

	// Algorithm 8b: Compute UE
	hashUE, err := computeHash2B(concat(userPasswordBytes, userKeySalt), userPasswordBytes, nil)
	if err != nil {
		return err
	}
	// "an initialization vector of zero"
	ue, err := aesCBCNoPadding(hashUE, make([]byte, 16), h.EncryptionKey(), false)
	if err != nil {
		logIfStrongEncryptionMissing()
		return err
	}

	// Algorithm 9a: Compute O
	ownerPasswordBytes := truncate127([]byte(ownerPassword))
	ownerValidationSalt := make([]byte, 8)
	ownerKeySalt := make([]byte, 8)
	if _, err := io.ReadFull(h.secureRandom(), ownerValidationSalt); err != nil {
		return err
	}
	if _, err := io.ReadFull(h.secureRandom(), ownerKeySalt); err != nil {
		return err
	}
	hashO, err := computeHash2B(concat3(ownerPasswordBytes, ownerValidationSalt, u),
		ownerPasswordBytes, u)
	if err != nil {
		return err
	}
	o := concat3(hashO, ownerValidationSalt, ownerKeySalt)

	// Algorithm 9b: Compute OE
	hashOE, err := computeHash2B(concat3(ownerPasswordBytes, ownerKeySalt, u),
		ownerPasswordBytes, u)
	if err != nil {
		return err
	}
	oe, err := aesCBCNoPadding(hashOE, make([]byte, 16), h.EncryptionKey(), false)
	if err != nil {
		logIfStrongEncryptionMissing()
		return err
	}

	// Set keys and other required constants in encryption dictionary
	encryptionDictionary.SetUserKey(u)
	encryptionDictionary.SetUserEncryptionKey(ue)
	encryptionDictionary.SetOwnerKey(o)
	encryptionDictionary.SetOwnerEncryptionKey(oe)

	prepareEncryptionDictAES(h, encryptionDictionary, cos.AESV3)

	// Algorithm 10: compute "Perms" value
	perms := make([]byte, 16)
	perms[0] = byte(permissionInt)
	perms[1] = byte(uint32(permissionInt) >> 8)
	perms[2] = byte(uint32(permissionInt) >> 16)
	perms[3] = byte(uint32(permissionInt) >> 24)
	perms[4] = 0xFF
	perms[5] = 0xFF
	perms[6] = 0xFF
	perms[7] = 0xFF
	perms[8] = 'T' // we always encrypt Metadata
	perms[9] = 'a'
	perms[10] = 'd'
	perms[11] = 'b'
	tail := make([]byte, 4)
	if _, err := io.ReadFull(h.secureRandom(), tail); err != nil {
		return err
	}
	// Java takes the low byte of a random int for each of the four.
	copy(perms[12:16], tail)

	permsEnc, err := aesCBCNoPadding(h.EncryptionKey(), make([]byte, 16), perms, false)
	if err != nil {
		logIfStrongEncryptionMissing()
		return err
	}

	encryptionDictionary.SetPerms(permsEnc)
	return nil
}

func (h *StandardSecurityHandler) prepareEncryptionDictRev234(ownerPassword, userPassword string,
	encryptionDictionary *PDEncryption, permissionInt int32, document PDDocumentLike,
	revision, length int) error {
	idArray := document.COSDocument().DocumentID()

	// check if the document has an id yet.  If it does not then generate one
	userPasswordBytes := encodeISO88591(userPassword)
	ownerPasswordBytes := encodeISO88591(ownerPassword)
	if idArray == nil || idArray.Size() < 2 {
		md := md5.New()
		timeValue := big.NewInt(time.Now().UnixMilli())
		md.Write(timeValue.Bytes())
		md.Write(ownerPasswordBytes)
		md.Write(userPasswordBytes)
		md.Write(encodeISO88591(document.COSDocument().String()))
		md.Write(encodeISO88591(h.String()))
		id := md.Sum(nil)
		idString := cos.NewStringObjBytes(id)
		idArray = cos.NewArrayOf([]cos.Base{idString, idString})
		document.COSDocument().SetDocumentID(idArray)
	}

	id := idArray.GetObject(0).(*cos.StringObj)
	ownerBytes, err := h.ComputeOwnerPassword(ownerPasswordBytes, userPasswordBytes, revision,
		length)
	if err != nil {
		return err
	}

	idBytes := id.Bytes()
	userBytes, err := h.ComputeUserPassword(userPasswordBytes, ownerBytes, int(permissionInt),
		idBytes, revision, length, true)
	if err != nil {
		return err
	}

	h.SetEncryptionKey(h.computeEncryptedKeyRev234(userPasswordBytes, ownerBytes,
		int(permissionInt), idBytes, true, length, revision))

	encryptionDictionary.SetOwnerKey(ownerBytes)
	encryptionDictionary.SetUserKey(userBytes)

	if revision == revision4 {
		prepareEncryptionDictAES(h, encryptionDictionary, cos.AESV2)
	}
	return nil
}

// String is what Java's Object.toString gives the ID digest, which is the class
// name and an identity hash. Go has neither, so the port writes the class name
// and the pointer, which is the same shape and just as arbitrary.
func (h *StandardSecurityHandler) String() string {
	return fmt.Sprintf("StandardSecurityHandler@%p", h)
}

func prepareEncryptionDictAES(h *StandardSecurityHandler, encryptionDictionary *PDEncryption,
	aesVName *cos.Name) {
	cryptFilterDictionary := NewPDCryptFilterDictionary()
	cryptFilterDictionary.SetCryptFilterMethod(aesVName)
	cryptFilterDictionary.SetLength(h.KeyLength())
	encryptionDictionary.SetStdCryptFilterDictionary(cryptFilterDictionary)
	encryptionDictionary.SetStreamFilterName(cos.StdCF)
	encryptionDictionary.SetStringFilterName(cos.StdCF)
	h.SetAES(true)
}

// IsOwnerPasswordBytes checks for owner password.
func (h *StandardSecurityHandler) IsOwnerPasswordBytes(ownerPassword, user, owner []byte,
	permissions int, id []byte, encRevision, keyLengthInBytes int,
	encryptMetadata bool) (bool, error) {
	switch encRevision {
	case revision2, revision3, revision4:
		return h.isOwnerPassword234(ownerPassword, user, owner, permissions, id, encRevision,
			keyLengthInBytes, encryptMetadata)
	case revision5, revision6:
		return h.isOwnerPassword56(ownerPassword, user, owner, encRevision)
	default:
		return false, fmt.Errorf("Unknown Encryption Revision %d", encRevision)
	}
}

func (h *StandardSecurityHandler) isOwnerPassword234(ownerPassword, user, owner []byte,
	permissions int, id []byte, encRevision, keyLengthInBytes int,
	encryptMetadata bool) (bool, error) {
	userPassword, err := h.getUserPassword234(ownerPassword, owner, encRevision, keyLengthInBytes)
	if err != nil {
		return false, err
	}
	return h.isUserPassword234(userPassword, user, owner, permissions, id, encRevision,
		keyLengthInBytes, encryptMetadata)
}

func (h *StandardSecurityHandler) isOwnerPassword56(ownerPassword, user, owner []byte,
	encRevision int) (bool, error) {
	if len(owner) < 40 {
		// PDFBOX-5104
		return false, errors.New("Owner password is too short")
	}
	truncatedOwnerPassword := truncate127(ownerPassword)

	oHash := make([]byte, 32)
	oValidationSalt := make([]byte, 8)
	copy(oHash, owner[:32])
	copy(oValidationSalt, owner[32:40])

	if encRevision == revision5 {
		computed, err := computeSHA256(truncatedOwnerPassword, oValidationSalt, user)
		if err != nil {
			return false, err
		}
		return messageDigestIsEqual(computed, oHash), nil
	}
	computed, err := computeHash2A(truncatedOwnerPassword, oValidationSalt, user)
	if err != nil {
		return false, err
	}
	return messageDigestIsEqual(computed, oHash), nil
}

// GetUserPassword returns the user password based on the owner password.
func (h *StandardSecurityHandler) GetUserPassword(ownerPassword, owner []byte,
	encRevision, length int) ([]byte, error) {
	// TODO ?!?!
	if encRevision == revision5 || encRevision == revision6 {
		return []byte{}, nil
	}
	return h.getUserPassword234(ownerPassword, owner, encRevision, length)
}

func (h *StandardSecurityHandler) getUserPassword234(ownerPassword, owner []byte,
	encRevision, length int) ([]byte, error) {
	var result bytes.Buffer
	rc4Key, err := h.computeRC4key(ownerPassword, encRevision, length)
	if err != nil {
		return nil, err
	}

	if encRevision == revision2 {
		if err := h.encryptDataRC4Bytes(rc4Key, owner, &result); err != nil {
			return nil, err
		}
	} else if encRevision == revision3 || encRevision == revision4 {
		iterationKey := make([]byte, len(rc4Key))
		otemp := make([]byte, len(owner))
		copy(otemp, owner)

		for i := 19; i >= 0; i-- {
			copy(iterationKey, rc4Key)
			for j := 0; j < len(iterationKey); j++ {
				iterationKey[j] = iterationKey[j] ^ byte(i)
			}
			result.Reset()
			if err := h.encryptDataRC4Bytes(iterationKey, otemp, &result); err != nil {
				return nil, err
			}
			otemp = append([]byte(nil), result.Bytes()...)
		}
	}
	return append([]byte(nil), result.Bytes()...), nil
}

// ComputeEncryptedKey computes the encrypted key.
func (h *StandardSecurityHandler) ComputeEncryptedKey(password, o, u, oe, ue []byte,
	permissions int, id []byte, encRevision, keyLengthInBytes int,
	encryptMetadata, isOwnerPassword bool) ([]byte, error) {
	if encRevision == revision5 || encRevision == revision6 {
		return h.computeEncryptedKeyRev56(password, isOwnerPassword, o, u, oe, ue, encRevision)
	}
	return h.computeEncryptedKeyRev234(password, o, permissions, id, encryptMetadata,
		keyLengthInBytes, encRevision), nil
}

func (h *StandardSecurityHandler) computeEncryptedKeyRev234(password, o []byte, permissions int,
	id []byte, encryptMetadata bool, length, encRevision int) []byte {
	// Algorithm 2, based on MD5

	// PDFReference 1.4 pg 78
	padded := truncateOrPad(password)

	md := md5.New()
	md.Write(padded)
	md.Write(o)
	md.Write([]byte{
		byte(permissions),
		byte(uint32(permissions) >> 8),
		byte(uint32(permissions) >> 16),
		byte(uint32(permissions) >> 24),
	})
	md.Write(id)

	// (Security handlers of revision 4 or greater) If document metadata is not
	// being encrypted, pass 4 bytes with the value 0xFFFFFFFF to the MD5 hash
	// function.
	// see 7.6.3.3 Algorithm 2 Step f of PDF 32000-1:2008
	if encRevision == revision4 && !encryptMetadata {
		md.Write([]byte{0xff, 0xff, 0xff, 0xff})
	}
	digest := md.Sum(nil)

	if encRevision == revision3 || encRevision == revision4 {
		for i := 0; i < 50; i++ {
			// Java calls md.update(digest, 0, length) then md.digest(), which
			// resets the digest after each round; a Go hash keeps its state, so
			// a fresh one is made instead.
			md = md5.New()
			md.Write(digest[:length])
			digest = md.Sum(nil)
		}
	}

	result := make([]byte, length)
	copy(result, digest[:length])
	return result
}

func (h *StandardSecurityHandler) computeEncryptedKeyRev56(password []byte, isOwnerPassword bool,
	o, u, oe, ue []byte, encRevision int) ([]byte, error) {
	var hashValue, fileKeyEnc []byte

	if isOwnerPassword {
		if oe == nil {
			return nil, errors.New("/Encrypt/OE entry is missing")
		}
		oKeySalt := make([]byte, 8)
		copy(oKeySalt, o[40:48])

		var err error
		if encRevision == revision5 {
			hashValue, err = computeSHA256(password, oKeySalt, u)
		} else {
			hashValue, err = computeHash2A(password, oKeySalt, u)
		}
		if err != nil {
			return nil, err
		}
		fileKeyEnc = oe
	} else {
		if ue == nil {
			return nil, errors.New("/Encrypt/UE entry is missing")
		}
		uKeySalt := make([]byte, 8)
		copy(uKeySalt, u[40:48])

		var err error
		if encRevision == revision5 {
			hashValue, err = computeSHA256(password, uKeySalt, nil)
		} else {
			hashValue, err = computeHash2A(password, uKeySalt, nil)
		}
		if err != nil {
			return nil, err
		}
		fileKeyEnc = ue
	}
	decrypted, err := aesCBCNoPadding(hashValue, make([]byte, 16), fileKeyEnc, true)
	if err != nil {
		logIfStrongEncryptionMissing()
		return nil, err
	}
	return decrypted, nil
}

// ComputeUserPassword computes the user password hash.
func (h *StandardSecurityHandler) ComputeUserPassword(password, owner []byte, permissions int,
	id []byte, encRevision, keyLengthInBytes int, encryptMetadata bool) ([]byte, error) {
	// TODO!?!?
	if encRevision == revision5 || encRevision == revision6 {
		return []byte{}, nil
	}

	var result bytes.Buffer
	encKey := h.computeEncryptedKeyRev234(password, owner, permissions, id, encryptMetadata,
		keyLengthInBytes, encRevision)

	if encRevision == revision2 {
		if err := h.encryptDataRC4Bytes(encKey, encryptPadding, &result); err != nil {
			return nil, err
		}
	} else if encRevision == revision3 || encRevision == revision4 {
		md := md5.New()
		md.Write(encryptPadding)
		md.Write(id)
		result.Write(md.Sum(nil))

		iterationKey := make([]byte, len(encKey))
		for i := 0; i < 20; i++ {
			copy(iterationKey, encKey)
			for j := 0; j < len(iterationKey); j++ {
				iterationKey[j] = iterationKey[j] ^ byte(i)
			}
			input := append([]byte(nil), result.Bytes()...)
			result.Reset()
			if err := h.encryptDataRC4Bytes(iterationKey, input, &result); err != nil {
				return nil, err
			}
		}

		finalResult := make([]byte, 32)
		copy(finalResult[:16], result.Bytes()[:16])
		copy(finalResult[16:32], encryptPadding[:16])
		result.Reset()
		result.Write(finalResult)
	}
	return append([]byte(nil), result.Bytes()...), nil
}

// ComputeOwnerPassword computes the owner entry in the encryption dictionary.
func (h *StandardSecurityHandler) ComputeOwnerPassword(ownerPassword, userPassword []byte,
	encRevision, length int) ([]byte, error) {
	if revision2 == encRevision && length != 5 {
		return nil, fmt.Errorf("Expected length=5 actual=%d", length)
	}

	rc4Key, err := h.computeRC4key(ownerPassword, encRevision, length)
	if err != nil {
		return nil, err
	}
	paddedUser := truncateOrPad(userPassword)

	var encrypted bytes.Buffer
	if err := h.encryptDataRC4Bytes(rc4Key, paddedUser, &encrypted); err != nil {
		return nil, err
	}

	if encRevision == revision3 || encRevision == revision4 {
		iterationKey := make([]byte, len(rc4Key))
		for i := 1; i < 20; i++ {
			copy(iterationKey, rc4Key)
			for j := 0; j < len(iterationKey); j++ {
				iterationKey[j] = iterationKey[j] ^ byte(i)
			}
			input := append([]byte(nil), encrypted.Bytes()...)
			encrypted.Reset()
			if err := h.encryptDataRC4Bytes(iterationKey, input, &encrypted); err != nil {
				return nil, err
			}
		}
	}

	return append([]byte(nil), encrypted.Bytes()...), nil
}

// computeRC4key is steps (a) to (d) of "Algorithm 3: Computing the encryption
// dictionary's O (owner password) value".
func (h *StandardSecurityHandler) computeRC4key(ownerPassword []byte, encRevision,
	length int) ([]byte, error) {
	// PDFBOX-6115: Java catches IllegalArgumentException here, which an illegal
	// key length raises; the port reports the same case as an error.
	if length < 0 || length > md5.Size {
		return nil, fmt.Errorf("encryption: illegal key length %d", length)
	}
	md := md5.New()
	md.Write(truncateOrPad(ownerPassword))
	digest := md.Sum(nil)

	if encRevision == revision3 || encRevision == revision4 {
		for i := 0; i < 50; i++ {
			// this deviates from the spec - however, omitting the length
			// parameter prevents the file to be opened in Adobe Reader
			// with the owner password when the key length is 40 bit (= 5 bytes)
			md = md5.New()
			md.Write(digest[:length])
			digest = md.Sum(nil)
		}
	}
	rc4Key := make([]byte, length)
	copy(rc4Key, digest[:length])
	return rc4Key, nil
}

// truncateOrPad pads or truncates the password to the 32-byte length the
// specification calls for.
func truncateOrPad(password []byte) []byte {
	padded := make([]byte, len(encryptPadding))
	bytesBeforePad := min(len(password), len(padded))
	copy(padded, password[:bytesBeforePad])
	copy(padded[bytesBeforePad:], encryptPadding[:len(encryptPadding)-bytesBeforePad])
	return padded
}

// IsUserPasswordBytes checks the user password against the user key.
func (h *StandardSecurityHandler) IsUserPasswordBytes(password, user, owner []byte,
	permissions int, id []byte, encRevision, keyLengthInBytes int,
	encryptMetadata bool) (bool, error) {
	switch encRevision {
	case revision2, revision3, revision4:
		return h.isUserPassword234(password, user, owner, permissions, id, encRevision,
			keyLengthInBytes, encryptMetadata)
	case revision5, revision6:
		return h.isUserPassword56(password, user, encRevision)
	default:
		return false, fmt.Errorf("Unknown Encryption Revision %d", encRevision)
	}
}

func (h *StandardSecurityHandler) isUserPassword234(password, user, owner []byte, permissions int,
	id []byte, encRevision, length int, encryptMetadata bool) (bool, error) {
	passwordBytes, err := h.ComputeUserPassword(password, owner, permissions, id, encRevision,
		length, encryptMetadata)
	if err != nil {
		return false, err
	}
	if encRevision == revision2 {
		return messageDigestIsEqual(user, passwordBytes), nil
	}
	// compare first 16 bytes only
	return messageDigestIsEqual(copyOf(user, 16), copyOf(passwordBytes, 16)), nil
}

func (h *StandardSecurityHandler) isUserPassword56(password, user []byte,
	encRevision int) (bool, error) {
	truncatedPassword := truncate127(password)
	uHash := make([]byte, 32)
	uValidationSalt := make([]byte, 8)
	copy(uHash, user[:32])
	copy(uValidationSalt, user[32:40])

	var hashValue []byte
	var err error
	if encRevision == revision5 {
		hashValue, err = computeSHA256(truncatedPassword, uValidationSalt, nil)
	} else {
		hashValue, err = computeHash2A(truncatedPassword, uValidationSalt, nil)
	}
	if err != nil {
		return false, err
	}
	return messageDigestIsEqual(hashValue, uHash), nil
}

// IsUserPassword checks the given text password against the user key.
func (h *StandardSecurityHandler) IsUserPassword(password string, user, owner []byte,
	permissions int, id []byte, encRevision, keyLengthInBytes int,
	encryptMetadata bool) (bool, error) {
	if encRevision == revision5 || encRevision == revision6 {
		return h.IsUserPasswordBytes([]byte(password), user, owner, permissions, id, encRevision,
			keyLengthInBytes, encryptMetadata)
	}
	return h.IsUserPasswordBytes(encodeISO88591(password), user, owner, permissions, id,
		encRevision, keyLengthInBytes, encryptMetadata)
}

// IsOwnerPassword checks the given text password against the owner key.
func (h *StandardSecurityHandler) IsOwnerPassword(password string, user, owner []byte,
	permissions int, id []byte, encRevision, keyLengthInBytes int,
	encryptMetadata bool) (bool, error) {
	return h.IsOwnerPasswordBytes(encodeISO88591(password), user, owner, permissions, id,
		encRevision, keyLengthInBytes, encryptMetadata)
}

// computeHash2A is Algorithm 2.A from ISO 32000-1.
func computeHash2A(password, salt, u []byte) ([]byte, error) {
	userKey, err := adjustUserKey(u)
	if err != nil {
		return nil, err
	}
	truncatedPassword := truncate127(password)
	input := concat3(truncatedPassword, salt, userKey)
	return computeHash2B(input, truncatedPassword, userKey)
}

// computeHash2B is Algorithm 2.B from ISO 32000-2.
func computeHash2B(input, password, userKey []byte) ([]byte, error) {
	sum := sha256.Sum256(input)
	k := sum[:]
	var e []byte

	for round := 0; round < 64 || int(e[len(e)-1]) > round-32; round++ {
		var k1 []byte
		if userKey != nil && len(userKey) >= 48 {
			k1 = make([]byte, 64*(len(password)+len(k)+48))
		} else {
			k1 = make([]byte, 64*(len(password)+len(k)))
		}

		pos := 0
		for i := 0; i < 64; i++ {
			copy(k1[pos:], password)
			pos += len(password)
			copy(k1[pos:], k)
			pos += len(k)
			if userKey != nil && len(userKey) >= 48 {
				copy(k1[pos:], userKey[:48])
				pos += 48
			}
		}

		kFirst := make([]byte, 16)
		kSecond := make([]byte, 16)
		copy(kFirst, k[:16])
		copy(kSecond, k[16:32])

		encrypted, err := aesCBCNoPadding(kFirst, kSecond, k1, false)
		if err != nil {
			logIfStrongEncryptionMissing()
			return nil, err
		}
		e = encrypted

		eFirst := make([]byte, 16)
		copy(eFirst, e[:16])
		bi := new(big.Int).SetBytes(eFirst)
		remainder := new(big.Int).Mod(bi, big.NewInt(3))
		nextHash := hashes2B[remainder.Int64()]()
		nextHash.Write(e)
		k = nextHash.Sum(nil)
	}

	if len(k) > 32 {
		return append([]byte(nil), k[:32]...), nil
	}
	return k, nil
}

func computeSHA256(input, password, userKey []byte) ([]byte, error) {
	adjusted, err := adjustUserKey(userKey)
	if err != nil {
		return nil, err
	}
	md := sha256.New()
	md.Write(input)
	md.Write(password)
	md.Write(adjusted)
	return md.Sum(nil), nil
}

func adjustUserKey(u []byte) ([]byte, error) {
	if u == nil {
		return []byte{}, nil
	}
	if len(u) < 48 {
		return nil, errors.New("Bad U length")
	}
	if len(u) > 48 {
		// must truncate
		userKey := make([]byte, 48)
		copy(userKey, u[:48])
		return userKey, nil
	}
	return u, nil
}

func concat(a, b []byte) []byte {
	o := make([]byte, len(a)+len(b))
	copy(o, a)
	copy(o[len(a):], b)
	return o
}

func concat3(a, b, c []byte) []byte {
	o := make([]byte, len(a)+len(b)+len(c))
	copy(o, a)
	copy(o[len(a):], b)
	copy(o[len(a)+len(b):], c)
	return o
}

func truncate127(in []byte) []byte {
	if len(in) <= 127 {
		return in
	}
	trunc := make([]byte, 127)
	copy(trunc, in[:127])
	return trunc
}

// logIfStrongEncryptionMissing warns where the JCE unlimited strength policy
// files are missing.
//
// Go has no key length policy, so there is nothing to check and nothing to
// warn about; the call sites keep it so that the two read the same.
func logIfStrongEncryptionMissing() {}

// messageDigestIsEqual is java.security.MessageDigest.isEqual, which compares in
// constant time and reports false for differing lengths.
func messageDigestIsEqual(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

// aesCBCNoPadding runs AES in CBC mode with no padding, which is what the JCE
// "AES/CBC/NoPadding" transformation does.
func aesCBCNoPadding(key, iv, data []byte, decrypt bool) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(data)%aes.BlockSize != 0 {
		// javax.crypto.IllegalBlockSizeException
		return nil, fmt.Errorf(
			"encryption: input length not a multiple of the block size: %d", len(data))
	}
	out := make([]byte, len(data))
	if decrypt {
		cipher.NewCBCDecrypter(block, iv).CryptBlocks(out, data)
	} else {
		cipher.NewCBCEncrypter(block, iv).CryptBlocks(out, data)
	}
	return out, nil
}

// encodeISO88591 is String.getBytes(ISO_8859_1), which keeps the low byte of
// each UTF-16 unit and writes '?' for anything above U+00FF.
func encodeISO88591(s string) []byte {
	out := make([]byte, 0, len(s))
	for _, r := range s {
		if r > 0xFF {
			out = append(out, '?')
		} else {
			out = append(out, byte(r))
		}
	}
	return out
}

// randomReader is the source prepareEncryptionDictRev6 draws from when no
// custom one was set, which is Java's static SecureRandom.
var randomReader io.Reader = rand.Reader

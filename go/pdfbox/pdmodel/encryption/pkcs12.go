package encryption

import (
	"crypto"
	"crypto/cipher"
	"crypto/des"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/x509"
	"encoding/asn1"
	"errors"
	"fmt"
	"io"
	"unicode/utf16"
)

// A PKCS#12 keystore reader.
//
// Java's KeyStore.getInstance("PKCS12") reads these; Go's standard library has
// no equivalent, and PDFBox has no code of its own to port here. This file is
// therefore infrastructure the port supplies rather than a migration of
// anything -- the same call the slice 9 rasteriser decision makes -- and it
// covers the algorithms the checked-in test keystores use: the SHA-1 based
// PKCS#12 derivation with 3DES for the shrouded key bags and 40-bit RC2 for the
// certificate bags. See migration/STATUS.md.

// The object identifiers a PKCS#12 file carries.
var (
	oidPKCS7Data             = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 1}
	oidPKCS7EncryptedData    = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 6}
	oidPBEWithSHAAnd3KeyDES  = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 12, 1, 3}
	oidPBEWithSHAAnd40BitRC2 = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 12, 1, 6}
	oidFriendlyName          = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 20}
	oidLocalKeyID            = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 21}
	oidCertTypeX509          = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 22, 1}
)

// The three purposes the PKCS#12 key derivation is run for.
const (
	pkcs12IDKey = 1
	pkcs12IDIV  = 2
	pkcs12IDMAC = 3
)

type pfx struct {
	Version  int
	AuthSafe contentInfo
	MacData  asn1.RawValue `asn1:"optional"`
}

type macData struct {
	Mac        digestInfo
	MacSalt    []byte
	Iterations int `asn1:"optional,default:1"`
}

type digestInfo struct {
	Algorithm algorithmIdentifier
	Digest    []byte
}

type encryptedData struct {
	Version              int
	EncryptedContentInfo encryptedContentInfo
}

type safeBag struct {
	ID         asn1.ObjectIdentifier
	Value      asn1.RawValue     `asn1:"explicit,tag:0"`
	Attributes []pkcs12Attribute `asn1:"set,optional"`
}

type pkcs12Attribute struct {
	ID     asn1.ObjectIdentifier
	Values asn1.RawValue `asn1:"set"`
}

type certBag struct {
	ID    asn1.ObjectIdentifier
	Value asn1.RawValue `asn1:"explicit,tag:0"`
}

type pbeParameter struct {
	Salt       []byte
	Iterations int
}

// pkcs12Entry is one alias of a keystore.
type pkcs12Entry struct {
	alias       string
	certificate *x509.Certificate
	privateKey  crypto.PrivateKey
}

// pkcs12KeyStore is a KeyStore read from a PKCS#12 file.
type pkcs12KeyStore struct {
	entries []*pkcs12Entry
}

var _ KeyStore = (*pkcs12KeyStore)(nil)

// LoadPKCS12 reads a PKCS#12 keystore, verifying its MAC against the given
// password.
//
// It stands for Java's KeyStore.getInstance("PKCS12") followed by load.
func LoadPKCS12(input io.Reader, password string) (KeyStore, error) {
	der, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}
	var file pfx
	if _, err := asn1.Unmarshal(der, &file); err != nil {
		return nil, fmt.Errorf("encryption: reading the PKCS#12 file: %w", err)
	}
	if !file.AuthSafe.ContentType.Equal(oidPKCS7Data) {
		return nil, fmt.Errorf("encryption: the PKCS#12 authenticated safe is %v",
			file.AuthSafe.ContentType)
	}
	var authSafeBytes []byte
	if _, err := asn1.Unmarshal(file.AuthSafe.Content.Bytes, &authSafeBytes); err != nil {
		return nil, fmt.Errorf("encryption: reading the PKCS#12 authenticated safe: %w", err)
	}

	bmpPassword := bmpString(password)

	if file.MacData.FullBytes != nil {
		var mac macData
		if _, err := asn1.Unmarshal(file.MacData.FullBytes, &mac); err != nil {
			return nil, fmt.Errorf("encryption: reading the PKCS#12 MAC: %w", err)
		}
		iterations := mac.Iterations
		if iterations == 0 {
			iterations = 1
		}
		key := pkcs12Derive(bmpPassword, mac.MacSalt, pkcs12IDMAC, iterations, sha1.Size)
		expected := hmac.New(sha1.New, key)
		expected.Write(authSafeBytes)
		if !hmac.Equal(expected.Sum(nil), mac.Mac.Digest) {
			return nil, errors.New(
				"encryption: PKCS#12 MAC verification failed, the password is probably wrong")
		}
	}

	var contents []contentInfo
	if _, err := asn1.Unmarshal(authSafeBytes, &contents); err != nil {
		return nil, fmt.Errorf("encryption: reading the PKCS#12 safe contents: %w", err)
	}

	store := &pkcs12KeyStore{}
	for _, content := range contents {
		var safeContents []byte
		switch {
		case content.ContentType.Equal(oidPKCS7Data):
			if _, err := asn1.Unmarshal(content.Content.Bytes, &safeContents); err != nil {
				return nil, fmt.Errorf("encryption: reading a PKCS#12 safe: %w", err)
			}
		case content.ContentType.Equal(oidPKCS7EncryptedData):
			var encrypted encryptedData
			if _, err := asn1.Unmarshal(content.Content.Bytes, &encrypted); err != nil {
				return nil, fmt.Errorf("encryption: reading an encrypted PKCS#12 safe: %w", err)
			}
			safeContents, err = pkcs12Decrypt(
				encrypted.EncryptedContentInfo.ContentEncryptionAlgorithm,
				encrypted.EncryptedContentInfo.EncryptedContent, bmpPassword)
			if err != nil {
				return nil, err
			}
		default:
			continue
		}
		if err := store.readSafeContents(safeContents, bmpPassword); err != nil {
			return nil, err
		}
	}
	store.pairKeysWithCertificates()
	return store, nil
}

// readSafeContents reads the bags of one safe into the store.
func (s *pkcs12KeyStore) readSafeContents(der []byte, bmpPassword []byte) error {
	var bags []safeBag
	if _, err := asn1.Unmarshal(der, &bags); err != nil {
		return fmt.Errorf("encryption: reading the PKCS#12 bags: %w", err)
	}
	for i := range bags {
		bag := &bags[i]
		alias, localKeyID := bagAttributes(bag)
		switch {
		case bag.ID.Equal(oidPKCS12CertBag):
			var cert certBag
			if _, err := asn1.Unmarshal(bag.Value.Bytes, &cert); err != nil {
				return fmt.Errorf("encryption: reading a PKCS#12 certificate bag: %w", err)
			}
			if !cert.ID.Equal(oidCertTypeX509) {
				continue
			}
			var certDER []byte
			if _, err := asn1.Unmarshal(cert.Value.Bytes, &certDER); err != nil {
				return fmt.Errorf("encryption: reading a PKCS#12 certificate: %w", err)
			}
			parsed, err := x509.ParseCertificate(certDER)
			if err != nil {
				return fmt.Errorf("encryption: parsing a PKCS#12 certificate: %w", err)
			}
			s.entries = append(s.entries, &pkcs12Entry{
				alias:       aliasOr(alias, localKeyID),
				certificate: parsed,
			})
		case bag.ID.Equal(oidPKCS12KeyBag):
			key, err := x509.ParsePKCS8PrivateKey(bag.Value.Bytes)
			if err != nil {
				return fmt.Errorf("encryption: reading a PKCS#12 key bag: %w", err)
			}
			s.entries = append(s.entries, &pkcs12Entry{
				alias:      aliasOr(alias, localKeyID),
				privateKey: key,
			})
		case bag.ID.Equal(oidPKCS12ShroudedKB):
			var encrypted struct {
				Algorithm algorithmIdentifier
				Data      []byte
			}
			if _, err := asn1.Unmarshal(bag.Value.FullBytes, &encrypted); err != nil {
				return fmt.Errorf("encryption: reading a PKCS#12 shrouded key bag: %w", err)
			}
			pkcs8, err := pkcs12Decrypt(encrypted.Algorithm, encrypted.Data, bmpPassword)
			if err != nil {
				return err
			}
			key, err := x509.ParsePKCS8PrivateKey(pkcs8)
			if err != nil {
				return fmt.Errorf("encryption: reading a PKCS#12 private key: %w", err)
			}
			s.entries = append(s.entries, &pkcs12Entry{
				alias:      aliasOr(alias, localKeyID),
				privateKey: key,
			})
		}
	}
	return nil
}

// pairKeysWithCertificates folds a key entry and a certificate entry that share
// an alias into one, which is the shape Java's KeyStore hands back.
func (s *pkcs12KeyStore) pairKeysWithCertificates() {
	byAlias := map[string]*pkcs12Entry{}
	var merged []*pkcs12Entry
	for _, entry := range s.entries {
		if existing, present := byAlias[entry.alias]; present {
			if entry.certificate != nil {
				existing.certificate = entry.certificate
			}
			if entry.privateKey != nil {
				existing.privateKey = entry.privateKey
			}
			continue
		}
		byAlias[entry.alias] = entry
		merged = append(merged, entry)
	}
	s.entries = merged
}

// bagAttributes returns the friendly name and the local key id of a bag.
func bagAttributes(bag *safeBag) (friendlyName string, localKeyID []byte) {
	for _, attribute := range bag.Attributes {
		switch {
		case attribute.ID.Equal(oidFriendlyName):
			var name asn1.RawValue
			if _, err := asn1.Unmarshal(attribute.Values.Bytes, &name); err == nil {
				friendlyName = decodeBMPString(name.Bytes)
			}
		case attribute.ID.Equal(oidLocalKeyID):
			var id []byte
			if _, err := asn1.Unmarshal(attribute.Values.Bytes, &id); err == nil {
				localKeyID = id
			}
		}
	}
	return friendlyName, localKeyID
}

func aliasOr(friendlyName string, localKeyID []byte) string {
	if friendlyName != "" {
		return friendlyName
	}
	return fmt.Sprintf("%x", localKeyID)
}

// pkcs12Decrypt decrypts a PKCS#12 blob with one of the SHA-1 based password
// based encryption schemes.
func pkcs12Decrypt(algorithm algorithmIdentifier, data []byte, bmpPassword []byte) ([]byte, error) {
	var params pbeParameter
	if _, err := asn1.Unmarshal(algorithm.Parameters.FullBytes, &params); err != nil {
		return nil, fmt.Errorf("encryption: reading the PKCS#12 PBE parameters: %w", err)
	}

	var block cipher.Block
	var err error
	switch {
	case algorithm.Algorithm.Equal(oidPBEWithSHAAnd3KeyDES):
		key := pkcs12Derive(bmpPassword, params.Salt, pkcs12IDKey, params.Iterations, 24)
		block, err = des.NewTripleDESCipher(key)
	case algorithm.Algorithm.Equal(oidPBEWithSHAAnd40BitRC2):
		key := pkcs12Derive(bmpPassword, params.Salt, pkcs12IDKey, params.Iterations, 5)
		block, err = newRC2Cipher(key, 40)
	default:
		return nil, fmt.Errorf("encryption: PKCS#12 algorithm %v is not supported",
			algorithm.Algorithm)
	}
	if err != nil {
		return nil, err
	}

	iv := pkcs12Derive(bmpPassword, params.Salt, pkcs12IDIV, params.Iterations, block.BlockSize())
	if len(data) == 0 || len(data)%block.BlockSize() != 0 {
		return nil, fmt.Errorf("encryption: the PKCS#12 blob is %d bytes", len(data))
	}
	plain := make([]byte, len(data))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plain, data)
	return pkcs7Unpad(plain, block.BlockSize())
}

// pkcs12Derive is the key derivation of RFC 7292 appendix B.2, with SHA-1.
func pkcs12Derive(bmpPassword, salt []byte, id byte, iterations, size int) []byte {
	const u = sha1.Size // hash output
	const v = 64        // hash block size

	d := make([]byte, v)
	for i := range d {
		d[i] = id
	}

	s := fillRepeating(salt, v)
	p := fillRepeating(bmpPassword, v)
	i := append(append([]byte{}, s...), p...)

	c := (size + u - 1) / u
	out := make([]byte, 0, c*u)

	for round := 0; round < c; round++ {
		hashed := sha1.Sum(append(append([]byte{}, d...), i...))
		a := hashed[:]
		for j := 1; j < iterations; j++ {
			next := sha1.Sum(a)
			a = next[:]
		}
		out = append(out, a...)
		if round == c-1 {
			break
		}

		b := fillRepeating(a, v)
		for j := 0; j < len(i); j += v {
			addBigEndian(i[j:j+v], b)
		}
	}
	return out[:size]
}

// fillRepeating repeats src until it fills a whole number of blocks of the
// given size, which is what the derivation does to the salt and the password.
func fillRepeating(src []byte, blockSize int) []byte {
	if len(src) == 0 {
		return []byte{}
	}
	length := ((len(src) + blockSize - 1) / blockSize) * blockSize
	out := make([]byte, length)
	for i := range out {
		out[i] = src[i%len(src)]
	}
	return out
}

// addBigEndian adds b to a in place, treating both as big-endian integers of
// the same length and dropping the carry out of the top, then adds one.
func addBigEndian(a, b []byte) {
	carry := 1
	for i := len(a) - 1; i >= 0; i-- {
		sum := int(a[i]) + int(b[i]) + carry
		a[i] = byte(sum)
		carry = sum >> 8
	}
}

// bmpString is Java's char[] password as PKCS#12 wants it: UTF-16BE with two
// terminating zero bytes.
func bmpString(password string) []byte {
	units := utf16.Encode([]rune(password))
	out := make([]byte, 0, len(units)*2+2)
	for _, unit := range units {
		out = append(out, byte(unit>>8), byte(unit))
	}
	return append(out, 0, 0)
}

// decodeBMPString reads a BMPString back into a Go string.
func decodeBMPString(b []byte) string {
	units := make([]uint16, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		units = append(units, uint16(b[i])<<8|uint16(b[i+1]))
	}
	return string(utf16.Decode(units))
}

// Size returns how many entries the store holds.
func (s *pkcs12KeyStore) Size() int { return len(s.entries) }

// Aliases returns the names of the entries, in the order the file lists them.
func (s *pkcs12KeyStore) Aliases() []string {
	aliases := make([]string, 0, len(s.entries))
	for _, entry := range s.entries {
		aliases = append(aliases, entry.alias)
	}
	return aliases
}

// Certificate returns the certificate of the named entry, or nil.
func (s *pkcs12KeyStore) Certificate(alias string) *x509.Certificate {
	if entry := s.entry(alias); entry != nil {
		return entry.certificate
	}
	return nil
}

// Key returns the private key of the named entry.
func (s *pkcs12KeyStore) Key(alias string, password string) (crypto.PrivateKey, error) {
	entry := s.entry(alias)
	if entry == nil {
		return nil, fmt.Errorf("encryption: the keystore has no entry named %q", alias)
	}
	if entry.privateKey == nil {
		return nil, fmt.Errorf("encryption: the keystore entry %q has no private key", alias)
	}
	return entry.privateKey, nil
}

// ContainsAlias reports whether the store holds an entry of that name.
func (s *pkcs12KeyStore) ContainsAlias(alias string) bool { return s.entry(alias) != nil }

func (s *pkcs12KeyStore) entry(alias string) *pkcs12Entry {
	for _, entry := range s.entries {
		if entry.alias == alias {
			return entry
		}
	}
	return nil
}

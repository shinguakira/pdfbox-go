package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"crypto/pbkdf2"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"fmt"
	"hash"
	"os"
	"unicode/utf16"
)

// A certificate line of a passwords table names a certificate and a private key,
// and neither loader takes those: PDFBox's takes a PKCS#12 keystore, and so does
// the port's. JavaCorpus builds the keystore with the JDK. keyStoreFor builds
// the same one-entry store for the port, from the same two files, in the one
// shape its PKCS#12 reader needs: a plain key bag and a certificate bag in an
// unencrypted safe, under one friendly name, with no MAC. That the store is
// built differently on the two sides does not reach the comparison, which is of
// what each side does with the document once it holds the key.

var (
	oidData         = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 1}
	oidKeyBag       = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 12, 10, 1, 1}
	oidCertBag      = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 12, 10, 1, 3}
	oidX509Cert     = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 22, 1}
	oidFriendlyName = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 20}
	oidLocalKeyID   = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 21}

	oidPBES2      = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 5, 13}
	oidPBKDF2     = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 5, 12}
	oidHMACSHA1   = asn1.ObjectIdentifier{1, 2, 840, 113549, 2, 7}
	oidHMACSHA256 = asn1.ObjectIdentifier{1, 2, 840, 113549, 2, 9}
	oidHMACSHA384 = asn1.ObjectIdentifier{1, 2, 840, 113549, 2, 10}
	oidHMACSHA512 = asn1.ObjectIdentifier{1, 2, 840, 113549, 2, 11}
	oidDESEDE3CBC = asn1.ObjectIdentifier{1, 2, 840, 113549, 3, 7}
	oidAES128CBC  = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 1, 2}
	oidAES192CBC  = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 1, 22}
	oidAES256CBC  = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 1, 42}
)

type pfxOut struct {
	Version  int
	AuthSafe contentInfoOut
}

type contentInfoOut struct {
	ContentType asn1.ObjectIdentifier
	Content     []byte `asn1:"explicit,tag:0"`
}

type safeBagOut struct {
	ID         asn1.ObjectIdentifier
	Value      asn1.RawValue
	Attributes []attributeOut `asn1:"set"`
}

type attributeOut struct {
	ID     asn1.ObjectIdentifier
	Values asn1.RawValue
}

type certBagOut struct {
	ID    asn1.ObjectIdentifier
	Value []byte `asn1:"explicit,tag:0"`
}

// keyStoreFor builds the PKCS#12 keystore a certificate line describes. The
// password decrypts the private key when its PEM is encrypted; the store itself
// carries no MAC, so nothing else reads it.
func keyStoreFor(o opening) ([]byte, error) {
	certificate, err := certificateDER(o.certificate)
	if err != nil {
		return nil, err
	}
	key, err := privateKeyDER(o.key, o.password)
	if err != nil {
		return nil, err
	}

	name := utf16.Encode([]rune("key"))
	bmp := make([]byte, 0, 2*len(name))
	for _, unit := range name {
		bmp = append(bmp, byte(unit>>8), byte(unit))
	}
	friendlyName, err := asn1.Marshal(asn1.RawValue{Tag: asn1.TagBMPString, Bytes: bmp})
	if err != nil {
		return nil, err
	}
	localKeyID, err := asn1.Marshal([]byte{1})
	if err != nil {
		return nil, err
	}
	attributes := []attributeOut{
		{ID: oidFriendlyName, Values: asn1.RawValue{Tag: asn1.TagSet, IsCompound: true, Bytes: friendlyName}},
		{ID: oidLocalKeyID, Values: asn1.RawValue{Tag: asn1.TagSet, IsCompound: true, Bytes: localKeyID}},
	}

	certBag, err := asn1.Marshal(certBagOut{ID: oidX509Cert, Value: certificate})
	if err != nil {
		return nil, err
	}
	explicit := func(body []byte) asn1.RawValue {
		return asn1.RawValue{Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true, Bytes: body}
	}
	safe, err := asn1.Marshal([]safeBagOut{
		{ID: oidKeyBag, Value: explicit(key), Attributes: attributes},
		{ID: oidCertBag, Value: explicit(certBag), Attributes: attributes},
	})
	if err != nil {
		return nil, err
	}
	authSafe, err := asn1.Marshal([]contentInfoOut{{ContentType: oidData, Content: safe}})
	if err != nil {
		return nil, err
	}
	return asn1.Marshal(pfxOut{Version: 3, AuthSafe: contentInfoOut{ContentType: oidData, Content: authSafe}})
}

// certificateDER answers the first certificate of a PEM file, or the file itself
// when it is not PEM.
func certificateDER(path string) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if block, _ := pem.Decode(raw); block == nil {
		if _, err := x509.ParseCertificate(raw); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		return raw, nil
	}
	for rest := raw; ; {
		var block *pem.Block
		if block, rest = pem.Decode(rest); block == nil {
			return nil, fmt.Errorf("%s: no certificate in it", path)
		}
		if block.Type == "CERTIFICATE" {
			return block.Bytes, nil
		}
	}
}

// privateKeyDER answers the first private key of a PEM file as PKCS#8,
// decrypting it with the password when it is encrypted, or the file itself
// when it is not PEM.
func privateKeyDER(path, password string) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if block, _ := pem.Decode(raw); block == nil {
		return raw, nil
	}
	for rest := raw; ; {
		var block *pem.Block
		if block, rest = pem.Decode(rest); block == nil {
			return nil, fmt.Errorf("%s: no private key in it", path)
		}
		switch block.Type {
		case "PRIVATE KEY":
			return block.Bytes, nil
		case "ENCRYPTED PRIVATE KEY":
			key, err := decryptPKCS8(block.Bytes, password)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			return key, nil
		case "RSA PRIVATE KEY":
			key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			return x509.MarshalPKCS8PrivateKey(key)
		case "EC PRIVATE KEY":
			key, err := x509.ParseECPrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			return x509.MarshalPKCS8PrivateKey(key)
		}
	}
}

type encryptedPrivateKeyInfo struct {
	Algorithm pkix.AlgorithmIdentifier
	Data      []byte
}

type pbes2Parameters struct {
	KeyDerivationFunc pkix.AlgorithmIdentifier
	EncryptionScheme  pkix.AlgorithmIdentifier
}

type pbkdf2Parameters struct {
	Salt           []byte
	IterationCount int
	KeyLength      int                      `asn1:"optional"`
	PRF            pkix.AlgorithmIdentifier `asn1:"optional"`
}

// decryptPKCS8 decrypts an EncryptedPrivateKeyInfo protected by PBES2 with
// PBKDF2, the scheme of every encrypted key in the corpora: RFC 8018, with the
// HMAC-SHA family as the pseudorandom function and AES or triple DES in CBC mode.
func decryptPKCS8(der []byte, password string) ([]byte, error) {
	var info encryptedPrivateKeyInfo
	if _, err := asn1.Unmarshal(der, &info); err != nil {
		return nil, fmt.Errorf("reading the encrypted private key: %w", err)
	}
	if !info.Algorithm.Algorithm.Equal(oidPBES2) {
		return nil, fmt.Errorf("private key encrypted with %v, not PBES2", info.Algorithm.Algorithm)
	}
	var scheme pbes2Parameters
	if _, err := asn1.Unmarshal(info.Algorithm.Parameters.FullBytes, &scheme); err != nil {
		return nil, fmt.Errorf("reading the PBES2 parameters: %w", err)
	}
	if !scheme.KeyDerivationFunc.Algorithm.Equal(oidPBKDF2) {
		return nil, fmt.Errorf("PBES2 key derivation %v, not PBKDF2", scheme.KeyDerivationFunc.Algorithm)
	}
	var kdf pbkdf2Parameters
	if _, err := asn1.Unmarshal(scheme.KeyDerivationFunc.Parameters.FullBytes, &kdf); err != nil {
		return nil, fmt.Errorf("reading the PBKDF2 parameters: %w", err)
	}

	var prf func() hash.Hash
	switch algorithm := kdf.PRF.Algorithm; {
	case len(algorithm) == 0, algorithm.Equal(oidHMACSHA1):
		prf = sha1.New
	case algorithm.Equal(oidHMACSHA256):
		prf = sha256.New
	case algorithm.Equal(oidHMACSHA384):
		prf = sha512.New384
	case algorithm.Equal(oidHMACSHA512):
		prf = sha512.New
	default:
		return nil, fmt.Errorf("PBKDF2 pseudorandom function %v", algorithm)
	}

	var keyLength int
	var newCipher func([]byte) (cipher.Block, error)
	switch algorithm := scheme.EncryptionScheme.Algorithm; {
	case algorithm.Equal(oidAES128CBC):
		keyLength, newCipher = 16, aes.NewCipher
	case algorithm.Equal(oidAES192CBC):
		keyLength, newCipher = 24, aes.NewCipher
	case algorithm.Equal(oidAES256CBC):
		keyLength, newCipher = 32, aes.NewCipher
	case algorithm.Equal(oidDESEDE3CBC):
		keyLength, newCipher = 24, des.NewTripleDESCipher
	default:
		return nil, fmt.Errorf("PBES2 encryption scheme %v", algorithm)
	}
	var iv []byte
	if _, err := asn1.Unmarshal(scheme.EncryptionScheme.Parameters.FullBytes, &iv); err != nil {
		return nil, fmt.Errorf("reading the PBES2 initialisation vector: %w", err)
	}

	key, err := pbkdf2.Key(prf, password, kdf.Salt, kdf.IterationCount, keyLength)
	if err != nil {
		return nil, err
	}
	block, err := newCipher(key)
	if err != nil {
		return nil, err
	}
	if len(iv) != block.BlockSize() || len(info.Data) == 0 || len(info.Data)%block.BlockSize() != 0 {
		return nil, errors.New("the encrypted private key is not whole blocks")
	}
	plain := make([]byte, len(info.Data))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plain, info.Data)
	padding := int(plain[len(plain)-1])
	if padding == 0 || padding > block.BlockSize() {
		return nil, errors.New("the private key did not decrypt; is the password right?")
	}
	for _, b := range plain[len(plain)-padding:] {
		if int(b) != padding {
			return nil, errors.New("the private key did not decrypt; is the password right?")
		}
	}
	return plain[:len(plain)-padding], nil
}

// Package keystore encrypts and decrypts secp256k1 private keys at rest using a
// password, in the Web3 Secret Storage (keystore v3) format. This is wire
// compatible with go-ethereum/go-stablenet keystore files, so keys produced by
// `gstable account` (and chainbench preset keys) can be read, and keys produced
// here can be imported by a node.
//
// Format: scrypt|pbkdf2 KDF -> AES-128-CTR cipher -> Keccak-256 MAC.
// Spec: docs/adr/ADR-0003.
package keystore

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/0xmhha/accounts/crypto"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"
)

// Default scrypt parameters (match go-ethereum "standard" strength).
const (
	StandardScryptN = 1 << 18
	StandardScryptP = 1
	// LightScryptN/P are cheaper params for tests and low-value keys.
	LightScryptN = 1 << 12
	LightScryptP = 6

	scryptR     = 8
	scryptDKLen = 32
	keyLen      = 32 // secp256k1 private key length
	aesKeyLen   = 16 // AES-128
)

type cipherparamsJSON struct {
	IV string `json:"iv"`
}

type cryptoJSON struct {
	Cipher       string                 `json:"cipher"`
	CipherText   string                 `json:"ciphertext"`
	CipherParams cipherparamsJSON       `json:"cipherparams"`
	KDF          string                 `json:"kdf"`
	KDFParams    map[string]interface{} `json:"kdfparams"`
	MAC          string                 `json:"mac"`
}

type encryptedKeyJSON struct {
	Address string     `json:"address"`
	Crypto  cryptoJSON `json:"crypto"`
	ID      string     `json:"id"`
	Version int        `json:"version"`
}

// Encrypt encrypts a 32-byte private key with the password using scrypt.
// n/p are the scrypt cost parameters (use StandardScryptN/P for production,
// LightScryptN/P for tests). The returned bytes are keystore-v3 JSON.
func Encrypt(privKey []byte, password string, n, p int) ([]byte, error) {
	if len(privKey) != keyLen {
		return nil, fmt.Errorf("private key must be %d bytes", keyLen)
	}
	salt := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	derived, err := scrypt.Key([]byte(password), salt, n, scryptR, p, scryptDKLen)
	if err != nil {
		return nil, err
	}
	encKey := derived[:aesKeyLen]

	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	cipherText, err := aesCTR(encKey, iv, privKey)
	if err != nil {
		return nil, err
	}
	mac := crypto.Keccak256(derived[16:32], cipherText)

	priv, err := crypto.PrivKeyFromBytes(privKey)
	if err != nil {
		return nil, err
	}
	addr := crypto.PrivKeyToAddress(priv)

	doc := encryptedKeyJSON{
		Address: hex.EncodeToString(addr.Bytes()),
		Crypto: cryptoJSON{
			Cipher:       "aes-128-ctr",
			CipherText:   hex.EncodeToString(cipherText),
			CipherParams: cipherparamsJSON{IV: hex.EncodeToString(iv)},
			KDF:          "scrypt",
			KDFParams: map[string]interface{}{
				"n": n, "r": scryptR, "p": p, "dklen": scryptDKLen,
				"salt": hex.EncodeToString(salt),
			},
			MAC: hex.EncodeToString(mac),
		},
		ID:      newUUID(),
		Version: 3,
	}
	return json.MarshalIndent(doc, "", "  ")
}

// Decrypt decrypts a keystore-v3 JSON document with the password and returns the
// 32-byte private key. Supports scrypt and pbkdf2 KDFs and aes-128-ctr.
func Decrypt(keyjson []byte, password string) ([]byte, error) {
	var doc encryptedKeyJSON
	if err := json.Unmarshal(keyjson, &doc); err != nil {
		return nil, fmt.Errorf("invalid keystore json: %w", err)
	}
	if doc.Version != 3 {
		return nil, fmt.Errorf("unsupported keystore version %d", doc.Version)
	}
	c := doc.Crypto
	if c.Cipher != "aes-128-ctr" {
		return nil, fmt.Errorf("unsupported cipher %q", c.Cipher)
	}
	cipherText, err := hex.DecodeString(c.CipherText)
	if err != nil {
		return nil, fmt.Errorf("bad ciphertext: %w", err)
	}
	iv, err := hex.DecodeString(c.CipherParams.IV)
	if err != nil {
		return nil, fmt.Errorf("bad iv: %w", err)
	}
	mac, err := hex.DecodeString(c.MAC)
	if err != nil {
		return nil, fmt.Errorf("bad mac: %w", err)
	}

	derived, err := deriveKey(c.KDF, c.KDFParams, password)
	if err != nil {
		return nil, err
	}
	calcMAC := crypto.Keccak256(derived[16:32], cipherText)
	if subtle.ConstantTimeCompare(calcMAC, mac) != 1 {
		return nil, errors.New("could not decrypt key with given password (MAC mismatch)")
	}
	key, err := aesCTR(derived[:aesKeyLen], iv, cipherText)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func deriveKey(kdf string, params map[string]interface{}, password string) ([]byte, error) {
	salt, err := hex.DecodeString(getString(params, "salt"))
	if err != nil {
		return nil, fmt.Errorf("bad salt: %w", err)
	}
	dklen := getInt(params, "dklen", scryptDKLen)
	switch kdf {
	case "scrypt":
		n := getInt(params, "n", 0)
		r := getInt(params, "r", scryptR)
		p := getInt(params, "p", 0)
		return scrypt.Key([]byte(password), salt, n, r, p, dklen)
	case "pbkdf2":
		c := getInt(params, "c", 0)
		if prf := getString(params, "prf"); prf != "" && prf != "hmac-sha256" {
			return nil, fmt.Errorf("unsupported pbkdf2 prf %q", prf)
		}
		return pbkdf2.Key([]byte(password), salt, c, dklen, sha256.New), nil
	default:
		return nil, fmt.Errorf("unsupported kdf %q", kdf)
	}
}

// aesCTR runs AES-128 in CTR mode over data (symmetric: encrypt == decrypt).
func aesCTR(key, iv, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	out := make([]byte, len(data))
	cipher.NewCTR(block, iv).XORKeyStream(out, data)
	return out, nil
}

func getString(m map[string]interface{}, k string) string {
	if v, ok := m[k].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]interface{}, k string, def int) int {
	switch v := m[k].(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return def
	}
}

// newUUID returns a random RFC-4122 v4 UUID string.
func newUUID() string {
	var u [16]byte
	_, _ = io.ReadFull(rand.Reader, u[:])
	u[6] = (u[6] & 0x0f) | 0x40
	u[8] = (u[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", u[0:4], u[4:6], u[6:8], u[8:10], u[10:16])
}

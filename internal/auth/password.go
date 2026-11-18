package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonMemory      = 19 * 1024
	argonIterations  = 2
	argonParallelism = 1
	saltLength       = 16
	keyLength        = 32
)

var ErrInvalidHash = errors.New("hash de senha em formato invalido")

var hashPrefix = fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$",
	argon2.Version, argonMemory, argonIterations, argonParallelism)

func deriveKey(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, argonIterations, argonMemory, argonParallelism, keyLength)
}

func HashPassword(password string) string {
	salt := make([]byte, saltLength)
	rand.Read(salt)
	key := deriveKey(password, salt)
	return hashPrefix + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(key)
}

func VerifyPassword(password, encoded string) (bool, error) {
	rest, ok := strings.CutPrefix(encoded, hashPrefix)
	if !ok {
		return false, ErrInvalidHash
	}
	saltText, keyText, ok := strings.Cut(rest, "$")
	if !ok {
		return false, ErrInvalidHash
	}
	salt, err := base64.RawStdEncoding.DecodeString(saltText)
	if err != nil || len(salt) != saltLength {
		return false, ErrInvalidHash
	}
	key, err := base64.RawStdEncoding.DecodeString(keyText)
	if err != nil || len(key) != keyLength {
		return false, ErrInvalidHash
	}
	return subtle.ConstantTimeCompare(deriveKey(password, salt), key) == 1, nil
}

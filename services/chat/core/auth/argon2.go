package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"strings"

	"golang.org/x/crypto/argon2"
)

func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	rand.Read(salt)

	hash := argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, 32)
	return base64.RawStdEncoding.EncodeToString(salt) + "." + base64.RawStdEncoding.EncodeToString(hash), nil
}

func VerifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, ".")
	salt, _ := base64.RawStdEncoding.DecodeString(parts[0])
	hash, _ := base64.RawStdEncoding.DecodeString(parts[1])

	testHash := argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, 32)

	return subtle.ConstantTimeCompare(hash, testHash) == 1
}

package encryption

import (
	"golang.org/x/crypto/bcrypt"
)

func Hashing(pass string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
}

func CmpHashAndPass(hash, pass string) bool {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pass)); err != nil {
		return false
	}
	return true
}

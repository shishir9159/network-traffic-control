package encryption

import (
	"golang.org/x/crypto/bcrypt"
	"log"
)

func GetStringInBytes(value string) []byte {
	return []byte(value)
}

func Encrypt(value string) string {
	valueInBytes := []byte(value)
	hash, err := bcrypt.GenerateFromPassword(valueInBytes, bcrypt.MinCost)
	if err != nil {
		log.Println(err)
	}
	return string(hash)
}

func CompareHashedVsOriginalValue(hashedPwd string, originalValue string) bool {
	originalValueInBytes := []byte(originalValue)
	byteHash := []byte(hashedPwd)
	err := bcrypt.CompareHashAndPassword(byteHash, originalValueInBytes)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

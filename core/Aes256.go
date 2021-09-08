package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/mergermarket/go-pkcs7"
	"io"
	"klovercloud/queue/config"
	"log"
)



func Encrypt(input string) (string, error) {
	key := []byte(config.AES256_CIPHER_KEY)
	plainText := []byte(input)
	plainText, err := pkcs7.Pad(plainText, aes.BlockSize)
	if err != nil {
		return "", fmt.Errorf(`plainText: "%s" has error`, plainText)
	}
	if len(plainText)%aes.BlockSize != 0 {
		err := fmt.Errorf(`plainText: "%s" has the wrong block size`, plainText)
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	cipherText := make([]byte, aes.BlockSize+len(plainText))
	iv := cipherText[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(cipherText[aes.BlockSize:], plainText)

	return fmt.Sprintf("%x", cipherText), nil
}

func  Decrypt(input string) (string, error) {
	key := []byte(config.AES256_CIPHER_KEY)
	cipherText, _ := hex.DecodeString(input)
	log.Println("input:", input, "decrypt keys: ", config.AES256_CIPHER_KEY, cipherText)


	block, err := aes.NewCipher(key)
	if err != nil {
		log.Println(err.Error())
		return "", err
	}

	if len(cipherText) < aes.BlockSize {
		err := fmt.Errorf("CipherText too short")
		log.Println(err.Error())
		return "", err
	}
	iv := cipherText[:aes.BlockSize]
	cipherText = cipherText[aes.BlockSize:]
	if len(cipherText)%aes.BlockSize != 0 {
		err := fmt.Errorf("CipherText is not a multiple of the block size")
		log.Println(err.Error())
		return "", err
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(cipherText, cipherText)

	cipherText, _ = pkcs7.Unpad(cipherText, aes.BlockSize)
	return fmt.Sprintf("%s", cipherText), nil
}
package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
)

// // use this to encrypt commands on the attacking machine
// func main() {
// 	// flags
// 	keyPtr := flag.String("key", "", "The 32-byte hex encryption key")
// 	filePtr := flag.String("file", "", "The txt file to encrypt")
// 	flag.Parse()

// 	// check flags
// 	if *keyPtr == "" || *filePtr == "" {
// 		fmt.Println("Usage: go run encrypt.go -key <HEX_KEY> -file <FILE_PATH>")
// 		os.Exit(1)
// 	}

// 	// decode key
// 	key, err := hex.DecodeString(*keyPtr)
// 	if err != nil || len(key) != 32 {
// 		fmt.Println("Error: Key must be a valid 64-character hex string (32 bytes).")
// 		os.Exit(1)
// 	}

// 	// read command file
// 	plaintext, err := os.ReadFile(*filePtr)
// 	if err != nil {
// 		fmt.Printf("Error reading file: %v\n", err)
// 		os.Exit(1)
// 	}

// 	// setup AES-GCM
// 	block, _ := aes.NewCipher(key)
// 	gcm, _ := cipher.NewGCM(block)

// 	// create nonce
// 	nonce := make([]byte, gcm.NonceSize())
// 	io.ReadFull(rand.Reader, nonce)

// 	// encrypt and encode
// 	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
// 	encoded := base64.StdEncoding.EncodeToString(ciphertext)

// 	// overwrite original file
// 	err = os.WriteFile(*filePtr, []byte(encoded), 0644)
// 	if err != nil {
// 		fmt.Printf("Error overwriting file: %v\n", err)
// 		os.Exit(1)
// 	}

// 	fmt.Printf("[+] Success: %s has been encrypted and overwritten.\n", *filePtr)
// }

// takes plaintext and hex key, return base64 ciphertext using AES-GCM
func EncryptString(plainText string, keyHex string) (string, error) {
	// decode key
	key, err := hex.DecodeString(keyHex)
	if err != nil || len(key) != 32 {
		return "", fmt.Errorf("invalid key: must be 32 bytes")
	}

	// setup AES-GCM
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// create nonce (this is for ensuring different ciphertexts for same plaintext to avoid pattern recognition, and it's not a secret)
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// encrypt and encode
	ciphertext := gcm.Seal(nonce, nonce, []byte(plainText), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// takes base64 ciphertext and hex key, return decrypted plaintext using AES-GCM
func DecryptString(cryptoText string, keyHex string) (string, error) {
	// decode key into bytes
	key, err := hex.DecodeString(keyHex)
	if err != nil || len(key) != 32 {
		return "", fmt.Errorf("invalid key: must be 32 bytes")
	}

	// decode base64 ciphertext
	ciphertext, err := base64.StdEncoding.DecodeString(cryptoText)
	if err != nil {
		return "", fmt.Errorf("base64 decode failed: %v", err)
	}

	// initialize cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// remove nonce
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// decrypt
	plaintext, err := gcm.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		// will fail if wrong key or tampered ciphertext
		return "", fmt.Errorf("decryption/auth failed: %v", err)
	}

	return string(plaintext), nil
}

// XORTransform toggles the bits. Running it twice restores the original.
func XORTransform(input []byte, key []byte) []byte {
	output := make([]byte, len(input))
	for i := 0; i < len(input); i++ {
		// Use the modulo operator (%) to loop the key if it's shorter than the input
		output[i] = input[i] ^ key[i%len(key)]
	}
	return output
}

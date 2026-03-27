package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
)

// use this to encrypt commands on the attacking machine
func main() {
	// flags
	keyPtr := flag.String("key", "", "The 32-byte hex encryption key")
	filePtr := flag.String("file", "", "The txt file to encrypt")
	flag.Parse()

	// check flags
	if *keyPtr == "" || *filePtr == "" {
		fmt.Println("Usage: go run encrypt.go -key <HEX_KEY> -file <FILE_PATH>")
		os.Exit(1)
	}

	// decode key
	key, err := hex.DecodeString(*keyPtr)
	if err != nil || len(key) != 32 {
		fmt.Println("Error: Key must be a valid 64-character hex string (32 bytes).")
		os.Exit(1)
	}

	// read command file
	plaintext, err := os.ReadFile(*filePtr)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	// setup AES-GCM
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)

	// create nonce
	nonce := make([]byte, gcm.NonceSize())
	io.ReadFull(rand.Reader, nonce)

	// encrypt and encode
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	encoded := base64.StdEncoding.EncodeToString(ciphertext)

	// overwrite original file
	err = os.WriteFile(*filePtr, []byte(encoded), 0644)
	if err != nil {
		fmt.Printf("Error overwriting file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[+] Success: %s has been encrypted and overwritten.\n", *filePtr)
}

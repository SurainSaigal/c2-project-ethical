COMPILE COMMAND:
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o sys_update backdoor/backdoor.go

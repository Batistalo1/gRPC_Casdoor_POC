package casdoor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	Organization string
	AppName      string
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	Error       string `json:"error"`
}

// Login authenticates a user with Casdoor
// Using the Resource Owner Password Credentials Grant and returns an access token.
func (c *Client) Login(email, password string) (string, error) {
	data := url.Values{
		"grant_type":    {"password"},
		"client_id":     {c.ClientID},
		"client_secret": {c.ClientSecret},
		"username":      {email},
		"password":      {password},
		"scope":         {"openid profile email"},
	}

	resp, err := http.PostForm(c.BaseURL+"/api/login/oauth/access_token", data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result TokenResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Error != "" {
		return "", fmt.Errorf("casdoor: %s", result.Error)
	}
	return result.AccessToken, nil
}

// Register creates a new user in Casdoor using the admin API.
func (c *Client) Register(username, email, password string) error {
	payload := map[string]interface{}{
		"owner":             c.Organization,
		"name":              username,
		"email":             email,
		"password":          password,
		"signupApplication": c.AppName,
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", c.BaseURL+"/api/add-user", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")

	// Set basic auth with client ID and secret for authentication
	req.SetBasicAuth(c.ClientID, c.ClientSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if status, ok := result["status"].(string); ok && status != "ok" {
		return fmt.Errorf("casdoor register failed: %v", result["msg"])
	}
	return nil
}

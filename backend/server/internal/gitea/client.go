package gitea

import (
	"net/http"
	"time"
)

// Client holds the configuration and HTTP client for Gitea API interactions
type Client struct {
	BaseURL    string
	AdminToken string
	HTTPClient *http.Client
}

// NewClient initializes a new Gitea API client
func NewClient(baseURL, adminToken string) *Client {
	return &Client{
		BaseURL:    baseURL,
		AdminToken: adminToken,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second, // Prevent hanging requests
		},
	}
}

// CreateUserOption represents the payload required to create a new Gitea user
type CreateUserOption struct {
	Username           string `json:"username"`
	Email              string `json:"email"`
	Password           string `json:"password"`
	MustChangePassword bool   `json:"must_change_password"`
	SendNotify         bool   `json:"send_notify"`
}

// EditUserOption represents the payload to modify an existing user (e.g., disabling them)
type EditUserOption struct {
	Active *bool `json:"active,omitempty"` // Pointer so we can omit it if nil, but send false if explicitly disabled
}
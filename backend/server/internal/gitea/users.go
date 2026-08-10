package gitea

import (
	"context"
	"fmt"
	"net/http"
)

// CreateUser provisions a new user account in Gitea
func (c *Client) CreateUser(ctx context.Context, username, email, password string, mustChangePassword bool) error {
	opt := CreateUserOption{
		Username:           username,
		Email:              email,
		Password:           password,
		MustChangePassword: mustChangePassword,
		SendNotify:         false, // Central Server handles emails
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/admin/users", opt)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("expected status 201 Created, got %d", resp.StatusCode)
	}
	return nil
}

// DeleteUser removes a user.
func (c *Client) DeleteUser(ctx context.Context, username string, purge bool) error {
	path := fmt.Sprintf("/admin/users/%s?purge=%t", username, purge)
	
	resp, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected status 204/200 on delete, got %d", resp.StatusCode)
	}
	return nil
}

// DisableUser locks the Gitea account (ARCHIVED state)
func (c *Client) DisableUser(ctx context.Context, username string) error {
	active := false
	opt := EditUserOption{
		Active: &active,
	}

	path := fmt.Sprintf("/admin/users/%s", username)
	resp, err := c.doRequest(ctx, http.MethodPatch, path, opt)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected status 200 OK on disable, got %d", resp.StatusCode)
	}
	return nil
}

// SetPassword updates a user's password and optional change requirement
func (c *Client) SetPassword(ctx context.Context, username, password string, mustChangePassword bool) error {
	opt := ChangePasswordOption{
		Password:           password,
		MustChangePassword: &mustChangePassword,
	}

	path := fmt.Sprintf("/admin/users/%s", username)
	resp, err := c.doRequest(ctx, http.MethodPatch, path, opt)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected status 200 OK on set password, got %d", resp.StatusCode)
	}

	return nil
}
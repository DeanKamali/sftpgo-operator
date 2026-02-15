/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sftpgo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	sftpgov1alpha1 "github.com/sftpgo/sftpgo-operator/api/v1alpha1"
)

// Client is an SFTPGO REST API client
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Username   string
	Password   string
}

// NewClient creates a new SFTPGO API client
func NewClient(baseURL, username, password string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		Username: username,
		Password: password,
	}
}

// ServiceURL returns the URL for an SFTPGO service in Kubernetes
func ServiceURL(name, namespace string, port int32) string {
	return fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", name, namespace, port)
}

// UserPayload represents the SFTPGO API user structure
type UserPayload struct {
	ID                int                 `json:"id,omitempty"`
	Username          string              `json:"username"`
	Status            int                 `json:"status"` // 1=enabled, 0=disabled
	Email             string              `json:"email,omitempty"`
	Password          string              `json:"password,omitempty"`
	PublicKeys        []string            `json:"public_keys,omitempty"`
	HomeDir           string              `json:"home_dir"`
	VirtualFolders    []VF                `json:"virtual_folders,omitempty"`
	Permissions       map[string][]string `json:"permissions,omitempty"`
	QuotaSize         int64               `json:"quota_size,omitempty"`
	QuotaFiles        int                 `json:"quota_files,omitempty"`
	UploadBandwidth   int64               `json:"upload_bandwidth,omitempty"`
	DownloadBandwidth int64               `json:"download_bandwidth,omitempty"`
	MaxSessions       int                 `json:"max_sessions,omitempty"`
	AllowedIP         []string            `json:"allowed_ip,omitempty"`
	DeniedIP          []string            `json:"denied_ip,omitempty"`
	Groups            []GM                `json:"groups,omitempty"`
}

type VF struct {
	VirtualPath string `json:"virtual_path"`
	MappedPath  string `json:"mapped_path"`
	QuotaSize   int64  `json:"quota_size,omitempty"`
	QuotaFiles  int    `json:"quota_files,omitempty"`
}

type GM struct {
	Name string `json:"name"`
	Type int    `json:"type"`
}

// GetUser fetches a user by username
func (c *Client) GetUser(username string) (*UserPayload, error) {
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+"/api/v2/users/"+username, nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned %d", resp.StatusCode)
	}

	var user UserPayload
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

// CreateUser creates a new user
func (c *Client) CreateUser(payload *UserPayload) (*UserPayload, error) {
	return c.upsertUser(http.MethodPost, "", payload)
}

// UpdateUser updates an existing user
func (c *Client) UpdateUser(username string, payload *UserPayload) (*UserPayload, error) {
	return c.upsertUser(http.MethodPut, username, payload)
}

func (c *Client) upsertUser(method, pathSuffix string, payload *UserPayload) (*UserPayload, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	url := c.BaseURL + "/api/v2/users"
	if pathSuffix != "" {
		url += "/" + pathSuffix
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	c.setAuth(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("API returned %d", resp.StatusCode)
	}

	var user UserPayload
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

// DeleteUser deletes a user
func (c *Client) DeleteUser(username string) error {
	req, err := http.NewRequest(http.MethodDelete, c.BaseURL+"/api/v2/users/"+username, nil)
	if err != nil {
		return err
	}
	c.setAuth(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) setAuth(req *http.Request) {
	if c.Username != "" && c.Password != "" {
		req.SetBasicAuth(c.Username, c.Password)
	}
}

// UserFromCR converts SftpGoUser CR to API payload
func UserFromCR(spec *sftpgov1alpha1.SftpGoUserSpec, password string, publicKeys []string) *UserPayload {
	status := 1
	if spec.Status == "disabled" {
		status = 0
	}

	perm := map[string][]string{}
	if len(spec.Permissions) > 0 {
		perm["/"] = spec.Permissions
	} else {
		perm["/"] = []string{"*"}
	}

	p := &UserPayload{
		Username:    spec.Username,
		Status:      status,
		Email:       spec.Email,
		HomeDir:     spec.HomeDir,
		Permissions: perm,
	}
	if password != "" {
		p.Password = password
	}
	if len(publicKeys) > 0 {
		p.PublicKeys = publicKeys
	}

	for _, vf := range spec.VirtualFolders {
		p.VirtualFolders = append(p.VirtualFolders, VF{
			VirtualPath: vf.VirtualPath,
			MappedPath:  vf.PhysicalPath,
			QuotaSize:   vf.Quota,
		})
	}

	if spec.Quota != nil {
		p.QuotaSize = spec.Quota.Size
		p.QuotaFiles = spec.Quota.Files
	}
	if spec.BandwidthLimits != nil {
		// SFTPGO API expects KB/s
		p.UploadBandwidth = spec.BandwidthLimits.Upload / 1024
		p.DownloadBandwidth = spec.BandwidthLimits.Download / 1024
	}
	if spec.MaxSessions > 0 {
		p.MaxSessions = spec.MaxSessions
	}
	if len(spec.AllowedIP) > 0 {
		p.AllowedIP = spec.AllowedIP
	}
	if len(spec.DeniedIP) > 0 {
		p.DeniedIP = spec.DeniedIP
	}
	for _, g := range spec.Groups {
		p.Groups = append(p.Groups, GM{Name: g, Type: 1})
	}

	return p
}

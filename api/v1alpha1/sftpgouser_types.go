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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SftpGoUserSpec defines the desired state of SftpGoUser
type SftpGoUserSpec struct {
	// Username is the SFTPGO username
	// +kubebuilder:validation:Required
	Username string `json:"username"`

	// Password is the user's password (required if not using public key)
	// +optional
	Password string `json:"password,omitempty"`

	// PasswordSecretRef is a reference to a secret containing the password
	// +optional
	PasswordSecretRef *SecretRef `json:"passwordSecretRef,omitempty"`

	// PublicKeys is a list of public keys for SSH authentication
	// +optional
	PublicKeys []string `json:"publicKeys,omitempty"`

	// PublicKeysSecretRef is a reference to a secret containing public keys
	// +optional
	PublicKeysSecretRef *SecretRef `json:"publicKeysSecretRef,omitempty"`

	// Email is the user's email address
	// +optional
	Email string `json:"email,omitempty"`

	// Status defines if the user is enabled or disabled
	// +optional
	// +kubebuilder:validation:Enum=enabled;disabled
	Status string `json:"status,omitempty"`

	// HomeDir is the user's home directory
	// +kubebuilder:validation:Required
	HomeDir string `json:"homeDir"`

	// VirtualFolders defines virtual folders for the user
	// +optional
	VirtualFolders []VirtualFolder `json:"virtualFolders,omitempty"`

	// Permissions defines the user's permissions
	// +optional
	Permissions []string `json:"permissions,omitempty"`

	// Quota defines storage quota (in bytes)
	// +optional
	Quota *Quota `json:"quota,omitempty"`

	// Bandwidth limits
	// +optional
	BandwidthLimits *BandwidthLimits `json:"bandwidthLimits,omitempty"`

	// Upload/Download limits (in bytes/sec)
	// +optional
	RateLimits *RateLimits `json:"rateLimits,omitempty"`

	// Max sessions allowed
	// +optional
	MaxSessions int `json:"maxSessions,omitempty"`

	// Concurrent transfers limit
	// +optional
	MaxConcurrentTransfers int `json:"maxConcurrentTransfers,omitempty"`

	// Allowed IP addresses (CIDR notation)
	// +optional
	AllowedIP []string `json:"allowedIP,omitempty"`

	// Denied IP addresses (CIDR notation)
	// +optional
	DeniedIP []string `json:"deniedIP,omitempty"`

	// Protocols allowed (SFTP, FTP, WebDAV, HTTP)
	// +optional
	Protocols []string `json:"protocols,omitempty"`

	// Groups the user belongs to
	// +optional
	Groups []string `json:"groups,omitempty"`

	// Additional settings
	// +optional
	Filters UserFilters `json:"filters,omitempty"`

	// Filesystem configuration
	// +optional
	Filesystem *FilesystemConfig `json:"filesystem,omitempty"`

	// The SftpGoServer this user belongs to
	// +kubebuilder:validation:Required
	ServerRef ServerRef `json:"serverRef"`
}

// SecretRef references a secret key
type SecretRef struct {
	// Name of the secret
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Key in the secret
	// +kubebuilder:validation:Required
	Key string `json:"key"`
}

// VirtualFolder defines a virtual folder
type VirtualFolder struct {
	// Virtual path
	// +kubebuilder:validation:Required
	VirtualPath string `json:"virtualPath"`

	// Physical path (on the host)
	// +kubebuilder:validation:Required
	PhysicalPath string `json:"physicalPath"`

	// Quota for this folder (in bytes)
	// +optional
	Quota int64 `json:"quota,omitempty"`
}

// Quota defines storage quota
type Quota struct {
	// Total size allowed (in bytes)
	// +optional
	Size int64 `json:"size,omitempty"`

	// Maximum number of files
	// +optional
	Files int `json:"files,omitempty"`
}

// BandwidthLimits defines bandwidth limits
type BandwidthLimits struct {
	// Upload speed limit (bytes/sec, 0 = unlimited)
	// +optional
	Upload int64 `json:"upload,omitempty"`

	// Download speed limit (bytes/sec, 0 = unlimited)
	// +optional
	Download int64 `json:"download,omitempty"`
}

// RateLimits defines rate limits
type RateLimits struct {
	// Average rate (bytes/sec)
	// +optional
	Average int64 `json:"average,omitempty"`

	// Burst size (bytes)
	// +optional
	Burst int64 `json:"burst,omitempty"`
}

// UserFilters defines additional user filters/restrictions
type UserFilters struct {
	// Require password change
	// +optional
	RequirePasswordChange bool `json:"requirePasswordChange,omitempty"`

	// Require TOTP
	// +optional
	RequireTOTP bool `json:"requireTOTP,omitempty"`

	// External auth hook
	// +optional
	ExternalAuthHook string `json:"externalAuthHook,omitempty"`

	// Command restrictions
	// +optional
	AllowedCommands []string `json:"allowedCommands,omitempty"`

	// Time-based access restrictions
	// +optional
	TimeIntervals []TimeInterval `json:"timeIntervals,omitempty"`
}

// TimeInterval defines a time interval for access
type TimeInterval struct {
	// Start hour (0-23)
	// +optional
	Start int `json:"start,omitempty"`

	// End hour (0-23)
	// +optional
	End int `json:"end,omitempty"`

	// Days of week (0=Sunday, 1=Monday, etc.)
	// +optional
	Days []int `json:"days,omitempty"`
}

// FilesystemConfig defines filesystem settings
type FilesystemConfig struct {
	// Provider type: osfs, s3fs, gcsfs, azureblob, crypt, encrypted, sftpfs
	// +optional
	// +kubebuilder:validation:Enum=osfs;s3fs;gcsfs;azureblob;crypt;encrypted;sftpfs
	Provider string `json:"provider,omitempty"`

	// S3 configuration
	// +optional
	S3 *S3FilesystemConfig `json:"s3,omitempty"`

	// GCS configuration
	// +optional
	GCS *GCSFilesystemConfig `json:"gcs,omitempty"`

	// Azure configuration
	// +optional
	Azure *AzureFilesystemConfig `json:"azure,omitempty"`

	// SFTP configuration
	// +optional
	SFTP *SFTPFilesystemConfig `json:"sftp,omitempty"`

	// Crypt configuration
	// +optional
	Crypt *CryptFilesystemConfig `json:"crypt,omitempty"`
}

// S3FilesystemConfig defines S3 filesystem settings
type S3FilesystemConfig struct {
	// Bucket name
	// +optional
	Bucket string `json:"bucket,omitempty"`

	// Region
	// +optional
	Region string `json:"region,omitempty"`

	// Access key
	// +optional
	AccessKey string `json:"accessKey,omitempty"`

	// Access secret reference
	// +optional
	AccessSecret *SecretRef `json:"accessSecret,omitempty"`

	// Endpoint URL
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// Storage class
	// +optional
	StorageClass string `json:"storageClass,omitempty"`

	// ACL
	// +optional
	ACL string `json:"acl,omitempty"`

	// Upload part size
	// +optional
	UploadPartSize int64 `json:"uploadPartSize,omitempty"`

	// Key prefix
	// +optional
	KeyPrefix string `json:"keyPrefix,omitempty"`
}

// GCSFilesystemConfig defines GCS filesystem settings
type GCSFilesystemConfig struct {
	// Bucket name
	// +optional
	Bucket string `json:"bucket,omitempty"`

	// Credentials file path
	// +optional
	CredentialsFile string `json:"credentialsFile,omitempty"`

	// Storage class
	// +optional
	StorageClass string `json:"storageClass,omitempty"`

	// Key prefix
	// +optional
	KeyPrefix string `json:"keyPrefix,omitempty"`
}

// AzureFilesystemConfig defines Azure Blob filesystem settings
type AzureFilesystemConfig struct {
	// Container name
	// +optional
	Container string `json:"container,omitempty"`

	// Account name
	// +optional
	AccountName string `json:"accountName,omitempty"`

	// Account key reference
	// +optional
	AccountKey *SecretRef `json:"accountKey,omitempty"`

	// SAS URL
	// +optional
	SASURL string `json:"sasURL,omitempty"`

	// Endpoint suffix
	// +optional
	EndpointSuffix string `json:"endpointSuffix,omitempty"`

	// Upload block size
	// +optional
	UploadBlockSize int64 `json:"uploadBlockSize,omitempty"`

	// Key prefix
	// +optional
	KeyPrefix string `json:"keyPrefix,omitempty"`
}

// SFTPFilesystemConfig defines SFTP filesystem settings
type SFTPFilesystemConfig struct {
	// Host
	// +optional
	Host string `json:"host,omitempty"`

	// Port
	// +optional
	Port int32 `json:"port,omitempty"`

	// Username
	// +optional
	Username string `json:"username,omitempty"`

	// Password reference
	// +optional
	Password *SecretRef `json:"password,omitempty"`

	// Private key reference
	// +optional
	PrivateKey *SecretRef `json:"privateKey,omitempty"`

	// Remote path
	// +optional
	RemotePath string `json:"remotePath,omitempty"`
}

// CryptFilesystemConfig defines encrypted filesystem settings
type CryptFilesystemConfig struct {
	// Passphrase reference
	// +optional
	Passphrase *SecretRef `json:"passphrase,omitempty"`

	// Physical path (to encrypt)
	// +optional
	PhysicalPath string `json:"physicalPath,omitempty"`
}

// ServerRef references the SftpGoServer
type ServerRef struct {
	// Name of the SftpGoServer
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace of the SftpGoServer (defaults to same namespace)
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// SftpGoUserStatus defines the observed state of SftpGoUser
type SftpGoUserStatus struct {
	// Phase is the current phase
	// +optional
	Phase string `json:"phase,omitempty"`

	// UserID is the SFTPGO internal user ID
	// +optional
	UserID int `json:"userID,omitempty"`

	// Conditions is the list of conditions
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastSynced is the last time the user was synced
	// +optional
	LastSynced *metav1.Time `json:"lastSynced,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Username",type="string",JSONPath=".spec.username"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".spec.status"
// +kubebuilder:printcolumn:name="HomeDir",type="string",JSONPath=".spec.homeDir"
// +kubebuilder:printcolumn:name="Server",type="string",JSONPath=".spec.serverRef.name"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// SftpGoUser is the Schema for the sftpgousers API
type SftpGoUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SftpGoUserSpec   `json:"spec,omitempty"`
	Status SftpGoUserStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SftpGoUserList contains a list of SftpGoUser
type SftpGoUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SftpGoUser `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SftpGoUser{}, &SftpGoUserList{})
}

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SftpGoServerSpec defines the desired state of SftpGoServer
type SftpGoServerSpec struct {
	// Image is the SFTPGO container image (default: docker.io/drakkan/sftpgo:latest)
	// +optional
	Image string `json:"image,omitempty"`

	// ImagePullPolicy is the image pull policy
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// Replicas is the desired number of replicas
	// +optional
	// +kubebuilder:validation:Minimum=1
	Replicas *int32 `json:"replicas,omitempty"`

	// ServiceAccount is the service account name to use for the deployment
	// +optional
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// SFTPGO Configuration
	// +optional
	Config SFTPGOConfig `json:"config,omitempty"`

	// SFTP Port (default: 2022)
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	SFTPPort int32 `json:"sftpPort,omitempty"`

	// Web Port (default: 8080)
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	WebPort int32 `json:"webPort,omitempty"`

	// Data Volume configuration
	// +optional
	DataVolume *VolumeConfig `json:"dataVolume,omitempty"`

	// Storage backend type: memory, sqlite, mysql, postgres
	// +optional
	// +kubebuilder:validation:Enum=memory;sqlite;mysql;postgres
	StorageBackend string `json:"storageBackend,omitempty"`

	// Database connection details (for mysql/postgres)
	// +optional
	Database *DatabaseConfig `json:"database,omitempty"`

	// Resources is the resource requirements for the container
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// NodeSelector is a selector which must be true for the pod to fit on a node
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Tolerations are tolerations to propagate to the pod
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Affinity is a group of affinity scheduling rules for the pods
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// AdminSecretRef is optional reference to a secret containing "username" and "password"
	// keys for the SFTPGO admin API (used by SftpGoUser controller to manage users)
	// +optional
	AdminSecretRef *corev1.LocalObjectReference `json:"adminSecretRef,omitempty"`
}

// VolumeConfig defines the data volume configuration
type VolumeConfig struct {
	// StorageClass to use for the PVC
	// +optional
	StorageClass *string `json:"storageClass,omitempty"`

	// Size of the volume (default: 10Gi)
	// +optional
	Size string `json:"size,omitempty"`

	// MountPath is the path where the volume will be mounted (default: /srv/sftpgo)
	// +optional
	MountPath string `json:"mountPath,omitempty"`
}

// DatabaseConfig defines database connection details
type DatabaseConfig struct {
	// Host of the database
	// +optional
	Host string `json:"host,omitempty"`

	// Port of the database
	// +optional
	Port int32 `json:"port,omitempty"`

	// Database name
	// +optional
	Database string `json:"database,omitempty"`

	// Username for database authentication
	// +optional
	Username string `json:"username,omitempty"`

	// Password reference (secret)
	// +optional
	PasswordSecret *corev1.SecretKeySelector `json:"passwordSecret,omitempty"`

	// SSL mode for postgres
	// +optional
	SSLMode string `json:"sslMode,omitempty"`
}

// SFTPGOConfig defines the SFTPGO configuration
type SFTPGOConfig struct {
	// Common settings
	// +optional
	Common *CommonConfig `json:"common,omitempty"`

	// SFTP settings
	// +optional
	SFTP *SFTPConfig `json:"sftp,omitempty"`

	// FTP settings
	// +optional
	FTP *FTPConfig `json:"ftp,omitempty"`

	// WebDAV settings
	// +optional
	WebDAV *WebDAVConfig `json:"webdav,omitempty"`

	// HTTP settings
	// +optional
	HTTP *HTTPConfig `json:"http,omitempty"`
}

// CommonConfig defines common SFTPGO settings
type CommonConfig struct {
	// Idle timeout in minutes (default: 15)
	// +optional
	IdleTimeout int `json:"idleTimeout,omitempty"`

	// Upload mode: 0=standard, 1=atomic, 2=resumable
	// +optional
	UploadMode int `json:"uploadMode,omitempty"`

	// Maximum total connections (0 = unlimited)
	// +optional
	MaxTotalConnections int `json:"maxTotalConnections,omitempty"`

	// Maximum connections per host
	// +optional
	MaxPerHostConnections int `json:"maxPerHostConnections,omitempty"`
}

// SFTPConfig defines SFTP server settings
type SFTPConfig struct {
	// Enable SFTP server (default: true)
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Port (default: 2022)
	// +optional
	Port int32 `json:"port,omitempty"`

	// Host keys (list of paths to SSH host keys)
	// +optional
	HostKeys []string `json:"hostKeys,omitempty"`

	// Maximum authentication attempts
	// +optional
	MaxAuthTries int `json:"maxAuthTries,omitempty"`

	// Allowed SSH commands
	// +optional
	AllowedSSHCommands []string `json:"allowedSSHCommands,omitempty"`

	// Password authentication enabled
	// +optional
	PasswordAuthentication bool `json:"passwordAuthentication,omitempty"`

	// Keyboard interactive authentication enabled
	// +optional
	KeyboardInteractiveAuth bool `json:"keyboardInteractiveAuth,omitempty"`
}

// FTPConfig defines FTP server settings
type FTPConfig struct {
	// Enable FTP server (default: false)
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Port (default: 2121)
	// +optional
	Port int32 `json:"port,omitempty"`

	// Passive port range
	// +optional
	PassivePortRange *PortRange `json:"passivePortRange,omitempty"`

	// Active port range
	// +optional
	ActivePortRange *PortRange `json:"activePortRange,omitempty"`
}

// PortRange defines a port range
type PortRange struct {
	Start int32 `json:"start"`
	End   int32 `json:"end"`
}

// WebDAVConfig defines WebDAV server settings
type WebDAVConfig struct {
	// Enable WebDAV server (default: true)
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Port (default: 8080)
	// +optional
	Port int32 `json:"port,omitempty"`

	// Enable HTTPS
	// +optional
	EnableHTTPS bool `json:"enableHTTPS,omitempty"`

	// Certificate file path
	// +optional
	CertificateFile string `json:"certificateFile,omitempty"`

	// Certificate key file path
	// +optional
	CertificateKeyFile string `json:"certificateKeyFile,omitempty"`
}

// HTTPConfig defines HTTP server settings
type HTTPConfig struct {
	// Enable HTTP API server (default: true)
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Port (default: 8080)
	// +optional
	Port int32 `json:"port,omitempty"`

	// Enable HTTPS
	// +optional
	EnableHTTPS bool `json:"enableHTTPS,omitempty"`

	// Certificate file path
	// +optional
	CertificateFile string `json:"certificateFile,omitempty"`

	// Certificate key file path
	// +optional
	CertificateKeyFile string `json:"certificateKeyFile,omitempty"`

	// Base URL for API
	// +optional
	BaseURL string `json:"baseURL,omitempty"`
}

// SftpGoServerStatus defines the observed state of SftpGoServer
type SftpGoServerStatus struct {
	// Replicas is the current number of replicas
	Replicas int32 `json:"replicas,omitempty"`

	// ReadyReplicas is the number of ready replicas
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// Conditions is the list of conditions
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Phase is the current phase of the deployment
	Phase string `json:"phase,omitempty"`

	// Service ports
	Ports ServicePorts `json:"ports,omitempty"`
}

// ServicePorts defines the service ports
type ServicePorts struct {
	SFTP int32 `json:"sftp,omitempty"`
	Web  int32 `json:"web,omitempty"`
	HTTP int32 `json:"http,omitempty"`
	FTP  int32 `json:"ftp,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Replicas",type="integer",JSONPath=".spec.replicas"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.readyReplicas"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// SftpGoServer is the Schema for the sftpgoservers API
type SftpGoServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SftpGoServerSpec   `json:"spec,omitempty"`
	Status SftpGoServerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SftpGoServerList contains a list of SftpGoServer
type SftpGoServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SftpGoServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SftpGoServer{}, &SftpGoServerList{})
}

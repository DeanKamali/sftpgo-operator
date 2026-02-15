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

package controller

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	sftpgov1alpha1 "github.com/sftpgo/sftpgo-operator/api/v1alpha1"
	"github.com/sftpgo/sftpgo-operator/internal/sftpgo"
)

const sftpgoUserFinalizer = "sftpgo.sftpgo.io/user-finalizer"

// SftpGoUserReconciler reconciles a SftpGoUser object
type SftpGoUserReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=sftpgo.sftpgo.io,resources=sftpgousers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sftpgo.sftpgo.io,resources=sftpgousers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=sftpgo.sftpgo.io,resources=sftpgousers/finalizers,verbs=update
// +kubebuilder:rbac:groups=sftpgo.sftpgo.io,resources=sftpgoservers,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get

func (r *SftpGoUserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	user := &sftpgov1alpha1.SftpGoUser{}
	if err := r.Get(ctx, req.NamespacedName, user); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Add finalizer for cleanup on delete
	if !controllerutil.ContainsFinalizer(user, sftpgoUserFinalizer) {
		controllerutil.AddFinalizer(user, sftpgoUserFinalizer)
		if err := r.Update(ctx, user); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Handle deletion - remove user from SFTPGO
	if !user.GetDeletionTimestamp().IsZero() {
		if err := r.deleteUserFromSFTPGO(ctx, user); err != nil {
			log.Error(err, "Failed to delete user from SFTPGO")
			return ctrl.Result{}, err
		}
		controllerutil.RemoveFinalizer(user, sftpgoUserFinalizer)
		if err := r.Update(ctx, user); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Resolve ServerRef namespace
	ns := user.Spec.ServerRef.Namespace
	if ns == "" {
		ns = user.Namespace
	}

	// Fetch SftpGoServer
	server := &sftpgov1alpha1.SftpGoServer{}
	if err := r.Get(ctx, types.NamespacedName{Name: user.Spec.ServerRef.Name, Namespace: ns}, server); err != nil {
		if errors.IsNotFound(err) {
			meta.SetStatusCondition(&user.Status.Conditions, metav1.Condition{
				Type:    "Ready",
				Status:  metav1.ConditionFalse,
				Reason:  "ServerNotFound",
				Message: fmt.Sprintf("SftpGoServer %s not found in namespace %s", user.Spec.ServerRef.Name, ns),
			})
			user.Status.Phase = "Error"
			_ = r.Status().Update(ctx, user)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Build API URL (service is same name as server)
	webPort := int32(8080)
	if server.Spec.WebPort > 0 {
		webPort = server.Spec.WebPort
	}
	baseURL := sftpgo.ServiceURL(server.Name, ns, webPort)

	// Get admin credentials
	username, password, err := r.getAdminCredentials(ctx, server)
	if err != nil {
		log.Error(err, "Failed to get admin credentials")
		meta.SetStatusCondition(&user.Status.Conditions, metav1.Condition{
			Type:    "Ready",
			Status:  metav1.ConditionFalse,
			Reason:  "AuthError",
			Message: err.Error(),
		})
		user.Status.Phase = "Error"
		_ = r.Status().Update(ctx, user)
		return ctrl.Result{}, err
	}
	if username == "" || password == "" {
		meta.SetStatusCondition(&user.Status.Conditions, metav1.Condition{
			Type:    "Ready",
			Status:  metav1.ConditionFalse,
			Reason:  "AuthNotConfigured",
			Message: "SftpGoServer AdminSecretRef not configured - cannot manage users via API",
		})
		user.Status.Phase = "Pending"
		_ = r.Status().Update(ctx, user)
		return ctrl.Result{}, nil
	}

	client := sftpgo.NewClient(baseURL, username, password)

	// Resolve user password
	userPassword, err := r.resolvePassword(ctx, user)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Resolve public keys
	publicKeys, err := r.resolvePublicKeys(ctx, user)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Build payload
	payload := sftpgo.UserFromCR(&user.Spec, userPassword, publicKeys)

	// Create or update
	existing, err := client.GetUser(user.Spec.Username)
	if err != nil {
		log.Error(err, "Failed to get user from SFTPGO")
		meta.SetStatusCondition(&user.Status.Conditions, metav1.Condition{
			Type:    "Ready",
			Status:  metav1.ConditionFalse,
			Reason:  "APIError",
			Message: err.Error(),
		})
		user.Status.Phase = "Error"
		_ = r.Status().Update(ctx, user)
		return ctrl.Result{}, err
	}

	if existing != nil {
		payload.ID = existing.ID
		if userPassword == "" {
			payload.Password = "" // Don't overwrite password if not provided
		}
		_, err = client.UpdateUser(user.Spec.Username, payload)
	} else {
		if userPassword == "" && len(publicKeys) == 0 {
			meta.SetStatusCondition(&user.Status.Conditions, metav1.Condition{
				Type:    "Ready",
				Status:  metav1.ConditionFalse,
				Reason:  "ValidationError",
				Message: "New user requires either password or publicKeys",
			})
			user.Status.Phase = "Error"
			_ = r.Status().Update(ctx, user)
			return ctrl.Result{}, nil
		}
		_, err = client.CreateUser(payload)
	}
	if err != nil {
		log.Error(err, "Failed to create/update user in SFTPGO")
		meta.SetStatusCondition(&user.Status.Conditions, metav1.Condition{
			Type:    "Ready",
			Status:  metav1.ConditionFalse,
			Reason:  "APIError",
			Message: err.Error(),
		})
		user.Status.Phase = "Error"
		_ = r.Status().Update(ctx, user)
		return ctrl.Result{}, err
	}

	// Update status
	now := metav1.Now()
	meta.SetStatusCondition(&user.Status.Conditions, metav1.Condition{
		Type:   "Ready",
		Status: metav1.ConditionTrue,
		Reason: "Synced",
	})
	user.Status.Phase = "Synced"
	user.Status.LastSynced = &now
	if existing != nil {
		user.Status.UserID = existing.ID
	}
	if err := r.Status().Update(ctx, user); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *SftpGoUserReconciler) getAdminCredentials(ctx context.Context, server *sftpgov1alpha1.SftpGoServer) (string, string, error) {
	if server.Spec.AdminSecretRef == nil || server.Spec.AdminSecretRef.Name == "" {
		return "", "", nil
	}

	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      server.Spec.AdminSecretRef.Name,
		Namespace: server.Namespace,
	}, secret); err != nil {
		return "", "", err
	}

	username := string(secret.Data["username"])
	password := string(secret.Data["password"])
	return username, password, nil
}

func (r *SftpGoUserReconciler) resolvePassword(ctx context.Context, user *sftpgov1alpha1.SftpGoUser) (string, error) {
	if user.Spec.Password != "" {
		return user.Spec.Password, nil
	}
	if user.Spec.PasswordSecretRef != nil {
		secret := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{
			Name:      user.Spec.PasswordSecretRef.Name,
			Namespace: user.Namespace,
		}, secret); err != nil {
			return "", err
		}
		return string(secret.Data[user.Spec.PasswordSecretRef.Key]), nil
	}
	return "", nil
}

func (r *SftpGoUserReconciler) resolvePublicKeys(ctx context.Context, user *sftpgov1alpha1.SftpGoUser) ([]string, error) {
	if len(user.Spec.PublicKeys) > 0 {
		return user.Spec.PublicKeys, nil
	}
	if user.Spec.PublicKeysSecretRef != nil {
		secret := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{
			Name:      user.Spec.PublicKeysSecretRef.Name,
			Namespace: user.Namespace,
		}, secret); err != nil {
			return nil, err
		}
		raw := string(secret.Data[user.Spec.PublicKeysSecretRef.Key])
		keys := strings.Split(raw, "\n")
		var out []string
		for _, k := range keys {
			k = strings.TrimSpace(k)
			if k != "" && !strings.HasPrefix(k, "#") {
				out = append(out, k)
			}
		}
		return out, nil
	}
	return nil, nil
}

func (r *SftpGoUserReconciler) deleteUserFromSFTPGO(ctx context.Context, user *sftpgov1alpha1.SftpGoUser) error {
	ns := user.Spec.ServerRef.Namespace
	if ns == "" {
		ns = user.Namespace
	}

	server := &sftpgov1alpha1.SftpGoServer{}
	if err := r.Get(ctx, types.NamespacedName{Name: user.Spec.ServerRef.Name, Namespace: ns}, server); err != nil {
		if errors.IsNotFound(err) {
			return nil // Server gone, nothing to delete
		}
		return err
	}

	username, password, err := r.getAdminCredentials(ctx, server)
	if err != nil || username == "" || password == "" {
		return nil // Can't authenticate, skip delete
	}

	webPort := int32(8080)
	if server.Spec.WebPort > 0 {
		webPort = server.Spec.WebPort
	}
	client := sftpgo.NewClient(sftpgo.ServiceURL(server.Name, ns, webPort), username, password)
	return client.DeleteUser(user.Spec.Username)
}

// SetupWithManager sets up the controller with the Manager.
func (r *SftpGoUserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&sftpgov1alpha1.SftpGoUser{}).
		Named("sftpgouser").
		Complete(r)
}

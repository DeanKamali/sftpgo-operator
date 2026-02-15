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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	sftpgov1alpha1 "github.com/sftpgo/sftpgo-operator/api/v1alpha1"
)

const (
	sftpgoServerFinalizer = "sftpgo.sftpgo.io/finalizer"
	sftpgoDefaultImage    = "docker.io/drakkan/sftpgo:latest"
)

// SftpGoServerReconciler reconciles a SftpGoServer object
type SftpGoServerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=sftpgo.sftpgo.io,resources=sftpgoservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sftpgo.sftpgo.io,resources=sftpgoservers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=sftpgo.sftpgo.io,resources=sftpgoservers/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch

func (r *SftpGoServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	server := &sftpgov1alpha1.SftpGoServer{}
	if err := r.Get(ctx, req.NamespacedName, server); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Add finalizer for cleanup on delete
	if !controllerutil.ContainsFinalizer(server, sftpgoServerFinalizer) {
		controllerutil.AddFinalizer(server, sftpgoServerFinalizer)
		if err := r.Update(ctx, server); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Handle deletion
	if !server.GetDeletionTimestamp().IsZero() {
		if controllerutil.ContainsFinalizer(server, sftpgoServerFinalizer) {
			controllerutil.RemoveFinalizer(server, sftpgoServerFinalizer)
			if err := r.Update(ctx, server); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Apply defaults
	spec := r.applyDefaults(server)

	// Create or update ConfigMap
	desiredCM := r.configMapForServer(server)
	configMap := &corev1.ConfigMap{}
	configMap.Name = desiredCM.Name
	configMap.Namespace = desiredCM.Namespace
	if err := r.createOrUpdate(ctx, server, configMap, func() error {
		configMap.Data = desiredCM.Data
		configMap.Labels = desiredCM.Labels
		configMap.Annotations = desiredCM.Annotations
		return controllerutil.SetControllerReference(server, configMap, r.Scheme)
	}); err != nil {
		log.Error(err, "Failed to create/update ConfigMap")
		meta.SetStatusCondition(&server.Status.Conditions, metav1.Condition{
			Type:    "Degraded",
			Status:  metav1.ConditionTrue,
			Reason:  "ConfigMapError",
			Message: err.Error(),
		})
		_ = r.Status().Update(ctx, server)
		return ctrl.Result{}, err
	}

	// Create or update PVC if data volume is configured
	if spec.DataVolume != nil {
		pvc := r.pvcForServer(server)
		if err := r.createOrUpdate(ctx, server, pvc, func() error {
			return controllerutil.SetControllerReference(server, pvc, r.Scheme)
		}); err != nil {
			log.Error(err, "Failed to create/update PVC")
			return ctrl.Result{}, err
		}
	}

	// Create or update Deployment
	desiredDep := r.deploymentForServer(server)
	deployment := &appsv1.Deployment{}
	deployment.Name = desiredDep.Name
	deployment.Namespace = desiredDep.Namespace
	if err := r.createOrUpdate(ctx, server, deployment, func() error {
		deployment.Labels = desiredDep.Labels
		deployment.Spec = desiredDep.Spec
		deployment.Annotations = desiredDep.Annotations
		return controllerutil.SetControllerReference(server, deployment, r.Scheme)
	}); err != nil {
		log.Error(err, "Failed to create/update Deployment")
		meta.SetStatusCondition(&server.Status.Conditions, metav1.Condition{
			Type:    "Degraded",
			Status:  metav1.ConditionTrue,
			Reason:  "DeploymentError",
			Message: err.Error(),
		})
		_ = r.Status().Update(ctx, server)
		return ctrl.Result{}, err
	}

	// Refresh deployment to get status
	if err := r.Get(ctx, types.NamespacedName{Name: server.Name, Namespace: server.Namespace}, deployment); err != nil {
		return ctrl.Result{}, err
	}

	// Create or update Service
	desiredSvc := r.serviceForServer(server)
	svc := &corev1.Service{}
	svc.Name = desiredSvc.Name
	svc.Namespace = desiredSvc.Namespace
	if err := r.createOrUpdate(ctx, server, svc, func() error {
		svc.Labels = desiredSvc.Labels
		svc.Spec.Ports = desiredSvc.Spec.Ports
		svc.Spec.Selector = desiredSvc.Spec.Selector
		svc.Spec.Type = desiredSvc.Spec.Type
		svc.Annotations = desiredSvc.Annotations
		return controllerutil.SetControllerReference(server, svc, r.Scheme)
	}); err != nil {
		log.Error(err, "Failed to create/update Service")
		return ctrl.Result{}, err
	}

	// Update status
	meta.SetStatusCondition(&server.Status.Conditions, metav1.Condition{
		Type:   "Ready",
		Status: metav1.ConditionTrue,
		Reason: "Reconciled",
	})
	server.Status.Phase = "Running"
	server.Status.Ports = sftpgov1alpha1.ServicePorts{
		SFTP: r.getSFTPPort(spec),
		Web:  r.getWebPort(spec),
		HTTP: r.getWebPort(spec),
	}

	if deployment.Status.Replicas > 0 {
		server.Status.Replicas = deployment.Status.Replicas
		server.Status.ReadyReplicas = deployment.Status.ReadyReplicas
	}

	if err := r.Status().Update(ctx, server); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *SftpGoServerReconciler) applyDefaults(s *sftpgov1alpha1.SftpGoServer) *sftpgov1alpha1.SftpGoServerSpec {
	spec := s.Spec.DeepCopy()
	if spec.Image == "" {
		spec.Image = sftpgoDefaultImage
	}
	if spec.Replicas == nil {
		one := int32(1)
		spec.Replicas = &one
	}
	if spec.SFTPPort == 0 {
		spec.SFTPPort = 2022
	}
	if spec.WebPort == 0 {
		spec.WebPort = 8080
	}
	if spec.StorageBackend == "" {
		spec.StorageBackend = "sqlite"
	}
	return spec
}

func (r *SftpGoServerReconciler) getSFTPPort(spec *sftpgov1alpha1.SftpGoServerSpec) int32 {
	if spec.Config.SFTP != nil && spec.Config.SFTP.Port > 0 {
		return spec.Config.SFTP.Port
	}
	return spec.SFTPPort
}

func (r *SftpGoServerReconciler) getWebPort(spec *sftpgov1alpha1.SftpGoServerSpec) int32 {
	if spec.Config.HTTP != nil && spec.Config.HTTP.Port > 0 {
		return spec.Config.HTTP.Port
	}
	return spec.WebPort
}

func (r *SftpGoServerReconciler) createOrUpdate(ctx context.Context, owner client.Object, obj client.Object, setOwnerRef func() error) error {
	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, obj, func() error {
		if setOwnerRef != nil {
			return setOwnerRef()
		}
		return nil
	})
	if err != nil {
		return err
	}
	if op != controllerutil.OperationResultNone {
		logf.FromContext(ctx).Info("Resource updated", "kind", obj.GetObjectKind().GroupVersionKind().Kind, "name", obj.GetName(), "operation", op)
	}
	return nil
}

func (r *SftpGoServerReconciler) configMapForServer(s *sftpgov1alpha1.SftpGoServer) *corev1.ConfigMap {
	spec := r.applyDefaults(s)
	config := sftpgoMinimalConfig(spec, spec.AdminSecretRef != nil)

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.Name,
			Namespace: s.Namespace,
		},
		Data: map[string]string{
			"sftpgo.json": config,
		},
	}
}

func sftpgoMinimalConfig(spec *sftpgov1alpha1.SftpGoServerSpec, createDefaultAdmin bool) string {
	sftpPort := int32(2022)
	if spec.SFTPPort > 0 {
		sftpPort = spec.SFTPPort
	}
	if spec.Config.SFTP != nil && spec.Config.SFTP.Port > 0 {
		sftpPort = spec.Config.SFTP.Port
	}
	dataProvider := fmt.Sprintf(`"driver": "%s",
    "name": "/srv/sftpgo/sftpgo.db"`, spec.StorageBackend)
	if createDefaultAdmin {
		dataProvider += `,
    "create_default_admin": true`
	}
	return fmt.Sprintf(`{
  "sftpd": {
    "bindings": [{"port": %d, "address": "", "apply_proxy_config": true}],
    "max_auth_tries": 0,
    "host_keys": [],
    "keyboard_interactive_authentication": true,
    "password_authentication": true
  },
  "data_provider": {
    %s
  },
  "httpd": {
    "bindings": [{"port": 8080, "address": "", "enable_web_admin": true, "enable_rest_api": true}]
  }
}`, sftpPort, dataProvider)
}

func (r *SftpGoServerReconciler) pvcForServer(s *sftpgov1alpha1.SftpGoServer) *corev1.PersistentVolumeClaim {
	spec := s.Spec.DataVolume
	size := "10Gi"
	if spec.Size != "" {
		size = spec.Size
	}
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.Name + "-data",
			Namespace: s.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: *resourceQuantity(size),
				},
			},
		},
	}
	if spec.StorageClass != nil && *spec.StorageClass != "" {
		pvc.Spec.StorageClassName = spec.StorageClass
	}
	return pvc
}

func (r *SftpGoServerReconciler) deploymentForServer(s *sftpgov1alpha1.SftpGoServer) *appsv1.Deployment {
	spec := r.applyDefaults(s)
	labels := map[string]string{
		"app":        "sftpgo",
		"controller": s.Name,
	}
	replicas := int32(1)
	if spec.Replicas != nil {
		replicas = *spec.Replicas
	}

	mountPath := "/srv/sftpgo"
	volumes := []corev1.Volume{
		{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: s.Name},
				},
			},
		},
	}
	volumeMounts := []corev1.VolumeMount{
		{Name: "config", MountPath: "/etc/sftpgo", ReadOnly: true},
	}

	if spec.DataVolume != nil {
		if spec.DataVolume.MountPath != "" {
			mountPath = spec.DataVolume.MountPath
		}
		volumes = append(volumes, corev1.Volume{
			Name: "data",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: s.Name + "-data",
				},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: "data", MountPath: mountPath})
	} else {
		volumes = append(volumes, corev1.Volume{
			Name: "data",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: "data", MountPath: mountPath})
	}

	container := corev1.Container{
		Name:            "sftpgo",
		Image:           spec.Image,
		ImagePullPolicy: spec.ImagePullPolicy,
		Args:            []string{"sftpgo", "serve", "--config-file", "/etc/sftpgo/sftpgo.json"},
		Ports: []corev1.ContainerPort{
			{Name: "sftp", ContainerPort: r.getSFTPPort(spec), Protocol: corev1.ProtocolTCP},
			{Name: "web", ContainerPort: r.getWebPort(spec), Protocol: corev1.ProtocolTCP},
		},
		VolumeMounts: volumeMounts,
	}
	if spec.AdminSecretRef != nil {
		secretName := spec.AdminSecretRef.Name
		container.Env = []corev1.EnvVar{
			{
				Name: "SFTPGO_DEFAULT_ADMIN_USERNAME",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
						Key:                  "username",
					},
				},
			},
			{
				Name: "SFTPGO_DEFAULT_ADMIN_PASSWORD",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
						Key:                  "password",
					},
				},
			},
		}
	}
	if spec.Resources != nil {
		container.Resources = *spec.Resources
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.Name,
			Namespace: s.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					ServiceAccountName: spec.ServiceAccount,
					Containers:         []corev1.Container{container},
					Volumes:            volumes,
					NodeSelector:       spec.NodeSelector,
					Tolerations:        spec.Tolerations,
					Affinity:           spec.Affinity,
				},
			},
		},
	}
	return dep
}

func (r *SftpGoServerReconciler) serviceForServer(s *sftpgov1alpha1.SftpGoServer) *corev1.Service {
	spec := r.applyDefaults(s)
	labels := map[string]string{
		"app":        "sftpgo",
		"controller": s.Name,
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.Name,
			Namespace: s.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{Name: "sftp", Port: r.getSFTPPort(spec), TargetPort: intStr(r.getSFTPPort(spec)), Protocol: corev1.ProtocolTCP},
				{Name: "web", Port: r.getWebPort(spec), TargetPort: intStr(r.getWebPort(spec)), Protocol: corev1.ProtocolTCP},
			},
		},
	}
}

func resourceQuantity(s string) *resource.Quantity {
	q, _ := resource.ParseQuantity(s)
	return &q
}

func intStr(i int32) intstr.IntOrString {
	return intstr.FromInt(int(i))
}

// SetupWithManager sets up the controller with the Manager.
func (r *SftpGoServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&sftpgov1alpha1.SftpGoServer{}).
		Named("sftpgoserver").
		Complete(r)
}

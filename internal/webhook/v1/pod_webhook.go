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

package v1

import (
	"context"
	"os"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var podlog = logf.Log.WithName("devworkspace-pod-mutator")

// SetupPodWebhookWithManager registers the webhook for Pod in the manager.
func SetupPodWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &corev1.Pod{}).
		WithDefaulter(&DevWorkspacePodMutator{
			Client: mgr.GetClient(),
			Reader: mgr.GetAPIReader(),
		}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate--v1-pod,mutating=true,failurePolicy=fail,sideEffects=None,groups=core,resources=pods,verbs=create;update,versions=v1,name=mpod-v1.kb.io,admissionReviewVersions=v1

// DevWorkspacePodMutator mutates pods created by the DevWorkspace operator.
// It assigns a restricted "no-access" service account to DevWorkspace pods,
// creating the service account if it doesn't exist in the target namespace.
type DevWorkspacePodMutator struct {
	Client client.Client
	Reader client.Reader
}

const (
	devworkspaceCreatorLabel = "controller.devfile.io/creator"
	wiperServiceAccountName  = "devworkspace-wiper"
	wiperRoleName            = "devworkspace-wiper"
	wiperRoleBindingName     = "devworkspace-wiper"
	wiperContainerName       = "devworkspace-wiper"
	defaultWiperImage        = "wiper:latest"
)

const (
	wiperImageEnv         = "WIPER_IMAGE"
	wiperIdleThresholdEnv = "WIPER_IDLE_THRESHOLD"
	wiperPollIntervalEnv  = "WIPER_POLL_INTERVAL"
	wiperLogEveryEnv      = "WIPER_LOG_EVERY"
	wiperMainContainerEnv = "WIPER_MAIN_CONTAINER_NAME"
)

// Default implements webhook.CustomDefaulter to mutate DevWorkspace pods.
func (m *DevWorkspacePodMutator) Default(ctx context.Context, obj *corev1.Pod) error {
	podlog.Info("Processing pod", "name", obj.GetName(), "namespace", obj.GetNamespace())

	// Only mutate pods that have the devworkspace creator label
	if _, hasLabel := obj.GetLabels()[devworkspaceCreatorLabel]; !hasLabel {
		podlog.Info("Pod does not have devworkspace creator label, skipping mutation", "name", obj.GetName())
		return nil
	}

	namespace := obj.GetNamespace()
	if namespace == "" {
		podlog.Info("Pod namespace is empty, skipping service account creation")
		return nil
	}

	if err := m.ensureWiperRBAC(ctx, namespace); err != nil {
		podlog.Error(err, "Failed to ensure wiper RBAC", "namespace", namespace)
		return err
	}

	obj.Spec.ServiceAccountName = wiperServiceAccountName
	obj.Spec.Containers = ensureWiperContainer(obj.Spec.Containers)
	enforceSecurityContext(obj)
	return nil
}

func (m *DevWorkspacePodMutator) ensureWiperRBAC(ctx context.Context, namespace string) error {
	// Ensure ServiceAccount exists
	sa := &corev1.ServiceAccount{}
	if err := m.Reader.Get(ctx, client.ObjectKey{Name: wiperServiceAccountName, Namespace: namespace}, sa); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		podlog.Info("Creating wiper ServiceAccount", "namespace", namespace)
		newSA := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      wiperServiceAccountName,
				Namespace: namespace,
			},
			AutomountServiceAccountToken: boolPtr(true),
		}
		if err := m.Client.Create(ctx, newSA); err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}
	}

	// Define the expected role rules
	expectedRules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"pods"},
			Verbs:     []string{"get"},
		},
		{
			APIGroups: []string{"workspace.devfile.io"},
			Resources: []string{"devworkspaces"},
			Verbs:     []string{"get", "patch"},
		},
	}

	// Ensure Role exists with correct permissions
	role := &rbacv1.Role{}
	if err := m.Reader.Get(ctx, client.ObjectKey{Name: wiperRoleName, Namespace: namespace}, role); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		podlog.Info("Creating wiper Role", "namespace", namespace)
		newRole := &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      wiperRoleName,
				Namespace: namespace,
			},
			Rules: expectedRules,
		}
		if err := m.Client.Create(ctx, newRole); err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}
	} else if !roleRulesMatch(role.Rules, expectedRules) {
		// Role exists but has incorrect permissions - update it
		podlog.Info("Updating wiper Role with correct permissions", "namespace", namespace)
		role.Rules = expectedRules
		if err := m.Client.Update(ctx, role); err != nil {
			return err
		}
	}

	// Define expected RoleBinding
	expectedSubjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      wiperServiceAccountName,
			Namespace: namespace,
		},
	}
	expectedRoleRef := rbacv1.RoleRef{
		APIGroup: rbacv1.GroupName,
		Kind:     "Role",
		Name:     wiperRoleName,
	}

	// Ensure RoleBinding exists with correct bindings
	roleBinding := &rbacv1.RoleBinding{}
	if err := m.Reader.Get(ctx, client.ObjectKey{Name: wiperRoleBindingName, Namespace: namespace}, roleBinding); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		podlog.Info("Creating wiper RoleBinding", "namespace", namespace)
		newRoleBinding := &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      wiperRoleBindingName,
				Namespace: namespace,
			},
			Subjects: expectedSubjects,
			RoleRef:  expectedRoleRef,
		}
		if err := m.Client.Create(ctx, newRoleBinding); err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}
	} else if !roleBindingMatches(roleBinding, expectedSubjects, expectedRoleRef) {
		// RoleBinding exists but has incorrect bindings - recreate it
		// Note: RoleRef is immutable, so we need to delete and recreate
		podlog.Info("Recreating wiper RoleBinding with correct bindings", "namespace", namespace)
		if err := m.Client.Delete(ctx, roleBinding); err != nil {
			return err
		}
		newRoleBinding := &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      wiperRoleBindingName,
				Namespace: namespace,
			},
			Subjects: expectedSubjects,
			RoleRef:  expectedRoleRef,
		}
		if err := m.Client.Create(ctx, newRoleBinding); err != nil {
			return err
		}
	}

	return nil
}

// roleRulesMatch checks if the actual rules match the expected rules
func roleRulesMatch(actual, expected []rbacv1.PolicyRule) bool {
	if len(actual) != len(expected) {
		return false
	}
	for i, expectedRule := range expected {
		if !policyRuleMatches(actual[i], expectedRule) {
			return false
		}
	}
	return true
}

// policyRuleMatches checks if two policy rules are equivalent
func policyRuleMatches(actual, expected rbacv1.PolicyRule) bool {
	if !stringSlicesEqual(actual.APIGroups, expected.APIGroups) {
		return false
	}
	if !stringSlicesEqual(actual.Resources, expected.Resources) {
		return false
	}
	if !stringSlicesEqual(actual.Verbs, expected.Verbs) {
		return false
	}
	return true
}

// stringSlicesEqual checks if two string slices contain the same elements
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// roleBindingMatches checks if a RoleBinding has the expected subjects and roleRef
func roleBindingMatches(rb *rbacv1.RoleBinding, expectedSubjects []rbacv1.Subject, expectedRoleRef rbacv1.RoleRef) bool {
	if rb.RoleRef.APIGroup != expectedRoleRef.APIGroup ||
		rb.RoleRef.Kind != expectedRoleRef.Kind ||
		rb.RoleRef.Name != expectedRoleRef.Name {
		return false
	}
	if len(rb.Subjects) != len(expectedSubjects) {
		return false
	}
	for i, expected := range expectedSubjects {
		actual := rb.Subjects[i]
		if actual.Kind != expected.Kind ||
			actual.Name != expected.Name ||
			actual.Namespace != expected.Namespace {
			return false
		}
	}
	return true
}

func ensureWiperContainer(containers []corev1.Container) []corev1.Container {
	for _, container := range containers {
		if container.Name == wiperContainerName {
			return containers
		}
	}

	image := os.Getenv(wiperImageEnv)
	if image == "" {
		image = defaultWiperImage
	}

	idleThreshold := os.Getenv(wiperIdleThresholdEnv)
	if idleThreshold == "" {
		idleThreshold = "5m"
	}
	pollInterval := os.Getenv(wiperPollIntervalEnv)
	if pollInterval == "" {
		pollInterval = "30s"
	}
	logEvery := os.Getenv(wiperLogEveryEnv)
	if logEvery == "" {
		logEvery = "1m"
	}
	mainContainer := os.Getenv(wiperMainContainerEnv)
	if mainContainer == "" {
		mainContainer = "dev"
	}

	container := corev1.Container{
		Name:            wiperContainerName,
		Image:           image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Env: []corev1.EnvVar{
			{
				Name: "POD_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"},
				},
			},
			{
				Name: "POD_NAMESPACE",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"},
				},
			},
			{
				Name:  "MAIN_CONTAINER_NAME",
				Value: mainContainer,
			},
			{
				Name:  "IDLE_THRESHOLD",
				Value: idleThreshold,
			},
			{
				Name:  "POLL_INTERVAL",
				Value: pollInterval,
			},
			{
				Name:  "LOG_EVERY",
				Value: logEvery,
			},
		},
	}

	return append(containers, container)
}

func boolPtr(b bool) *bool {
	return &b
}

// enforceSecurityContext sets privileged: false on all containers' security context
func enforceSecurityContext(pod *corev1.Pod) {
	for i := range pod.Spec.Containers {
		if pod.Spec.Containers[i].SecurityContext == nil {
			pod.Spec.Containers[i].SecurityContext = &corev1.SecurityContext{}
		}
		pod.Spec.Containers[i].SecurityContext.Privileged = boolPtr(false)
	}

	for i := range pod.Spec.InitContainers {
		if pod.Spec.InitContainers[i].SecurityContext == nil {
			pod.Spec.InitContainers[i].SecurityContext = &corev1.SecurityContext{}
		}
		pod.Spec.InitContainers[i].SecurityContext.Privileged = boolPtr(false)
	}
}

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

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	shipv1 "github.com/svelama/mutator/api/v1"
)

// nolint:unused
// log is for logging in this package.
var frigatelog = logf.Log.WithName("frigate-resource")

// SetupFrigateWebhookWithManager registers the webhook for Frigate in the manager.
func SetupFrigateWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &shipv1.Frigate{}).
		WithDefaulter(&FrigateCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-ship-svelama-com-v1-frigate,mutating=true,failurePolicy=fail,sideEffects=None,groups=ship.svelama.com,resources=frigates,verbs=create;update,versions=v1,name=mfrigate-v1.kb.io,admissionReviewVersions=v1

// FrigateCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind Frigate when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type FrigateCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind Frigate.
func (d *FrigateCustomDefaulter) Default(_ context.Context, obj *shipv1.Frigate) error {
	frigatelog.Info("Defaulting for Frigate", "name", obj.GetName())

	// TODO(user): fill in your defaulting logic.

	return nil
}

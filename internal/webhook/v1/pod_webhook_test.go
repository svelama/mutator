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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Pod Webhook", func() {
	var (
		obj       *corev1.Pod
		oldObj    *corev1.Pod
		mutator DevWorkspacePodMutator
	)

	BeforeEach(func() {
		obj = &corev1.Pod{}
		oldObj = &corev1.Pod{}
		mutator = DevWorkspacePodMutator{}
		Expect(mutator).NotTo(BeNil(), "Expected mutator to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
	})

	AfterEach(func() {
		// TODO (user): Add any teardown logic common to all tests
	})

	Context("When creating Pod under Defaulting Webhook", func() {
		It("should set the custom ServiceAccount when empty", func() {
			By("setting an empty ServiceAccountName")
			obj.Spec.ServiceAccountName = ""

			By("calling the Default method to apply defaults")
			Expect(mutator.Default(context.Background(), obj)).To(Succeed())

			By("verifying the ServiceAccountName is set to No-Access")
			Expect(obj.Spec.ServiceAccountName).To(Equal("No-Access"))
		})

		It("should replace the default ServiceAccount", func() {
			By("setting the ServiceAccountName to default")
			obj.Spec.ServiceAccountName = "default"

			By("calling the Default method to apply defaults")
			Expect(mutator.Default(context.Background(), obj)).To(Succeed())

			By("verifying the ServiceAccountName is set to No-Access")
			Expect(obj.Spec.ServiceAccountName).To(Equal("No-Access"))
		})

		It("should keep a custom ServiceAccount unchanged", func() {
			By("setting a custom ServiceAccountName")
			obj.Spec.ServiceAccountName = "custom-sa"

			By("calling the Default method to apply defaults")
			Expect(mutator.Default(context.Background(), obj)).To(Succeed())

			By("verifying the ServiceAccountName is unchanged")
			Expect(obj.Spec.ServiceAccountName).To(Equal("custom-sa"))
		})
	})

})

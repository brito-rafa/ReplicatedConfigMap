/*

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
// +kubebuilder:docs-gen:collapse=Apache License

/*
Ideally, we should have one `<kind>_controller_test.go` for each controller scaffolded and called in the `suite_test.go`.
So, let's write our example test for the CronJob controller (`cronjob_controller_test.go.`)
*/

/*
As usual, we start with the necessary imports. We also define some utility variables.
*/
package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	replicationsv1alpha1 "rcm/api/v1alpha1"
)

// +kubebuilder:docs-gen:collapse=Imports

/*
The first step to writing a simple integration test is to actually create an instance of ReplicatedConfigMap.

*/
var _ = Describe("ReplicatedConfigMap controller", func() {

	const (
		rcmName  = "test-rcm"
		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When updating rcm Status", func() {
		It("Should set ReplicatedConfigMap finalizer", func() {
			By("By creating a new ReplicatedConfigMap")
			ctx := context.Background()
			rcmObj := &replicationsv1alpha1.ReplicatedConfigMap{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "replications.example.io/v1alpha1",
					Kind:       "ReplicatedConfigMap",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: rcmName,
				},
				Data: map[string]string{"file": "test"},
			}
			Expect(k8sClient.Create(ctx, rcmObj)).Should(Succeed())

			/*
				After creating this ReplicatedConfigMap, let's check that the Status fields match what we passed in.
				Note that, because the k8s apiserver may not have finished creating a rcm after our `Create()` call from earlier, we will use Gomega’s Eventually() testing function instead of Expect() to give the apiserver an opportunity to finish creating our CronJob.

				`Eventually()` will repeatedly run the function provided as an argument every interval seconds until
				(a) the function’s output matches what’s expected in the subsequent `Should()` call, or
				(b) the number of attempts * interval period exceed the provided timeout value.

				In the examples below, timeout and interval are Go Duration values of our choosing.
			*/

			rcmLookupKey := types.NamespacedName{Name: rcmName, Namespace: ""}
			createdRcm := &replicationsv1alpha1.ReplicatedConfigMap{}

			// We'll need to retry getting this newly created CronJob, given that creation may not immediately happen.
			Eventually(func() bool {
				err := k8sClient.Get(ctx, rcmLookupKey, createdRcm)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			By("By creating a rcm, it has empty MatchedNamespaces")
			Consistently(func() (int, error) {
				err := k8sClient.Get(ctx, rcmLookupKey, createdRcm)
				if err != nil {
					return -1, err
				}
				return len(createdRcm.Status.MatchingNamespaces), nil
			}, duration, interval).Should(Equal(0))
		})
	})

})

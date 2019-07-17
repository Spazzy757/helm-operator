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

package controllers

import (
	"bytes"
	"context"
	"fmt"
	stablev1 "github.com/Spazzy757/helm-operator/api/v1"
	"github.com/go-logr/logr"
	"strings"
	//"io"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	//"k8s.io/apimachinery/pkg/runtime/schema"
	//ref "k8s.io/client-go/tools/reference"
	//"k8s.io/apimachinery/pkg/runtime/schema"
	ref "k8s.io/client-go/tools/reference"
	"os/exec"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// ChartReconciler reconciles a Chart object
type ChartReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=stable.helm.operator.io,resources=charts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=stable.helm.operator.io,resources=charts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=statefulsets;deployment,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets/status;deployment/status,verbs=get;list;watch;create;update;patch;delete
func (r *ChartReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {

	ctx := context.Background()
	log := r.Log.WithValues("chart", req.NamespacedName)
	instance := &stablev1.Chart{}
	// your logic here

	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		log.Error(err, "unable to fetch Chart")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, ignoreNotFound(err)
	}
	getChart(instance)
	yamlString, err := templateChart(instance)
	if err != nil {
		fmt.Println(err)
	}
	resources := bytes.Split(yamlString, []byte(`---`))
	// your logic here
	for _, resource := range resources {
		if strings.Contains(string(resource), "kind") {
			// Decode the YAML to an object.
			u := &unstructured.Unstructured{Object: map[string]interface{}{}}
			if err := yaml.Unmarshal(resource, &u.Object); err != nil {
				fmt.Println(err)
			}
			if err := ctrl.SetControllerReference(instance, u, r.Scheme); err != nil {
				return ctrl.Result{}, err
			}
			u.SetNamespace("default")
			objRef, err := ref.GetReference(r.Scheme, u)
			if err != nil {
				log.Error(err, "unable to make reference to obj", "Object", u.GetName())
			}
			if err := r.Client.Get(ctx, client.ObjectKey{Namespace: u.GetNamespace(), Name: u.GetName()}, u); err != nil {
				// we'll ignore not-found errors, since they can't be fixed by an immediate
				// requeue (we'll need to wait for a new notification), and we can get them
				// on deleted requests.
				if apierrs.IsNotFound(err) {
					if err := r.Create(ctx, u); err != nil {
						log.Error(err, fmt.Sprintf("unable to apply %T", u))
						return ctrl.Result{}, err
					}
					log.V(1).Info(fmt.Sprintf("Applying %T", u))
					if !refInSlice(*objRef, instance.Status.Resource) {
						instance.Status.Resource = append(instance.Status.Resource, *objRef)
					}

				} else {
					log.Error(err, "unable to get object, unknown error occured")
					return ctrl.Result{}, err
				}
			}

		}
	}
	instance.Status.Status = "Deployed"

	if err := r.Status().Update(ctx, instance); err != nil {
		log.Error(err, "unable to update Chart status")
		return ctrl.Result{}, err
	}
	log.V(1).Info("reconciling the configmap")
	return ctrl.Result{}, nil

}

func (r *ChartReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&stablev1.Chart{}).
		Complete(r)
}

//func ignoreNotFound(err error) error {
//	if apierrs.IsNotFound(err) {
//		return nil
//	}
//	return err
//}
func getChart(c *stablev1.Chart) {
	exec.Command("helm", "repo", "update")
	cmd := exec.Command("helm", "fetch", "--untar", "--untardir=.", "stable/"+c.Spec.Chart)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
		return
	}
	fmt.Println("Pulled Chart: " + c.Spec.Chart)
}

// template helm
func templateChart(c *stablev1.Chart) ([]byte, error) {
	cmd := exec.Command("helm", "template", "--name=test-chart", "--namespace=default", c.Spec.Chart)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
		return nil, err
	}
	return out.Bytes(), nil
}

// Store in Configmaps
// run the configs that have been generated
// Store status of helm deployment

func ignoreNotFound(err error) error {
	if apierrs.IsNotFound(err) {
		return nil
	}
	return err
}

func refInSlice(a corev1.ObjectReference, list []corev1.ObjectReference) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

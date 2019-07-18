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
	"os"
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

var ctx = context.Background()

// +kubebuilder:rbac:groups=stable.helm.operator.io,resources=charts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=stable.helm.operator.io,resources=charts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=statefulsets;deployment,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets/status;deployment/status,verbs=get;list;watch;create;update;patch;delete
func (r *ChartReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {

	log := r.Log.WithValues("chart", req.NamespacedName)
	instance := &stablev1.Chart{}
	finalizer := "helm.operator.finalizer.io"
	forGroundFinalizer := "foregroundDeletion"
	// your logic here

	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		return ctrl.Result{}, ignoreNotFound(err)
	}
	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		if !containsString(instance.ObjectMeta.Finalizers, finalizer) {
			instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, finalizer)
			if err := r.Update(context.Background(), instance); err != nil {
				return ctrl.Result{}, err
			}
		}
		if err := getChart(instance); err != nil {
			return ctrl.Result{}, err
		}
		yamlString, err := templateChart(instance)
		if err != nil {
			return ctrl.Result{}, err
		}
		resources := bytes.Split(yamlString, []byte(`---`))
		// your logic here
		for _, resource := range resources {
			// Helm sometimes templates just comments so skip these
			if !strings.Contains(string(resource), "kind") {
				continue
			}
			// Decode the YAML to an object.
			u := &unstructured.Unstructured{Object: map[string]interface{}{}}
			if err := yaml.Unmarshal(resource, &u.Object); err != nil {
				fmt.Println(err)
			}
			// set controller reference
			if err := ctrl.SetControllerReference(instance, u, r.Scheme); err != nil {
				return ctrl.Result{}, err
			}

			// set namespace of the resource (by default helm does not template this out)
			u.SetNamespace(instance.Spec.NameSpaceSelector)
			// Get the reference of the resource to attach to the chart instance
			objRef, err := ref.GetReference(r.Scheme, u)
			if err != nil {
				log.Error(err, "unable to make reference", "Object", u.GetName())
			}
			// Get Key to fetch resource if exists
			key, err := client.ObjectKeyFromObject(u)
			if err != nil {
				return ctrl.Result{}, err
			}

			// Get resource
			if err := r.Client.Get(ctx, key, u); err != nil {
				// if error is anything but is not found, return error
				if !apierrs.IsNotFound(err) {
					log.Error(err, "unable to get object, unknown error occured")
					return ctrl.Result{}, err
				}

				// set finalizer of resource
				u.SetFinalizers([]string{forGroundFinalizer})

				// Create Object
				if err := r.Create(ctx, u); err != nil {
					log.Error(err, fmt.Sprintf("unable to apply %v", u.GroupVersionKind()))
					instance.Status.Status = "Failed"
					if err := r.UpdateStatus(instance); err != nil {
						return ctrl.Result{}, err
					}
					return ctrl.Result{}, err
				}
				log.V(1).Info(fmt.Sprintf("Applying: %v", u.GroupVersionKind()))

				// Check if resource reference is attached to instance, if not add it
				if !refInSlice(*objRef, instance.Status.Resource) {
					instance.Status.Resource = append(instance.Status.Resource, *objRef)
					if err := r.UpdateStatus(instance); err != nil {
						return ctrl.Result{}, err
					}
				}
				continue
			}
			continue
			// Implement Patch if resource already exist
			//log.V(1).Info(fmt.Sprintf("Updating: %v", u.GroupVersionKind()))
		}

		instance.Status.Status = "Deployed"
		if err := r.UpdateStatus(instance); err != nil {
			return ctrl.Result{}, err
		}
		log.V(1).Info("reconciling the Chart")
		return ctrl.Result{}, nil
	} else {
		if containsString(instance.ObjectMeta.Finalizers, finalizer) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.deleteExternalResources(instance); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return ctrl.Result{}, err
			}

			// remove our finalizer from the list and update it.
			instance.ObjectMeta.Finalizers = removeString(instance.ObjectMeta.Finalizers, finalizer)
			if err := r.Update(context.Background(), instance); err != nil {
				return ctrl.Result{}, err
			}
		}
	}
	return ctrl.Result{}, nil
}

// Deletes all resources attached to the instance
func (r *ChartReconciler) deleteExternalResources(instance *stablev1.Chart) error {
	for _, resource := range instance.Status.Resource {
		u := &unstructured.Unstructured{}
		u.Object = map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      resource.Name,
				"namespace": resource.Namespace,
			},
		}
		key, err := client.ObjectKeyFromObject(u)
		if err != nil {
			return err
		}
		u.SetGroupVersionKind(resource.GroupVersionKind())
		if err := r.Get(ctx, key, u); err != nil {
			return err
		}
		if err := r.Delete(ctx, u); err != nil {
			return err
		}
	}
	return nil
}

func (r *ChartReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&stablev1.Chart{}).
		Complete(r)
}

// Updates the status of the instance on the kube api server
func (r *ChartReconciler) UpdateStatus(c *stablev1.Chart) error {
	if err := r.Status().Update(ctx, c); err != nil {
		return err
	}
	return nil
}

// Fetch the chart specified on the instance
func getChart(c *stablev1.Chart) error {
	exec.Command("helm", "repo", "update")
	if err := os.MkdirAll("chart/"+c.Spec.Version, os.ModePerm); err != nil {
		fmt.Printf("%v\n", err)
	}
	cmd := exec.Command("helm",
		"fetch",
		"--untar",
		"--version="+c.Spec.Version,
		"--untardir=chart/"+c.Spec.Version,
		c.Spec.Repo+"/"+c.Spec.Chart)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
		return err
	}
	return nil
}

// template out the yaml files from the chart
func templateChart(c *stablev1.Chart) ([]byte, error) {
	values := buildValuesString(c)
	cmd := exec.Command("helm",
		"template",
		"--name="+c.GetName(),
		"--set="+values,
		"--namespace="+c.Spec.NameSpaceSelector,
		"chart/"+c.Spec.Version+"/"+c.Spec.Chart)
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

// Builds a string representation of the values on the instance
func buildValuesString(c *stablev1.Chart) string {
	var buildString string
	for _, valuePair := range c.Spec.Values {
		buildString += valuePair.Name + "=" + valuePair.Value + ","
	}
	if last := len(buildString) - 1; last >= 0 && buildString[last] == ',' {
		buildString = buildString[:last]
	}
	return buildString
}

// Ignores not found error
func ignoreNotFound(err error) error {
	if apierrs.IsNotFound(err) {
		return nil
	}
	return err
}

// Checks if an object ref is in a slice of object refs
func refInSlice(a corev1.ObjectReference, list []corev1.ObjectReference) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// Checks if string in a slice of strings
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// Removes string from a slice of strings
func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

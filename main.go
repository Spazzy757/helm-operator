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

package main

import (
	"flag"
	"os"

	helmv1 "github.com/Spazzy757/helm-operator/api/v1"
	"github.com/Spazzy757/helm-operator/controllers"
	appsv1 "k8s.io/api/apps/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	auditregistrationv1alpha1 "k8s.io/api/auditregistration/v1alpha1"
	authenticationv1 "k8s.io/api/authentication/v1"
	authenticationv1beta1 "k8s.io/api/authentication/v1beta1"
	authorizationv1 "k8s.io/api/authorization/v1"
	authorizationv1beta1 "k8s.io/api/authorization/v1beta1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	autoscalingv2beta1 "k8s.io/api/autoscaling/v2beta1"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	batchv1 "k8s.io/api/batch/v1"
	kbatchv1beta1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	batchv2alpha1 "k8s.io/api/batch/v2alpha1"
	certificatesv1beta1 "k8s.io/api/certificates/v1beta1"
	coordinationv1 "k8s.io/api/coordination/v1"
	coordinationv1beta1 "k8s.io/api/coordination/v1beta1"
	corev1 "k8s.io/api/core/v1"
	eventsv1beta1 "k8s.io/api/events/v1beta1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	nodev1alpha1 "k8s.io/api/node/v1alpha1"
	nodev1beta1 "k8s.io/api/node/v1beta1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	rbacv1alpha1 "k8s.io/api/rbac/v1alpha1"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	schedulingv1alpha1 "k8s.io/api/scheduling/v1alpha1"
	schedulingv1beta1 "k8s.io/api/scheduling/v1beta1"
	settingsv1alpha1 "k8s.io/api/settings/v1alpha1"
	storagev1 "k8s.io/api/storage/v1"
	storagev1alpha1 "k8s.io/api/storage/v1alpha1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	kbatchv1beta1.AddToScheme(scheme)
	helmv1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)
	appsv1.AddToScheme(scheme)
	appsv1beta1.AddToScheme(scheme)
	appsv1beta2.AddToScheme(scheme)
	auditregistrationv1alpha1.AddToScheme(scheme)
	authenticationv1.AddToScheme(scheme)
	authenticationv1beta1.AddToScheme(scheme)
	authorizationv1.AddToScheme(scheme)
	authorizationv1beta1.AddToScheme(scheme)
	autoscalingv1.AddToScheme(scheme)
	autoscalingv2beta1.AddToScheme(scheme)
	autoscalingv2beta2.AddToScheme(scheme)
	batchv1.AddToScheme(scheme)
	batchv1beta1.AddToScheme(scheme)
	batchv2alpha1.AddToScheme(scheme)
	certificatesv1beta1.AddToScheme(scheme)
	coordinationv1.AddToScheme(scheme)
	coordinationv1beta1.AddToScheme(scheme)
	eventsv1beta1.AddToScheme(scheme)
	extensionsv1beta1.AddToScheme(scheme)
	networkingv1.AddToScheme(scheme)
	networkingv1beta1.AddToScheme(scheme)
	nodev1alpha1.AddToScheme(scheme)
	nodev1beta1.AddToScheme(scheme)
	policyv1beta1.AddToScheme(scheme)
	rbacv1.AddToScheme(scheme)
	rbacv1alpha1.AddToScheme(scheme)
	rbacv1beta1.AddToScheme(scheme)
	schedulingv1.AddToScheme(scheme)
	schedulingv1alpha1.AddToScheme(scheme)
	schedulingv1beta1.AddToScheme(scheme)
	settingsv1alpha1.AddToScheme(scheme)
	storagev1.AddToScheme(scheme)
	storagev1alpha1.AddToScheme(scheme)
	storagev1beta1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	ctrl.SetLogger(zap.Logger(true))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		LeaderElection:     enableLeaderElection,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	err = (&controllers.ChartReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Chart"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Chart")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

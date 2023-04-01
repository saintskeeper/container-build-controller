/*
Copyright 2023.

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

package buildtoolkitcloudmasondev

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	buildtoolkitcloudmasondevv1 "github.com/saintskeeper/container-build-controller/apis/build.toolkit.cloudmason.dev/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildDefinitionReconciler reconciles a BuildDefinition object
type BuildDefinitionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

//+kubebuilder:rbac:groups=build.toolkit.cloudmason.dev,resources=builddefinitions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=build.toolkit.cloudmason.dev,resources=builddefinitions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=build.toolkit.cloudmason.dev,resources=builddefinitions/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the BuildDefinition object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *BuildDefinitionReconciler) CreateContainer(ctx context.Context, log logr.Logger, buildDefinition *buildtoolkitcloudmasondevv1.BuildDefinition) (*batchv1.Job, error) {
	// Default builder image
	defaultBuilderImage := "gcr.io/kaniko-project/executor:latest"
	args := append(buildDefinition.Spec.Args, "executor")

	// check to see if hte builder is defined
	builderImage := buildDefinition.Spec.BuilderImage
	if builderImage == "" {
		builderImage = defaultBuilderImage
	}

	jobSpec := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildDefinition.Name + "-build-job",
			Namespace: buildDefinition.Namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyOnFailure,
					Containers: []corev1.Container{
						{
							Name:  "build-and-push",
							Image: builderImage,
							Args:  args,
							Env: []corev1.EnvVar{
								{Name: "GIT_REPOSITORY", Value: buildDefinition.Spec.GitRepository},
								{Name: "CONTEXT_PATH", Value: buildDefinition.Spec.ContextPath},
								{Name: "REGISTRY", Value: buildDefinition.Spec.Registry},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "docker-config", MountPath: "/kaniko/.docker/"},
							},
						},
					},

					Volumes: []corev1.Volume{
						{
							Name: "docker-config",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: buildDefinition.Spec.RegistrySecret,
								},
							},
						},
					},
				},
			},
		},
	}

	if err := r.Create(ctx, jobSpec); err != nil {
		log.Error(err, "unable to create Kubernetes Job")
		return nil, client.IgnoreNotFound(err)
	}

	return jobSpec, nil
}
func (r *BuildDefinitionReconciler) Reconcile(_ context.Context, req ctrl.Request) (ctrl.Result, error) {
	var buildDefinition buildtoolkitcloudmasondevv1.BuildDefinition
	if err := r.Get(context.TODO(), req.NamespacedName, &buildDefinition); err != nil {
		log := r.Log.WithValues("BuildDefinition", req.NamespacedName)
		log.Error(err, "unable to build JobSpec")

		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	jobSpec, err := r.CreateContainer(context.TODO(), r.Log, &buildDefinition)
	if err != nil {
		r.Log.Error(err, "unable to build JobSpec")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := r.Create(context.TODO(), jobSpec); err != nil {
		r.Log.Error(err, "unable to create Kubernetes Job")
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BuildDefinitionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&buildtoolkitcloudmasondevv1.BuildDefinition{}).
		Complete(r)
}

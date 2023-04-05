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
	"fmt"
	"math/rand"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	buildtoolkitcloudmasondevv1 "github.com/saintskeeper/container-build-controller/apis/build.toolkit.cloudmason.dev/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Constants
const jobOwnerKey = ".metadata.controller"
const apiGVStr = "build.toolkit.cloudmason.dev/v1"

// Numbers
var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

// BuildDefinitionReconciler reconciles a BuildDefinition object
type BuildDefinitionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

func NewBuildDefinitionReconciler(client client.Client, scheme *runtime.Scheme) *BuildDefinitionReconciler {
	return &BuildDefinitionReconciler{
		Client: client,
		Scheme: scheme,
		Log:    ctrl.Log.WithName("controllers").WithName("BuildDefinition"),
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func randString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
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
func (r *BuildDefinitionReconciler) CreateContainer(ctx context.Context, buildDefinition *buildtoolkitcloudmasondevv1.BuildDefinition) (*batchv1.Job, error) {
	// Default builder image

	if buildDefinition.ObjectMeta.Namespace == "" || buildDefinition.ObjectMeta.Name == "" {
		return nil, fmt.Errorf("buildDefinition metadata is invalid")
	}

	if buildDefinition.Spec.GitRepository == "" || buildDefinition.Spec.ContextPath == "" || buildDefinition.Spec.Registry == "" {
		return nil, fmt.Errorf("invalid values for GitRepository, ContextPath, or Registry")
	}

	// Get the dockerfilePath from the BuildDefinition object or use the default value

	dockerfilePath := buildDefinition.Spec.ContextPath + "/Dockerfile"

	// Use the provided build file path if not empty
	if buildDefinition.Spec.BuildFile != "" {
		dockerfilePath = buildDefinition.Spec.ContextPath + "/" + buildDefinition.Spec.BuildFile
	}

	defaultBuilderImage := "gcr.io/kaniko-project/executor:latest"
	args := append(buildDefinition.Spec.Args,
		"--context=git://"+strings.TrimPrefix(buildDefinition.Spec.GitRepository, "https://"),
		"--dockerfile="+dockerfilePath,
	)

	// check to see if hte builder is defined
	builderImage := buildDefinition.Spec.BuilderImage
	if builderImage == "" {
		builderImage = defaultBuilderImage
	}

	// lets create a function to create a random 8 chracter string only using numbers and lowercase letters

	jobSpec := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildDefinition.Name + "-build-" + randString(8),
			Namespace: buildDefinition.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: buildDefinition.APIVersion,
					Kind:       buildDefinition.Kind,
					Name:       buildDefinition.Name,
					UID:        buildDefinition.UID,
				},
			},
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
								{Name: "DOCKERFILE_PATH", Value: dockerfilePath},
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

	/*	if err := r.Create(ctx, jobSpec); err != nil {
		r.Log.Error(err, "unable to create Kubernetes Job")
		return nil, client.IgnoreNotFound(err)
	}*/

	return jobSpec, nil
}

func (r *BuildDefinitionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log.Info("Debug: Checking the state of BuildDefinitionReconciler struct", "r", r)
	var buildDefinition buildtoolkitcloudmasondevv1.BuildDefinition
	if err := r.Get(ctx, req.NamespacedName, &buildDefinition); err != nil {
		r.Log.Error(err, "unable to build JobSpec", "BuildDefinition", req.NamespacedName)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	r.Log.Info("Debug: Successfully fetched BuildDefinition", "buildDefinition", buildDefinition)
	if buildDefinition.ObjectMeta.Namespace == "" || buildDefinition.ObjectMeta.Name == "" {
		r.Log.Error(fmt.Errorf("buildDefinition metadata is invalid"), "unable to build JobSpec", "BuildDefinition", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	// Delete any existing Jobs with the same OwnerReference
	err := r.DeleteAllOf(ctx, &batchv1.Job{}, client.InNamespace(req.Namespace), client.MatchingFields{jobOwnerKey: req.Name})
	if err != nil {
		r.Log.Error(err, "failed to delete old Jobs")
		return ctrl.Result{}, err
	}

	// Create a new Job
	jobSpec, err := r.CreateContainer(ctx, &buildDefinition)
	if err != nil {
		r.Log.Error(err, "unable to build JobSpec")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	r.Log.Info("Debug: Successfully created JobSpec", "jobSpec", jobSpec)
	if err := r.Create(ctx, jobSpec); err != nil {
		r.Log.Error(err, "unable to create Kubernetes Job")
		return ctrl.Result{}, nil
	}

	r.Log.Info("Debug: Successfully created Kubernetes Job")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BuildDefinitionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &batchv1.Job{}, jobOwnerKey, func(rawObj client.Object) []string {
    job := rawObj.(*batchv1.Job)
    owner := metav1.GetControllerOf(job)
    if owner == nil || owner.APIVersion != apiGVStr || owner.Kind != "BuildDefinition" {
        return nil
    }
    return []string{owner.Name}
}); err != nil {
    return err
}

		// grab the job object, extract the owner...
		job := rawObj.(*batchv1.Job)
		owner := metav1.GetControllerOf(job)
		if owner == nil {
			return nil
		}
		// ...make sure it's a BuildDefinition...
		if owner.APIVersion != apiGVStr || owner.Kind != "BuildDefinition" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&buildtoolkitcloudmasondevv1.BuildDefinition{}).
		Owns(&batchv1.Job{}). // Add this line to watch owned Jobs
		Complete(r)
}

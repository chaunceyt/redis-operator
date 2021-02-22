/*
Copyright 2021.

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
	"fmt"
	"reflect"
	"text/template"
	"time"

	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	cachev1alpha1 "github.com/chaunceyt/redis-operator/api/v1alpha1"
)

// RedisReconciler reconciles a Redis object
type RedisReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=redis.module.operator.io,resources=redis,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=credis.module.operator.io,resources=redis/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=redis.module.operator.io,resources=redis/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Redis object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *RedisReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("redis", req.NamespacedName)

	redis := &cachev1alpha1.Redis{}
	err := r.Get(ctx, req.NamespacedName, redis)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Redis resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Redis")
		return ctrl.Result{}, err
	}

	found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: redis.Name, Namespace: redis.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Define a new deployment
		dep := r.deploymentForRedis(redis)
		log.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.Create(ctx, dep)
		if err != nil {
			log.Error(err, "Failed to create new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}

	// Standalone Redis - ensure the deployment size is always 1
	size := int32(1)
	if *found.Spec.Replicas != size {
		found.Spec.Replicas = &size
		err = r.Update(ctx, found)
		if err != nil {
			log.Error(err, "Failed to update Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
			return ctrl.Result{}, err
		}
		// Spec updated - return and requeue
		return ctrl.Result{Requeue: true}, nil
	}

	configmap := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{Name: redis.Name, Namespace: redis.Namespace}, configmap)
	if err != nil && errors.IsNotFound(err) {
		// Define a new deployment
		cm := r.redisConfigMapForRedis(redis)
		log.Info("Creating a new ConfigMap", "ConfigMap.Namespace", cm.Namespace, "Configmap.Name", cm.Name)
		err = r.Create(ctx, cm)
		if err != nil {
			log.Error(err, "Failed to create new ConfigMap", "ConfigMap.Namespace", cm.Namespace, "Configmap.Name", cm.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}

	desiredConfigMap := r.redisConfigMapForRedis(redis)
	if !reflect.DeepEqual(configmap.Data, desiredConfigMap.Data) {
		err := r.Client.Update(context.Background(), desiredConfigMap)
		if err != nil {
			log.Error(err, "Failed to update Configmap")
			return ctrl.Result{}, err
		}

		// Since we updated the configMap. Trigger a restart of the pod.
		// update annotation configMapHash
		found.Spec.Template.Annotations = map[string]string{
			"configMapHash": time.Now().String(),
		}

		// Update deployment
		desiredDeploy := found
		errDeploy := r.Client.Update(context.Background(), desiredDeploy)
		if errDeploy != nil {
			log.Error(errDeploy, "Failed to update Deployment")
			return ctrl.Result{}, errDeploy
		}

		return ctrl.Result{Requeue: true}, nil
	}

	// Update the Redis status with the pod names
	// List the pods for this redis's deployment
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(redis.Namespace),
		client.MatchingLabels(labelsForRedis(redis, "deployment")),
	}
	if err = r.List(ctx, podList, listOpts...); err != nil {
		log.Error(err, "Failed to list pods", "Redis.Namespace", redis.Namespace, "Redis.Name", redis.Name)
		return ctrl.Result{}, err
	}
	podNames := getPodNames(podList.Items)

	// Update status.Nodes if needed
	if !reflect.DeepEqual(podNames, redis.Status.Nodes) {
		redis.Status.Nodes = podNames
		err := r.Status().Update(ctx, redis)
		if err != nil {
			log.Error(err, "Failed to update Memcached status")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// deploymentForMemcached returns a memcached Deployment object
// https://blog.questionable.services/article/kubernetes-deployments-configmap-change/
func (r *RedisReconciler) deploymentForRedis(redis *cachev1alpha1.Redis) *appsv1.Deployment {
	containerImage := "chaunceyt/custom-redis:v0.0.1"
	ls := labelsForRedis(redis, "deployment")
	replicas := int32(1)
	command := []string{"redis-server", "/opt/redis/redis.conf"}

	passwordParam := ""
	password := redis.Spec.GlobalConfig.Password

	if password != "" {
		passwordParam = " -a " + password
	}
	probeCmd := []string{"sh", "-c", "redis-cli", passwordParam, "ping"}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      redis.Name,
			Namespace: redis.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
					Annotations: map[string]string{
						"configMapHash": time.Now().String(),
					},
				},
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken: createBool(false),
					Containers: []corev1.Container{{
						Image:           containerImage,
						ImagePullPolicy: corev1.PullPolicy("Always"),
						Name:            "redis",
						Command:         command,
						Resources:       redis.Spec.GlobalConfig.Resources,
						SecurityContext: &corev1.SecurityContext{
							AllowPrivilegeEscalation: createBool(false),
							ReadOnlyRootFilesystem:   createBool(true),
							RunAsNonRoot:             createBool(true),
							RunAsUser:                createInt64(1000),
							RunAsGroup:               createInt64(1000),
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{
									"ALL",
								},
							},
						},
						ReadinessProbe: &corev1.Probe{
							InitialDelaySeconds: 15,
							TimeoutSeconds:      5,
							Handler: corev1.Handler{
								Exec: &corev1.ExecAction{
									Command: probeCmd,
								},
							},
						},
						LivenessProbe: &corev1.Probe{
							InitialDelaySeconds: 15,
							TimeoutSeconds:      5,
							Handler: corev1.Handler{
								Exec: &corev1.ExecAction{
									Command: probeCmd,
								},
							},
						},
						Env: []corev1.EnvVar{
							{
								Name: "POD_NAME",
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "metadata.name",
									},
								},
							},
							{
								Name: "POD_ID",
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "metadata.uid",
									},
								},
							},
							{
								Name: "POD_NAMESPACE",
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "metadata.namespace",
									},
								},
							},
							{
								Name: "POD_IP",
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "status.podIP",
									},
								},
							},
							{
								Name: "NODE_NAME",
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "spec.nodeName",
									},
								},
							},
							{
								Name: "SERVICE_ACCOUNT",
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "spec.serviceAccountName",
									},
								},
							},
							{
								Name: "CONTAINER_CPU_REQUEST_MILLICORES",
								ValueFrom: &corev1.EnvVarSource{
									ResourceFieldRef: &corev1.ResourceFieldSelector{
										Resource: "requests.cpu",
										Divisor:  resource.MustParse("1"),
									},
								},
							},
							{
								Name: "CONTAINER_MEMORY_REQUEST_MILLICORES",
								ValueFrom: &corev1.EnvVarSource{
									ResourceFieldRef: &corev1.ResourceFieldSelector{
										Resource: "requests.memory",
										Divisor:  resource.MustParse("1Mi"),
									},
								},
							},
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "redis-conf",
								MountPath: "/opt/redis/redis.conf",
								SubPath:   "redis.conf",
							},
							{
								Name:      "log-dir",
								MountPath: "/var/log/redis-app",
							},
						},
						Ports: []corev1.ContainerPort{{
							ContainerPort: 6379,
							Name:          "redis",
						}},
					}},
					Volumes: []corev1.Volume{
						{
							Name: "redis-conf",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: redis.Name,
									},
									DefaultMode: createInt32(0644),
								},
							},
						},
						{
							Name: "log-dir",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}
	// Set Memcached instance as the owner and controller
	ctrl.SetControllerReference(redis, dep, r.Scheme)
	return dep
}

// redisConfigMapForWebproject - configmap that contains configuration for redis caching.
func (r *RedisReconciler) redisConfigMapForRedis(redis *cachev1alpha1.Redis) *corev1.ConfigMap {
	// create template for redis.
	redisConfigTemplate := `loglevel notice
logfile /dev/stdout
timeout 300
tcp-keepalive 0
databases 1
maxmemory 2mb
maxmemory-policy allkeys-lru
`

	tmpl, err := template.New("redis").Parse(redisConfigTemplate)
	if err != nil {
		panic(err)
	}

	var tplOutput bytes.Buffer
	if err := tmpl.Execute(&tplOutput, redis); err != nil {
		panic(err)
	}

	redisConfigFileContent := tplOutput.String()

	// Enable Redis password support
	if redis.Spec.GlobalConfig.Password != "" {
		redisConfigFileContent = fmt.Sprintf("%srequirepass %s\n", redisConfigFileContent, redis.Spec.GlobalConfig.Password)
	}

	// Enable redisgraph
	// https://github.com/RedisGraph/RedisGraph
	if redis.Spec.RedisGraph.Enabled {
		redisConfigFileContent = fmt.Sprintf("%sloadmodule %s\n", redisConfigFileContent, "/usr/lib/redis/modules/redisgraph.so")
	}

	// Enable redisjson
	// https://github.com/RedisJSON/RedisJSON
	if redis.Spec.RedisJSON.Enabled {
		redisConfigFileContent = fmt.Sprintf("%sloadmodule %s\n", redisConfigFileContent, "/usr/lib/redis/modules/rejson.so")
	}

	// Enable redisSearch
	// https://github.com/RediSearch/RediSearch
	if redis.Spec.RedisSearch.Enabled {
		redisConfigFileContent = fmt.Sprintf("%sloadmodule %s\n", redisConfigFileContent, "/usr/lib/redis/modules/redisearch.so")
	}

	dep := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      redis.Name,
			Namespace: redis.Namespace,
			Labels:    labelsForRedis(redis, "configmap"),
		},
		Data: map[string]string{
			"redis.conf": redisConfigFileContent,
		},
	}

	controllerutil.SetControllerReference(redis, dep, r.Scheme)
	return dep
}

// SetupWithManager sets up the controller with the Manager.
func (r *RedisReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1alpha1.Redis{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}

// helper functions
func labelsForRedis(redis *cachev1alpha1.Redis, role string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":      redis.Name,
		"app.kubernetes.io/component": role,
		"provider":                    "redis-operator",
	}
}

// getPodNames returns the pod names of the array of pods passed in
func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}

func createInt32(x int32) *int32 {
	return &x
}

func createBool(x bool) *bool {
	return &x
}

func createInt64(i int64) *int64 {
	return &i
}

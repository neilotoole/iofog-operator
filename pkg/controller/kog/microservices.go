/*
 *  *******************************************************************************
 *  * Copyright (c) 2019 Edgeworx, Inc.
 *  *
 *  * This program and the accompanying materials are made available under the
 *  * terms of the Eclipse Public License v. 2.0 which is available at
 *  * http://www.eclipse.org/legal/epl-2.0
 *  *
 *  * SPDX-License-Identifier: EPL-2.0
 *  *******************************************************************************
 *
 */

package kog

import (
	"errors"
	"fmt"
	iofogv1 "github.com/eclipse-iofog/iofog-operator/pkg/apis/iofog/v1"
	"k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
	"strconv"
	"strings"
)

func getConnectorNamePrefix() string {
	return "connector-"
}

func prefixConnectorName(name string) string {
	return "connector-" + name
}

func removeConnectorNamePrefix(name string) string {
	pos := strings.Index(name, "-")
	if pos == -1 || pos >= len(name)-1 {
		return name
	}
	return name[pos+1:]
}

type microservice struct {
	name            string
	loadBalancerIP  string
	serviceType     string
	trafficPolicy   string
	imagePullSecret string
	ports           []int
	replicas        int32
	containers      []container
	labels          map[string]string
	annotations     map[string]string
	secrets         []v1.Secret
	volumes         []v1.Volume
	rbacRules       []rbacv1.PolicyRule
}

type container struct {
	name            string
	image           string
	imagePullPolicy string
	args            []string
	livenessProbe   *v1.Probe
	readinessProbe  *v1.Probe
	env             []v1.EnvVar
	command         []string
	ports           []v1.ContainerPort
	resources       v1.ResourceRequirements
	volumeMounts    []v1.VolumeMount
}

func newControllerMicroservice(replicas int32, image, imagePullSecret string, db *iofogv1.Database, svcType, loadBalancerIP string) *microservice {
	if replicas == 0 {
		replicas = 1
	}
	return &microservice{
		name: "controller",
		labels: map[string]string{
			"name": "controller",
		},
		ports: []int{
			51121,
			80,
		},
		imagePullSecret: imagePullSecret,
		replicas:        replicas,
		serviceType:     svcType,
		trafficPolicy:   getTrafficPolicy(svcType),
		loadBalancerIP:  loadBalancerIP,
		containers: []container{
			{
				name:            "controller",
				image:           image,
				imagePullPolicy: "Always",
				readinessProbe: &v1.Probe{
					Handler: v1.Handler{
						HTTPGet: &v1.HTTPGetAction{
							Path: "/api/v3/status",
							Port: intstr.FromInt(51121),
						},
					},
					InitialDelaySeconds: 1,
					PeriodSeconds:       4,
					FailureThreshold:    3,
				},
				env: []v1.EnvVar{
					{
						Name:  "DB_PROVIDER",
						Value: db.Provider,
					},
					{
						Name:  "DB_NAME",
						Value: db.DatabaseName,
					},
					{
						Name:  "DB_USERNAME",
						Value: db.User,
					},
					{
						Name:  "DB_PASSWORD",
						Value: db.Password,
					},
					{
						Name:  "DB_HOST",
						Value: db.Host,
					},
					{
						Name:  "DB_PORT",
						Value: strconv.Itoa(db.Port),
					},
				},
				resources: v1.ResourceRequirements{
					Limits: v1.ResourceList{
						"cpu":    resource.MustParse("1800m"),
						"memory": resource.MustParse("3Gi"),
					},
					Requests: v1.ResourceList{
						"cpu":    resource.MustParse("400m"),
						"memory": resource.MustParse("1Gi"),
					},
				},
			},
		},
	}
}

func newConnectorMicroservice(image, svcType string) *microservice {
	return &microservice{
		name: "connector",
		labels: map[string]string{
			"name": "connector",
		},
		ports: []int{
			8080,
			6000, 6001, 6002, 6003, 6004, 6005, 6006, 6007, 6008, 6009,
			6010, 6011, 6012, 6013, 6014, 6015, 6016, 6017, 6018, 6019,
			6020, 6021, 6022, 6023, 6024, 6025, 6026, 6027, 6028, 6029,
			6030, 6031, 6032, 6033, 6034, 6035, 6036, 6037, 6038, 6039,
			6040, 6041, 6042, 6043, 6044, 6045, 6046, 6047, 6048, 6049,
			6050,
		},
		replicas:      1,
		serviceType:   svcType,
		trafficPolicy: getTrafficPolicy(svcType),
		containers: []container{
			{
				name:            "connector",
				image:           image,
				imagePullPolicy: "Always",
				resources: v1.ResourceRequirements{
					Limits: v1.ResourceList{
						"cpu":    resource.MustParse("200m"),
						"memory": resource.MustParse("1Gi"),
					},
					Requests: v1.ResourceList{
						"cpu":    resource.MustParse("50m"),
						"memory": resource.MustParse("200Mi"),
					},
				},
			},
		},
	}
}

func getKubeletToken(containers []corev1.Container) (token string, err error) {
	if len(containers) != 1 {
		err = errors.New(fmt.Sprintf("Expected 1 container in Kubelet deployment config. Found %d", len(containers)))
		return
	}
	if len(containers[0].Args) != 6 {
		err = errors.New(fmt.Sprintf("Expected 6 args in Kubelet deployment config. Found %d", len(containers[0].Args)))
		return
	}
	token = containers[0].Args[3]
	return
}

func newKubeletMicroservice(image, namespace, token, controllerEndpoint string) *microservice {
	return &microservice{
		name: "kubelet",
		labels: map[string]string{
			"name": "kubelet",
		},
		ports:    []int{60000},
		replicas: 1,
		containers: []container{
			{
				name:            "kubelet",
				image:           image,
				imagePullPolicy: "Always",
				args: []string{
					"--namespace",
					namespace,
					"--iofog-token",
					token,
					"--iofog-url",
					fmt.Sprintf("http://%s", controllerEndpoint),
				},
				resources: v1.ResourceRequirements{
					Limits: v1.ResourceList{
						"cpu":    resource.MustParse("200m"),
						"memory": resource.MustParse("1Gi"),
					},
					Requests: v1.ResourceList{
						"cpu":    resource.MustParse("50m"),
						"memory": resource.MustParse("200Mi"),
					},
				},
			},
		},
	}
}

func newPortManagerMicroservice(image, watchNamespace, iofogUserEmail, iofogUserPass string) *microservice {
	return &microservice{
		name: "port-manager",
		labels: map[string]string{
			"name": "port-manager",
		},
		replicas: 1,
		containers: []container{
			{
				name:            "port-manager",
				image:           image,
				imagePullPolicy: "Always",
				readinessProbe: &v1.Probe{
					Handler: v1.Handler{
						Exec: &v1.ExecAction{
							Command: []string{
								"stat",
								"/tmp/operator-sdk-ready",
							},
						},
					},
					InitialDelaySeconds: 4,
					PeriodSeconds:       10,
					FailureThreshold:    1,
				},
				resources: v1.ResourceRequirements{
					Limits: v1.ResourceList{
						"cpu":    resource.MustParse("200m"),
						"memory": resource.MustParse("1Gi"),
					},
					Requests: v1.ResourceList{
						"cpu":    resource.MustParse("50m"),
						"memory": resource.MustParse("200Mi"),
					},
				},
				env: []v1.EnvVar{
					{
						Name:  "WATCH_NAMESPACE",
						Value: watchNamespace,
					},
					{
						Name:  "IOFOG_USER_EMAIL",
						Value: iofogUserEmail,
					},
					{
						Name:  "IOFOG_USER_PASS",
						Value: iofogUserPass,
					},
				},
			},
		},
	}
}

func newSkupperMicroservice(image, volumeMountPath string) *microservice {
	return &microservice{
		name: "skupper",
		labels: map[string]string{
			"name":                 "skupper",
			"application":          "skupper-router",
			"skupper.io/component": "router",
		},
		annotations: map[string]string{
			"prometheus.io/port":   "9090",
			"prometheus.io/scrape": "true",
		},
		ports: []int{
			5671,  // amqps
			9090,  // http
			55671, // interior
			45671, // edge
		},
		replicas:      1,
		serviceType:   "LoadBalancer",
		trafficPolicy: "Local",
		rbacRules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"get", "list", "watch"},
				APIGroups: []string{""},
				Resources: []string{"pods"},
			},
		},
		volumes: []v1.Volume{
			{
				Name: "skupper-internal",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: "skupper-internal",
					},
				},
			},
			{
				Name: "skupper-amqps",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: "skupper-amqps",
					},
				},
			},
		},
		containers: []container{
			{
				name:            "skupper",
				image:           image,
				imagePullPolicy: "Always",
				livenessProbe: &corev1.Probe{
					InitialDelaySeconds: 60,
					Handler: corev1.Handler{
						HTTPGet: &corev1.HTTPGetAction{
							Port: intstr.FromInt(9090),
							Path: "/healthz",
						},
					},
				},
				env: []v1.EnvVar{
					{
						Name:  "APPLICATION_NAME",
						Value: "skupper-router",
					},
					{
						Name:  "QDROUTERD_AUTO_MESH_DISCOVERY",
						Value: "QUERY",
					},
					{
						Name:  "QDROUTERD_CONF",
						Value: routerConfig,
					},
					{
						Name: "POD_NAMESPACE",
						ValueFrom: &v1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: "metadata.namespace",
							},
						},
					},
					{
						Name: "POD_IP",
						ValueFrom: &v1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: "status.podIP",
							},
						},
					},
				},
				volumeMounts: []v1.VolumeMount{
					{
						Name:      "skupper-internal",
						MountPath: volumeMountPath + "/skupper-internal",
					},
					{
						Name:      "skupper-amqps",
						MountPath: volumeMountPath + "/skupper-amqps",
					},
				},
				resources: v1.ResourceRequirements{
					Limits: v1.ResourceList{
						"cpu":    resource.MustParse("200m"),
						"memory": resource.MustParse("1Gi"),
					},
					Requests: v1.ResourceList{
						"cpu":    resource.MustParse("50m"),
						"memory": resource.MustParse("200Mi"),
					},
				},
			},
		},
	}
}

func getTrafficPolicy(serviceType string) string {
	if serviceType == string(corev1.ServiceTypeLoadBalancer) {
		return string(corev1.ServiceExternalTrafficPolicyTypeLocal)
	}
	return string(corev1.ServiceExternalTrafficPolicyTypeCluster)
}

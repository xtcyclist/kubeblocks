/*
Copyright ApeCloud Inc.

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

package dbaas

import (
	"fmt"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	dbaasv1alpha1 "github.com/apecloud/kubeblocks/apis/dbaas/v1alpha1"
)

type insClassType struct {
	memSize int64
	cpu     int64
	// recommended buffer size
	bufferSize string

	maxBufferSize int
}

var _ = Describe("tpl template", func() {

	var (
		podSpec     *corev1.PodSpec
		cfgTemplate []dbaasv1alpha1.ConfigTemplate
		component   *Component
	)

	const (
		mysqlCfgName       = "my.cfg"
		mysqlCfgTmpContext = `
#test
cluster_name = {{ $.cluster.metadata.name }}
cluster_namespace = {{ $.cluster.metadata.namespace }}
component_name = {{ $.component.name }}
component_replica = {{ $.component.replicas }}
containers = {{ (index $.podSpec.containers 0 ).name }}
{{- $buffer_pool_size_tmp := 2147483648 -}}
{{- if $.componentResource -}}
{{- $buffer_pool_size_tmp = $.componentResource.memorySize }}
{{- end }}
innodb_buffer_pool_size = {{ $buffer_pool_size_tmp | int64 }}
`
		mysqlCfgRenderedContext = `
#test
cluster_name = my_test
cluster_namespace = default
component_name = replicasets
component_replica = 5
containers = mytest
innodb_buffer_pool_size = 8589934592
`
	)

	BeforeEach(func() {
		// Add any steup steps that needs to be executed before each test
		podSpec = &corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "mytest",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "data",
							MountPath: "/data/mysql",
						},
						{
							Name:      "log",
							MountPath: "/log/mysql",
						},
					},
					Env: []corev1.EnvVar{
						{
							Name:  "t1",
							Value: "value1",
						},
						{
							Name:  "t2",
							Value: "value2",
						},
						{
							Name:  "a",
							Value: "b",
						},
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          "mysql",
							ContainerPort: 3356,
							Protocol:      "TCP",
						},
						{
							Name:          "paxos",
							ContainerPort: 3356,
							Protocol:      "TCP",
						},
					},
					Resources: corev1.ResourceRequirements{
						Limits: map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceMemory: resource.MustParse("8Gi"),
							corev1.ResourceCPU:    resource.MustParse("4"),
						},
					},
				},
				{
					Name: "invalid_contaienr",
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "config",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "cluster_name_for_test",
							},
						},
					},
				},
			},
		}
		component = &Component{
			ClusterDefName: "mysql-three-node-definition",
			ClusterType:    "state.mysql-8",
			Name:           "replicasets",
			Type:           "replicasets",
			Replicas:       5,
		}
		cfgTemplate = []dbaasv1alpha1.ConfigTemplate{
			{
				Name:       "mysql-config-8.0.2",
				VolumeName: "config1",
			},
		}
	})

	// for test GetContainerWithVolumeMount
	Context("ConfigTemplateBuilder sample test", func() {
		It("test render", func() {
			cfgBuilder := newCfgTemplateBuilder(
				"my_test",
				"default",
				&dbaasv1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my_test",
						Namespace: "default",
					},
				},
				nil,
			)

			Expect(cfgBuilder.injectBuiltInObjectsAndFunctions(
				podSpec, cfgTemplate, component)).Should(BeNil())

			cfgBuilder.componentValues.Resource = &ResourceDefinition{
				MemorySize: 8 * 1024 * 1024 * 1024,
				CoreNum:    4,
			}

			cfgBuilder.setTplName("for_test")
			rendered, err := cfgBuilder.render(map[string]string{
				mysqlCfgName: mysqlCfgTmpContext,
			})

			// Debug
			fmt.Println(rendered[mysqlCfgName])
			fmt.Printf("%s\n", mysqlCfgRenderedContext)

			Expect(err).Should(BeNil())
			Expect(rendered[mysqlCfgName]).Should(Equal(mysqlCfgRenderedContext))
		})
		It("test built-in function", func() {
			cfgBuilder := newCfgTemplateBuilder(
				"my_test",
				"default",
				&dbaasv1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my_test",
						Namespace: "default",
					},
				},
				nil,
			)

			Expect(cfgBuilder.injectBuiltInObjectsAndFunctions(podSpec, cfgTemplate, component)).Should(BeNil())

			rendered, err := cfgBuilder.render(map[string]string{
				"a":                 "{{ getVolumePathByName ( index $.podSpec.containers 0 ) \"log\" }}",
				"b":                 "{{ getVolumePathByName ( index $.podSpec.containers 0 ) \"data\" }}",
				"c":                 "{{ ( getPortByName ( index $.podSpec.containers 0 ) \"mysql\" ).containerPort }}",
				"d":                 "{{ callBufferSizeByResource ( index $.podSpec.containers 0 ) }}",
				"e":                 "{{ getArgByName ( index $.podSpec.containers 0 ) \"User\" }}",
				"f":                 "{{ getVolumePathByName ( getContainerByName $.podSpec.containers \"mytest\") \"data\" }}",
				"i":                 "{{ getEnvByName ( index $.podSpec.containers 0 ) \"a\" }}",
				"j":                 "{{ ( getPVCByName $.podSpec.volumes \"config\" ).configMap.name }}",
				"invalid_volume":    "{{ getVolumePathByName ( index $.podSpec.containers 0 ) \"invalid\" }}",
				"invalid_port":      "{{ getPortByName ( index $.podSpec.containers 0 ) \"invalid\" }}",
				"invalid_container": "{{ getContainerByName $.podSpec.containers  \"invalid\" }}",
				"invalid_resource":  "{{ callBufferSizeByResource ( index $.podSpec.containers 1 ) }}",
				"invalid_env":       "{{ getEnvByName ( index $.podSpec.containers 0 ) \"invalid\" }}",
				"invalid_pvc":       "{{ getPVCByName $.podSpec.volumes \"invalid\" }}",
			})

			Expect(err).Should(BeNil())
			// for test volumeMounts
			Expect(rendered["a"]).Should(BeEquivalentTo("/log/mysql"))
			// for test volumeMounts
			Expect(rendered["b"]).Should(BeEquivalentTo("/data/mysql"))
			// for test port
			Expect(rendered["c"]).Should(BeEquivalentTo("3356"))
			// for test resource
			Expect(rendered["d"]).Should(BeEquivalentTo("4096M"))
			// for test args
			Expect(rendered["e"]).Should(BeEquivalentTo(""))
			// for test volumeMounts
			Expect(rendered["f"]).Should(BeEquivalentTo("/data/mysql"))
			// for test env
			Expect(rendered["i"]).Should(BeEquivalentTo("b"))
			// for test volume
			Expect(rendered["j"]).Should(BeEquivalentTo("cluster_name_for_test"))
			Expect(rendered["invalid_volume"]).Should(BeEquivalentTo(""))
			Expect(rendered["invalid_port"]).Should(BeEquivalentTo("<no value>"))
			Expect(rendered["invalid_container"]).Should(BeEquivalentTo("<no value>"))
			Expect(rendered["invalid_env"]).Should(BeEquivalentTo(""))
			Expect(rendered["invalid_pvc"]).Should(BeEquivalentTo("<no value>"))
			Expect(rendered["invalid_resource"]).Should(BeEquivalentTo(""))
		})
	})

	Context("calMysqlPoolSizeByResource test", func() {
		It("mysql test", func() {
			Expect(calMysqlPoolSizeByResource(nil, false)).Should(Equal("128M"))

			Expect(calMysqlPoolSizeByResource(nil, true)).Should(Equal("128M"))

			// for small instance class
			Expect(calMysqlPoolSizeByResource(&ResourceDefinition{
				MemorySize: 1024 * 1024 * 1024,
				CoreNum:    1,
			}, false)).Should(Equal("128M"))

			Expect(calMysqlPoolSizeByResource(&ResourceDefinition{
				MemorySize: 2 * 1024 * 1024 * 1024,
				CoreNum:    2,
			}, false)).Should(Equal("256M"))

			// for shard
			Expect(calMysqlPoolSizeByResource(&ResourceDefinition{
				MemorySize: 2 * 1024 * 1024 * 1024,
				CoreNum:    2,
			}, true)).Should(Equal("1024M"))

			insClassTest := []insClassType{
				// for 2 core
				{
					memSize:       4,
					cpu:           2,
					bufferSize:    "1024M",
					maxBufferSize: 1024,
				},
				{
					memSize:       8,
					cpu:           2,
					bufferSize:    "4096M",
					maxBufferSize: 4096,
				},
				{
					memSize:       16,
					cpu:           2,
					bufferSize:    "9216M",
					maxBufferSize: 10240,
				},
				// for 4 core
				{
					memSize:       8,
					cpu:           4,
					bufferSize:    "4096M",
					maxBufferSize: 4096,
				},
				{
					memSize:       16,
					cpu:           4,
					bufferSize:    "9216M",
					maxBufferSize: 10240,
				},
				{
					memSize:       32,
					cpu:           4,
					bufferSize:    "21504M",
					maxBufferSize: 22528,
				},
				// for 8 core
				{
					memSize:       16,
					cpu:           8,
					bufferSize:    "9216M",
					maxBufferSize: 10240,
				},
				{
					memSize:       32,
					cpu:           8,
					bufferSize:    "21504M",
					maxBufferSize: 22528,
				},
				{
					memSize:       64,
					cpu:           8,
					bufferSize:    "45056M",
					maxBufferSize: 48128,
				},
				// for 12 core
				{
					memSize:       24,
					cpu:           12,
					bufferSize:    "15360M",
					maxBufferSize: 16384,
				},
				{
					memSize:       48,
					cpu:           12,
					bufferSize:    "33792M",
					maxBufferSize: 35840,
				},
				{
					memSize:       96,
					cpu:           12,
					bufferSize:    "69632M",
					maxBufferSize: 73728,
				},
				// for 16 core
				{
					memSize:       32,
					cpu:           16,
					bufferSize:    "21504M",
					maxBufferSize: 22528,
				},
				{
					memSize:       64,
					cpu:           16,
					bufferSize:    "45056M",
					maxBufferSize: 48128,
				},
				{
					memSize:       128,
					cpu:           16,
					bufferSize:    "93184M",
					maxBufferSize: 99328,
				},
				// for 24 core
				{
					memSize:       48,
					cpu:           24,
					bufferSize:    "32768M",
					maxBufferSize: 34816,
				},
				{
					memSize:       96,
					cpu:           24,
					bufferSize:    "69632M",
					maxBufferSize: 73728,
				},
				{
					memSize:       192,
					cpu:           24,
					bufferSize:    "140288M",
					maxBufferSize: 149504,
				},
				// for 32 core
				{
					memSize:       64,
					cpu:           32,
					bufferSize:    "45056M",
					maxBufferSize: 47104,
				},
				{
					memSize:       128,
					cpu:           32,
					bufferSize:    "93184M",
					maxBufferSize: 99328,
				},
				{
					memSize:       256,
					cpu:           32,
					bufferSize:    "188416M",
					maxBufferSize: 200704,
				},
				// for 52 core
				{
					memSize:       96,
					cpu:           52,
					bufferSize:    "67584M",
					maxBufferSize: 72704,
				},
				{
					memSize:       192,
					cpu:           52,
					bufferSize:    "140288M",
					maxBufferSize: 149504,
				},
				{
					memSize:       384,
					cpu:           52,
					bufferSize:    "283648M",
					maxBufferSize: 302080,
				},
				// for 64 core
				{
					memSize:       256,
					cpu:           64,
					bufferSize:    "188416M",
					maxBufferSize: 200704,
				},
				{
					memSize:       512,
					cpu:           64,
					bufferSize:    "378880M",
					maxBufferSize: 403456,
				},
				// for 102
				{
					memSize:       768,
					cpu:           102,
					bufferSize:    "569344M",
					maxBufferSize: 607232,
				},
				// for 104 core
				{
					memSize:       192,
					cpu:           104,
					bufferSize:    "138240M",
					maxBufferSize: 147456,
				},
				{
					memSize:       384,
					cpu:           104,
					bufferSize:    "282624M",
					maxBufferSize: 302080,
				},
			}

			// for debug
			for _, r := range insClassTest {
				fmt.Printf("cal : %s, expect [%v] [%s]\n", calMysqlPoolSizeByResource(&ResourceDefinition{
					MemorySize: r.memSize * 1024 * 1024 * 1024, // 4G
					CoreNum:    r.cpu,                          // 2core
				}, false), r, r.bufferSize)
			}

			for _, r := range insClassTest {
				ret := calMysqlPoolSizeByResource(&ResourceDefinition{
					MemorySize: r.memSize * 1024 * 1024 * 1024, // 4G
					CoreNum:    r.cpu,                          // 2core
				}, false)
				Expect(ret).Should(Equal(r.bufferSize))
				Expect(strconv.ParseInt(strings.Trim(ret, "M"), 10, 64)).Should(BeNumerically("<=", r.maxBufferSize))
			}
		})
	})

})

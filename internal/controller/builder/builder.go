/*
Copyright (C) 2022-2023 ApeCloud Co., Ltd

This file is part of KubeBlocks project

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package builder

import (
	"embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	snapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v6/apis/volumesnapshot/v1"
	"github.com/leaanthony/debme"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1alpha1 "github.com/apecloud/kubeblocks/apis/apps/v1alpha1"
	dataprotectionv1alpha1 "github.com/apecloud/kubeblocks/apis/dataprotection/v1alpha1"
	componentutil "github.com/apecloud/kubeblocks/controllers/apps/components/util"
	cfgcm "github.com/apecloud/kubeblocks/internal/configuration/config_manager"
	"github.com/apecloud/kubeblocks/internal/constant"
	"github.com/apecloud/kubeblocks/internal/controller/component"
	intctrlutil "github.com/apecloud/kubeblocks/internal/controllerutil"
)

const (
	VolumeName = "tls"
	CAName     = "ca.crt"
	CertName   = "tls.crt"
	KeyName    = "tls.key"
	MountPath  = "/etc/pki/tls"
)

var (
	//go:embed cue/*
	cueTemplates embed.FS
	cacheCtx     = map[string]interface{}{}
)

func getCacheCUETplValue(key string, valueCreator func() (*intctrlutil.CUETpl, error)) (*intctrlutil.CUETpl, error) {
	vIf, ok := cacheCtx[key]
	if ok {
		return vIf.(*intctrlutil.CUETpl), nil
	}
	v, err := valueCreator()
	if err != nil {
		return nil, err
	}
	cacheCtx[key] = v
	return v, err
}

func buildFromCUE(tplName string, fillMap map[string]any, lookupKey string, target any) error {
	cueFS, _ := debme.FS(cueTemplates, "cue")
	cueTpl, err := getCacheCUETplValue(tplName, func() (*intctrlutil.CUETpl, error) {
		return intctrlutil.NewCUETplFromBytes(cueFS.ReadFile(tplName))
	})
	if err != nil {
		return err
	}
	cueValue := intctrlutil.NewCUEBuilder(*cueTpl)

	for k, v := range fillMap {
		if err := cueValue.FillObj(k, v); err != nil {
			return err
		}
	}

	b, err := cueValue.Lookup(lookupKey)
	if err != nil {
		return err
	}

	if err = json.Unmarshal(b, target); err != nil {
		return err
	}

	return nil
}

func processContainersInjection(reqCtx intctrlutil.RequestCtx,
	cluster *appsv1alpha1.Cluster,
	component *component.SynthesizedComponent,
	envConfigName string,
	podSpec *corev1.PodSpec) error {
	for _, cc := range []*[]corev1.Container{
		&podSpec.Containers,
		&podSpec.InitContainers,
	} {
		for i := range *cc {
			if err := injectEnvs(cluster, component, envConfigName, &(*cc)[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

func injectEnvs(cluster *appsv1alpha1.Cluster, component *component.SynthesizedComponent, envConfigName string, c *corev1.Container) error {
	// can not use map, it is unordered
	envFieldPathSlice := []struct {
		name      string
		fieldPath string
	}{
		{name: "KB_POD_NAME", fieldPath: "metadata.name"},
		{name: "KB_POD_UID", fieldPath: "metadata.uid"},
		{name: "KB_NAMESPACE", fieldPath: "metadata.namespace"},
		{name: "KB_SA_NAME", fieldPath: "spec.serviceAccountName"},
		{name: "KB_NODENAME", fieldPath: "spec.nodeName"},
		{name: "KB_HOST_IP", fieldPath: "status.hostIP"},
		{name: "KB_POD_IP", fieldPath: "status.podIP"},
		{name: "KB_POD_IPS", fieldPath: "status.podIPs"},
		// TODO: need to deprecate following
		{name: "KB_HOSTIP", fieldPath: "status.hostIP"},
		{name: "KB_PODIP", fieldPath: "status.podIP"},
		{name: "KB_PODIPS", fieldPath: "status.podIPs"},
	}

	toInjectEnvs := make([]corev1.EnvVar, 0, len(envFieldPathSlice)+len(c.Env))
	for _, v := range envFieldPathSlice {
		toInjectEnvs = append(toInjectEnvs, corev1.EnvVar{
			Name: v.name,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					APIVersion: "v1",
					FieldPath:  v.fieldPath,
				},
			},
		})
	}

	var kbClusterPostfix8 string
	if len(cluster.UID) > 8 {
		kbClusterPostfix8 = string(cluster.UID)[len(cluster.UID)-8:]
	} else {
		kbClusterPostfix8 = string(cluster.UID)
	}
	toInjectEnvs = append(toInjectEnvs, []corev1.EnvVar{
		{Name: "KB_CLUSTER_NAME", Value: cluster.Name},
		{Name: "KB_COMP_NAME", Value: component.Name},
		{Name: "KB_CLUSTER_COMP_NAME", Value: cluster.Name + "-" + component.Name},
		{Name: "KB_CLUSTER_UID_POSTFIX_8", Value: kbClusterPostfix8},
		{Name: "KB_POD_FQDN", Value: fmt.Sprintf("%s.%s-headless.%s.svc", "$(KB_POD_NAME)",
			"$(KB_CLUSTER_COMP_NAME)", "$(KB_NAMESPACE)")},
	}...)

	if component.TLS {
		toInjectEnvs = append(toInjectEnvs, []corev1.EnvVar{
			{Name: "KB_TLS_CERT_PATH", Value: MountPath},
			{Name: "KB_TLS_CA_FILE", Value: CAName},
			{Name: "KB_TLS_CERT_FILE", Value: CertName},
			{Name: "KB_TLS_KEY_FILE", Value: KeyName},
		}...)
	}

	if udeValue, ok := cluster.Annotations[constant.ExtraEnvAnnotationKey]; ok {
		udeMap := make(map[string]string)
		if err := json.Unmarshal([]byte(udeValue), &udeMap); err != nil {
			return err
		}
		for k, v := range udeMap {
			if k == "" || v == "" {
				continue
			}
			toInjectEnvs = append(toInjectEnvs, corev1.EnvVar{
				Name:  k,
				Value: v,
			})
		}
	}
	// have injected variables placed at the front of the slice
	if len(c.Env) == 0 {
		c.Env = toInjectEnvs
	} else {
		c.Env = append(toInjectEnvs, c.Env...)
	}
	if envConfigName == "" {
		return nil
	}
	c.EnvFrom = append(c.EnvFrom, corev1.EnvFromSource{
		ConfigMapRef: &corev1.ConfigMapEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: envConfigName,
			},
		},
	})

	return nil
}

// BuildPersistentVolumeClaimLabels builds a pvc name label, and synchronize the labels from sts to pvc.
func BuildPersistentVolumeClaimLabels(component *component.SynthesizedComponent, pvc *corev1.PersistentVolumeClaim,
	pvcTplName string) {
	// strict args checking.
	if pvc == nil || component == nil {
		return
	}
	if pvc.Labels == nil {
		pvc.Labels = make(map[string]string)
	}
	pvc.Labels[constant.VolumeClaimTemplateNameLabelKey] = pvcTplName

	if component.VolumeTypes != nil {
		for _, t := range component.VolumeTypes {
			if t.Name == pvcTplName {
				pvc.Labels[constant.VolumeTypeLabelKey] = string(t.Type)
				break
			}
		}
	}
}

func BuildSvcListWithCustomAttributes(cluster *appsv1alpha1.Cluster, component *component.SynthesizedComponent,
	customAttributeSetter func(*corev1.Service)) ([]*corev1.Service, error) {
	services, err := BuildSvcList(cluster, component)
	if err != nil {
		return nil, err
	}
	if customAttributeSetter != nil {
		for _, svc := range services {
			customAttributeSetter(svc)
		}
	}
	return services, nil
}

func BuildSvcList(cluster *appsv1alpha1.Cluster, component *component.SynthesizedComponent) ([]*corev1.Service, error) {
	const tplFile = "service_template.cue"
	var result = make([]*corev1.Service, 0)
	for _, item := range component.Services {
		if len(item.Spec.Ports) == 0 {
			continue
		}
		svc := corev1.Service{}
		if err := buildFromCUE(tplFile, map[string]any{
			"cluster":   cluster,
			"service":   item,
			"component": component,
		}, "svc", &svc); err != nil {
			return nil, err
		}
		result = append(result, &svc)
	}
	return result, nil
}

func BuildHeadlessSvc(cluster *appsv1alpha1.Cluster, component *component.SynthesizedComponent) (*corev1.Service, error) {
	const tplFile = "headless_service_template.cue"
	service := corev1.Service{}
	if err := buildFromCUE(tplFile, map[string]any{
		"cluster":   cluster,
		"component": component,
	}, "service", &service); err != nil {
		return nil, err
	}
	return &service, nil
}

func BuildSts(reqCtx intctrlutil.RequestCtx, cluster *appsv1alpha1.Cluster,
	component *component.SynthesizedComponent, envConfigName string) (*appsv1.StatefulSet, error) {
	const tplFile = "statefulset_template.cue"

	sts := appsv1.StatefulSet{}
	if err := buildFromCUE(tplFile, map[string]any{
		"cluster":   cluster,
		"component": component,
	}, "statefulset", &sts); err != nil {
		return nil, err
	}

	if component.StatefulSetWorkload != nil {
		sts.Spec.PodManagementPolicy, sts.Spec.UpdateStrategy = component.StatefulSetWorkload.FinalStsUpdateStrategy()
	}

	// update sts.spec.volumeClaimTemplates[].metadata.labels
	if len(sts.Spec.VolumeClaimTemplates) > 0 && len(sts.GetLabels()) > 0 {
		for index, vct := range sts.Spec.VolumeClaimTemplates {
			BuildPersistentVolumeClaimLabels(component, &vct, vct.Name)
			sts.Spec.VolumeClaimTemplates[index] = vct
		}
	}

	if err := processContainersInjection(reqCtx, cluster, component, envConfigName, &sts.Spec.Template.Spec); err != nil {
		return nil, err
	}
	return &sts, nil
}

func randomString(length int) string {
	return rand.String(length)
}

func BuildConnCredential(clusterDefiniiton *appsv1alpha1.ClusterDefinition, cluster *appsv1alpha1.Cluster,
	component *component.SynthesizedComponent) (*corev1.Secret, error) {
	const tplFile = "conn_credential_template.cue"

	connCredential := corev1.Secret{}
	if err := buildFromCUE(tplFile, map[string]any{
		"clusterdefinition": clusterDefiniiton,
		"cluster":           cluster,
	}, "secret", &connCredential); err != nil {
		return nil, err
	}

	if len(connCredential.StringData) == 0 {
		return &connCredential, nil
	}

	replaceVarObjects := func(k, v *string, i int, origValue string, varObjectsMap map[string]string) {
		toReplace := origValue
		for j, r := range varObjectsMap {
			replaced := strings.ReplaceAll(toReplace, j, r)
			if replaced == toReplace {
				continue
			}
			toReplace = replaced
			// replace key
			if i == 0 {
				delete(connCredential.StringData, origValue)
				*k = replaced
			} else {
				*v = replaced
			}
		}
	}

	// REVIEW: perhaps handles value replacement at `func mergeComponents`
	replaceData := func(varObjectsMap map[string]string) {
		copyStringData := connCredential.DeepCopy().StringData
		for k, v := range copyStringData {
			for i, vv := range []string{k, v} {
				if !strings.Contains(vv, "$(") {
					continue
				}
				replaceVarObjects(&k, &v, i, vv, varObjectsMap)
			}
			connCredential.StringData[k] = v
		}
	}

	// TODO: do JIT value generation for lower CPU resources
	// 1st pass replace variables
	uuidVal := uuid.New()
	uuidBytes := uuidVal[:]
	uuidStr := uuidVal.String()
	uuidB64 := base64.RawStdEncoding.EncodeToString(uuidBytes)
	uuidStrB64 := base64.RawStdEncoding.EncodeToString([]byte(strings.ReplaceAll(uuidStr, "-", "")))
	uuidHex := hex.EncodeToString(uuidBytes)
	m := map[string]string{
		"$(RANDOM_PASSWD)":        randomString(8),
		"$(UUID)":                 uuidStr,
		"$(UUID_B64)":             uuidB64,
		"$(UUID_STR_B64)":         uuidStrB64,
		"$(UUID_HEX)":             uuidHex,
		"$(SVC_FQDN)":             fmt.Sprintf("%s-%s.%s.svc", cluster.Name, component.Name, cluster.Namespace),
		"$(KB_CLUSTER_COMP_NAME)": cluster.Name + "-" + component.Name,
		"$(HEADLESS_SVC_FQDN)":    fmt.Sprintf("%s-%s-headless.%s.svc", cluster.Name, component.Name, cluster.Namespace),
	}
	if len(component.Services) > 0 {
		for _, p := range component.Services[0].Spec.Ports {
			m[fmt.Sprintf("$(SVC_PORT_%s)", p.Name)] = strconv.Itoa(int(p.Port))
		}
	}
	replaceData(m)

	// 2nd pass replace $(CONN_CREDENTIAL) variables
	m = map[string]string{}
	for k, v := range connCredential.StringData {
		m[fmt.Sprintf("$(CONN_CREDENTIAL).%s", k)] = v
	}
	replaceData(m)
	return &connCredential, nil
}

func BuildPDB(cluster *appsv1alpha1.Cluster, component *component.SynthesizedComponent) (*policyv1.PodDisruptionBudget, error) {
	const tplFile = "pdb_template.cue"
	pdb := policyv1.PodDisruptionBudget{}
	if err := buildFromCUE(tplFile, map[string]any{
		"cluster":   cluster,
		"component": component,
	}, "pdb", &pdb); err != nil {
		return nil, err
	}
	return &pdb, nil
}

func BuildDeploy(reqCtx intctrlutil.RequestCtx, cluster *appsv1alpha1.Cluster, component *component.SynthesizedComponent) (*appsv1.Deployment, error) {
	const tplFile = "deployment_template.cue"
	deploy := appsv1.Deployment{}
	if err := buildFromCUE(tplFile, map[string]any{
		"cluster":   cluster,
		"component": component,
	}, "deployment", &deploy); err != nil {
		return nil, err
	}

	if component.StatelessSpec != nil {
		deploy.Spec.Strategy = component.StatelessSpec.UpdateStrategy
	}
	if err := processContainersInjection(reqCtx, cluster, component, "", &deploy.Spec.Template.Spec); err != nil {
		return nil, err
	}
	return &deploy, nil
}

func BuildPVC(cluster *appsv1alpha1.Cluster,
	component *component.SynthesizedComponent,
	vct *corev1.PersistentVolumeClaimTemplate,
	pvcKey types.NamespacedName,
	snapshotName string) (*corev1.PersistentVolumeClaim, error) {
	pvc := corev1.PersistentVolumeClaim{}
	if err := buildFromCUE("pvc_template.cue", map[string]any{
		"cluster":             cluster,
		"component":           component,
		"volumeClaimTemplate": vct,
		"pvc_key":             pvcKey,
		"snapshot_name":       snapshotName,
	}, "pvc", &pvc); err != nil {
		return nil, err
	}
	BuildPersistentVolumeClaimLabels(component, &pvc, vct.Name)
	return &pvc, nil
}

// BuildEnvConfig builds cluster component context ConfigMap object, which is to be used in workload container's
// envFrom.configMapRef with name of "$(cluster.metadata.name)-$(component.name)-env" pattern.
func BuildEnvConfig(reqCtx intctrlutil.RequestCtx, cli client.Client, cluster *appsv1alpha1.Cluster, component *component.SynthesizedComponent) (*corev1.ConfigMap, error) {
	const tplFile = "env_config_template.cue"
	envData := map[string]string{}

	// build common env
	commonEnv := buildWorkloadCommonEnv(cluster, component)
	for k, v := range commonEnv {
		envData[k] = v
	}

	// build env for replication workload
	replicationEnv := buildReplicationSetEnv(reqCtx, cli, cluster, component)
	for k, v := range replicationEnv {
		envData[k] = v
	}

	// TODO following code seems to be redundant with updateConsensusRoleInfo in consensus_set_utils.go
	// build consensus env from cluster.status
	consensusEnv := buildConsensusSetEnv(cluster, component)
	for k, v := range consensusEnv {
		envData[k] = v
	}

	config := corev1.ConfigMap{}
	if err := buildFromCUE(tplFile, map[string]any{
		"cluster":     cluster,
		"component":   component,
		"config.data": envData,
	}, "config", &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// buildWorkloadCommonEnv builds common env for all workload types.
func buildWorkloadCommonEnv(cluster *appsv1alpha1.Cluster, component *component.SynthesizedComponent) map[string]string {
	prefix := constant.KBPrefix + "_"
	cnt := strconv.Itoa(int(component.Replicas))
	svcName := strings.Join([]string{cluster.Name, component.Name, "headless"}, "-")
	suffixes := make([]string, 0, 4+component.Replicas)
	env := map[string]string{
		prefix + "REPLICA_COUNT": cnt,
	}

	for j := 0; j < int(component.Replicas); j++ {
		toA := strconv.Itoa(j)
		suffix := toA + "_HOSTNAME"
		value := fmt.Sprintf("%s.%s", cluster.Name+"-"+component.Name+"-"+toA, svcName)
		env[prefix+suffix] = value
		suffixes = append(suffixes, suffix)
	}

	// set cluster uid to let pod know if the cluster is recreated
	env[prefix+"CLUSTER_UID"] = string(cluster.UID)
	suffixes = append(suffixes, "CLUSTER_UID")

	// have backward compatible handling for CM key with 'compDefName' being part of the key name
	// TODO: need to deprecate 'compDefName' being part of variable name, as it's redundant
	// and introduce env/cm key naming reference complexity
	prefixWithCompDefName := prefix + strings.ToUpper(component.CompDefName) + "_"
	for _, s := range suffixes {
		env[prefixWithCompDefName+s] = env[prefix+s]
	}
	env[prefixWithCompDefName+"N"] = env[prefix+"REPLICA_COUNT"]
	return env
}

// buildReplicationSetEnv builds env for replication workload.
func buildReplicationSetEnv(reqCtx intctrlutil.RequestCtx,
	cli client.Client,
	cluster *appsv1alpha1.Cluster,
	component *component.SynthesizedComponent) map[string]string {
	if component.WorkloadType != appsv1alpha1.Replication {
		return nil
	}
	env := map[string]string{}
	svcName := strings.Join([]string{cluster.Name, component.Name, "headless"}, "-")
	podList, _ := componentutil.GetComponentPodListWithRole(reqCtx.Ctx, cli, *cluster, component.Name, constant.Primary)
	if len(podList.Items) > 0 {
		env[constant.KBReplicationSetPrimaryPodName] = podList.Items[0].Name
		env[constant.KBReplicationSetPrimaryPodFQDN] = fmt.Sprintf("%s.%s.%s.svc", podList.Items[0].Name, svcName, cluster.Namespace)
	} else {
		// If there is no primaryPod in the cluster, it means that the cluster is new created for the first time,
		// and index=0 is used as the primary pod by default.
		primaryPodName := fmt.Sprintf("%s-%s-%d", cluster.Name, component.Name, 0)
		env[constant.KBReplicationSetPrimaryPodName] = primaryPodName
		env[constant.KBReplicationSetPrimaryPodFQDN] = fmt.Sprintf("%s.%s.%s.svc", primaryPodName, svcName, cluster.Namespace)
	}
	return env
}

// buildConsensusSetEnv builds env for consensus workload.
func buildConsensusSetEnv(cluster *appsv1alpha1.Cluster, component *component.SynthesizedComponent) map[string]string {
	env := map[string]string{}
	prefix := constant.KBPrefix + "_"
	prefixWithCompDefName := prefix + strings.ToUpper(component.CompDefName) + "_"
	if v, ok := cluster.Status.Components[component.Name]; ok && v.ConsensusSetStatus != nil {
		consensusSetStatus := v.ConsensusSetStatus
		if consensusSetStatus.Leader.Pod != constant.ComponentStatusDefaultPodName {
			env[prefix+"LEADER"] = consensusSetStatus.Leader.Pod
			env[prefixWithCompDefName+"LEADER"] = env[prefix+"LEADER"]
		}
		followers := make([]string, 0, len(consensusSetStatus.Followers))
		for _, follower := range consensusSetStatus.Followers {
			if follower.Pod == constant.ComponentStatusDefaultPodName {
				continue
			}
			followers = append(followers, follower.Pod)
		}
		env[prefix+"FOLLOWERS"] = strings.Join(followers, ",")
		env[prefixWithCompDefName+"FOLLOWERS"] = env[prefix+"FOLLOWERS"]
	}
	return env
}

func BuildBackup(cluster *appsv1alpha1.Cluster,
	component *component.SynthesizedComponent,
	backupPolicyName string,
	backupKey types.NamespacedName,
	backupType string) (*dataprotectionv1alpha1.Backup, error) {
	backup := dataprotectionv1alpha1.Backup{}
	if err := buildFromCUE("backup_job_template.cue", map[string]any{
		"cluster":          cluster,
		"component":        component,
		"backupPolicyName": backupPolicyName,
		"backupJobKey":     backupKey,
		"backupType":       backupType,
	}, "backupJob", &backup); err != nil {
		return nil, err
	}
	return &backup, nil
}

func BuildVolumeSnapshot(snapshotKey types.NamespacedName,
	pvcName string,
	sts *appsv1.StatefulSet) (*snapshotv1.VolumeSnapshot, error) {
	snapshot := snapshotv1.VolumeSnapshot{}
	if err := buildFromCUE("snapshot_template.cue", map[string]any{
		"snapshot_key": snapshotKey,
		"pvc_name":     pvcName,
		"sts":          sts,
	}, "snapshot", &snapshot); err != nil {
		return nil, err
	}
	return &snapshot, nil
}

func BuildConfigMapWithTemplate(cluster *appsv1alpha1.Cluster,
	component *component.SynthesizedComponent,
	configs map[string]string,
	cmName string,
	configConstraintName string,
	configTemplateSpec appsv1alpha1.ComponentTemplateSpec) (*corev1.ConfigMap, error) {
	const tplFile = "config_template.cue"
	cueFS, _ := debme.FS(cueTemplates, "cue")
	cueTpl, err := getCacheCUETplValue(tplFile, func() (*intctrlutil.CUETpl, error) {
		return intctrlutil.NewCUETplFromBytes(cueFS.ReadFile(tplFile))
	})
	if err != nil {
		return nil, err
	}

	cueValue := intctrlutil.NewCUEBuilder(*cueTpl)
	// prepare cue data
	configMeta := map[string]map[string]string{
		"clusterDefinition": {
			"name": cluster.Spec.ClusterDefRef,
		},
		"cluster": {
			"name":      cluster.GetName(),
			"namespace": cluster.GetNamespace(),
		},
		"component": {
			"name":                  component.Name,
			"compDefName":           component.CompDefName,
			"characterType":         component.CharacterType,
			"configName":            cmName,
			"templateName":          configTemplateSpec.TemplateRef,
			"configConstraintsName": configConstraintName,
			"configTemplateName":    configTemplateSpec.Name,
		},
	}
	configBytes, err := json.Marshal(configMeta)
	if err != nil {
		return nil, err
	}

	// Generate config files context by rendering cue template
	if err = cueValue.Fill("meta", configBytes); err != nil {
		return nil, err
	}

	configStrByte, err := cueValue.Lookup("config")
	if err != nil {
		return nil, err
	}

	cm := corev1.ConfigMap{}
	if err = json.Unmarshal(configStrByte, &cm); err != nil {
		return nil, err
	}

	// Update rendered config
	cm.Data = configs
	return &cm, nil
}

func BuildCfgManagerContainer(sidecarRenderedParam *cfgcm.CfgManagerBuildParams, component *component.SynthesizedComponent) (*corev1.Container, error) {
	const tplFile = "config_manager_sidecar.cue"
	cueFS, _ := debme.FS(cueTemplates, "cue")
	cueTpl, err := getCacheCUETplValue(tplFile, func() (*intctrlutil.CUETpl, error) {
		return intctrlutil.NewCUETplFromBytes(cueFS.ReadFile(tplFile))
	})
	if err != nil {
		return nil, err
	}

	cueValue := intctrlutil.NewCUEBuilder(*cueTpl)
	paramBytes, err := json.Marshal(sidecarRenderedParam)
	if err != nil {
		return nil, err
	}

	if err = cueValue.Fill("parameter", paramBytes); err != nil {
		return nil, err
	}

	containerStrByte, err := cueValue.Lookup("template")
	if err != nil {
		return nil, err
	}
	container := corev1.Container{}
	if err = json.Unmarshal(containerStrByte, &container); err != nil {
		return nil, err
	}

	if err := injectEnvs(sidecarRenderedParam.Cluster, component, sidecarRenderedParam.EnvConfigName, &container); err != nil {
		return nil, err
	}
	return &container, nil
}

func BuildBackupManifestsJob(key types.NamespacedName, backup *dataprotectionv1alpha1.Backup, podSpec *corev1.PodSpec) (*batchv1.Job, error) {
	const tplFile = "backup_manifests_template.cue"
	job := &batchv1.Job{}
	if err := buildFromCUE(tplFile,
		map[string]any{
			"job.metadata.name":      key.Name,
			"job.metadata.namespace": key.Namespace,
			"backup":                 backup,
			"podSpec":                podSpec,
		},
		"job", job); err != nil {
		return nil, err
	}
	return job, nil
}

func BuildRestoreJob(name, namespace string, image string, command []string,
	volumes []corev1.Volume, volumeMounts []corev1.VolumeMount, env []corev1.EnvVar, resources *corev1.ResourceRequirements) (*batchv1.Job, error) {
	const tplFile = "restore_job_template.cue"
	job := &batchv1.Job{}
	fillMaps := map[string]any{
		"job.metadata.name":              name,
		"job.metadata.namespace":         namespace,
		"job.spec.template.spec.volumes": volumes,
		"container.image":                image,
		"container.command":              command,
		"container.volumeMounts":         volumeMounts,
		"container.env":                  env,
	}
	if resources != nil {
		fillMaps["container.resources"] = *resources
	}

	if err := buildFromCUE(tplFile, fillMaps, "job", job); err != nil {
		return nil, err
	}
	return job, nil
}

func BuildRestoreJobForFullBackup(
	restoreJobName string,
	component *component.SynthesizedComponent,
	backup *dataprotectionv1alpha1.Backup,
	backupTool *dataprotectionv1alpha1.BackupTool,
	pvcName string) (*batchv1.Job, error) {
	const tplFile = "restore_full_backup_job.cue"
	job := batchv1.Job{}
	if err := buildFromCUE(tplFile, map[string]any{
		"restoreJobName": restoreJobName,
		"component":      component,
		"backup":         backup,
		"backupTool":     backupTool,
		"pvcName":        pvcName,
	}, "job", &job); err != nil {
		return nil, err
	}
	return &job, nil
}

func BuildCfgManagerToolsContainer(sidecarRenderedParam *cfgcm.CfgManagerBuildParams, component *component.SynthesizedComponent, toolsMetas []appsv1alpha1.ToolConfig) ([]corev1.Container, error) {
	toolContainers := make([]corev1.Container, 0, len(toolsMetas))
	for _, toolConfig := range toolsMetas {
		toolContainer := corev1.Container{
			Name:            toolConfig.Name,
			Command:         toolConfig.Command,
			ImagePullPolicy: corev1.PullIfNotPresent,
			VolumeMounts:    sidecarRenderedParam.Volumes,
		}
		if toolConfig.Image != "" {
			toolContainer.Image = toolConfig.Image
		}
		toolContainers = append(toolContainers, toolContainer)
	}
	for i := range toolContainers {
		if err := injectEnvs(sidecarRenderedParam.Cluster, component, sidecarRenderedParam.EnvConfigName, &toolContainers[i]); err != nil {
			return nil, err
		}
	}
	return toolContainers, nil
}

func BuildVolumeSnapshotClass(name string, driver string) (*snapshotv1.VolumeSnapshotClass, error) {
	const tplFile = "volumesnapshotclass.cue"
	vsc := &snapshotv1.VolumeSnapshotClass{}
	if err := buildFromCUE(tplFile,
		map[string]any{
			"class.metadata.name": name,
			"class.driver":        driver,
		},
		"class", vsc); err != nil {
		return nil, err
	}
	return vsc, nil
}

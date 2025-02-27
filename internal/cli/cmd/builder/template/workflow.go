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

package template

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appsv1alpha1 "github.com/apecloud/kubeblocks/apis/apps/v1alpha1"
	"github.com/apecloud/kubeblocks/controllers/apps/components"
	"github.com/apecloud/kubeblocks/internal/cli/printer"
	cfgcore "github.com/apecloud/kubeblocks/internal/configuration"
	"github.com/apecloud/kubeblocks/internal/controller/builder"
	intctrlutil "github.com/apecloud/kubeblocks/internal/controllerutil"
	"github.com/apecloud/kubeblocks/internal/generics"
)

type componentedConfigSpec struct {
	component  string
	configSpec appsv1alpha1.ComponentTemplateSpec
}

type templateRenderWorkflow struct {
	renderedOpts  RenderedOptions
	clusterYaml   string
	clusterDefObj *appsv1alpha1.ClusterDefinition
	localObjects  []client.Object

	clusterDefComponents []appsv1alpha1.ClusterComponentDefinition
}

func (w *templateRenderWorkflow) Do(outputDir string) error {
	var err error
	var cluster *appsv1alpha1.Cluster
	var configSpecs []componentedConfigSpec

	cli := newMockClient(w.localObjects)
	ctx := intctrlutil.RequestCtx{
		Ctx: context.Background(),
		Log: log.Log.WithName("ctool"),
	}

	if cluster, err = w.createClusterObject(); err != nil {
		return err
	}
	ctx.Log.V(1).Info(fmt.Sprintf("cluster object : %v", cluster))

	if configSpecs, err = w.getRenderedConfigSpec(); err != nil {
		return err
	}

	ctx.Log.Info("rendering template:")
	for _, tplSpec := range configSpecs {
		ctx.Log.Info(fmt.Sprintf("config spec: %s, template name: %s in the component[%s]",
			tplSpec.configSpec.Name,
			tplSpec.configSpec.TemplateRef,
			tplSpec.component))
	}

	cache := make(map[string][]client.Object)
	for _, configSpec := range configSpecs {
		compName, err := w.getComponentName(configSpec.component, cluster)
		if err != nil {
			return err
		}
		objects, ok := cache[configSpec.component]
		if !ok {
			objs, err := createComponentObjects(w, ctx, cli, configSpec.component, cluster)
			if err != nil {
				return err
			}
			cache[configSpec.component] = objs
			objects = objs
		}
		if err := renderTemplates(configSpec.configSpec, outputDir, cluster.Name, compName, objects, configSpec.component); err != nil {
			return err
		}
	}
	return nil
}

func (w *templateRenderWorkflow) getComponentName(componentType string, cluster *appsv1alpha1.Cluster) (string, error) {
	clusterCompSpec := cluster.Spec.GetDefNameMappingComponents()[componentType]
	if len(clusterCompSpec) == 0 {
		return "", cfgcore.MakeError("component[%s] is not defined in cluster definition", componentType)
	}
	return clusterCompSpec[0].Name, nil
}

func (w *templateRenderWorkflow) getRenderedConfigSpec() ([]componentedConfigSpec, error) {
	foundSpec := func(com appsv1alpha1.ClusterComponentDefinition, specName string) (appsv1alpha1.ComponentTemplateSpec, bool) {
		for _, spec := range com.ConfigSpecs {
			if spec.Name == specName {
				return spec.ComponentTemplateSpec, true
			}
		}
		for _, spec := range com.ScriptSpecs {
			if spec.Name == specName {
				return spec, true
			}
		}
		return appsv1alpha1.ComponentTemplateSpec{}, false
	}

	if w.renderedOpts.ConfigSpec != "" {
		for _, com := range w.clusterDefComponents {
			if spec, ok := foundSpec(com, w.renderedOpts.ConfigSpec); ok {
				return []componentedConfigSpec{{com.Name, spec}}, nil
			}
		}
		return nil, cfgcore.MakeError("config spec[%s] is not found", w.renderedOpts.ConfigSpec)
	}

	if !w.renderedOpts.AllConfigSpecs {
		return nil, cfgcore.MakeError("AllConfigSpecs should be set while config spec is unset")
	}
	configSpecs := make([]componentedConfigSpec, 0)
	for _, com := range w.clusterDefComponents {
		for _, configSpec := range com.ConfigSpecs {
			configSpecs = append(configSpecs, componentedConfigSpec{com.Name, configSpec.ComponentTemplateSpec})
		}
		for _, configSpec := range com.ScriptSpecs {
			configSpecs = append(configSpecs, componentedConfigSpec{com.Name, configSpec})
		}
	}
	return configSpecs, nil
}

func (w *templateRenderWorkflow) createClusterObject() (*appsv1alpha1.Cluster, error) {
	if w.clusterYaml != "" {
		return CustomizedObjFromYaml(w.clusterYaml, generics.ClusterSignature)
	}

	clusterVersionObj := GetTypedResourceObjectBySignature(w.localObjects, generics.ClusterVersionSignature)
	return mockClusterObject(w.clusterDefObj, w.renderedOpts, clusterVersionObj), nil
}

func NewWorkflowTemplateRender(helmTemplateDir string, opts RenderedOptions) (*templateRenderWorkflow, error) {
	if _, err := os.Stat(helmTemplateDir); err != nil {
		panic("cluster definition yaml file is required")
	}

	allObjects, err := CreateObjectsFromDirectory(helmTemplateDir)
	if err != nil {
		return nil, err
	}

	clusterDefObj := GetTypedResourceObjectBySignature(allObjects, generics.ClusterDefinitionSignature)
	if clusterDefObj == nil {
		return nil, cfgcore.MakeError("cluster definition object is not found in helm template directory[%s]", helmTemplateDir)
	}
	// hack apiserver auto filefield
	checkAndFillPortProtocol(clusterDefObj.Spec.ComponentDefs)

	components := clusterDefObj.Spec.ComponentDefs
	if opts.ComponentName != "" {
		component := clusterDefObj.GetComponentDefByName(opts.ComponentName)
		if component == nil {
			return nil, cfgcore.MakeError("component[%s] is not defined in cluster definition", opts.ComponentName)
		}
		components = []appsv1alpha1.ClusterComponentDefinition{*component}
	}
	return &templateRenderWorkflow{
		renderedOpts:         opts,
		clusterDefObj:        clusterDefObj,
		localObjects:         allObjects,
		clusterDefComponents: components,
	}, nil
}

func checkAndFillPortProtocol(clusterDefComponents []appsv1alpha1.ClusterComponentDefinition) {
	// set a default protocol with 'TCP' to avoid failure in BuildHeadlessSvc
	for i := range clusterDefComponents {
		for j := range clusterDefComponents[i].PodSpec.Containers {
			container := &clusterDefComponents[i].PodSpec.Containers[j]
			for k := range container.Ports {
				port := &container.Ports[k]
				if port.Protocol == "" {
					port.Protocol = corev1.ProtocolTCP
				}
			}
		}
	}
}

func renderTemplates(configSpec appsv1alpha1.ComponentTemplateSpec, outputDir, clusterName, compName string, objects []client.Object, componentDefName string) error {
	cfgName := cfgcore.GetComponentCfgName(clusterName, compName, configSpec.Name)
	output := filepath.Join(outputDir, cfgName)
	fmt.Printf("dump rendering template spec: %s, output directory: %s\n",
		printer.BoldYellow(fmt.Sprintf("%s.%s", componentDefName, configSpec.Name)), output)

	if err := os.MkdirAll(output, 0755); err != nil {
		return err
	}

	var ok bool
	var cm *corev1.ConfigMap
	for _, obj := range objects {
		if cm, ok = obj.(*corev1.ConfigMap); !ok || cm.Name != cfgName {
			continue
		}
		for file, val := range cm.Data {
			if err := os.WriteFile(filepath.Join(output, file), []byte(val), 0755); err != nil {
				return err
			}
		}
		break
	}
	return nil
}

func createComponentObjects(w *templateRenderWorkflow, ctx intctrlutil.RequestCtx, cli client.Client,
	componentType string, cluster *appsv1alpha1.Cluster) ([]client.Object, error) {
	compName, err := w.getComponentName(componentType, cluster)
	if err != nil {
		return nil, err
	}
	clusterVersionObj := GetTypedResourceObjectBySignature(w.localObjects, generics.ClusterVersionSignature)
	component, err := components.NewComponent(ctx, cli, w.clusterDefObj, clusterVersionObj, cluster, compName, nil)
	if err != nil {
		return nil, err
	}

	objs := make([]client.Object, 0)
	secret, err := builder.BuildConnCredential(w.clusterDefObj, cluster, component.GetSynthesizedComponent())
	if err != nil {
		return nil, err
	}
	objs = append(objs, secret)

	compObjs, err := component.GetBuiltObjects(ctx, cli)
	if err != nil {
		return nil, err
	}
	objs = append(objs, compObjs...)

	return objs, nil
}

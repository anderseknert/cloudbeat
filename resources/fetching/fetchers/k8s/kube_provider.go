// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package fetchers

import (
	"reflect"

	"github.com/elastic/elastic-agent-autodiscover/kubernetes"
	"github.com/elastic/elastic-agent-libs/logp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/elastic/cloudbeat/resources/fetching"
)

type K8sResource struct {
	log  *logp.Logger
	Data interface{}
}

const (
	k8sObjMetadataField  = "ObjectMeta"
	k8sTypeMetadataField = "TypeMeta"
	K8sObjType           = "k8s_object"
)

func getKubeData(log *logp.Logger, watchers []kubernetes.Watcher, resCh chan fetching.ResourceInfo, cMetadata fetching.CycleMetadata) {
	log.Debug("Starting getKubeData")

	for _, watcher := range watchers {
		rs := watcher.Store().List()

		for _, r := range rs {
			nullifyManagedFields(r)
			resource, ok := r.(kubernetes.Resource)

			if !ok {
				log.Errorf("Bad resource: %#v does not implement kubernetes.Resource", r)
				continue
			}

			err := addTypeInformationToKubeResource(resource)
			if err != nil {
				log.Errorf("Bad resource: %v", err)
				continue
			} // See https://github.com/kubernetes/kubernetes/issues/3030
			resCh <- fetching.ResourceInfo{Resource: K8sResource{log, resource}, CycleMetadata: cMetadata}
		}
	}
}

func (r K8sResource) GetData() interface{} {
	return r.Data
}

func (r K8sResource) GetMetadata() (fetching.ResourceMetadata, error) {
	k8sObj := reflect.Indirect(reflect.ValueOf(r.Data))
	k8sObjMeta := getK8sObjectMeta(r.log, k8sObj)
	resourceID := k8sObjMeta.UID
	resourceName := k8sObjMeta.Name

	return fetching.ResourceMetadata{
		ID:      string(resourceID),
		Type:    K8sObjType,
		SubType: getK8sSubType(r.log, k8sObj),
		Name:    resourceName,
	}, nil
}

func (r K8sResource) GetElasticCommonData() (map[string]any, error) { return nil, nil }

func getK8sObjectMeta(log *logp.Logger, k8sObj reflect.Value) metav1.ObjectMeta {
	metadata, ok := k8sObj.FieldByName(k8sObjMetadataField).Interface().(metav1.ObjectMeta)
	if !ok {
		log.Errorf("Failed to retrieve object metadata, Resource: %#v", k8sObj)
		return metav1.ObjectMeta{}
	}

	return metadata
}

func getK8sSubType(log *logp.Logger, k8sObj reflect.Value) string {
	typeMeta, ok := k8sObj.FieldByName(k8sTypeMetadataField).Interface().(metav1.TypeMeta)
	if !ok {
		log.Errorf("Failed to retrieve type metadata, Resource: %#v", k8sObj)
		return ""
	}

	return typeMeta.Kind
}

// nullifyManagedFields ManagedFields field contains fields with dot that prevent from elasticsearch to index
// the events.
func nullifyManagedFields(resource interface{}) {
	switch val := resource.(type) {
	case *kubernetes.Pod:
		val.ManagedFields = nil
	case *kubernetes.Role:
		val.ManagedFields = nil
	case *kubernetes.RoleBinding:
		val.ManagedFields = nil
	case *kubernetes.ClusterRole:
		val.ManagedFields = nil
	case *kubernetes.ClusterRoleBinding:
		val.ManagedFields = nil
	case *kubernetes.PodSecurityPolicy:
		val.ManagedFields = nil
	case *kubernetes.ServiceAccount:
		val.ManagedFields = nil
	case *kubernetes.NetworkPolicy:
		val.ManagedFields = nil
	case *kubernetes.Node:
		val.ManagedFields = nil
	}
}

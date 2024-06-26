/*
Copyright 2023 The Godel Scheduler Authors.

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

package helper

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	appslisters "k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
)

// DefaultSelector returns a selector deduced from the Services, Replication
// Controllers, Replica Sets, and Stateful Sets matching the given pod.
func DefaultSelector(pod *v1.Pod, sl corelisters.ServiceLister, cl corelisters.ReplicationControllerLister, rsl appslisters.ReplicaSetLister, ssl appslisters.StatefulSetLister) labels.Selector {
	labelSet := make(labels.Set)
	// Since services, RCs, RSs and SSs match the pod, they won't have conflicting
	// labels. Merging is safe.

	if services, err := GetPodServices(sl, pod); err == nil {
		for _, service := range services {
			labelSet = labels.Merge(labelSet, service.Spec.Selector)
		}
	}

	if rcs, err := cl.GetPodControllers(pod); err == nil {
		for _, rc := range rcs {
			labelSet = labels.Merge(labelSet, rc.Spec.Selector)
		}
	}

	selector := labels.NewSelector()
	if len(labelSet) != 0 {
		selector = labelSet.AsSelector()
	}

	if rss, err := rsl.GetPodReplicaSets(pod); err == nil {
		for _, rs := range rss {
			selector = addPodSelectorFromLabelSelector(selector, rs.Spec.Selector)
		}
	}

	if sss, err := ssl.GetPodStatefulSets(pod); err == nil {
		for _, ss := range sss {
			selector = addPodSelectorFromLabelSelector(selector, ss.Spec.Selector)
		}
	}

	return selector
}

func addPodSelectorFromLabelSelector(selector labels.Selector, labelSelector *metav1.LabelSelector) labels.Selector {
	if other, err := metav1.LabelSelectorAsSelector(labelSelector); err == nil {
		if r, ok := other.Requirements(); ok {
			selector = selector.Add(r...)
		}
	}
	return selector
}

// GetPodServices gets the services that have the selector that match the labels on the given pod.
func GetPodServices(sl corelisters.ServiceLister, pod *v1.Pod) ([]*v1.Service, error) {
	allServices, err := sl.Services(pod.Namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	var services []*v1.Service
	for i := range allServices {
		service := allServices[i]
		if service.Spec.Selector == nil {
			// services with nil selectors match nothing, not everything.
			continue
		}
		selector := labels.Set(service.Spec.Selector).AsSelectorPreValidated()
		if selector.Matches(labels.Set(pod.Labels)) {
			services = append(services, service)
		}
	}

	return services, nil
}

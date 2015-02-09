/*
Copyright 2014 Google Inc. All rights reserved.

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

package v1beta2_test

import (
	"encoding/json"
	"testing"

	newer "github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api/resource"
	current "github.com/GoogleCloudPlatform/kubernetes/pkg/api/v1beta2"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"
)

func TestServiceEmptySelector(t *testing.T) {
	// Nil map should be preserved
	svc := &current.Service{Selector: nil}
	data, err := newer.Scheme.EncodeToVersion(svc, "v1beta2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	obj, err := newer.Scheme.Decode(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	selector := obj.(*newer.Service).Spec.Selector
	if selector != nil {
		t.Errorf("unexpected selector: %#v", obj)
	}

	// Empty map should be preserved
	svc2 := &current.Service{Selector: map[string]string{}}
	data, err = newer.Scheme.EncodeToVersion(svc2, "v1beta2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	obj, err = newer.Scheme.Decode(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	selector = obj.(*newer.Service).Spec.Selector
	if selector == nil || len(selector) != 0 {
		t.Errorf("unexpected selector: %#v", obj)
	}
}

func TestNodeConversion(t *testing.T) {
	version, kind, err := newer.Scheme.ObjectVersionAndKind(&current.Minion{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version != "v1beta2" || kind != "Minion" {
		t.Errorf("unexpected version and kind: %s %s", version, kind)
	}

	newer.Scheme.Log(t)
	obj, err := current.Codec.Decode([]byte(`{"kind":"Node","apiVersion":"v1beta2"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := obj.(*newer.Node); !ok {
		t.Errorf("unexpected type: %#v", obj)
	}

	obj, err = current.Codec.Decode([]byte(`{"kind":"NodeList","apiVersion":"v1beta2"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := obj.(*newer.NodeList); !ok {
		t.Errorf("unexpected type: %#v", obj)
	}

	obj = &newer.Node{}
	if err := current.Codec.DecodeInto([]byte(`{"kind":"Node","apiVersion":"v1beta2"}`), obj); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	obj = &newer.Node{}
	data, err := current.Codec.Encode(obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := map[string]interface{}{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m["kind"] != "Minion" {
		t.Errorf("unexpected encoding: %s - %#v", m["kind"], string(data))
	}
}

func TestPullPolicyConversion(t *testing.T) {
	table := []struct {
		versioned current.PullPolicy
		internal  newer.PullPolicy
	}{
		{
			versioned: current.PullAlways,
			internal:  newer.PullAlways,
		}, {
			versioned: current.PullNever,
			internal:  newer.PullNever,
		}, {
			versioned: current.PullIfNotPresent,
			internal:  newer.PullIfNotPresent,
		}, {
			versioned: "",
			internal:  "",
		}, {
			versioned: "invalid value",
			internal:  "invalid value",
		},
	}
	for _, item := range table {
		var got newer.PullPolicy
		err := newer.Scheme.Convert(&item.versioned, &got)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
			continue
		}
		if e, a := item.internal, got; e != a {
			t.Errorf("Expected: %q, got %q", e, a)
		}
	}
	for _, item := range table {
		var got current.PullPolicy
		err := newer.Scheme.Convert(&item.internal, &got)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
			continue
		}
		if e, a := item.versioned, got; e != a {
			t.Errorf("Expected: %q, got %q", e, a)
		}
	}
}

func getResourceRequirements(cpu, memory resource.Quantity) current.ResourceRequirements {
	res := current.ResourceRequirements{}
	res.Limits = current.ResourceList{}
	if cpu.Value() > 0 {
		res.Limits[current.ResourceCPU] = util.NewIntOrStringFromInt(int(cpu.Value()))
	}
	if memory.Value() > 0 {
		res.Limits[current.ResourceMemory] = util.NewIntOrStringFromInt(int(memory.Value()))
	}

	return res
}

func TestContainerConversion(t *testing.T) {
	cpuLimit := resource.MustParse("10")
	memoryLimit := resource.MustParse("10M")
	null := resource.Quantity{}
	testCases := []current.Container{
		{
			Name:      "container",
			Resources: getResourceRequirements(cpuLimit, memoryLimit),
		},
		{
			Name:      "container",
			CPU:       int(cpuLimit.MilliValue()),
			Resources: getResourceRequirements(null, memoryLimit),
		},
		{
			Name:      "container",
			Memory:    memoryLimit.Value(),
			Resources: getResourceRequirements(cpuLimit, null),
		},
		{
			Name:   "container",
			CPU:    int(cpuLimit.MilliValue()),
			Memory: memoryLimit.Value(),
		},
		{
			Name:      "container",
			Memory:    memoryLimit.Value(),
			Resources: getResourceRequirements(cpuLimit, resource.MustParse("100M")),
		},
		{
			Name:      "container",
			CPU:       int(cpuLimit.MilliValue()),
			Resources: getResourceRequirements(resource.MustParse("500"), memoryLimit),
		},
	}

	for i, tc := range testCases {
		got := newer.Container{}
		if err := newer.Scheme.Convert(&tc, &got); err != nil {
			t.Errorf("[Case: %d] Unexpected error: %v", i, err)
			continue
		}
		if cpu := got.Resources.Limits.Cpu(); cpu.Value() != cpuLimit.Value() {
			t.Errorf("[Case: %d] Expected cpu: %v, got: %v", i, cpuLimit, *cpu)
		}
		if memory := got.Resources.Limits.Memory(); memory.Value() != memoryLimit.Value() {
			t.Errorf("[Case: %d] Expected memory: %v, got: %v", i, memoryLimit, *memory)
		}
	}
}

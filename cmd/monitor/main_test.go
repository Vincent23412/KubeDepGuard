package main

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestSchemeRegistersEventSources(t *testing.T) {
	tests := []struct {
		name   string
		object runtime.Object
	}{
		{name: "Pod", object: &corev1.Pod{}},
		{name: "Service", object: &corev1.Service{}},
		{name: "Deployment", object: &appsv1.Deployment{}},
		{name: "Ingress", object: &networkingv1.Ingress{}},
		{name: "NetworkPolicy", object: &networkingv1.NetworkPolicy{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kinds, _, err := scheme().ObjectKinds(tt.object)
			if err != nil {
				t.Fatalf("event source is not registered: %v", err)
			}
			if len(kinds) == 0 {
				t.Fatal("event source has no registered kind")
			}
		})
	}
}

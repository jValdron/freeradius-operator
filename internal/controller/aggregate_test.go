package controller

import "testing"

func TestResourceBelongsToCluster(t *testing.T) {
	tests := []struct {
		name        string
		labels      map[string]string
		cluster     string
		isDefault   bool
		wantBelongs bool
	}{
		{"unlabeled default", nil, "primary", true, true},
		{"unlabeled non-default", nil, "primary", false, false},
		{"clusterName match", map[string]string{"clusterName": "private"}, "private", false, true},
		{"clusterName mismatch", map[string]string{"clusterName": "other"}, "private", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resourceBelongsToCluster(tt.labels, tt.cluster, tt.isDefault)
			if got != tt.wantBelongs {
				t.Fatalf("got %v want %v", got, tt.wantBelongs)
			}
		})
	}
}

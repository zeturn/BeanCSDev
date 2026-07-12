package service

import (
	"testing"

	"github.com/zeturn/beancs-controller/internal/model"
)

func TestNormalizeProjectVolumes(t *testing.T) {
	volumes, err := normalizeProjectVolumes("api", []model.ProjectVolume{
		{Name: "cache", Type: "emptyDir", MountPath: "/cache", Size: "1Gi"},
		{Name: "data", Type: "PVC", MountPath: "/var/lib/app", Size: "10Gi", StorageClassName: "local-path", AccessModes: []string{"ReadWriteOnce", "ReadWriteOnce"}},
		{Name: "archive", Type: "existingPVC", MountPath: "/archive", ClaimName: "shared-archive"},
	})
	if err != nil {
		t.Fatalf("normalizeProjectVolumes() error = %v", err)
	}
	if len(volumes) != 3 || volumes[0].Type != "emptyDir" || volumes[0].Size != "" || len(volumes[0].AccessModes) != 0 {
		t.Fatalf("unexpected emptyDir normalization: %#v", volumes[0])
	}
	if volumes[1].Type != "pvc" || volumes[1].Size != "10Gi" || len(volumes[1].AccessModes) != 1 {
		t.Fatalf("unexpected PVC normalization: %#v", volumes[1])
	}
	if volumes[2].Type != "existingPVC" || volumes[2].ClaimName != "shared-archive" {
		t.Fatalf("unexpected existing PVC normalization: %#v", volumes[2])
	}
}

func TestNormalizeProjectVolumesRejectsInvalidPVC(t *testing.T) {
	_, err := normalizeProjectVolumes("api", []model.ProjectVolume{{Name: "data", Type: "pvc", MountPath: "relative", Size: "0Gi"}})
	if err == nil {
		t.Fatal("normalizeProjectVolumes() error = nil, want validation failure")
	}
}

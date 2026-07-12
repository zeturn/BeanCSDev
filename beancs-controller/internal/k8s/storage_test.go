package k8s

import (
	"context"
	"testing"
	"time"

	"github.com/zeturn/beancs-controller/internal/model"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestStorageOverviewSummarizesResourcesAndMountedClaims(t *testing.T) {
	allowExpansion := true
	mode := corev1.PersistentVolumeFilesystem
	client := fake.NewSimpleClientset(
		&corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: "data", Namespace: "apps", CreationTimestamp: metav1.NewTime(time.Now().Add(-time.Hour))},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				StorageClassName: stringPtr("local-path"),
				VolumeName:       "pv-data",
				VolumeMode:       &mode,
				Resources:        corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("10Gi")}},
			},
			Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound, Capacity: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("10Gi")}},
		},
		&corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{Name: "pv-data"},
			Spec: corev1.PersistentVolumeSpec{
				Capacity:                      corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("10Gi")},
				AccessModes:                   []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				StorageClassName:              "local-path",
				PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
				ClaimRef:                      &corev1.ObjectReference{Namespace: "apps", Name: "data"},
				VolumeMode:                    &mode,
			},
			Status: corev1.PersistentVolumeStatus{Phase: corev1.VolumeBound},
		},
		&storagev1.StorageClass{
			ObjectMeta:           metav1.ObjectMeta{Name: "local-path", Annotations: map[string]string{"storageclass.kubernetes.io/is-default-class": "true"}},
			Provisioner:          "rancher.io/local-path",
			ReclaimPolicy:        persistentVolumeReclaimPolicyPtr(corev1.PersistentVolumeReclaimDelete),
			VolumeBindingMode:    volumeBindingModePtr(storagev1.VolumeBindingWaitForFirstConsumer),
			AllowVolumeExpansion: &allowExpansion,
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "apps"},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Name: "api", VolumeMounts: []corev1.VolumeMount{{Name: "data", MountPath: "/data"}}}},
				Volumes:    []corev1.Volume{{Name: "data", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "data"}}}},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "unmounted", Namespace: "apps"},
			Spec:       corev1.PodSpec{Volumes: []corev1.Volume{{Name: "data", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "data"}}}}},
		},
	)

	out, err := (&Manager{Clientset: client}).StorageOverview(context.Background())
	if err != nil {
		t.Fatalf("StorageOverview() error = %v", err)
	}
	if len(out.PersistentVolumeClaims) != 1 || len(out.PersistentVolumes) != 1 || len(out.StorageClasses) != 1 {
		t.Fatalf("unexpected resource counts: %#v", out)
	}
	pvc := out.PersistentVolumeClaims[0]
	if pvc.RequestedCapacity != "10Gi" || pvc.Capacity != "10Gi" || pvc.StorageClass != "local-path" || pvc.VolumeName != "pv-data" {
		t.Fatalf("unexpected PVC summary: %#v", pvc)
	}
	if len(pvc.MountedBy) != 1 || pvc.MountedBy[0] != "apps/api" {
		t.Fatalf("unexpected PVC mount users: %#v", pvc.MountedBy)
	}
	class := out.StorageClasses[0]
	if !class.Default || !class.AllowVolumeExpansion || class.VolumeBindingMode != string(storagev1.VolumeBindingWaitForFirstConsumer) {
		t.Fatalf("unexpected StorageClass summary: %#v", class)
	}
}

func TestReconcileProjectVolumesUsesExistingPVCWithoutCreatingClaim(t *testing.T) {
	client := fake.NewSimpleClientset()
	manager := &Manager{Clientset: client}
	volumes, mounts, err := manager.reconcileProjectVolumes(context.Background(), "apps", "api", []model.ProjectVolume{{
		Name:      "shared",
		Type:      "existingPVC",
		MountPath: "/shared",
		ClaimName: "shared-data",
	}})
	if err != nil {
		t.Fatalf("reconcileProjectVolumes() error = %v", err)
	}
	if len(volumes) != 1 || volumes[0].PersistentVolumeClaim == nil || volumes[0].PersistentVolumeClaim.ClaimName != "shared-data" {
		t.Fatalf("unexpected volumes: %#v", volumes)
	}
	if len(mounts) != 1 || mounts[0].MountPath != "/shared" {
		t.Fatalf("unexpected mounts: %#v", mounts)
	}
	claims, err := client.CoreV1().PersistentVolumeClaims("apps").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("list claims: %v", err)
	}
	if len(claims.Items) != 0 {
		t.Fatalf("existing PVC mount created claims: %#v", claims.Items)
	}
}

func stringPtr(value string) *string { return &value }

func persistentVolumeReclaimPolicyPtr(value corev1.PersistentVolumeReclaimPolicy) *corev1.PersistentVolumeReclaimPolicy {
	return &value
}

func volumeBindingModePtr(value storagev1.VolumeBindingMode) *storagev1.VolumeBindingMode {
	return &value
}

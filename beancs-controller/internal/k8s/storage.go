package k8s

import (
	"context"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type StorageOverview struct {
	PersistentVolumeClaims []PersistentVolumeClaimSummary `json:"persistent_volume_claims"`
	PersistentVolumes      []PersistentVolumeSummary      `json:"persistent_volumes"`
	StorageClasses         []StorageClassSummary          `json:"storage_classes"`
	CheckedAt              time.Time                      `json:"checked_at"`
}

type PersistentVolumeClaimSummary struct {
	Namespace         string    `json:"namespace"`
	Name              string    `json:"name"`
	Phase             string    `json:"phase"`
	RequestedCapacity string    `json:"requested_capacity"`
	Capacity          string    `json:"capacity"`
	AccessModes       []string  `json:"access_modes"`
	StorageClass      string    `json:"storage_class"`
	VolumeName        string    `json:"volume_name"`
	VolumeMode        string    `json:"volume_mode"`
	MountedBy         []string  `json:"mounted_by"`
	AgeSeconds        int64     `json:"age_seconds"`
	CreatedAt         time.Time `json:"created_at"`
}

type PersistentVolumeSummary struct {
	Name          string    `json:"name"`
	Phase         string    `json:"phase"`
	Capacity      string    `json:"capacity"`
	AccessModes   []string  `json:"access_modes"`
	StorageClass  string    `json:"storage_class"`
	Claim         string    `json:"claim"`
	ReclaimPolicy string    `json:"reclaim_policy"`
	VolumeMode    string    `json:"volume_mode"`
	AgeSeconds    int64     `json:"age_seconds"`
	CreatedAt     time.Time `json:"created_at"`
}

type StorageClassSummary struct {
	Name                 string    `json:"name"`
	Provisioner          string    `json:"provisioner"`
	Default              bool      `json:"default"`
	ReclaimPolicy        string    `json:"reclaim_policy"`
	VolumeBindingMode    string    `json:"volume_binding_mode"`
	AllowVolumeExpansion bool      `json:"allow_volume_expansion"`
	AgeSeconds           int64     `json:"age_seconds"`
	CreatedAt            time.Time `json:"created_at"`
}

func (m *Manager) StorageOverview(ctx context.Context) (*StorageOverview, error) {
	if err := m.ensure(); err != nil {
		return nil, err
	}

	pvcs, err := m.Clientset.CoreV1().PersistentVolumeClaims("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	pvs, err := m.Clientset.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	classes, err := m.Clientset.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	pods, err := m.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	mountedBy := mountedPVCsByPod(pods.Items)
	out := &StorageOverview{
		PersistentVolumeClaims: make([]PersistentVolumeClaimSummary, 0, len(pvcs.Items)),
		PersistentVolumes:      make([]PersistentVolumeSummary, 0, len(pvs.Items)),
		StorageClasses:         make([]StorageClassSummary, 0, len(classes.Items)),
		CheckedAt:              now,
	}
	for _, pvc := range pvcs.Items {
		key := pvc.Namespace + "/" + pvc.Name
		out.PersistentVolumeClaims = append(out.PersistentVolumeClaims, PersistentVolumeClaimSummary{
			Namespace:         pvc.Namespace,
			Name:              pvc.Name,
			Phase:             string(pvc.Status.Phase),
			RequestedCapacity: storageQuantity(pvc.Spec.Resources.Requests),
			Capacity:          storageQuantity(pvc.Status.Capacity),
			AccessModes:       accessModes(pvc.Spec.AccessModes),
			StorageClass:      stringValue(pvc.Spec.StorageClassName),
			VolumeName:        pvc.Spec.VolumeName,
			VolumeMode:        volumeMode(pvc.Spec.VolumeMode),
			MountedBy:         mountedBy[key],
			AgeSeconds:        ageSeconds(now, pvc.CreationTimestamp.Time),
			CreatedAt:         pvc.CreationTimestamp.Time,
		})
	}
	for _, pv := range pvs.Items {
		claim := ""
		if pv.Spec.ClaimRef != nil {
			claim = pv.Spec.ClaimRef.Namespace + "/" + pv.Spec.ClaimRef.Name
		}
		out.PersistentVolumes = append(out.PersistentVolumes, PersistentVolumeSummary{
			Name:          pv.Name,
			Phase:         string(pv.Status.Phase),
			Capacity:      storageQuantity(pv.Spec.Capacity),
			AccessModes:   accessModes(pv.Spec.AccessModes),
			StorageClass:  pv.Spec.StorageClassName,
			Claim:         claim,
			ReclaimPolicy: string(pv.Spec.PersistentVolumeReclaimPolicy),
			VolumeMode:    volumeMode(pv.Spec.VolumeMode),
			AgeSeconds:    ageSeconds(now, pv.CreationTimestamp.Time),
			CreatedAt:     pv.CreationTimestamp.Time,
		})
	}
	for _, class := range classes.Items {
		out.StorageClasses = append(out.StorageClasses, storageClassSummary(class, now))
	}
	sort.Slice(out.PersistentVolumeClaims, func(i, j int) bool {
		left, right := out.PersistentVolumeClaims[i], out.PersistentVolumeClaims[j]
		if left.Namespace == right.Namespace {
			return left.Name < right.Name
		}
		return left.Namespace < right.Namespace
	})
	sort.Slice(out.PersistentVolumes, func(i, j int) bool { return out.PersistentVolumes[i].Name < out.PersistentVolumes[j].Name })
	sort.Slice(out.StorageClasses, func(i, j int) bool { return out.StorageClasses[i].Name < out.StorageClasses[j].Name })
	return out, nil
}

func mountedPVCsByPod(pods []corev1.Pod) map[string][]string {
	claims := map[string][]string{}
	for _, pod := range pods {
		if pod.DeletionTimestamp != nil || pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			continue
		}
		mountedVolumes := map[string]bool{}
		for _, container := range append(append([]corev1.Container{}, pod.Spec.InitContainers...), pod.Spec.Containers...) {
			for _, mount := range container.VolumeMounts {
				mountedVolumes[mount.Name] = true
			}
		}
		for _, volume := range pod.Spec.Volumes {
			if volume.PersistentVolumeClaim == nil || !mountedVolumes[volume.Name] {
				continue
			}
			key := pod.Namespace + "/" + volume.PersistentVolumeClaim.ClaimName
			claims[key] = append(claims[key], pod.Namespace+"/"+pod.Name)
		}
	}
	for key := range claims {
		sort.Strings(claims[key])
	}
	return claims
}

func storageClassSummary(class storagev1.StorageClass, now time.Time) StorageClassSummary {
	return StorageClassSummary{
		Name:                 class.Name,
		Provisioner:          class.Provisioner,
		Default:              class.Annotations["storageclass.kubernetes.io/is-default-class"] == "true" || class.Annotations["storageclass.beta.kubernetes.io/is-default-class"] == "true",
		ReclaimPolicy:        stringValue(class.ReclaimPolicy),
		VolumeBindingMode:    stringValue(class.VolumeBindingMode),
		AllowVolumeExpansion: class.AllowVolumeExpansion != nil && *class.AllowVolumeExpansion,
		AgeSeconds:           ageSeconds(now, class.CreationTimestamp.Time),
		CreatedAt:            class.CreationTimestamp.Time,
	}
}

func storageQuantity(values corev1.ResourceList) string {
	if quantity, ok := values[corev1.ResourceStorage]; ok {
		return quantity.String()
	}
	return ""
}

func accessModes(modes []corev1.PersistentVolumeAccessMode) []string {
	out := make([]string, len(modes))
	for i, mode := range modes {
		out[i] = string(mode)
	}
	return out
}

func volumeMode(mode *corev1.PersistentVolumeMode) string {
	return stringValue(mode)
}

func stringValue[T ~string](value *T) string {
	if value == nil {
		return ""
	}
	return string(*value)
}

func ageSeconds(now, createdAt time.Time) int64 {
	if createdAt.IsZero() {
		return 0
	}
	return int64(now.Sub(createdAt).Seconds())
}

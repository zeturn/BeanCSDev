package model

import "encoding/json"

type ProjectHealthCheck struct {
	Type                string `json:"type"`
	Path                string `json:"path,omitempty"`
	Port                any    `json:"port,omitempty"`
	InitialDelaySeconds *int   `json:"initialDelaySeconds,omitempty"`
	PeriodSeconds       *int   `json:"periodSeconds,omitempty"`
	TimeoutSeconds      *int   `json:"timeoutSeconds,omitempty"`
}

type ProjectVolume struct {
	Name             string   `json:"name"`
	Type             string   `json:"type"`
	MountPath        string   `json:"mountPath"`
	ClaimName        string   `json:"claimName,omitempty"`
	Size             string   `json:"size,omitempty"`
	StorageClassName string   `json:"storageClassName,omitempty"`
	AccessModes      []string `json:"accessModes,omitempty"`
}

func (p Project) HealthCheckConfig() *ProjectHealthCheck {
	if len(p.HealthCheck) == 0 {
		return nil
	}
	raw, err := json.Marshal(p.HealthCheck)
	if err != nil {
		return nil
	}
	var out ProjectHealthCheck
	if err := json.Unmarshal(raw, &out); err != nil || out.Type == "" {
		return nil
	}
	return &out
}

func (p Project) VolumeConfig() []ProjectVolume {
	if len(p.Volumes) == 0 {
		return nil
	}
	raw, err := json.Marshal(p.Volumes["items"])
	if err != nil {
		return nil
	}
	var out []ProjectVolume
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	for i := range out {
		if len(out[i].AccessModes) == 0 {
			out[i].AccessModes = []string{"ReadWriteOnce"}
		}
	}
	return out
}

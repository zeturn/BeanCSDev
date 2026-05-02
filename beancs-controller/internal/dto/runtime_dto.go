package dto

type RuntimeLabelPatchRequest struct {
	Labels map[string]string `json:"labels" validate:"omitempty"`
}

type CreateNamespaceRequest struct {
	Name   string            `json:"name" validate:"required,hostname_rfc1123,max=63"`
	Labels map[string]string `json:"labels" validate:"omitempty"`
}

type CreateServiceRequest struct {
	Namespace string            `json:"namespace" validate:"required,hostname_rfc1123,max=63"`
	Name      string            `json:"name" validate:"required,hostname_rfc1123,max=63"`
	Type      string            `json:"type" validate:"omitempty,oneof=ClusterIP NodePort LoadBalancer"`
	Selector  map[string]string `json:"selector" validate:"omitempty"`
	Ports     []ServicePortSpec `json:"ports" validate:"required,min=1"`
	Labels    map[string]string `json:"labels" validate:"omitempty"`
}

type UpdateServiceRequest struct {
	Type     string            `json:"type" validate:"omitempty,oneof=ClusterIP NodePort LoadBalancer"`
	Selector map[string]string `json:"selector" validate:"omitempty"`
	Ports    []ServicePortSpec `json:"ports" validate:"required,min=1"`
	Labels   map[string]string `json:"labels" validate:"omitempty"`
}

type ServicePortSpec struct {
	Name       string `json:"name" validate:"omitempty,max=15"`
	Port       int32  `json:"port" validate:"required,min=1,max=65535"`
	TargetPort int32  `json:"target_port" validate:"omitempty,min=1,max=65535"`
	Protocol   string `json:"protocol" validate:"omitempty,oneof=TCP UDP SCTP"`
}

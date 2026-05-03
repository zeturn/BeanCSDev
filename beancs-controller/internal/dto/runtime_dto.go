package dto

type RuntimeLabelPatchRequest struct {
	Labels map[string]string `json:"labels" validate:"omitempty"`
}

type RuntimeTaintRequest struct {
	Taints []RuntimeTaintSpec `json:"taints" validate:"omitempty,dive"`
}

type RuntimeTaintSpec struct {
	Key    string `json:"key" validate:"required"`
	Value  string `json:"value" validate:"omitempty"`
	Effect string `json:"effect" validate:"required,oneof=NoSchedule PreferNoSchedule NoExecute"`
}

type DrainNodeRequest struct {
	Force              bool  `json:"force"`
	IgnoreDaemonSets   bool  `json:"ignore_daemonsets"`
	DeleteEmptyDirData bool  `json:"delete_emptydir_data"`
	GracePeriodSeconds int64 `json:"grace_period_seconds" validate:"omitempty,min=0,max=3600"`
}

type CreateNamespaceRequest struct {
	Name   string            `json:"name" validate:"required,hostname_rfc1123,max=63"`
	Labels map[string]string `json:"labels" validate:"omitempty"`
}

type ResourceQuotaRequest struct {
	Name string            `json:"name" validate:"required,hostname_rfc1123,max=63"`
	Hard map[string]string `json:"hard" validate:"required"`
}

type LimitRangeRequest struct {
	Name           string            `json:"name" validate:"required,hostname_rfc1123,max=63"`
	Type           string            `json:"type" validate:"required,oneof=Container Pod PersistentVolumeClaim"`
	Default        map[string]string `json:"default" validate:"omitempty"`
	DefaultRequest map[string]string `json:"default_request" validate:"omitempty"`
	Min            map[string]string `json:"min" validate:"omitempty"`
	Max            map[string]string `json:"max" validate:"omitempty"`
}

type NamespacePermissionRequest struct {
	Name      string                       `json:"name" validate:"required,hostname_rfc1123,max=63"`
	Subjects  []NamespacePermissionSubject `json:"subjects" validate:"required,min=1,dive"`
	Verbs     []string                     `json:"verbs" validate:"required,min=1,dive,required"`
	Resources []string                     `json:"resources" validate:"required,min=1,dive,required"`
	APIGroups []string                     `json:"api_groups" validate:"omitempty,dive"`
}

type NamespacePermissionSubject struct {
	Kind      string `json:"kind" validate:"required,oneof=User Group ServiceAccount"`
	Name      string `json:"name" validate:"required"`
	Namespace string `json:"namespace" validate:"omitempty"`
}

type NamespaceIsolationRequest struct {
	Enabled            bool `json:"enabled"`
	AllowSameNamespace bool `json:"allow_same_namespace"`
	AllowDNS           bool `json:"allow_dns"`
}

type CreateServiceRequest struct {
	Namespace             string            `json:"namespace" validate:"required,hostname_rfc1123,max=63"`
	Name                  string            `json:"name" validate:"required,hostname_rfc1123,max=63"`
	Type                  string            `json:"type" validate:"omitempty,oneof=ClusterIP NodePort LoadBalancer"`
	Selector              map[string]string `json:"selector" validate:"omitempty"`
	Ports                 []ServicePortSpec `json:"ports" validate:"required,min=1"`
	Labels                map[string]string `json:"labels" validate:"omitempty"`
	LoadBalancerIP        string            `json:"load_balancer_ip" validate:"omitempty,ip"`
	ExternalIPs           []string          `json:"external_ips" validate:"omitempty,dive,ip"`
	ExternalTrafficPolicy string            `json:"external_traffic_policy" validate:"omitempty,oneof=Cluster Local"`
}

type UpdateServiceRequest struct {
	Type                  string            `json:"type" validate:"omitempty,oneof=ClusterIP NodePort LoadBalancer"`
	Selector              map[string]string `json:"selector" validate:"omitempty"`
	Ports                 []ServicePortSpec `json:"ports" validate:"required,min=1"`
	Labels                map[string]string `json:"labels" validate:"omitempty"`
	LoadBalancerIP        string            `json:"load_balancer_ip" validate:"omitempty,ip"`
	ExternalIPs           []string          `json:"external_ips" validate:"omitempty,dive,ip"`
	ExternalTrafficPolicy string            `json:"external_traffic_policy" validate:"omitempty,oneof=Cluster Local"`
}

type ServicePortSpec struct {
	Name       string `json:"name" validate:"omitempty,max=15"`
	Port       int32  `json:"port" validate:"required,min=1,max=65535"`
	TargetPort int32  `json:"target_port" validate:"omitempty,min=1,max=65535"`
	NodePort   int32  `json:"node_port" validate:"omitempty,min=30000,max=32767"`
	Protocol   string `json:"protocol" validate:"omitempty,oneof=TCP UDP SCTP"`
}

type CreateIngressRequest struct {
	Namespace     string            `json:"namespace" validate:"required,hostname_rfc1123,max=63"`
	Name          string            `json:"name" validate:"required,hostname_rfc1123,max=63"`
	ClassName     string            `json:"class_name" validate:"omitempty,max=63"`
	Host          string            `json:"host" validate:"required,hostname_rfc1123"`
	Path          string            `json:"path" validate:"omitempty"`
	ServiceName   string            `json:"service_name" validate:"required,hostname_rfc1123,max=63"`
	ServicePort   int32             `json:"service_port" validate:"required,min=1,max=65535"`
	TLSSecretName string            `json:"tls_secret_name" validate:"omitempty,max=253"`
	Annotations   map[string]string `json:"annotations" validate:"omitempty"`
	Labels        map[string]string `json:"labels" validate:"omitempty"`
}

type UpdateIngressRequest struct {
	ClassName     string            `json:"class_name" validate:"omitempty,max=63"`
	Host          string            `json:"host" validate:"required,hostname_rfc1123"`
	Path          string            `json:"path" validate:"omitempty"`
	ServiceName   string            `json:"service_name" validate:"required,hostname_rfc1123,max=63"`
	ServicePort   int32             `json:"service_port" validate:"required,min=1,max=65535"`
	TLSSecretName string            `json:"tls_secret_name" validate:"omitempty,max=253"`
	Annotations   map[string]string `json:"annotations" validate:"omitempty"`
	Labels        map[string]string `json:"labels" validate:"omitempty"`
}

type UpsertNetworkPolicyRequest struct {
	Namespace          string            `json:"namespace" validate:"required,hostname_rfc1123,max=63"`
	Name               string            `json:"name" validate:"required,hostname_rfc1123,max=63"`
	PodSelector        map[string]string `json:"pod_selector" validate:"omitempty"`
	PolicyTypes        []string          `json:"policy_types" validate:"omitempty,dive,oneof=Ingress Egress"`
	AllowSameNamespace bool              `json:"allow_same_namespace"`
	AllowDNS           bool              `json:"allow_dns"`
	Labels             map[string]string `json:"labels" validate:"omitempty"`
}

type UpdateNetworkPolicyRequest struct {
	PodSelector        map[string]string `json:"pod_selector" validate:"omitempty"`
	PolicyTypes        []string          `json:"policy_types" validate:"omitempty,dive,oneof=Ingress Egress"`
	AllowSameNamespace bool              `json:"allow_same_namespace"`
	AllowDNS           bool              `json:"allow_dns"`
	Labels             map[string]string `json:"labels" validate:"omitempty"`
}

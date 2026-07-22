package spec

type ApplicationSpecDocument struct {
	APIVersion string              `yaml:"apiVersion" json:"apiVersion"`
	Kind       string              `yaml:"kind" json:"kind"`
	Metadata   ApplicationMetadata `yaml:"metadata" json:"metadata"`
	Spec       ApplicationSpec     `yaml:"spec" json:"spec"`
}

type ApplicationMetadata struct {
	Name        string            `yaml:"name" json:"name"`
	DisplayName string            `yaml:"displayName,omitempty" json:"displayName,omitempty"`
	Description string            `yaml:"description,omitempty" json:"description,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
}

type ApplicationSpec struct {
	Type         string           `yaml:"type" json:"type"`
	Repo         RepoSpec         `yaml:"repo" json:"repo"`
	Namespace    *NamespaceSpec   `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	AutoDeploy   *AutoDeploySpec  `yaml:"autoDeploy,omitempty" json:"autoDeploy,omitempty"`
	Dependencies []DependencySpec `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	Components   []ComponentSpec  `yaml:"components" json:"components"`
}

type RepoSpec struct {
	Provider string `yaml:"provider,omitempty" json:"provider,omitempty"`
	Name     string `yaml:"name" json:"name"`
	Branch   string `yaml:"branch,omitempty" json:"branch,omitempty"`
}

type NamespaceSpec struct {
	Strategy string `yaml:"strategy,omitempty" json:"strategy,omitempty"`
	Name     string `yaml:"name,omitempty" json:"name,omitempty"`
}

type AutoDeploySpec struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Mode    string `yaml:"mode,omitempty" json:"mode,omitempty"`
}

type ComponentSpec struct {
	Name                string                  `yaml:"name" json:"name"`
	Kind                string                  `yaml:"kind" json:"kind"`
	ProjectName         string                  `yaml:"projectName" json:"projectName"`
	BasaltPass          *BasaltPassSpec         `yaml:"basaltPass,omitempty" json:"basaltPass,omitempty"`
	Build               *BuildSpec              `yaml:"build,omitempty" json:"build,omitempty"`
	Image               *ImageSpec              `yaml:"image,omitempty" json:"image,omitempty"`
	Ports               []PortSpec              `yaml:"ports,omitempty" json:"ports,omitempty"`
	HealthCheck         *HealthCheckSpec        `yaml:"healthCheck,omitempty" json:"healthCheck,omitempty"`
	DependsOn           []string                `yaml:"dependsOn,omitempty" json:"dependsOn,omitempty"`
	EnvFromDependencies []EnvFromDependencySpec `yaml:"envFromDependencies,omitempty" json:"envFromDependencies,omitempty"`
	Env                 map[string]any          `yaml:"env,omitempty" json:"env,omitempty"`
	Secrets             []SecretSpec            `yaml:"secrets,omitempty" json:"secrets,omitempty"`
	Volumes             []VolumeSpec            `yaml:"volumes,omitempty" json:"volumes,omitempty"`
	Replicas            *int                    `yaml:"replicas,omitempty" json:"replicas,omitempty"`
	WatchPaths          []string                `yaml:"watchPaths,omitempty" json:"watchPaths,omitempty"`
}

type BasaltPassSpec struct {
	CallbackPath   string   `yaml:"callbackPath,omitempty" json:"callbackPath,omitempty"`
	RedirectURIs   []string `yaml:"redirectURIs,omitempty" json:"redirectURIs,omitempty"`
	AllowedOrigins []string `yaml:"allowedOrigins,omitempty" json:"allowedOrigins,omitempty"`
	Scopes         []string `yaml:"scopes,omitempty" json:"scopes,omitempty"`
}

type BuildSpec struct {
	Context    string            `yaml:"context" json:"context"`
	Dockerfile string            `yaml:"dockerfile" json:"dockerfile"`
	Args       map[string]string `yaml:"args,omitempty" json:"args,omitempty"`
}

type ImageSpec struct {
	Repository string `yaml:"repository,omitempty" json:"repository,omitempty"`
	TagPolicy  string `yaml:"tagPolicy,omitempty" json:"tagPolicy,omitempty"`
}

type PortSpec struct {
	Name     string `yaml:"name" json:"name"`
	Port     int    `yaml:"port" json:"port"`
	Protocol string `yaml:"protocol,omitempty" json:"protocol,omitempty"`
	Exposure string `yaml:"exposure,omitempty" json:"exposure,omitempty"`
	Domain   string `yaml:"domain,omitempty" json:"domain,omitempty"`
}

type HealthCheckSpec struct {
	Type                string `yaml:"type" json:"type"`
	Path                string `yaml:"path,omitempty" json:"path,omitempty"`
	Port                any    `yaml:"port,omitempty" json:"port,omitempty"`
	InitialDelaySeconds *int   `yaml:"initialDelaySeconds,omitempty" json:"initialDelaySeconds,omitempty"`
	PeriodSeconds       *int   `yaml:"periodSeconds,omitempty" json:"periodSeconds,omitempty"`
	TimeoutSeconds      *int   `yaml:"timeoutSeconds,omitempty" json:"timeoutSeconds,omitempty"`
}

type DependencySpec struct {
	Name                 string                    `yaml:"name" json:"name"`
	Type                 string                    `yaml:"type,omitempty" json:"type,omitempty"`
	DeployMethod         string                    `yaml:"deployMethod,omitempty" json:"deployMethod,omitempty"`
	Version              string                    `yaml:"version,omitempty" json:"version,omitempty"`
	Config               map[string]any            `yaml:"config,omitempty" json:"config,omitempty"`
	Shared               bool                      `yaml:"shared,omitempty" json:"shared,omitempty"`
	External             bool                      `yaml:"external,omitempty" json:"external,omitempty"`
	ExistingDependencyID uint                      `yaml:"existingDependencyID,omitempty" json:"existingDependencyID,omitempty"`
	Credential           *DependencyCredentialSpec `yaml:"credential,omitempty" json:"credential,omitempty"`
	Outputs              *DependencyOutputsSpec    `yaml:"outputs,omitempty" json:"outputs,omitempty"`
}

type DependencyOutputsSpec struct {
	Items map[string]any `yaml:",inline" json:"items,omitempty"`
}

type EnvFromDependencySpec struct {
	Dependency   string                `yaml:"dependency" json:"dependency"`
	DependencyID uint                  `yaml:"dependencyID,omitempty" json:"dependencyID,omitempty"`
	Credential   string                `yaml:"credential,omitempty" json:"credential,omitempty"`
	CredentialID uint                  `yaml:"credentialID,omitempty" json:"credentialID,omitempty"`
	Preset       string                `yaml:"preset,omitempty" json:"preset,omitempty"`
	Mappings     map[string]EnvMapping `yaml:"mappings,omitempty" json:"mappings,omitempty"`
}

type DependencyCredentialSpec struct {
	Name        string         `yaml:"name" json:"name"`
	Description string         `yaml:"description,omitempty" json:"description,omitempty"`
	Config      map[string]any `yaml:"config,omitempty" json:"config,omitempty"`
}

type EnvMapping struct {
	Output string `yaml:"output" json:"output"`
	Secret bool   `yaml:"secret,omitempty" json:"secret,omitempty"`
}

type SecretSpec struct {
	Name          string         `yaml:"name" json:"name"`
	Generate      *GenerateSpec  `yaml:"generate,omitempty" json:"generate,omitempty"`
	ValueFrom     map[string]any `yaml:"valueFrom,omitempty" json:"valueFrom,omitempty"`
	FromComponent string         `yaml:"fromComponent,omitempty" json:"fromComponent,omitempty"`
}

type GenerateSpec struct {
	Length int `yaml:"length" json:"length"`
}

type VolumeSpec struct {
	Name        string   `yaml:"name" json:"name"`
	Type        string   `yaml:"type" json:"type"`
	MountPath   string   `yaml:"mountPath" json:"mountPath"`
	Size        string   `yaml:"size,omitempty" json:"size,omitempty"`
	AccessModes []string `yaml:"accessModes,omitempty" json:"accessModes,omitempty"`
}

type ValidationIssue struct {
	Field    string `json:"field"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

type ValidationResult struct {
	Valid    bool              `json:"valid"`
	Errors   []ValidationIssue `json:"errors"`
	Warnings []ValidationIssue `json:"warnings"`
}

type ApplicationPlan struct {
	Application            PlannedApplication    `json:"application"`
	WillCreateDependencies []PlannedDependency   `json:"willCreateDependencies"`
	WillCreateProjects     []PlannedProject      `json:"willCreateProjects"`
	WillInjectEnv          []PlannedEnvInjection `json:"willInjectEnv"`
	Warnings               []ValidationIssue     `json:"warnings,omitempty"`
}

type PlannedApplication struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type PlannedDependency struct {
	Name         string         `json:"name"`
	Type         string         `json:"type"`
	DeployMethod string         `json:"deployMethod"`
	Version      string         `json:"version,omitempty"`
	Config       map[string]any `json:"config,omitempty"`
}

type PlannedProject struct {
	Name         string            `json:"name"`
	Component    string            `json:"component"`
	Kind         string            `json:"kind"`
	Dockerfile   string            `json:"dockerfile,omitempty"`
	BuildContext string            `json:"buildContext,omitempty"`
	BuildArgs    map[string]string `json:"buildArgs,omitempty"`
	Ports        []PortSpec        `json:"ports,omitempty"`
	HealthCheck  *HealthCheckSpec  `json:"healthCheck,omitempty"`
	Volumes      []VolumeSpec      `json:"volumes,omitempty"`
	WatchPaths   []string          `json:"watchPaths,omitempty"`
}

type PlannedEnvInjection struct {
	Component  string   `json:"component"`
	Dependency string   `json:"dependency"`
	Env        []string `json:"env"`
}

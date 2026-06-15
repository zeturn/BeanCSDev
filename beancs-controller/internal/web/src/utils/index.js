export function filterNavItems(items, query) {
  const needle = String(query || "")
    .trim()
    .toLowerCase();
  if (!needle) return items;
  return items.filter((item) =>
    `${item.label} ${item.id}`.toLowerCase().includes(needle),
  );
}

export function filterNavSections(sections, query) {
  const needle = String(query || "")
    .trim()
    .toLowerCase();
  if (!needle) return sections;
  return sections
    .map((section) => {
      const sectionMatches = `${section.label} ${section.id}`
        .toLowerCase()
        .includes(needle);
      return {
        ...section,
        items: sectionMatches
          ? section.items
          : filterNavItems(section.items, needle),
      };
    })
    .filter((section) => section.items.length > 0);
}

export function shouldShowSkeleton(view, dashboard, network) {
  if (["dashboard", "alerts", "events", "metrics"].includes(view))
    return !dashboard;
  if (view === "networking") return !network;
  return false;
}

export function processJobsFromRecord(process) {
  const jobs = (process?.jobs || [])
    .slice()
    .sort((a, b) => Number(a.step_index || 0) - Number(b.step_index || 0));
  if (!jobs.length) {
    return [
      {
        id: "queued",
        label: "queued",
        status: process?.status || "queued",
        detail: "no jobs yet",
        description:
          "The process has been created and is waiting for the executor.",
        steps: [
          {
            label: "Waiting for executor",
            status: "queued",
            expanded: true,
            log: "No process jobs have been persisted yet.",
          },
        ],
      },
    ];
  }
  return jobs.map((job) => ({
    id: String(job.id),
    label: job.display_name || job.name,
    status: normalizeProcessStatus(job.status),
    detail: job.finished_at
      ? formatTime(job.finished_at)
      : job.started_at
        ? "running"
        : "queued",
    description: `${job.name} · ${job.status}`,
    steps: [
      {
        label: job.display_name || job.name,
        status: normalizeProcessStatus(job.status),
        expanded: true,
        duration: jobDuration(job),
        kind: "process",
        log:
          job.logs ||
          job.failure_reason ||
          "No log output has been written yet.",
      },
    ],
  }));
}

export function progressJobs(
  progress,
  installProgress,
  readyPods,
  pods,
  deployments,
  events,
) {
  if (!progress && !installProgress) {
    return [
      {
        id: "waiting",
        label: "waiting",
        status: "pending",
        detail: "no project",
        description: "Choose a project to inspect its deployment process.",
        steps: [
          {
            label: "Choose project",
            status: "pending",
            expanded: true,
            log: "Select a project from the toolbar to load process details.",
          },
        ],
      },
    ];
  }
  const installSteps = (installProgress?.steps || []).map((step) => ({
    label: step.label,
    status: step.state,
    duration: step.state === "running" ? "now" : "",
    log: step.log || `${step.label}: ${step.state}`,
  }));
  const buildSteps =
    deployments.length > 0
      ? deployments.slice(0, 8).map((deployment, index) => ({
          label:
            deployment.image_ref ||
            deployment.tag ||
            deployment.commit_sha ||
            `Deployment ${deployment.id}`,
          status:
            deployment.status === "failed"
              ? "failed"
              : ["deployed", "running"].includes(deployment.status)
                ? "done"
                : deployment.status === "provisioned"
                  ? "running"
                  : "running",
          duration: index === 0 ? "latest" : "",
          log: [
            `status=${deployment.status || "pending"}`,
            `image=${deployment.image_ref || deployment.tag || "-"}`,
            `commit=${deployment.commit_sha || "-"}`,
            deployment.workflow_url
              ? `workflow=${deployment.workflow_url}`
              : "",
            deployment.failure_reason
              ? `error=${deployment.failure_reason}`
              : "",
            deployment.status === "provisioned"
              ? "note=control-plane resources were created; waiting for workload build/sync/runtime readiness"
              : "",
          ]
            .filter(Boolean)
            .join("\n"),
        }))
      : [
          {
            label: "No build record yet",
            status: "pending",
            log: "BeanCS has not recorded a build/deployment record yet.",
          },
        ];
  const hasWarningEvents = events.some((event) => event.type === "Warning");
  const missingDeployment = progress && !progress.deployment;
  const runtimeStatus =
    progress?.error ||
    hasWarningEvents ||
    pods.some((pod) => pod.status === "Failed")
      ? "failed"
      : readyPods >= pods.length && pods.length > 0
        ? "done"
        : "running";
  const installStatus = installSteps.some((step) => step.status === "failed")
    ? "failed"
    : installSteps.some((step) => step.status === "running")
      ? "running"
      : "done";
  const buildStatus = buildSteps.some((step) => step.status === "failed")
    ? "failed"
    : buildSteps.some((step) => step.status === "running")
      ? "running"
      : buildSteps.some((step) => step.status === "pending")
        ? "pending"
        : "done";
  return [
    {
      id: "install",
      label: "install",
      status: installStatus,
      detail: installSteps.length ? `${installSteps.length} steps` : "created",
      description:
        "Project creation, namespace preparation, and traffic route setup.",
      steps: installSteps.length
        ? installSteps
        : [
            {
              label: "Project already created",
              status: "done",
              log: "No active install step is running.",
            },
          ],
    },
    {
      id: "runtime",
      label: "runtime",
      status: runtimeStatus,
      detail: `${readyPods}/${pods.length} pods`,
      description: "Live Kubernetes workload readiness for this project.",
      steps: [
        {
          label: "Load project status",
          status: progress ? "done" : "pending",
          duration: "0s",
          log: `namespace=${progress?.project?.namespace || "-"}\nproject=${progress?.project?.name || "-"}\nchecked_at=${formatTime(progress?.checked_at)}`,
        },
        ...(missingDeployment
          ? [
              {
                label: "Find Kubernetes Deployment",
                status: "running",
                expanded: true,
                log: `deployment=${progress?.project?.namespace || "-"}/${progress?.project?.name || "-"} not found yet\nservices=${(progress?.services || []).length}\ningresses=${(progress?.ingresses || []).length}\nThis usually means the image build or GitOps/Argo CD sync has not created the workload yet.`,
              },
            ]
          : []),
        {
          label: "Replica readiness",
          status: runtimeStatus,
          expanded: !missingDeployment,
          log: `ready_pods=${readyPods}\ntotal_pods=${pods.length}\nready_replicas=${progress?.deployment?.ready_replicas ?? 0}\ndesired_replicas=${progress?.deployment?.replicas ?? progress?.project?.replicas ?? 0}\nerror=${progress?.error || "-"}`,
        },
        ...(hasWarningEvents
          ? [
              {
                label: "Kubernetes warnings",
                status: "failed",
                log: `${events.filter((event) => event.type === "Warning").length} warning event(s). See Kubernetes events below for full messages.`,
              },
            ]
          : []),
        ...pods.slice(0, 8).map((pod) => ({
          label: pod.name || "pod",
          status:
            pod.status === "Running" &&
            Number(pod.ready_containers) === Number(pod.total_containers)
              ? "done"
              : pod.status === "Failed"
                ? "failed"
                : "running",
          log: `status=${pod.status || "-"}\nreason=${pod.reason || "-"}\nmessage=${pod.message || "-"}\nready=${pod.ready_containers}/${pod.total_containers}\nrestarts=${pod.restarts || 0}\ncontainers=${(pod.containers || []).join(", ") || "-"}`,
        })),
      ],
    },
    {
      id: "build",
      label: "build",
      status: buildStatus,
      detail: `${deployments.length} records`,
      description: "Build, GitOps, and rollout deployment records.",
      steps: buildSteps,
    },
  ];
}

export function filterLogLines(logs, query) {
  const text = String(logs || "");
  const needle = String(query || "")
    .trim()
    .toLowerCase();
  if (!needle) return text;
  return text
    .split("\n")
    .filter((line) => line.toLowerCase().includes(needle))
    .join("\n");
}

export function canContinueDeployStep(
  stepID,
  form,
  selectedCredential,
  analysis,
) {
  if (stepID === "method") return Boolean(form.deploy_target || form.deploy_source);
  if (stepID === "source") {
    if (form.deploy_target === "basaltpass") {
      return Boolean(selectedCredential && form.github_repo);
    }
    if (form.deploy_source === "gitops") {
      if (form.repo_type === "git-url") return false;
      return Boolean(selectedCredential && form.github_repo);
    }
    if (form.image_choice === "new")
      return Boolean(form.selected_image_id && form.image_reference);
    return Boolean(form.image_reference);
  }
  if (stepID === "update")
    return form.deploy_source === "registry" || Boolean(form.update_mode);
  if (stepID === "check")
    if (form.deploy_target === "basaltpass") return true;
    return form.application_type === "monorepo"
      ? Boolean(analysis?.is_monorepo && analysis?.deployable !== false)
      : Boolean(analysis?.deployable);
  if (stepID === "params") {
    if (form.deploy_target === "basaltpass") {
      return Boolean(
        form.name &&
          form.tenant_name &&
          form.tenant_code &&
          ((form.exposure_mode === "private" && form.public_host) ||
            (form.exposure_mode !== "private" &&
              form.cloudflare_credential_id &&
              form.cloudflare_zone_id &&
              form.subdomain)) &&
          form.backend_image &&
          form.frontend_image,
      );
    }
    if (form.application_type === "monorepo") {
      return Boolean(
        form.name &&
        (form.components || []).some(
          (component) =>
            component.enabled !== false &&
            component.project_name &&
            component.dockerfile_path,
        ),
      );
    }
    return Boolean(
      form.name && Number(form.port || 0) > 0 && Number(form.replicas || 0) > 0,
    );
  }
  if (stepID === "dependencies") {
    if (form.deploy_target === "basaltpass") {
      return Boolean(
        form.database_binding &&
          form.owner_email &&
          form.service_token,
      );
    }
    return true;
  }
  if (stepID === "domain") {
    if (form.deploy_target === "basaltpass") {
      return form.exposure_mode === "private" || Boolean(form.public_host);
    }
    if (form.application_type === "monorepo") {
      const publicComponents = (form.components || []).some(
        (component) =>
          component.enabled !== false && component.exposure_mode === "public",
      );
      return publicComponents
        ? Boolean(form.cloudflare_credential_id && form.cloudflare_zone_id)
        : true;
    }
    if (form.exposure_mode === "public")
      return Boolean(
        form.cloudflare_credential_id &&
        form.cloudflare_zone_id &&
        form.subdomain,
      );
    if (form.exposure_mode === "private") return Boolean(form.private_host);
  }
  return true;
}

export function sourceLabel(source) {
  return (
    {
      github: "GitHub",
      dockerhub: "Docker Hub",
      ghcr: "Container registry",
      registry: "Container registry",
      "source-upload": "Source upload",
    }[source || "github"] || source
  );
}

export function sourceSummary(form) {
  if (form.deploy_source === "gitops")
    return form.repo_type === "git-url"
      ? form.git_url || "-"
      : `${form.github_repo || "-"} @ ${form.github_branch || "main"}`;
  return form.image_reference || "-";
}

export function detailTitle(kind) {
  return (
    {
      pod: "Pod",
      node: "Node",
      services: "Service",
      ingresses: "Ingress",
      endpoints: "Endpoints",
      namespaces: "Namespace",
      "namespace-detail": "Namespace",
      "network-policy": "NetworkPolicy",
      "service-access": "Service access",
      "service-edit": "Service",
    }[kind] || kind
  );
}

export function podContainers(pod) {
  return (pod.containers || [])
    .map((value) => {
      const text = String(value || "");
      const [name, ...rest] = text.split(":");
      return { name: name || text, image: rest.join(":") };
    })
    .filter((container) => container.name);
}

export function defaultDeployForm() {
  return {
    deploy_target: "project",
    deploy_source: "gitops",
    build_source: "github",
    application_type: "single",
    repo_type: "github",
    git_url: "",
    update_mode: "argocd",
    image_choice: "",
    selected_image_id: "",
    new_image_registry_id: "",
    new_image_repository: "",
    name: "",
    namespace: "",
    github_repo: "",
    github_branch: "main",
    dockerfile_path: "Dockerfile",
    build_context: ".",
    auto_deploy: true,
    image_reference: "",
    source_archive_name: "",
    basaltpass_instance_id: "",
    cloudflare_credential_id: "",
    cloudflare_zone_id: "",
    exposure_mode: "private",
    subdomain: "",
    private_host: "",
    port: 8080,
    replicas: 1,
    resource_preset: "small",
    env_entries: [],
    components: [],
    dependencies: [],
    base_url: "",
    public_host: "",
    backend_image: "",
    frontend_image: "",
    database_binding: "",
    tenant_name: "",
    owner_email: "",
    tenant_code: "",
    description: "",
    max_apps: "",
    max_users: "",
    max_tokens_per_hour: "",
    service_token: "",
    automation_token: "",
    jwt_secret: "",
    cors_allow_origins: "",
  };
}

export function buildProjectPayload(form, githubCredentialID, credentials) {
  const exposure = form.exposure_mode;
  const selectedCF =
    (credentials.domains || []).find(
      (domain) =>
        String(domain.credential_id) ===
          String(form.cloudflare_credential_id) &&
        String(domain.zone_id) === String(form.cloudflare_zone_id),
    ) ||
    credentials.cloudflare.find(
      (cred) => String(cred.id) === String(form.cloudflare_credential_id),
    );
  const domain =
    exposure === "public" && selectedCF
      ? `${form.subdomain}.${selectedCF.domain}`
      : exposure === "private"
        ? form.private_host
        : "";
  const source = form.deploy_source === "registry" ? "registry" : "github";
  return {
    build_source: source,
    name: form.name,
    namespace: form.namespace || undefined,
    image_reference: form.image_reference || undefined,
    source_archive_name: form.source_archive_name || undefined,
    github_credential_id:
      source === "github" ? Number(githubCredentialID) : undefined,
    github_repo: source === "github" ? form.github_repo : undefined,
    github_branch: form.github_branch || "main",
    dockerfile_path: form.dockerfile_path || undefined,
    build_context: form.build_context || ".",
    auto_deploy: source === "github" ? form.update_mode === "argocd" : false,
    basaltpass_instance_id: form.basaltpass_instance_id
      ? Number(form.basaltpass_instance_id)
      : undefined,
    cloudflare_credential_id:
      exposure === "public" ? Number(form.cloudflare_credential_id) : undefined,
    cloudflare_zone_id:
      exposure === "public" ? form.cloudflare_zone_id : undefined,
    exposure_mode: exposure,
    subdomain: form.subdomain || undefined,
    resource_preset: form.resource_preset || "small",
    port: Number(form.port || 8080),
    replicas: Number(form.replicas || 1),
    ports: [
      {
        name: "http",
        port: Number(form.port || 8080),
        protocol: "http",
        exposure,
        domain,
      },
    ],
    env: envObjectFromEntries(form.env_entries || []),
  };
}

export function monorepoComponentsFromAnalysis(
  applicationName,
  components = [],
) {
  return components.map((component) => {
    const name = slugify(component.name || component.path || "component");
    const suggestedPort = Number(component.suggested_port || 0);
    return {
      enabled: true,
      name,
      kind: component.kind || "service",
      path: component.path || "",
      component_path: component.path || "",
      project_name: slugify(`${applicationName}-${name}`),
      dockerfile_path: component.dockerfile_path || "",
      build_context: component.build_context || ".",
      port: suggestedPort || "",
      exposure_mode: suggestedPort
        ? component.kind === "frontend"
          ? "public"
          : "private"
        : "internal-only",
      replicas: 1,
      env_entries: [],
      dependency_links: [],
    };
  });
}

export function applicationSpecAnalysis(data) {
  const doc = data.document || {};
  const plan = data.plan || {};
  const projects = plan.willCreateProjects || [];
  return {
    source: "beancs_spec",
    config_path: data.config_path || ".beancs/app.yaml",
    is_monorepo: doc.spec?.type === "monorepo" || projects.length > 1,
    deployable: Boolean(data.validation?.valid),
    document: doc,
    plan,
    components: projects.map((project) => ({
      name: project.component,
      path: project.buildContext,
      dockerfile_path: project.dockerfile,
      build_context: project.buildContext,
      suggested_port: project.ports?.[0]?.port || 0,
      kind: project.kind,
    })),
    signals: [
      `Application: ${plan.application?.name || doc.metadata?.name || "-"}`,
      `Projects: ${projects.map((project) => project.name).join(", ") || "none"}`,
      `Dependencies: ${(plan.willCreateDependencies || []).map((dep) => dep.name).join(", ") || "none"}`,
    ],
    warnings: [
      ...(data.validation?.warnings || []).map(
        (warning) => `${warning.field}: ${warning.message}`,
      ),
      ...(plan.warnings || []).map(
        (warning) => `${warning.field}: ${warning.message}`,
      ),
    ],
  };
}

export function deployFormFromApplicationSpec(
  current,
  repoFullName,
  branch,
  data,
) {
  const doc = data.document || {};
  const spec = doc.spec || {};
  const metadata = doc.metadata || {};
  const plan = data.plan || {};
  const namespace =
    spec.namespace?.strategy === "shared" ? spec.namespace?.name || "" : "";
  return {
    ...current,
    application_type: "monorepo",
    name: slugify(
      metadata.name ||
        plan.application?.name ||
        repoFullName.split("/")[1] ||
        current.name,
    ),
    github_repo: repoFullName,
    github_branch: spec.repo?.branch || branch || "main",
    namespace,
    auto_deploy: Boolean(spec.autoDeploy?.enabled),
    update_mode: spec.autoDeploy?.enabled ? "argocd" : "passive",
    components: (spec.components || []).map((component) =>
      componentFromApplicationSpec(
        metadata.name || plan.application?.name || repoFullName,
        component,
      ),
    ),
    dependencies: (spec.dependencies || []).map((dependency) => ({
      name: dependency.name,
      type: dependency.type,
      deploy_method: dependency.deployMethod || "helm",
      version: dependency.version || "",
      config: dependency.config || {},
    })),
  };
}

export function componentFromApplicationSpec(applicationName, component) {
  const firstPort = (component.ports || [])[0];
  const exposure =
    firstPort?.exposure === "internal"
      ? "internal-only"
      : firstPort?.exposure || (firstPort ? "private" : "internal-only");
  return {
    enabled: true,
    name: slugify(component.name),
    kind: component.kind || "service",
    path: component.build?.context || "",
    component_path: component.build?.context || "",
    project_name: slugify(
      component.projectName || `${applicationName}-${component.name}`,
    ),
    dockerfile_path: component.build?.dockerfile || "",
    build_context: component.build?.context || ".",
    build_args: component.build?.args || {},
    port: firstPort?.port || "",
    exposure_mode: exposure,
    replicas: component.replicas || 1,
    health_check: component.healthCheck || null,
    volumes: component.volumes || [],
    watch_paths: component.watchPaths || [],
    env_entries: envEntriesFromObject(component.env || {}),
    dependency_links: (component.envFromDependencies || []).map((ref) => ({
      dependency: ref.dependency,
      preset: ref.preset || "",
    })),
  };
}

export function indexForComponent(components, target) {
  return (components || []).findIndex(
    (component) =>
      component === target || component.project_name === target.project_name,
  );
}

export function monorepoComponentHost(
  component,
  form,
  selectedCloudflareDomain,
) {
  const exposure =
    component.exposure_mode || (component.port ? "private" : "internal-only");
  if (exposure === "public") {
    const subdomain = slugify(component.subdomain ?? component.project_name);
    return selectedCloudflareDomain?.domain
      ? `${subdomain}.${selectedCloudflareDomain.domain}`
      : "Choose a Cloudflare zone";
  }
  if (exposure === "private") {
    return (
      component.private_host || monorepoDefaultPrivateHost(component, form)
    );
  }
  return "internal only";
}

export function monorepoDefaultPrivateHost(component, form) {
  return `${component.project_name}.${form.namespace || `proj-${component.project_name}`}.ts.net`;
}

export function monorepoComponentDomainOverrides(form, credentials) {
  const selectedCF =
    (credentials.domains || []).find(
      (domain) =>
        String(domain.credential_id) ===
          String(form.cloudflare_credential_id) &&
        String(domain.zone_id) === String(form.cloudflare_zone_id),
    ) ||
    credentials.cloudflare.find(
      (cred) => String(cred.id) === String(form.cloudflare_credential_id),
    );
  const out = {};
  for (const component of form.components || []) {
    if (component.enabled === false || Number(component.port || 0) <= 0)
      continue;
    const exposure = component.exposure_mode || "internal-only";
    let host = "";
    if (exposure === "public" && selectedCF?.domain) {
      host = `${slugify(component.subdomain ?? component.project_name)}.${selectedCF.domain}`;
    } else if (exposure === "private") {
      host =
        component.private_host || monorepoDefaultPrivateHost(component, form);
    }
    if (host) out[component.project_name] = host;
  }
  return out;
}

export function buildMonorepoApplicationPayload(
  form,
  githubCredentialID,
  credentials,
) {
  const selectedCF =
    (credentials.domains || []).find(
      (domain) =>
        String(domain.credential_id) ===
          String(form.cloudflare_credential_id) &&
        String(domain.zone_id) === String(form.cloudflare_zone_id),
    ) ||
    credentials.cloudflare.find(
      (cred) => String(cred.id) === String(form.cloudflare_credential_id),
    );
  return {
    name: form.name,
    display_name: form.name,
    github_credential_id: Number(githubCredentialID),
    github_repo: form.github_repo,
    github_branch: form.github_branch || "main",
    auto_deploy: form.update_mode === "argocd",
    namespace: form.namespace || undefined,
    basaltpass_instance_id: form.basaltpass_instance_id
      ? Number(form.basaltpass_instance_id)
      : undefined,
    cloudflare_credential_id: form.cloudflare_credential_id
      ? Number(form.cloudflare_credential_id)
      : undefined,
    cloudflare_zone_id: form.cloudflare_zone_id || undefined,
    resource_preset: form.resource_preset || "small",
    dependencies: (form.dependencies || []).map((dependency) => ({
      name: dependency.name,
      type: dependency.type,
      version: dependency.version || undefined,
      deploy_method:
        dependency.source === "existing"
          ? undefined
          : dependency.deploy_method || "helm",
      existing_dependency_id:
        dependency.source === "existing" && dependency.existing_dependency_id
          ? Number(dependency.existing_dependency_id)
          : undefined,
      config: dependency.source === "existing" ? undefined : dependency.config || {},
      external: dependency.source === "existing" ? undefined : dependency.external,
      controlled:
        dependency.source === "existing" ? undefined : dependency.controlled,
    })),
    components: (form.components || [])
      .filter((component) => component.enabled !== false)
      .map((component) => {
        const exposure =
          component.exposure_mode ||
          (component.port ? "private" : "internal-only");
        const port = Number(component.port || 0);
        const domain =
          exposure === "public" && selectedCF
            ? `${slugify(component.subdomain ?? component.project_name)}.${selectedCF.domain}`
            : exposure === "private"
              ? component.private_host ||
                monorepoDefaultPrivateHost(component, form)
              : "";
        return {
          name: component.name,
          kind: component.kind || "service",
          project_name: component.project_name,
          dockerfile_path: component.dockerfile_path,
          build_context: component.build_context || ".",
          component_path: component.component_path || component.path || "",
          namespace: form.namespace || undefined,
          exposure_mode: exposure,
          resource_preset: form.resource_preset || "small",
          replicas: Number(component.replicas || 1),
          ports:
            port > 0
              ? [{ name: "http", port, protocol: "http", exposure, domain }]
              : [],
          env: envObjectFromEntries(
            component.env_entries || form.env_entries || [],
          ),
          depends_on: (component.dependency_links || []).map(
            (link) => link.dependency,
          ),
          env_from_dependencies: (component.dependency_links || []).map(
            (link) => ({
              dependency: link.dependency,
              credential: link.credential || undefined,
              credential_id: link.credential_id
                ? Number(link.credential_id)
                : undefined,
              preset: link.preset,
            }),
          ),
        };
      }),
  };
}

export function definitionForDependency(definitions, type) {
  return (definitions || []).find(
    (definition) => definition.name === type || definition.type === type,
  );
}

export function normalizeDependencyDefinition(definition) {
  if (!definition?.metadata && definition?.name) return definition;
  return {
    raw: definition,
    name: definition?.metadata?.name || "",
    display_name:
      definition?.metadata?.display_name ||
      definition?.metadata?.displayName ||
      definition?.metadata?.name ||
      "",
    category: definition?.metadata?.category || "",
    type: definition?.spec?.type || definition?.metadata?.name || "",
    supported_deploy_methods:
      definition?.spec?.supported_deploy_methods ||
      definition?.spec?.supportedDeployMethods ||
      [],
    default_deploy_method:
      definition?.spec?.default_deploy_method ||
      definition?.spec?.defaultDeployMethod ||
      "helm",
    config_schema:
      definition?.spec?.config_schema || definition?.spec?.configSchema || {},
    env_presets:
      definition?.spec?.env_presets || definition?.spec?.envPresets || {},
  };
}

export function dependencyDefaultConfig(definition) {
  const out = {};
  const properties = definition?.config_schema?.properties || {};
  for (const [key, schema] of Object.entries(properties)) {
    if (schema.type === "object") {
      out[key] = dependencyDefaultConfig({
        config_schema: { properties: schema.properties || {} },
      });
      continue;
    }
    if (schema.generate) {
      continue;
    }
    if (schema.default !== undefined) out[key] = schema.default;
  }
  return out;
}

export function uniqueDependencyName(dependencies, base) {
  const root = slugify(base || "dependency") || "dependency";
  const existing = new Set(
    (dependencies || []).map((dependency) => dependency.name),
  );
  if (!existing.has(root)) return root;
  let index = 2;
  while (existing.has(`${root}-${index}`)) index += 1;
  return `${root}-${index}`;
}

export function firstEnvPreset(definition) {
  return Object.keys(definition?.env_presets || {})[0] || "";
}

export function labelize(value) {
  return String(value || "")
    .replace(/([a-z])([A-Z])/g, "$1 $2")
    .replace(/[_-]+/g, " ")
    .replace(/\b\w/g, (char) => char.toUpperCase());
}

export function envObjectFromEntries(entries) {
  const out = {};
  for (const entry of entries || []) {
    const key = String(entry.key || "").trim();
    if (!key) continue;
    out[key] = String(entry.value ?? "");
  }
  return out;
}

export function envEntriesFromObject(obj = {}) {
  return Object.entries(obj)
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([key, value]) => ({ key, value: String(value ?? "") }));
}

export function parseDotEnv(text) {
  const entries = [];
  String(text || "")
    .split(/\r?\n/)
    .forEach((line) => {
      const trimmed = line.trim();
      if (!trimmed || trimmed.startsWith("#")) return;
      const normalized = trimmed.startsWith("export ")
        ? trimmed.slice(7).trim()
        : trimmed;
      const index = normalized.indexOf("=");
      if (index <= 0) return;
      const key = normalized.slice(0, index).trim();
      let value = normalized.slice(index + 1).trim();
      if (
        (value.startsWith('"') && value.endsWith('"')) ||
        (value.startsWith("'") && value.endsWith("'"))
      ) {
        value = value.slice(1, -1);
      }
      entries.push({ key, value });
    });
  return entries;
}

export function imageName(image) {
  const withoutDigest = String(image || "").split("@")[0];
  const slash = withoutDigest.lastIndexOf("/");
  const colon = withoutDigest.lastIndexOf(":");
  const value = colon > slash ? withoutDigest.slice(0, colon) : withoutDigest;
  return value.split("/").filter(Boolean).pop() || "app";
}

export function imageReferenceFromTrackedImage(image, tag = "") {
  if (!image) return "";
  const registry = registryHostFromAPIBase(image.registry?.api_base || "");
  const repository = String(image.repository || "").replace(/^\/+/, "");
  const normalizedTag = tag || (image.tags || [])[0] || "latest";
  return `${registry ? `${registry}/` : ""}${repository}:${normalizedTag}`;
}

export function registryHostFromAPIBase(apiBase) {
  const value = String(apiBase || "").trim();
  if (!value) return "";
  try {
    const url = new URL(value);
    return url.host;
  } catch {
    return value
      .replace(/^https?:\/\//, "")
      .replace(/\/v2\/?$/, "")
      .replace(/\/+$/, "");
  }
}

export function imageTagFromReference(image) {
  const value = String(image || "");
  const withoutDigest = value.split("@")[0];
  const slash = withoutDigest.lastIndexOf("/");
  const colon = withoutDigest.lastIndexOf(":");
  return colon > slash ? withoutDigest.slice(colon + 1) : "latest";
}

export function formatRepoDate(repo) {
  const value = repo.pushed_at || repo.updated_at || repo.created_at;
  if (!value) return repo.default_branch || "main";
  const diff = Date.now() - new Date(value).getTime();
  const day = 24 * 60 * 60 * 1000;
  if (Number.isFinite(diff) && diff >= 0 && diff < 7 * day) {
    const days = Math.max(1, Math.round(diff / day));
    return `${days}d ago`;
  }
  return new Date(value).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
  });
}

export function profileFromToken(token) {
  const fallback = {
    name: "Signed in user",
    detail: "BeanCS session",
    initial: "U",
    avatar: "",
    scopes: [],
  };
  if (!token || !token.includes(".")) return fallback;
  try {
    const payload = JSON.parse(base64URLDecode(token.split(".")[1]));
    const pick = (values) =>
      values.map((value) => String(value || "").trim()).find(Boolean) || "";
    const name =
      pick([
        payload.name,
        payload.preferred_username,
        payload.username,
        payload.user,
        payload.login,
        payload.email,
        payload.sub,
      ]) || fallback.name;
    const detail =
      pick(
        [
          payload.email,
          payload.preferred_username,
          payload.username,
          payload.login,
          payload.sub,
        ].filter(
          (value) =>
            String(value || "").trim() && String(value || "").trim() !== name,
        ),
      ) || "BeanCS session";
    const avatar = pick([
      payload.picture,
      payload.avatar,
      payload.avatar_url,
      payload.image,
      payload.image_url,
      payload.profile_picture,
    ]);
    return {
      name,
      detail,
      avatar,
      initial: String(name).trim().slice(0, 1).toUpperCase() || "U",
      scopes: String(payload.scope || "")
        .split(/\s+/)
        .filter(Boolean),
    };
  } catch {
    return fallback;
  }
}

export function profileFromBasalt(profile, token) {
  const fallback = profileFromToken(token);
  if (!profile) return fallback;
  const pick = (values) =>
    values.map((value) => String(value || "").trim()).find(Boolean) || "";
  const name =
    pick([
      profile.name,
      profile.nickname,
      profile.preferred_username,
      profile.username,
      profile.email,
      profile.sub,
    ]) || fallback.name;
  const detail =
    pick(
      [
        profile.email,
        profile.phone_number,
        profile.tenant_code,
        profile.tenant_id,
      ].filter(
        (value) =>
          String(value || "").trim() && String(value || "").trim() !== name,
      ),
    ) || fallback.detail;
  const avatar =
    pick([
      profile.picture,
      profile.avatar_url,
      profile.avatar,
      profile.image,
      profile.image_url,
    ]) || fallback.avatar;
  return {
    ...fallback,
    name,
    detail,
    avatar,
    initial: String(name).trim().slice(0, 1).toUpperCase() || fallback.initial,
  };
}

export function base64URLDecode(value) {
  const normalized = String(value || "")
    .replace(/-/g, "+")
    .replace(/_/g, "/");
  const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, "=");
  return decodeURIComponent(
    Array.from(
      atob(padded),
      (char) => `%${char.charCodeAt(0).toString(16).padStart(2, "0")}`,
    ).join(""),
  );
}

export function trimLiveLog(value) {
  const maxLength = 200000;
  if (value.length <= maxLength) return value;
  return value.slice(value.length - maxLength);
}

export function claimFromJwt(token, key) {
  try {
    const payload = token.split(".")[1] || "";
    const normalized = payload.replace(/-/g, "+").replace(/_/g, "/");
    const padded = normalized + "=".repeat((4 - (normalized.length % 4)) % 4);
    return JSON.parse(atob(padded))[key];
  } catch {
    return undefined;
  }
}

export function titleFor(view) {
  const map = {
    dashboard: "Overview",
    deploy: "Deploy",
    applications: "Applications",
    dependencies: "Dependencies",
    progress: "Progress",
    projects: "Projects",
    deployments: "Deployments",
    pods: "Pods",
    services: "Services",
    ingresses: "Ingresses",
    workloadImage: "Image",
    nodes: "Nodes",
    namespaces: "Namespaces",
    networking: "Networking",
    storage: "Storage",
    github: "GitHub",
    cloudflare: "Cloudflare",
    domains: "Domains",
    registries: "Image Registry",
    apiKeys: "API Keys",
    secrets: "Secrets",
    accessControl: "Access Control",
    alerts: "Alerts",
    events: "Events",
    logs: "Logs",
    metrics: "Metrics",
    settings: "Settings",
  };
  return map[view] || "BeanCS";
}

export function subtitleFor(view, runtime, projects) {
  if (view === "dashboard")
    return "Real-time cluster health and operating signals";
  if (view === "networking")
    return "Service, Ingress, Endpoint, NetworkPolicy, Traefik and Tailscale operations";
  if (view === "projects") return `${projects.length} managed projects`;
  if (view === "applications") return "Monorepo and multi-project application records";
  if (view === "dependencies")
    return "Reusable managed and external service dependencies";
  if (view === "progress") return "Watch installs and runtime readiness";
  if (view === "registries")
    return "Register OCI mirrors and sync image tags for this account";
  if (view === "workloadImage")
    return "Tracked registry tags; use Image Registry to add mirrors";
  if (view === "apiKeys") return "Issue and revoke API keys for automation";
  if (view === "accessControl") return "BasaltPass and access integrations";
  if (view === "settings") return "Workspace and version information";
  if (view === "storage" || view === "secrets")
    return "Planned console capabilities";
  if (view === "alerts")
    return "Active warning signals and degraded runtime objects";
  if (view === "events")
    return "Recent Kubernetes warning events and reason groups";
  if (view === "metrics")
    return "Cluster resource utilization and node readings";
  if (view === "logs") return "Project log snapshots and live streaming";
  if (runtime[view]) return `${(runtime[view] || []).length} cluster resources`;
  return "Operate k3s, GitHub, DNS, and traffic from one console.";
}

export function formatTime(value) {
  if (!value) return "-";
  return new Date(value).toLocaleString();
}

export function formatDeploymentDate(value) {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "-";
  return date.toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

export function shortRelativeDuration(value) {
  if (!value) return "-";
  const elapsed = Math.max(
    1,
    Math.floor((Date.now() - new Date(value).getTime()) / 1000),
  );
  if (elapsed < 60) return `${elapsed}s`;
  if (elapsed < 3600) return `${Math.floor(elapsed / 60)}m ${elapsed % 60}s`;
  if (elapsed < 86400)
    return `${Math.floor(elapsed / 3600)}h ${Math.floor((elapsed % 3600) / 60)}m`;
  return `${Math.floor(elapsed / 86400)}d`;
}

export function normalizeDeploymentStatus(status) {
  const value = String(status || "").toLowerCase();
  if (["failed", "error", "degraded"].some((item) => value.includes(item)))
    return "error";
  if (
    ["building", "deploying", "pending", "progress"].some((item) =>
      value.includes(item),
    )
  )
    return "building";
  return "ready";
}

export function normalizeProcessStatus(status) {
  const value = String(status || "").toLowerCase();
  if (["failed", "error"].includes(value)) return "failed";
  if (["succeeded", "success", "done", "completed", "running"].includes(value))
    return value === "running" ? "running" : "done";
  return value || "pending";
}

export function jobDuration(job) {
  if (!job?.started_at) return "";
  const start = new Date(job.started_at);
  const end = job.finished_at ? new Date(job.finished_at) : new Date();
  if (Number.isNaN(start.getTime()) || Number.isNaN(end.getTime())) return "";
  const seconds = Math.max(
    0,
    Math.round((end.getTime() - start.getTime()) / 1000),
  );
  return `${seconds}s`;
}

export function imageRepoName(value) {
  if (!value) return "";
  const withoutTag = String(value).split("@")[0].split(":")[0];
  const parts = withoutTag.split("/").filter(Boolean);
  return parts.slice(-2).join("/") || withoutTag;
}

export function deploymentShortID(name, fallback) {
  const base = String(name || fallback || "deployment").replace(
    /[^a-zA-Z0-9]/g,
    "",
  );
  return (base.slice(0, 9) || String(fallback || "deploy")).padEnd(7, "x");
}

export function truncateMiddle(value, max = 28) {
  const text = String(value || "-");
  if (text.length <= max) return text;
  const head = Math.max(8, Math.ceil((max - 3) * 0.58));
  const tail = Math.max(6, max - 3 - head);
  return `${text.slice(0, head)}...${text.slice(-tail)}`;
}

export function formatBytes(value) {
  const bytes = Number(value || 0);
  if (!bytes) return "-";
  const units = ["B", "KiB", "MiB", "GiB", "TiB"];
  let size = bytes;
  let index = 0;
  while (size >= 1024 && index < units.length - 1) {
    size /= 1024;
    index += 1;
  }
  return `${size.toFixed(index === 0 ? 0 : 1)} ${units[index]}`;
}

export function formatPercent(value) {
  return Number(value || 0).toFixed(0);
}

export function formatDuration(seconds) {
  const value = Number(seconds || 0);
  if (!value) return "-";
  const days = Math.floor(value / 86400);
  const hours = Math.floor((value % 86400) / 3600);
  const minutes = Math.floor((value % 3600) / 60);
  if (days > 0) return `${days}d ${hours}h`;
  if (hours > 0) return `${hours}h ${minutes}m`;
  return `${minutes}m`;
}

export function formatCell(value) {
  if (Array.isArray(value)) return value.join(", ") || "-";
  if (typeof value === "object" && value !== null)
    return formatKeyValues(value);
  if (typeof value === "boolean") return value ? "Yes" : "No";
  if (value === null || value === undefined || value === "") return "-";
  return String(value);
}

export function parseKeyValues(value) {
  if (!value) return {};
  return String(value)
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean)
    .reduce((out, item) => {
      const [key, ...rest] = item.split("=");
      if (key?.trim()) out[key.trim()] = rest.join("=").trim();
      return out;
    }, {});
}

export function formatKeyValues(value) {
  if (!value || typeof value !== "object") return "";
  return Object.entries(value)
    .map(([key, val]) => `${key}=${val}`)
    .join(",");
}

export function parseTaints(value) {
  return String(value || "")
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean)
    .map((item) => {
      const [left, effect = "NoSchedule"] = item.split(":");
      const [key, ...valueParts] = left.split("=");
      return {
        key: key.trim(),
        value: valueParts.join("=").trim(),
        effect: effect.trim() || "NoSchedule",
      };
    })
    .filter((taint) => taint.key);
}

export function parseCSV(value) {
  return String(value || "")
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

export function parsePermissionSubjects(value, namespace) {
  return parseCSV(value)
    .map((item) => {
      const [kind = "User", name = item, subjectNamespace = ""] =
        item.split(":");
      return {
        kind: kind.trim(),
        name: name.trim(),
        namespace:
          subjectNamespace.trim() ||
          (kind.trim() === "ServiceAccount" ? namespace : ""),
      };
    })
    .filter((subject) => subject.name);
}

export function taintsToForm(taints) {
  return (taints || []).join(",");
}

export function parseServicePorts(value) {
  return String(value || "")
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean)
    .map((item) => {
      const [left, protocol = "TCP"] = item.split("/");
      const parts = left.split(":");
      const hasName = parts.length > 1 && Number.isNaN(Number(parts[0]));
      const port = hasName ? Number(parts[1]) : Number(parts[0]);
      return {
        name: hasName ? parts[0] : "",
        port,
        target_port: Number(
          hasName ? parts[2] || parts[1] : parts[1] || parts[0],
        ),
        node_port: Number(hasName ? parts[3] || 0 : parts[2] || 0),
        protocol: protocol || "TCP",
      };
    });
}

export function portsToForm(ports) {
  if (!Array.isArray(ports)) return "";
  return ports.map((port) => String(port)).join(",");
}

export function localDateTimeToRFC3339(value) {
  if (!value) return "";
  return new Date(value).toISOString();
}

export function slugify(value) {
  return String(value || "")
    .toLowerCase()
    .replace(/[^a-z0-9-]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 63);
}

export function trimSlash(value) {
  return String(value || "").replace(/\/+$/, "");
}

export function browserRedirectURI() {
  return `${location.origin}/api/v1/ui/oauth/callback`;
}

export function randomString(length) {
  const bytes = new Uint8Array(length);
  crypto.getRandomValues(bytes);
  return Array.from(
    bytes,
    (byte) =>
      "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~"[
        byte % 66
      ],
  ).join("");
}

export async function codeChallenge(verifier) {
  const encoded = new TextEncoder().encode(verifier);
  const digest = await crypto.subtle.digest("SHA-256", encoded);
  return btoa(String.fromCharCode(...new Uint8Array(digest)))
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "");
}

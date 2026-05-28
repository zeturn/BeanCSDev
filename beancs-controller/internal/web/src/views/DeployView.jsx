import React, { useEffect, useMemo, useRef, useState } from "react";
import * as LucideIcons from "lucide-react";
import {
  canContinueDeployStep,
  sourceLabel,
  sourceSummary,
  defaultDeployForm,
  indexForComponent,
  monorepoComponentHost,
  monorepoDefaultPrivateHost,
  definitionForDependency,
  dependencyDefaultConfig,
  uniqueDependencyName,
  imageName,
  imageReferenceFromTrackedImage,
  imageTagFromReference,
  slugify,
} from "../utils/index";
import {
  RepoListSkeleton,
  ApplicationSpecPlanSummary,
  DependencyConfigEditor,
  DependencyLinksEditor,
  EnvEditor,
  Field,
  ChevronIcon,
  Button,
  Input,
  Select,
  Checkbox,
} from "../components/index";
import {
  Activity,
  AlertTriangle,
  Bell,
  Boxes,
  Box,
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  Cloud,
  Coffee,
  Code2,
  Container,
  Cpu,
  Database,
  Edit3,
  FileText,
  GitBranch,
  Github,
  Globe2,
  HardDrive,
  Image as ImageIcon,
  KeyRound,
  Layers3,
  LayoutDashboard,
  LineChart,
  ListRestart,
  LoaderCircle,
  Lock,
  Menu,
  MemoryStick,
  MoreHorizontal,
  Network,
  Package,
  Play,
  Plus,
  RefreshCw,
  RotateCcw,
  Rocket,
  ScrollText,
  Search,
  Server,
  Settings,
  Shield,
  ShieldCheck,
  Trash2,
  Upload,
  X,
} from "lucide-react";
const deploySteps = [
  {
    id: "method",
    label: "Source type",
    title: "Choose deployment source",
  },
  {
    id: "source",
    label: "Source",
    title: "Choose deployment source details",
  },
  {
    id: "update",
    label: "Update",
    title: "Choose update mode",
  },
  {
    id: "check",
    label: "Check",
    title: "Check installability",
  },
  {
    id: "params",
    label: "Params",
    title: "Configure parameters",
  },
  {
    id: "dependencies",
    label: "Dependencies",
    title: "Configure dependencies",
  },
  {
    id: "namespace",
    label: "Namespace",
    title: "Choose namespace",
  },
  {
    id: "ingress",
    label: "Ingress",
    title: "Choose ingress mode",
  },
  {
    id: "domain",
    label: "Domain",
    title: "Choose domain",
  },
  {
    id: "env",
    label: "Env",
    title: "Add runtime variables",
  },
  {
    id: "confirm",
    label: "Confirm",
    title: "Confirm and build",
  },
];
const deploySourceOptions = [
  {
    id: "gitops",
    label: "GitOps repository",
    icon: GitBranch,
    description:
      "Use a GitHub repository as source and publish runtime images to GHCR.",
  },
  {
    id: "registry",
    label: "Container registry",
    icon: Package,
    description: "Deploy an existing or newly tracked container image object.",
  },
];
const updateModeOptions = [
  {
    id: "argocd",
    label: "Argo CD",
    icon: GitBranch,
    description:
      "Create GitOps manifests, register an Argo CD app, and let GitHub Actions build the first GHCR image.",
  },
  {
    id: "passive",
    label: "Passive update",
    icon: RefreshCw,
    description: "Create the project without automatic GitHub push deployment.",
  },
];
export default function DeployView({
  credentials,
  domains,
  namespaces,
  selectedCredential,
  setSelectedCredential,
  repos,
  selectedRepo,
  analysis,
  setAnalysis,
  form,
  setForm,
  loadRepos,
  analyzeRepo,
  checkInstallSource,
  deployProject,
  containerRegistries,
  containerImages,
  dependencyDefinitions,
  createTrackedImageFromDeploy,
  onConnectGitHub,
  reposLoading,
}) {
  const [stepIndex, setStepIndex] = useState(0);
  const [creatingImage, setCreatingImage] = useState(false);
  const [checkingInstall, setCheckingInstall] = useState(false);
  const [repoSearch, setRepoSearch] = useState("");
  const [accountMenuOpen, setAccountMenuOpen] = useState(false);
  const selectedCloudflareDomain = (domains || []).find(
    (domain) =>
      String(domain.credential_id) === String(form.cloudflare_credential_id) &&
      String(domain.zone_id) === String(form.cloudflare_zone_id),
  );
  const selectedGitHubCredential = credentials.github.find(
    (cred) => String(cred.id) === String(selectedCredential),
  );
  const visibleRepos = repos.filter((repo) =>
    `${repo.full_name || ""} ${repo.name || ""}`
      .toLowerCase()
      .includes(repoSearch.toLowerCase()),
  );
  const publicHost =
    form.subdomain && selectedCloudflareDomain
      ? `${form.subdomain}.${selectedCloudflareDomain.domain}`
      : "";
  const step = deploySteps[stepIndex];
  const canContinue = canContinueDeployStep(
    step.id,
    form,
    selectedCredential,
    analysis,
  );
  const ghcrPreview = form.github_repo
    ? `ghcr.io/${form.github_repo.toLowerCase()}:beancs-<build>`
    : "ghcr.io/<owner>/<repo>:beancs-<build>";
  const setDeploySource = (deploySource) => {
    setAnalysis(null);
    setForm({
      ...defaultDeployForm(),
      deploy_source: deploySource,
      build_source: deploySource === "gitops" ? "github" : "ghcr",
      repo_type: deploySource === "gitops" ? "github" : "",
      update_mode: deploySource === "gitops" ? "argocd" : "passive",
      image_choice: deploySource === "registry" ? "existing" : "",
      github_branch: form.github_branch || "main",
      port: form.port || 8080,
    });
    setStepIndex(1);
  };
  const updateSourceForm = (nextForm) => {
    setAnalysis(null);
    setForm(nextForm);
  };
  const setRepoType = (repoType) => {
    setAnalysis(null);
    setForm({
      ...form,
      repo_type: repoType,
      github_repo: "",
      git_url: "",
      update_mode:
        repoType === "github" ? form.update_mode || "argocd" : "passive",
    });
  };
  const setApplicationType = (applicationType) => {
    setAnalysis(null);
    setForm({
      ...form,
      application_type: applicationType,
      components: [],
      name: applicationType === "monorepo" ? form.name : form.name,
    });
  };
  const setUpdateMode = (updateMode) => {
    setForm({
      ...form,
      update_mode: form.deploy_source === "registry" ? "passive" : updateMode,
      auto_deploy: updateMode === "argocd",
    });
    setStepIndex((current) =>
      deploySteps[current]?.id === "update"
        ? Math.min(current + 1, deploySteps.length - 1)
        : current,
    );
  };
  const selectTrackedImage = (image, tag = "") => {
    const ref = imageReferenceFromTrackedImage(image, tag);
    updateSourceForm({
      ...form,
      selected_image_id: String(image.id),
      image_reference: ref,
      name: form.name || slugify(imageName(ref)),
    });
  };
  const createImage = async () => {
    if (!form.new_image_registry_id || !form.new_image_repository) return;
    setCreatingImage(true);
    try {
      const created = await createTrackedImageFromDeploy({
        registry_id: Number(form.new_image_registry_id),
        repository: form.new_image_repository.trim(),
      });
      const ref = imageReferenceFromTrackedImage(created, "");
      updateSourceForm({
        ...form,
        image_choice: "existing",
        selected_image_id: String(created.id),
        image_reference: ref,
        name: form.name || slugify(imageName(ref)),
      });
    } finally {
      setCreatingImage(false);
    }
  };
  const runInstallCheck = async () => {
    setCheckingInstall(true);
    try {
      return await checkInstallSource(form);
    } finally {
      setCheckingInstall(false);
    }
  };
  useEffect(() => {
    if (step.id !== "check") return;
    if (checkingInstall || analysis) return;
    runInstallCheck();
  }, [step.id]);
  const next = async () => {
    if (step.id === "check") {
      const result = await runInstallCheck();
      if (result && stepIndex < deploySteps.length - 1)
        setStepIndex(stepIndex + 1);
      return;
    }
    if (stepIndex < deploySteps.length - 1) setStepIndex(stepIndex + 1);
  };
  const back = () => setStepIndex(Math.max(0, stepIndex - 1));
  const updateComponent = (index, patch) => {
    setForm({
      ...form,
      components: (form.components || []).map((component, i) =>
        i === index
          ? {
              ...component,
              ...patch,
            }
          : component,
      ),
    });
  };
  const addDependency = () => {
    const definition = dependencyDefinitions[0];
    if (!definition) return;
    const name = uniqueDependencyName(form.dependencies || [], definition.name);
    setForm({
      ...form,
      dependencies: [
        ...(form.dependencies || []),
        {
          name,
          type: definition.name,
          deploy_method: definition.default_deploy_method || "helm",
          version: "",
          config: dependencyDefaultConfig(definition),
        },
      ],
    });
  };
  const updateDependency = (index, patch) => {
    setForm({
      ...form,
      dependencies: (form.dependencies || []).map((dependency, i) =>
        i === index
          ? {
              ...dependency,
              ...patch,
            }
          : dependency,
      ),
    });
  };
  const deleteDependency = (index) => {
    const removed = (form.dependencies || [])[index]?.name;
    setForm({
      ...form,
      dependencies: (form.dependencies || []).filter((_, i) => i !== index),
      components: (form.components || []).map((component) => ({
        ...component,
        dependency_links: (component.dependency_links || []).filter(
          (link) => link.dependency !== removed,
        ),
      })),
    });
  };
  return (
    <div className="deploy-wizard">
      <section className="panel wizard-progress-panel">
        <div className="wizard-progress-head">
          <span>{step.label}</span>
          <b>
            {stepIndex + 1} / {deploySteps.length}
          </b>
        </div>
        <div className="wizard-progress-track">
          <span
            style={{
              width: `${((stepIndex + 1) / deploySteps.length) * 100}%`,
            }}
          />
        </div>
        <div className="wizard-step-labels">
          {deploySteps.map((item, index) => (
            <span
              key={item.id}
              className={
                index === stepIndex ? "active" : index < stepIndex ? "done" : ""
              }
            >
              {item.label}
            </span>
          ))}
        </div>
      </section>
      <form className="panel deploy-form wizard-panel" onSubmit={deployProject}>
        <h2>
          <Rocket size={18} /> {step.title}
        </h2>
        {step.id === "method" && (
          <div className="method-grid">
            {deploySourceOptions.map((method) => {
              const Icon = method.icon;
              return (
                <Button
                  key={method.id}
                  type="button"
                  className={
                    form.deploy_source === method.id
                      ? "method-card active"
                      : "method-card"
                  }
                  onClick={() => setDeploySource(method.id)}
                >
                  <Icon size={22} />
                  <b>{method.label}</b>
                  <span>{method.description}</span>
                </Button>
              );
            })}
          </div>
        )}
        {step.id === "source" && (
          <div className="form-grid">
            {form.deploy_source === "gitops" && (
              <>
                <label>Repository type</label>
                <div className="segmented-control">
                  <Button
                    type="button"
                    className={form.repo_type === "github" ? "active" : ""}
                    onClick={() => setRepoType("github")}
                  >
                    <Github size={15} /> GitHub
                  </Button>
                  <Button
                    type="button"
                    className={form.repo_type === "git-url" ? "active" : ""}
                    onClick={() => setRepoType("git-url")}
                  >
                    <GitBranch size={15} /> Git link
                  </Button>
                </div>
                <label>Repository layout</label>
                <div className="segmented-control">
                  <Button
                    type="button"
                    className={
                      form.application_type !== "monorepo" ? "active" : ""
                    }
                    onClick={() => setApplicationType("single")}
                  >
                    <Box size={15} /> Single service
                  </Button>
                  <Button
                    type="button"
                    className={
                      form.application_type === "monorepo" ? "active" : ""
                    }
                    onClick={() => setApplicationType("monorepo")}
                  >
                    <Layers3 size={15} /> Monorepo
                  </Button>
                </div>
                {form.repo_type === "github" && (
                  <>
                    <div className="import-repo-panel">
                      <h3>Import Git Repository</h3>
                      <div className="import-repo-toolbar">
                        <div className="account-picker">
                          <Button
                            type="button"
                            className="account-picker-button"
                            onClick={() => setAccountMenuOpen(!accountMenuOpen)}
                          >
                            <Github size={18} />
                            <span>
                              {selectedGitHubCredential?.account_login ||
                                selectedGitHubCredential?.name ||
                                "Choose account"}
                            </span>
                            <ChevronIcon open={accountMenuOpen} />
                          </Button>
                          {accountMenuOpen && (
                            <div className="account-menu">
                              {credentials.github.map((cred) => (
                                <Button
                                  key={cred.id}
                                  type="button"
                                  className={
                                    String(cred.id) ===
                                    String(selectedCredential)
                                      ? "active"
                                      : ""
                                  }
                                  onClick={() => {
                                    updateSourceForm({
                                      ...form,
                                      github_repo: "",
                                      github_branch: "main",
                                    });
                                    setSelectedCredential(String(cred.id));
                                    setAccountMenuOpen(false);
                                    loadRepos(cred.id);
                                  }}
                                >
                                  <Github size={16} />
                                  <span>{cred.account_login || cred.name}</span>
                                  {String(cred.id) ===
                                    String(selectedCredential) && (
                                    <CheckCircle2 size={16} />
                                  )}
                                </Button>
                              ))}
                              <Button
                                type="button"
                                onClick={() => {
                                  setAccountMenuOpen(false);
                                  onConnectGitHub?.();
                                }}
                              >
                                <Plus size={16} />
                                <span>Add GitHub Account</span>
                              </Button>
                              <Button
                                type="button"
                                onClick={() => {
                                  setAccountMenuOpen(false);
                                  setRepoType("git-url");
                                }}
                              >
                                <ListRestart size={16} />
                                <span>Switch Git Provider</span>
                              </Button>
                            </div>
                          )}
                        </div>
                        <div className="repo-search-box">
                          <Search size={18} />
                          <Input
                            value={repoSearch}
                            onChange={(event) =>
                              setRepoSearch(event.target.value)
                            }
                            placeholder="Search..."
                          />
                        </div>
                      </div>
                      <div className="import-repo-list">
                        {reposLoading && <RepoListSkeleton />}
                        {!reposLoading &&
                          visibleRepos.map((repo) => {
                            const isSelected =
                              form.github_repo === repo.full_name ||
                              selectedRepo === repo.full_name;
                            const repoName =
                              repo.name || repo.full_name.split("/")[1];
                            const branch = repo.default_branch || "main";
                            return (
                              <div
                                key={repo.full_name}
                                className={
                                  isSelected
                                    ? "import-repo-row active"
                                    : "import-repo-row"
                                }
                              >
                                <div>
                                  <Github size={17} />
                                  <span>{repoName}</span>
                                  <small>· {branch}</small>
                                  {isSelected && (
                                    <b className="selected-repo-pill">
                                      <CheckCircle2 size={14} /> Selected
                                    </b>
                                  )}
                                </div>
                                <Button
                                  type="button"
                                  onClick={() => {
                                    setForm({
                                      ...form,
                                      github_repo: repo.full_name,
                                      github_branch: branch,
                                      name: form.name || slugify(repoName),
                                    });
                                    analyzeRepo(repo.full_name, branch);
                                  }}
                                >
                                  {isSelected ? "Selected" : "Import"}
                                </Button>
                              </div>
                            );
                          })}
                        {!reposLoading && visibleRepos.length === 0 && (
                          <div className="empty">
                            {selectedCredential
                              ? "No repositories match this search."
                              : "Choose a GitHub account to load repositories."}
                          </div>
                        )}
                      </div>
                      {form.github_repo && (
                        <div className="selected-repo-summary">
                          <CheckCircle2 size={16} />
                          <span>Selected repository</span>
                          <b>
                            {form.github_repo} @ {form.github_branch || "main"}
                          </b>
                        </div>
                      )}
                    </div>
                  </>
                )}
                {form.repo_type === "git-url" && (
                  <>
                    <Field
                      label="Git URL"
                      value={form.git_url}
                      onChange={(v) =>
                        updateSourceForm({
                          ...form,
                          git_url: v.trim(),
                        })
                      }
                      required
                    />
                    <p className="warning-note">
                      当前部署模式展示不支持直接的 git 链接。请改用已连接的
                      GitHub 仓库继续部署。
                    </p>
                  </>
                )}
              </>
            )}
            {form.deploy_source === "registry" && (
              <>
                <label>Image object</label>
                <div className="segmented-control">
                  <Button
                    type="button"
                    className={form.image_choice === "existing" ? "active" : ""}
                    onClick={() =>
                      updateSourceForm({
                        ...form,
                        image_choice: "existing",
                        image_reference: "",
                      })
                    }
                  >
                    <Package size={15} /> Existing
                  </Button>
                  <Button
                    type="button"
                    className={form.image_choice === "new" ? "active" : ""}
                    onClick={() =>
                      updateSourceForm({
                        ...form,
                        image_choice: "new",
                        selected_image_id: "",
                        image_reference: "",
                      })
                    }
                  >
                    <Plus size={15} /> New object
                  </Button>
                </div>
                {form.image_choice === "existing" && (
                  <>
                    <div className="image-picker">
                      {containerImages.map((image) => (
                        <Button
                          key={image.id}
                          type="button"
                          className={
                            String(form.selected_image_id) === String(image.id)
                              ? "image-option active"
                              : "image-option"
                          }
                          onClick={() =>
                            selectTrackedImage(
                              image,
                              (image.tags || [])[0] || "",
                            )
                          }
                        >
                          <b>{image.repository}</b>
                          <span>
                            {image.registry?.name ||
                              `registry #${image.registry_id}`}
                          </span>
                          <small>
                            {(image.tags || []).length
                              ? `${(image.tags || []).length} tags cached`
                              : "No cached tags"}
                          </small>
                        </Button>
                      ))}
                      {containerImages.length === 0 && (
                        <div className="empty">
                          No image objects yet. Create one below or open Image
                          Registry.
                        </div>
                      )}
                    </div>
                    {form.selected_image_id && (
                      <>
                        <label>Tag</label>
                        <Select
                          value={imageTagFromReference(form.image_reference)}
                          onChange={(event) => {
                            const image = containerImages.find(
                              (item) =>
                                String(item.id) ===
                                String(form.selected_image_id),
                            );
                            if (image)
                              selectTrackedImage(image, event.target.value);
                          }}
                        >
                          {(
                            containerImages.find(
                              (item) =>
                                String(item.id) ===
                                String(form.selected_image_id),
                            )?.tags || ["latest"]
                          ).map((tag) => (
                            <option key={tag} value={tag}>
                              {tag}
                            </option>
                          ))}
                        </Select>
                      </>
                    )}
                    <Field
                      label="Image reference"
                      value={form.image_reference}
                      onChange={(v) =>
                        updateSourceForm({
                          ...form,
                          image_reference: v.trim(),
                          name: form.name || slugify(imageName(v)),
                        })
                      }
                      required
                    />
                  </>
                )}
                {form.image_choice === "new" && (
                  <>
                    <label>Registry</label>
                    <Select
                      value={form.new_image_registry_id}
                      onChange={(event) =>
                        updateSourceForm({
                          ...form,
                          new_image_registry_id: event.target.value,
                        })
                      }
                      required
                    >
                      <option value="">Choose registry</option>
                      {containerRegistries.map((registry) => (
                        <option key={registry.id} value={registry.id}>
                          {registry.name} ({registry.kind})
                        </option>
                      ))}
                    </Select>
                    <Field
                      label="Repository path"
                      value={form.new_image_repository}
                      onChange={(v) =>
                        updateSourceForm({
                          ...form,
                          new_image_repository: v.trim(),
                        })
                      }
                      required
                    />
                    <Button
                      type="button"
                      className="inline-primary"
                      disabled={
                        creatingImage ||
                        !form.new_image_registry_id ||
                        !form.new_image_repository
                      }
                      onClick={createImage}
                      variant="primary"
                    >
                      <Plus size={15} /> Create image object
                    </Button>
                    <p className="muted">
                      保存对象后会回到对象选择，并使用该镜像进行被动更新部署。
                    </p>
                  </>
                )}
              </>
            )}
          </div>
        )}
        {step.id === "update" && (
          <div className="method-grid two-up">
            {form.deploy_source === "gitops" &&
              updateModeOptions.map((mode) => {
                const Icon = mode.icon;
                return (
                  <Button
                    key={mode.id}
                    type="button"
                    className={
                      form.update_mode === mode.id
                        ? "method-card active"
                        : "method-card"
                    }
                    onClick={() => setUpdateMode(mode.id)}
                  >
                    <Icon size={22} />
                    <b>{mode.label}</b>
                    <span>{mode.description}</span>
                  </Button>
                );
              })}
            {form.deploy_source === "registry" && (
              <Button
                type="button"
                className="method-card active"
                onClick={() => setUpdateMode("passive")}
              >
                <RefreshCw size={22} />
                <b>Passive update</b>
                <span>
                  Registry deployments only support passive updates in the
                  current flow.
                </span>
              </Button>
            )}
          </div>
        )}
        {step.id === "check" && (
          <div className="readiness-card">
            {!analysis && (
              <p className="muted">
                {checkingInstall
                  ? "Checking installability..."
                  : "BeanCS will verify repository signals or image/source inputs before continuing."}
              </p>
            )}
            {analysis && form.application_type === "monorepo" && (
              <>
                <div
                  className={
                    analysis.is_monorepo ? "status good" : "status bad"
                  }
                >
                  {analysis.source === "beancs_spec"
                    ? `.beancs spec found: ${analysis.config_path}`
                    : analysis.is_monorepo
                      ? `${analysis.components?.length || 0} components detected`
                      : "No components detected"}
                </div>
                {analysis.source === "beancs_spec" && (
                  <ApplicationSpecPlanSummary analysis={analysis} />
                )}
                <div className="signal-list">
                  {analysis.package_manager && (
                    <span>Package manager: {analysis.package_manager}</span>
                  )}
                  {(analysis.signals || []).map((signal) => (
                    <span key={signal}>{signal}</span>
                  ))}
                  {(analysis.warnings || []).map((warning) => (
                    <span className="warning" key={warning}>
                      {warning}
                    </span>
                  ))}
                </div>
              </>
            )}
            {analysis && form.application_type !== "monorepo" && (
              <>
                <div
                  className={analysis.deployable ? "status good" : "status bad"}
                >
                  {analysis.containerized
                    ? "Deployable"
                    : analysis.scaffoldable
                      ? "Source detected"
                      : "Needs containerization"}
                </div>
                <div className="signal-list">
                  {(analysis.signals || []).map((signal) => (
                    <span key={signal}>{signal}</span>
                  ))}
                  {analysis.compose_path && (
                    <span>Compose: {analysis.compose_path}</span>
                  )}
                  {analysis.ports?.length > 0 && (
                    <span>Ports: {analysis.ports.join(", ")}</span>
                  )}
                  {(analysis.warnings || []).map((warning) => (
                    <span className="warning" key={warning}>
                      {warning}
                    </span>
                  ))}
                </div>
              </>
            )}
          </div>
        )}
        {step.id === "params" && (
          <div className="form-grid">
            <Field
              label={
                form.application_type === "monorepo"
                  ? "Application name"
                  : "Project name"
              }
              value={form.name}
              onChange={(v) =>
                setForm({
                  ...form,
                  name: slugify(v),
                })
              }
              required
            />
            {form.application_type !== "monorepo" && (
              <>
                <Field
                  label="Port"
                  type="number"
                  value={form.port}
                  onChange={(v) =>
                    setForm({
                      ...form,
                      port: Number(v),
                    })
                  }
                />
                <Field
                  label="Replicas"
                  type="number"
                  value={form.replicas}
                  onChange={(v) =>
                    setForm({
                      ...form,
                      replicas: Number(v),
                    })
                  }
                />
              </>
            )}
            <label>Resource preset</label>
            <Select
              value={form.resource_preset}
              onChange={(event) =>
                setForm({
                  ...form,
                  resource_preset: event.target.value,
                })
              }
            >
              <option value="nano">Nano</option>
              <option value="small">Small</option>
              <option value="medium">Medium</option>
              <option value="large">Large</option>
            </Select>
            <label>BasaltPass tenant</label>
            <Select
              value={form.basaltpass_instance_id}
              onChange={(event) =>
                setForm({
                  ...form,
                  basaltpass_instance_id: event.target.value,
                })
              }
            >
              <option value="">Do not register OAuth app</option>
              {credentials.basaltpass.map((cred) => (
                <option key={cred.id} value={cred.id}>
                  {[cred.name, cred.tenant_code || cred.tenant_id]
                    .filter(Boolean)
                    .join(" / ")}
                </option>
              ))}
            </Select>
            {form.application_type === "monorepo" && (
              <div className="component-list">
                {analysis?.source === "beancs_spec" && (
                  <p className="muted">
                    These components are declared by repo config. Edit{" "}
                    <span className="mono">{analysis.config_path}</span> in the
                    repository to change build args, health checks, volumes, or
                    dependency bindings.
                  </p>
                )}
                {(form.components || []).map((component, index) => (
                  <div
                    key={`${component.path}-${index}`}
                    className={
                      component.enabled
                        ? "component-card active"
                        : "component-card"
                    }
                  >
                    <div className="component-card-head">
                      <label className="checkbox-label">
                        <Checkbox
                          type="checkbox"
                          checked={component.enabled !== false}
                          onChange={(event) =>
                            updateComponent(index, {
                              enabled: event.target.checked,
                            })
                          }
                        />
                        <b>{component.name}</b>
                      </label>
                      <span>{component.kind || "service"}</span>
                    </div>
                    <div className="component-grid">
                      <Field
                        label="Project name"
                        value={component.project_name}
                        onChange={(v) =>
                          updateComponent(index, {
                            project_name: slugify(v),
                          })
                        }
                        required
                      />
                      <Field
                        label="Component path"
                        value={component.component_path || component.path}
                        onChange={(v) =>
                          updateComponent(index, {
                            component_path: v.trim(),
                          })
                        }
                      />
                      <Field
                        label="Dockerfile"
                        value={component.dockerfile_path}
                        onChange={(v) =>
                          updateComponent(index, {
                            dockerfile_path: v.trim(),
                          })
                        }
                        required
                      />
                      <Field
                        label="Build context"
                        value={component.build_context || "."}
                        onChange={(v) =>
                          updateComponent(index, {
                            build_context: v.trim() || ".",
                          })
                        }
                      />
                      <Field
                        label="Port"
                        type="number"
                        value={component.port || ""}
                        onChange={(v) =>
                          updateComponent(index, {
                            port: Number(v || 0),
                            exposure_mode:
                              Number(v || 0) > 0
                                ? component.exposure_mode || "private"
                                : "internal-only",
                          })
                        }
                      />
                      <Field
                        label="Replicas"
                        type="number"
                        value={component.replicas || 1}
                        onChange={(v) =>
                          updateComponent(index, {
                            replicas: Number(v || 1),
                          })
                        }
                      />
                      <label>Exposure</label>
                      <Select
                        value={
                          component.exposure_mode ||
                          (component.port ? "private" : "internal-only")
                        }
                        onChange={(event) =>
                          updateComponent(index, {
                            exposure_mode: event.target.value,
                          })
                        }
                      >
                        <option value="public">Public</option>
                        <option value="private">Private</option>
                        <option value="internal-only">Internal only</option>
                      </Select>
                    </div>
                    {analysis?.source === "beancs_spec" && (
                      <div className="spec-component-meta">
                        {component.build_args &&
                          Object.keys(component.build_args).length > 0 && (
                            <span>
                              Build args:{" "}
                              {Object.entries(component.build_args)
                                .map(([key, value]) => `${key}=${value}`)
                                .join(", ")}
                            </span>
                          )}
                        {component.health_check?.type && (
                          <span>
                            Health: {component.health_check.type}
                            {component.health_check.path
                              ? ` ${component.health_check.path}`
                              : ""}
                          </span>
                        )}
                        {(component.volumes || []).length > 0 && (
                          <span>
                            Volumes:{" "}
                            {(component.volumes || [])
                              .map(
                                (volume) =>
                                  `${volume.name}:${volume.mountPath || volume.mount_path}`,
                              )
                              .join(", ")}
                          </span>
                        )}
                        {(component.watch_paths || []).length > 0 && (
                          <span>Watch: {component.watch_paths.join(", ")}</span>
                        )}
                      </div>
                    )}
                  </div>
                ))}
                {(form.components || []).length === 0 && (
                  <div className="empty">
                    Run repository analysis to detect deployable components.
                  </div>
                )}
              </div>
            )}
          </div>
        )}
        {step.id === "dependencies" && (
          <div className="form-grid">
            {form.application_type !== "monorepo" && (
              <p className="muted">
                Managed dependency components are currently available for
                monorepo applications.
              </p>
            )}
            {form.application_type === "monorepo" && (
              <>
                <div className="section-head">
                  <div>
                    <h3>Dependency components</h3>
                    <p className="muted">
                      Definitions drive config, outputs, and env presets for
                      application components.
                    </p>
                  </div>
                  <Button
                    type="button"
                    onClick={addDependency}
                    disabled={!dependencyDefinitions.length}
                  >
                    <Plus size={15} /> Add dependency
                  </Button>
                </div>
                <div className="dependency-list">
                  {analysis?.source === "beancs_spec" && (
                    <p className="muted">
                      Dependencies are declared by repo config and will be
                      created from the spec during deploy.
                    </p>
                  )}
                  {(form.dependencies || []).map((dependency, index) => {
                    const definition = definitionForDependency(
                      dependencyDefinitions,
                      dependency.type,
                    );
                    return (
                      <div
                        className="dependency-card"
                        key={`${dependency.name}-${index}`}
                      >
                        <div className="component-card-head">
                          <b>{dependency.name || "dependency"}</b>
                          <Button
                            type="button"
                            onClick={() => deleteDependency(index)}
                            title="Remove dependency"
                            variant="danger"
                          >
                            <Trash2 size={15} />
                          </Button>
                        </div>
                        <div className="component-grid">
                          <Field
                            label="Name"
                            value={dependency.name}
                            onChange={(v) =>
                              updateDependency(index, {
                                ...dependency,
                                name: slugify(v),
                              })
                            }
                            required
                          />
                          <label>Type</label>
                          <Select
                            value={dependency.type}
                            onChange={(event) => {
                              const nextDefinition = definitionForDependency(
                                dependencyDefinitions,
                                event.target.value,
                              );
                              updateDependency(index, {
                                type: event.target.value,
                                name: uniqueDependencyName(
                                  (form.dependencies || []).filter(
                                    (_, i) => i !== index,
                                  ),
                                  event.target.value,
                                ),
                                deploy_method:
                                  nextDefinition?.default_deploy_method ||
                                  "helm",
                                config: dependencyDefaultConfig(nextDefinition),
                              });
                            }}
                          >
                            {dependencyDefinitions.map((definition) => (
                              <option
                                key={definition.name}
                                value={definition.name}
                              >
                                {definition.display_name || definition.name}
                              </option>
                            ))}
                          </Select>
                          <label>Deploy method</label>
                          <Select
                            value={
                              dependency.deploy_method ||
                              definition?.default_deploy_method ||
                              "helm"
                            }
                            onChange={(event) =>
                              updateDependency(index, {
                                deploy_method: event.target.value,
                              })
                            }
                          >
                            {(
                              definition?.supported_deploy_methods || ["helm"]
                            ).map((method) => (
                              <option key={method} value={method}>
                                {method}
                              </option>
                            ))}
                          </Select>
                          <Field
                            label="Version"
                            value={dependency.version || ""}
                            onChange={(v) =>
                              updateDependency(index, {
                                version: v.trim(),
                              })
                            }
                          />
                        </div>
                        <DependencyConfigEditor
                          definition={definition}
                          value={dependency.config || {}}
                          onChange={(config) =>
                            updateDependency(index, {
                              config,
                            })
                          }
                        />
                      </div>
                    );
                  })}
                  {(form.dependencies || []).length === 0 && (
                    <div className="empty">
                      No managed dependencies selected.
                    </div>
                  )}
                </div>
                {(form.dependencies || []).length > 0 && (
                  <div className="component-list">
                    {(form.components || []).map((component, index) => (
                      <DependencyLinksEditor
                        key={`${component.project_name}-${index}`}
                        component={component}
                        dependencies={form.dependencies || []}
                        definitions={dependencyDefinitions}
                        onChange={(dependency_links) =>
                          updateComponent(index, {
                            dependency_links,
                          })
                        }
                      />
                    ))}
                  </div>
                )}
              </>
            )}
          </div>
        )}
        {step.id === "namespace" && (
          <div className="form-grid">
            <label>Namespace</label>
            <Input
              list="namespace-options"
              value={form.namespace}
              placeholder={form.name ? `proj-${form.name}` : "proj-my-app"}
              onChange={(event) =>
                setForm({
                  ...form,
                  namespace: slugify(event.target.value),
                })
              }
            />
            <datalist id="namespace-options">
              {namespaces.map((ns) => (
                <option key={ns.name} value={ns.name} />
              ))}
            </datalist>
            <p className="muted">
              Leave empty to create{" "}
              {form.name ? <b>proj-{form.name}</b> : "a project namespace"}{" "}
              automatically.
            </p>
          </div>
        )}
        {step.id === "ingress" && (
          <div className="form-grid">
            {form.application_type === "monorepo" ? (
              <>
                <p className="muted">
                  Traffic mode is configured per component in the parameters
                  step.
                </p>
                <div className="signal-list">
                  {(form.components || [])
                    .filter((component) => component.enabled !== false)
                    .map((component) => (
                      <span key={component.project_name}>
                        {component.project_name}:{" "}
                        {component.port
                          ? component.exposure_mode
                          : "internal-only"}
                      </span>
                    ))}
                </div>
              </>
            ) : (
              <>
                <label>Traffic</label>
                <Select
                  value={form.exposure_mode}
                  onChange={(event) =>
                    setForm({
                      ...form,
                      exposure_mode: event.target.value,
                    })
                  }
                >
                  <option value="public">Traefik public ingress</option>
                  <option value="private">Tailscale private ingress</option>
                  <option value="internal-only">Cluster internal only</option>
                </Select>
              </>
            )}
          </div>
        )}
        {step.id === "domain" && (
          <div className="form-grid">
            {form.application_type === "monorepo" &&
              (form.components || []).some(
                (component) =>
                  component.enabled !== false &&
                  component.exposure_mode === "public",
              ) && (
                <>
                  <label>Cloudflare credential</label>
                  <Select
                    value={
                      form.cloudflare_zone_id
                        ? `${form.cloudflare_credential_id}:${form.cloudflare_zone_id}`
                        : ""
                    }
                    onChange={(event) => {
                      const [credentialID, zoneID] =
                        event.target.value.split(":");
                      setForm({
                        ...form,
                        cloudflare_credential_id: credentialID || "",
                        cloudflare_zone_id: zoneID || "",
                      });
                    }}
                    required
                  >
                    <option value="">Choose Cloudflare zone</option>
                    {(domains || []).map((domain) => (
                      <option
                        key={`${domain.credential_id}:${domain.zone_id}`}
                        value={`${domain.credential_id}:${domain.zone_id}`}
                      >
                        {domain.credential} · {domain.domain}
                      </option>
                    ))}
                  </Select>
                  <p className="muted">
                    Public component hostnames use the component project name
                    under the selected zone.
                  </p>
                </>
              )}
            {form.application_type === "monorepo" && (
              <>
                {!(form.components || []).some(
                  (component) =>
                    component.enabled !== false &&
                    component.exposure_mode === "public",
                ) && (
                  <p className="muted">
                    No public DNS zone is required for the selected components.
                  </p>
                )}
                <div className="component-list">
                  {(form.components || [])
                    .filter(
                      (component) =>
                        component.enabled !== false &&
                        Number(component.port || 0) > 0,
                    )
                    .map((component) => (
                      <div
                        className="component-card active"
                        key={component.project_name}
                      >
                        <div className="component-card-head">
                          <b>{component.project_name}</b>
                          <span>
                            {component.exposure_mode || "internal-only"}
                          </span>
                        </div>
                        {component.exposure_mode === "public" ? (
                          <div className="component-grid">
                            <Field
                              label="Subdomain"
                              value={
                                component.subdomain ?? component.project_name
                              }
                              onChange={(v) =>
                                updateComponent(
                                  indexForComponent(form.components, component),
                                  {
                                    subdomain: slugify(v),
                                  },
                                )
                              }
                            />
                            <div className="computed-host">
                              {monorepoComponentHost(
                                component,
                                form,
                                selectedCloudflareDomain,
                              )}
                            </div>
                          </div>
                        ) : component.exposure_mode === "private" ? (
                          <div className="component-grid">
                            <Field
                              label="Tailscale host"
                              value={
                                component.private_host ||
                                monorepoDefaultPrivateHost(component, form)
                              }
                              onChange={(v) =>
                                updateComponent(
                                  indexForComponent(form.components, component),
                                  {
                                    private_host: v.trim().toLowerCase(),
                                  },
                                )
                              }
                            />
                            <div className="computed-host">
                              {monorepoComponentHost(
                                component,
                                form,
                                selectedCloudflareDomain,
                              )}
                            </div>
                          </div>
                        ) : (
                          <p className="muted">Internal-only component.</p>
                        )}
                      </div>
                    ))}
                </div>
              </>
            )}
            {form.application_type !== "monorepo" &&
              form.exposure_mode === "public" && (
                <>
                  <label>Cloudflare credential</label>
                  <Select
                    value={
                      form.cloudflare_zone_id
                        ? `${form.cloudflare_credential_id}:${form.cloudflare_zone_id}`
                        : ""
                    }
                    onChange={(event) => {
                      const [credentialID, zoneID] =
                        event.target.value.split(":");
                      setForm({
                        ...form,
                        cloudflare_credential_id: credentialID || "",
                        cloudflare_zone_id: zoneID || "",
                      });
                    }}
                    required
                  >
                    <option value="">Choose Cloudflare zone</option>
                    {(domains || []).map((domain) => (
                      <option
                        key={`${domain.credential_id}:${domain.zone_id}`}
                        value={`${domain.credential_id}:${domain.zone_id}`}
                      >
                        {domain.credential} · {domain.domain}
                      </option>
                    ))}
                  </Select>
                  <Field
                    label="Subdomain"
                    value={form.subdomain}
                    onChange={(v) =>
                      setForm({
                        ...form,
                        subdomain: slugify(v),
                      })
                    }
                    required
                  />
                  <div className="computed-host">
                    {publicHost || "Subdomain preview"}
                  </div>
                </>
              )}
            {form.application_type !== "monorepo" &&
              form.exposure_mode === "private" && (
                <Field
                  label="Tailscale host"
                  value={form.private_host}
                  onChange={(v) =>
                    setForm({
                      ...form,
                      private_host: v.trim().toLowerCase(),
                    })
                  }
                  required
                />
              )}
            {form.application_type !== "monorepo" &&
              form.exposure_mode === "internal-only" && (
                <p className="muted">
                  No domain is required for internal-only projects.
                </p>
              )}
          </div>
        )}
        {step.id === "env" && (
          <EnvEditor
            entries={form.env_entries || []}
            onChange={(entries) =>
              setForm({
                ...form,
                env_entries: entries,
              })
            }
            masked={false}
            title="Runtime environment"
          />
        )}
        {step.id === "confirm" && (
          <div className="detail-list">
            <span>
              Install method <b>{sourceLabel(form.build_source)}</b>
            </span>
            <span>
              Source <b>{sourceSummary(form)}</b>
            </span>
            <span>
              {form.application_type === "monorepo" ? "Application" : "Project"}{" "}
              <b>{form.name || "-"}</b>
            </span>
            <span>
              Namespace{" "}
              <b>{form.namespace || (form.name ? `proj-${form.name}` : "-")}</b>
            </span>
            <span>
              Ingress <b>{form.exposure_mode}</b>
            </span>
            <span>
              Domain <b>{publicHost || form.private_host || "internal only"}</b>
            </span>
            {form.application_type === "monorepo" ? (
              <span>
                Components{" "}
                <b>
                  {
                    (form.components || []).filter(
                      (component) => component.enabled !== false,
                    ).length
                  }
                </b>
              </span>
            ) : (
              <span>
                Port <b>{form.port}</b>
              </span>
            )}
            {form.application_type === "monorepo" && (
              <span>
                Dependencies <b>{(form.dependencies || []).length}</b>
              </span>
            )}
            <span>
              Runtime variables{" "}
              <b>
                {
                  (form.env_entries || []).filter((entry) => entry.key.trim())
                    .length
                }
              </b>
            </span>
            <span>
              Update mode{" "}
              <b>
                {form.deploy_source === "registry"
                  ? "Passive"
                  : form.update_mode === "argocd"
                    ? "Argo CD"
                    : "Passive"}
              </b>
            </span>
            {form.deploy_source === "gitops" &&
              form.update_mode === "argocd" && (
                <span>
                  Future GHCR image <b>{ghcrPreview}</b>
                </span>
              )}
          </div>
        )}
        {step.id !== "method" && step.id !== "update" && (
          <div className="wizard-actions">
            <Button type="button" onClick={back} disabled={stepIndex === 0}>
              Back
            </Button>
            {step.id === "confirm" ? (
              <Button
                disabled={
                  form.application_type === "monorepo"
                    ? !(analysis?.is_monorepo && analysis?.deployable !== false)
                    : !analysis?.deployable
                }
                type="submit"
                variant="primary"
              >
                <Play size={16} /> Build
              </Button>
            ) : (
              <Button
                type="button"
                disabled={!canContinue || checkingInstall}
                onClick={next}
                variant="primary"
              >
                {checkingInstall ? (
                  <LoaderCircle className="spin" size={16} />
                ) : null}{" "}
                Next
              </Button>
            )}
          </div>
        )}
      </form>
    </div>
  );
}

import React, { useEffect, useMemo, useRef, useState } from "react";
import * as LucideIcons from "lucide-react";
import {
  canContinueDeployStep,
  basaltPassStepBlockers,
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
import { t } from "../i18n/index";
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
  Modal,
  Select,
  Checkbox,
} from "../components/index";
import DependencyCreateForm from "./DependencyCreateForm";
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
    label: t("Target"),
    title: t("Choose deployment target"),
  },
  {
    id: "source",
    label: t("Source"),
    title: t("Choose deployment source details"),
  },
  {
    id: "update",
    label: t("Update"),
    title: t("Choose update mode"),
  },
  {
    id: "check",
    label: t("Check"),
    title: t("Check installability"),
  },
  {
    id: "params",
    label: t("Params"),
    title: t("Configure parameters"),
  },
  {
    id: "dependencies",
    label: t("Dependencies"),
    title: t("Configure dependencies"),
  },
  {
    id: "namespace",
    label: t("Namespace"),
    title: t("Choose namespace"),
  },
  {
    id: "ingress",
    label: t("Ingress"),
    title: t("Choose ingress mode"),
  },
  {
    id: "domain",
    label: t("Domain"),
    title: t("Choose domain"),
  },
  {
    id: "env",
    label: t("Env"),
    title: t("Add runtime variables"),
  },
  {
    id: "confirm",
    label: t("Confirm"),
    title: t("Confirm and build"),
  },
];
const basaltPassDeploySteps = [
  {
    id: "method",
    label: t("Target"),
    title: t("Choose deployment target"),
  },
  {
    id: "source",
    label: t("Repository"),
    title: t("Choose BasaltPass repository"),
  },
  {
    id: "params",
    label: t("Runtime"),
    title: t("Configure BasaltPass runtime"),
  },
  {
    id: "dependencies",
    label: t("Tenant"),
    title: t("Create tenant and credentials"),
  },
  {
    id: "confirm",
    label: t("Confirm"),
    title: t("Confirm BasaltPass deployment"),
  },
];
const deployTargetOptions = [
  {
    id: "project",
    label: t("Application"),
    icon: Rocket,
    description: t("Deploy an application service or monorepo workload."),
  },
  {
    id: "basaltpass",
    label: t("BasaltPass"),
    icon: ShieldCheck,
    description: t("Deploy a BasaltPass platform and store the new tenant."),
  },
];
const deploySourceOptions = [
  {
    id: "gitops",
    label: t("GitOps repository"),
    icon: GitBranch,
    description: t(
      "Use a GitHub repository as source and publish runtime images to BeanCS Harbor.",
    ),
  },
  {
    id: "registry",
    label: t("Container registry"),
    icon: Package,
    description: t("Deploy an existing or newly tracked container image object."),
  },
];
const updateModeOptions = [
  {
    id: "argocd",
    label: t("Argo CD"),
    icon: GitBranch,
    description: t(
      "Create GitOps manifests, register an Argo CD app, and let GitHub Actions build the first Harbor image.",
    ),
  },
  {
    id: "passive",
    label: t("Passive update"),
    icon: RefreshCw,
    description: t("Create the project without automatic GitHub push deployment."),
  },
];
export default function DeployView({
  config,
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
  reusableDependencies,
  createTrackedImageFromDeploy,
  deployBasaltPass,
  onDeployDependency,
  onConnectGitHub,
  reposLoading,
}) {
  const [stepIndex, setStepIndex] = useState(0);
  const [creatingImage, setCreatingImage] = useState(false);
  const [checkingInstall, setCheckingInstall] = useState(false);
  const [repoSearch, setRepoSearch] = useState("");
  const [accountMenuOpen, setAccountMenuOpen] = useState(false);
  const [dependencyCreateOpen, setDependencyCreateOpen] = useState(false);
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
  const registryHost = String(config?.registry_host || "")
    .replace(/^https?:\/\//, "")
    .replace(/\/+$/, "");
  const basaltpassImageProject = slugify(
    String(config?.basaltpass_image_project || "basaltpass"),
  );
  const basaltpassImageBase =
    registryHost && basaltpassImageProject
      ? `${registryHost}/${basaltpassImageProject}`
      : "";
  const isBasaltPassDeploy = form.deploy_target === "basaltpass";
  const databaseDependencies = (reusableDependencies || []).filter(
    (dependency) => ["mysql", "postgresql", "timescaledb"].includes(dependency.type),
  );
  const selectedDatabaseDependency = databaseDependencies.find(
    (dependency) => String(dependency.id) === String(form.database_dependency_id),
  );
  const selectedDatabaseCredentials =
    selectedDatabaseDependency?.credentials || [];
  const publicHost =
    form.subdomain && selectedCloudflareDomain
      ? `${form.subdomain}.${selectedCloudflareDomain.domain}`
      : "";
  const basaltPassPublicHost =
    isBasaltPassDeploy && form.subdomain && selectedCloudflareDomain
      ? `${form.subdomain}.${selectedCloudflareDomain.domain}`
      : form.public_host || "";
  const basaltPassBaseURL = basaltPassPublicHost
    ? `https://${basaltPassPublicHost}`
    : form.base_url || "";
  const activeSteps = isBasaltPassDeploy ? basaltPassDeploySteps : deploySteps;
  const step = activeSteps[stepIndex] || activeSteps[0];
  const stepBlockers = isBasaltPassDeploy
    ? basaltPassStepBlockers(step.id, form, selectedCredential)
    : [];
  const basaltPassBuildBlockers = isBasaltPassDeploy
    ? basaltPassStepBlockers("dependencies", form, selectedCredential)
    : [];
  const canContinue = isBasaltPassDeploy
    ? stepBlockers.length === 0
    : canContinueDeployStep(step.id, form, selectedCredential, analysis);
  const harborPreviewRepo = slugify(form.name || form.github_repo?.split("/").pop() || "app");
  const harborPreview = registryHost
    ? `${registryHost}/<tenant>/${harborPreviewRepo}:beancs-<build>`
    : "<BeanCS Harbor>/<tenant>/<app>:beancs-<build>";
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
  const setDeployTarget = (deployTarget) => {
    setAnalysis(null);
    if (deployTarget === "basaltpass") {
      setForm({
        ...defaultDeployForm(),
        deploy_target: "basaltpass",
        deploy_source: "gitops",
        build_source: "github",
        repo_type: "github",
        application_type: "single",
        exposure_mode: "public",
      });
      setStepIndex(1);
      return;
    }
    setForm({
      ...defaultDeployForm(),
      deploy_target: "project",
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
    if (step.id === "check" && !isBasaltPassDeploy) {
      const result = await runInstallCheck();
      if (result && stepIndex < activeSteps.length - 1)
        setStepIndex(stepIndex + 1);
      return;
    }
    if (stepIndex < activeSteps.length - 1) setStepIndex(stepIndex + 1);
  };
  const selectRepository = (repo, repoName, branch) => {
    const nextName = form.name || slugify(repoName);
    const nextForm = {
      ...form,
      github_repo: repo.full_name,
      github_branch: branch,
      name: nextName,
      tenant_name: form.tenant_name || nextName,
    };
    if (isBasaltPassDeploy) {
      const defaultSlug = slugify(nextName);
      const backendImage = basaltpassImageBase
        ? `${basaltpassImageBase}/basaltpass-backend:latest`
        : "";
      const frontendImage = basaltpassImageBase
        ? `${basaltpassImageBase}/basaltpass-frontend:latest`
        : "";
      setForm({
        ...nextForm,
        backend_image: form.backend_image || backendImage,
        frontend_image: form.frontend_image || frontendImage,
        tenant_code: form.tenant_code || defaultSlug,
        subdomain: form.subdomain || defaultSlug,
      });
      return;
    }
    setForm(nextForm);
    analyzeRepo(repo.full_name, branch);
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
          source: "new",
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
      <div className="deploy-wizard-actions">
        <Button type="button" onClick={() => setDependencyCreateOpen(true)}>
          <Database size={15} /> {t("Deploy dependency")}
        </Button>
      </div>
      <section className="panel wizard-progress-panel">
        <div className="wizard-progress-head">
          <span>{step.label}</span>
          <b>
            {stepIndex + 1} / {activeSteps.length}
          </b>
        </div>
        <div className="wizard-progress-track">
          <span
            style={{
              width: `${((stepIndex + 1) / activeSteps.length) * 100}%`,
            }}
          />
        </div>
        <div className="wizard-step-labels">
          {activeSteps.map((item, index) => (
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
      <form
        className="panel deploy-form wizard-panel"
        onSubmit={isBasaltPassDeploy ? deployBasaltPass : deployProject}
      >
        <h2>
          <Rocket size={18} /> {step.title}
        </h2>
        {step.id === "method" && (
          <div className="method-grid">
            {deployTargetOptions.map((method) => {
              const Icon = method.icon;
              return (
                <Button
                  key={method.id}
                  type="button"
                  className={
                    (form.deploy_target || "project") === method.id
                      ? "method-card active"
                      : "method-card"
                  }
                  onClick={() => setDeployTarget(method.id)}
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
            {!isBasaltPassDeploy && (
              <>
                <label>{t("Deployment source")}</label>
                <div className="method-grid two-up">
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
              </>
            )}
            {form.deploy_source === "gitops" && (
              <>
                {!isBasaltPassDeploy && (
                  <>
                    <label>{t("Repository type")}</label>
                    <div className="segmented-control">
                      <Button
                        type="button"
                        className={form.repo_type === "github" ? "active" : ""}
                        onClick={() => setRepoType("github")}
                      >
                        <Github size={15} /> {t("GitHub")}
                      </Button>
                      <Button
                        type="button"
                        className={form.repo_type === "git-url" ? "active" : ""}
                        onClick={() => setRepoType("git-url")}
                      >
                        <GitBranch size={15} /> {t("Git link")}
                      </Button>
                    </div>
                    <label>{t("Repository layout")}</label>
                    <div className="segmented-control">
                      <Button
                        type="button"
                        className={
                          form.application_type !== "monorepo" ? "active" : ""
                        }
                        onClick={() => setApplicationType("single")}
                      >
                        <Box size={15} /> {t("Single service")}
                      </Button>
                      <Button
                        type="button"
                        className={
                          form.application_type === "monorepo" ? "active" : ""
                        }
                        onClick={() => setApplicationType("monorepo")}
                      >
                        <Layers3 size={15} /> {t("Monorepo")}
                      </Button>
                    </div>
                  </>
                )}
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
                                <span>{t("Add GitHub Account")}</span>
                              </Button>
                              <Button
                                type="button"
                                onClick={() => {
                                  setAccountMenuOpen(false);
                                  setRepoType("git-url");
                                }}
                              >
                                <ListRestart size={16} />
                                <span>{t("Switch Git Provider")}</span>
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
                                      <CheckCircle2 size={14} /> {t("Selected")}
                                    </b>
                                  )}
                                </div>
                                <Button
                                  type="button"
                                  onClick={() => {
                                    selectRepository(repo, repoName, branch);
                                  }}
                                >
                                  {isSelected ? t("Selected") : t("Import")}
                                </Button>
                              </div>
                            );
                          })}
                        {!reposLoading && visibleRepos.length === 0 && (
                          <div className="empty">
                            {selectedCredential
                              ? t("No repositories match this search.")
                              : t("Choose a GitHub account to load repositories.")}
                          </div>
                        )}
                      </div>
                      {form.github_repo && (
                        <div className="selected-repo-summary">
                          <CheckCircle2 size={16} />
                          <span>{t("Selected repository")}</span>
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
                      label={t("Git URL")}
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
                      {t(
                        "The current deployment flow does not support a direct git link. Use a connected GitHub repo instead.",
                      )}
                    </p>
                  </>
                )}
              </>
            )}
            {form.deploy_source === "registry" && (
              <>
                <label>{t("Image object")}</label>
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
                    <Package size={15} /> {t("Existing")}
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
                    <Plus size={15} /> {t("New object")}
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
                              ? t("{count} tags cached", {
                                  count: (image.tags || []).length,
                                })
                              : t("No cached tags")}
                          </small>
                        </Button>
                      ))}
                      {containerImages.length === 0 && (
                        <div className="empty">
                          {t(
                            "No image objects yet. Create one below or open Image Registry.",
                          )}
                        </div>
                      )}
                    </div>
                    {form.selected_image_id && (
                      <>
                        <label>{t("Tag")}</label>
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
                          label={t("Image reference")}
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
                    <label>{t("Registry")}</label>
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
                      <option value="">{t("Choose registry")}</option>
                      {containerRegistries.map((registry) => (
                        <option key={registry.id} value={registry.id}>
                          {registry.name} ({registry.kind})
                        </option>
                      ))}
                    </Select>
                    <Field
                      label={t("Repository path")}
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
                      <Plus size={15} /> {t("Create image object")}
                    </Button>
                    <p className="muted">
                      {t(
                        "After saving the object you return to selection and use this image for a passive update deployment.",
                      )}
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
                <b>{t("Passive update")}</b>
                <span>
                  {t(
                    "Registry deployments only support passive updates in the current flow.",
                  )}
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
                  ? t("Checking installability...")
                  : t(
                      "BeanCS will verify repository signals or image/source inputs before continuing.",
                    )}
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
                    ? t(".beancs spec found: {path}", {
                        path: analysis.config_path,
                      })
                    : analysis.is_monorepo
                      ? t("{count} components detected", {
                          count: analysis.components?.length || 0,
                        })
                      : t("No components detected")}
                </div>
                {analysis.source === "beancs_spec" && (
                  <ApplicationSpecPlanSummary analysis={analysis} />
                )}
                <div className="signal-list">
                  {analysis.package_manager && (
                    <span>
                      {t("Package manager:")} {analysis.package_manager}
                    </span>
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
                    ? t("Deployable")
                    : analysis.scaffoldable
                      ? t("Source detected")
                      : t("Needs containerization")}
                </div>
                <div className="signal-list">
                  {(analysis.signals || []).map((signal) => (
                    <span key={signal}>{signal}</span>
                  ))}
                  {analysis.compose_path && (
                    <span>
                      {t("Compose:")} {analysis.compose_path}
                    </span>
                  )}
                  {analysis.ports?.length > 0 && (
                    <span>
                      {t("Ports:")} {analysis.ports.join(", ")}
                    </span>
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
        {step.id === "params" && isBasaltPassDeploy && (
          <div className="form-grid">
            <Field
              label={t("Deployment name")}
              value={form.name}
              onChange={(v) =>
                setForm({
                  ...form,
                  name: slugify(v),
                  tenant_name: form.tenant_name || v.trim(),
                  tenant_code: form.tenant_code || slugify(v),
                  subdomain: form.subdomain || slugify(v),
                })
              }
              required
            />
            <Field
              label={t("Tenant name")}
              value={form.tenant_name}
              onChange={(v) =>
                setForm({
                  ...form,
                  tenant_name: v.trim(),
                  tenant_code: form.tenant_code || slugify(v),
                })
              }
              required
            />
            <Field
              label={t("Namespace")}
              value={form.namespace}
              onChange={(v) =>
                setForm({
                  ...form,
                  namespace: slugify(v),
                })
              }
              placeholder={form.name ? `bp-${form.name}` : "bp-basaltpass"}
            />
            <Field
              label={t("Backend image")}
              value={form.backend_image}
              onChange={(v) =>
                setForm({
                  ...form,
                  backend_image: v.trim(),
                })
              }
              required
            />
            <Field
              label={t("Frontend image")}
              value={form.frontend_image}
              onChange={(v) =>
                setForm({
                  ...form,
                  frontend_image: v.trim(),
                })
              }
              required
            />
            <label>{t("Traffic")}</label>
            <Select
              value={form.exposure_mode}
              onChange={(event) =>
                setForm({
                  ...form,
                  exposure_mode: event.target.value,
                })
              }
            >
              <option value="public">{t("Traefik public ingress")}</option>
              <option value="private">{t("Tailscale private ingress")}</option>
            </Select>
            {form.exposure_mode === "public" && (
              <>
                <label>{t("Domain")}</label>
                <Select
                  value={
                    form.cloudflare_zone_id
                      ? `${form.cloudflare_credential_id}:${form.cloudflare_zone_id}`
                      : ""
                  }
                  onChange={(event) => {
                    const [credentialID, zoneID] = event.target.value.split(":");
                    setForm({
                      ...form,
                      cloudflare_credential_id: credentialID || "",
                      cloudflare_zone_id: zoneID || "",
                    });
                  }}
                  required
                >
                  <option value="">{t("Choose Cloudflare zone")}</option>
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
                  label={t("Subdomain")}
                  value={form.subdomain}
                  onChange={(v) =>
                    setForm({
                      ...form,
                      subdomain: slugify(v),
                    })
                  }
                  placeholder="basaltpasstest"
                  required
                />
                <div className="computed-host">
                  {basaltPassPublicHost || t("Choose a domain")}
                </div>
                <div className="computed-host">
                  {basaltPassBaseURL || t("Base URL preview")}
                </div>
              </>
            )}
            {form.exposure_mode === "private" && (
              <Field
                label={t("Private host")}
                value={form.public_host}
                onChange={(v) =>
                  setForm({
                    ...form,
                    public_host: v.trim().toLowerCase(),
                  })
                }
                placeholder="basaltpass.internal.example"
                required
              />
            )}
            <Field
              label={t("CORS origins")}
              value={form.cors_allow_origins}
              onChange={(v) =>
                setForm({
                  ...form,
                  cors_allow_origins: v.trim(),
                })
              }
              placeholder={t("Defaults to Base URL")}
            />
            <Field
              label={t("Platform admin email")}
              type="email"
              value={form.platform_admin_email}
              onChange={(v) => {
                const value = v.trim();
                setForm({
                  ...form,
                  platform_admin_email: value,
                  platform_admin_username:
                    form.platform_admin_username || value.split("@")[0],
                });
              }}
              required
            />
            <Field
              label={t("Platform admin username")}
              value={form.platform_admin_username}
              onChange={(v) =>
                setForm({
                  ...form,
                  platform_admin_username: v.trim(),
                })
              }
              required
            />
            <Field
              label={t("Platform admin password")}
              type="password"
              value={form.platform_admin_password}
              onChange={(v) =>
                setForm({
                  ...form,
                  platform_admin_password: v,
                })
              }
              required
            />
            <Field
              label={t("JWT secret")}
              type="password"
              value={form.jwt_secret}
              onChange={(v) =>
                setForm({
                  ...form,
                  jwt_secret: v,
                })
              }
              placeholder={t("Generated if empty")}
            />
          </div>
        )}
        {step.id === "params" && !isBasaltPassDeploy && (
          <div className="form-grid">
            <Field
              label={
                form.application_type === "monorepo"
                  ? t("Application name")
                  : t("Project name")
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
                  label={t("Port")}
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
                  label={t("Replicas")}
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
            <label>{t("Resource preset")}</label>
            <Select
              value={form.resource_preset}
              onChange={(event) =>
                setForm({
                  ...form,
                  resource_preset: event.target.value,
                })
              }
            >
              <option value="nano">{t("Nano")}</option>
              <option value="small">{t("Small")}</option>
              <option value="medium">{t("Medium")}</option>
              <option value="large">{t("Large")}</option>
            </Select>
            <label>{t("BasaltPass tenant")}</label>
            <Select
              value={form.basaltpass_instance_id}
              onChange={(event) =>
                setForm({
                  ...form,
                  basaltpass_instance_id: event.target.value,
                })
              }
            >
              <option value="">{t("Do not register OAuth app")}</option>
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
                    {t(
                      "These components are declared by repo config. Edit {path} in the repository to change build args, health checks, volumes, or dependency bindings.",
                      { path: analysis.config_path },
                    )}
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
                        label={t("Project name")}
                        value={component.project_name}
                        onChange={(v) =>
                          updateComponent(index, {
                            project_name: slugify(v),
                          })
                        }
                        required
                      />
                      <Field
                        label={t("Component path")}
                        value={component.component_path || component.path}
                        onChange={(v) =>
                          updateComponent(index, {
                            component_path: v.trim(),
                          })
                        }
                      />
                      <Field
                        label={t("Dockerfile")}
                        value={component.dockerfile_path}
                        onChange={(v) =>
                          updateComponent(index, {
                            dockerfile_path: v.trim(),
                          })
                        }
                        required
                      />
                      <Field
                        label={t("Build context")}
                        value={component.build_context || "."}
                        onChange={(v) =>
                          updateComponent(index, {
                            build_context: v.trim() || ".",
                          })
                        }
                      />
                      <Field
                        label={t("Port")}
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
                        label={t("Replicas")}
                        type="number"
                        value={component.replicas || 1}
                        onChange={(v) =>
                          updateComponent(index, {
                            replicas: Number(v || 1),
                          })
                        }
                      />
                      <label>{t("Exposure")}</label>
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
                        <option value="public">{t("Public")}</option>
                        <option value="private">{t("Private")}</option>
                        <option value="internal-only">{t("Internal only")}</option>
                      </Select>
                    </div>
                    {analysis?.source === "beancs_spec" && (
                      <div className="spec-component-meta">
                        {component.build_args &&
                          Object.keys(component.build_args).length > 0 && (
                            <span>
                              {t("Build args:")}{" "}
                              {Object.entries(component.build_args)
                                .map(([key, value]) => `${key}=${value}`)
                                .join(", ")}
                            </span>
                          )}
                        {component.health_check?.type && (
                          <span>
                            {t("Health:")} {component.health_check.type}
                            {component.health_check.path
                              ? ` ${component.health_check.path}`
                              : ""}
                          </span>
                        )}
                        {(component.volumes || []).length > 0 && (
                          <span>
                            {t("Volumes:")}{" "}
                            {(component.volumes || [])
                              .map(
                                (volume) =>
                                  `${volume.name}:${volume.mountPath || volume.mount_path}`,
                              )
                              .join(", ")}
                          </span>
                        )}
                        {(component.watch_paths || []).length > 0 && (
                          <span>{t("Watch:")} {component.watch_paths.join(", ")}</span>
                        )}
                      </div>
                    )}
                  </div>
                ))}
                {(form.components || []).length === 0 && (
                  <div className="empty">
                    {t("Run repository analysis to detect deployable components.")}
                  </div>
                )}
              </div>
            )}
          </div>
        )}
        {step.id === "dependencies" && isBasaltPassDeploy && (
          <div className="form-grid">
            <label>{t("Database")}</label>
            <Select
              value={form.database_dependency_id}
              onChange={(event) => {
                const dependencyID = event.target.value;
                const dependency = databaseDependencies.find(
                  (item) => String(item.id) === String(dependencyID),
                );
                const firstCredential = (dependency?.credentials || [])[0];
                setForm({
                  ...form,
                  database_dependency_id: dependencyID,
                  database_binding: firstCredential
                    ? `${dependencyID}:${firstCredential.id}`
                    : "",
                });
              }}
              required
            >
              <option value="">{t("Choose MySQL or PostgreSQL")}</option>
              {databaseDependencies.map((dependency) => (
                <option key={dependency.id} value={dependency.id}>
                  {dependency.name} · {dependency.type}
                </option>
              ))}
            </Select>
            <label>{t("Credential mode")}</label>
            <div className="segmented-control">
              <Button
                type="button"
                className={
                  form.database_credential_mode !== "new" ? "active" : ""
                }
                onClick={() =>
                  setForm({
                    ...form,
                    database_credential_mode: "existing",
                  })
                }
              >
                <ShieldCheck size={15} /> {t("Existing")}
              </Button>
              <Button
                type="button"
                className={form.database_credential_mode === "new" ? "active" : ""}
                onClick={() =>
                  setForm({
                    ...form,
                    database_credential_mode: "new",
                    database_binding: "",
                  })
                }
              >
                <Plus size={15} /> {t("New")}
              </Button>
            </div>
            {form.database_credential_mode === "new" ? (
              <>
                <Field
                  label={t("Credential name")}
                  value={form.database_credential_name}
                  onChange={(v) =>
                    setForm({
                      ...form,
                      database_credential_name: v.trim(),
                    })
                  }
                  required
                />
                <Field
                  label={t("Database name")}
                  value={form.database_name}
                  onChange={(v) =>
                    setForm({
                      ...form,
                      database_name: v.trim(),
                    })
                  }
                  required
                />
                <Field
                  label={t("Database username")}
                  value={form.database_username}
                  onChange={(v) =>
                    setForm({
                      ...form,
                      database_username: v.trim(),
                    })
                  }
                  required
                />
                <Field
                  label={t("Database password")}
                  type="password"
                  value={form.database_password}
                  onChange={(v) =>
                    setForm({
                      ...form,
                      database_password: v,
                    })
                  }
                  required
                />
                <Field
                  label={t("Credential description")}
                  value={form.database_credential_description}
                  onChange={(v) =>
                    setForm({
                      ...form,
                      database_credential_description: v,
                    })
                  }
                />
              </>
            ) : (
              <>
                <label>{t("Database credential")}</label>
                <Select
                  value={form.database_binding}
                  onChange={(event) =>
                    setForm({
                      ...form,
                      database_binding: event.target.value,
                    })
                  }
                  required
                  disabled={!selectedDatabaseDependency}
                >
                  <option value="">{t("Choose existing credential")}</option>
                  {selectedDatabaseCredentials.map((credential) => (
                    <option
                      key={`${selectedDatabaseDependency.id}:${credential.id}`}
                      value={`${selectedDatabaseDependency.id}:${credential.id}`}
                    >
                      {credential.name}
                    </option>
                  ))}
                </Select>
              </>
            )}
            <Field
              label={t("Tenant code")}
              value={form.tenant_code}
              onChange={(v) =>
                setForm({
                  ...form,
                  tenant_code: slugify(v),
                })
              }
              required
            />
            <Field
              label={t("Tenant admin email")}
              type="email"
              value={form.owner_email}
              onChange={(v) => {
                const value = v.trim();
                setForm({
                  ...form,
                  owner_email: value,
                  owner_username: form.owner_username || value.split("@")[0],
                });
              }}
              required
            />
            <Field
              label={t("Tenant admin username")}
              value={form.owner_username}
              onChange={(v) =>
                setForm({
                  ...form,
                  owner_username: v.trim(),
                })
              }
              required
            />
            <Field
              label={t("Tenant admin password")}
              type="password"
              value={form.owner_password}
              onChange={(v) =>
                setForm({
                  ...form,
                  owner_password: v,
                })
              }
              required
            />
            <Field
              label={t("Description")}
              value={form.description}
              onChange={(v) =>
                setForm({
                  ...form,
                  description: v,
                })
              }
            />
            <Field
              label={t("Max apps")}
              type="number"
              value={form.max_apps}
              onChange={(v) =>
                setForm({
                  ...form,
                  max_apps: v,
                })
              }
              placeholder="50"
            />
            <Field
              label={t("Max users")}
              type="number"
              value={form.max_users}
              onChange={(v) =>
                setForm({
                  ...form,
                  max_users: v,
                })
              }
              placeholder="500"
            />
            <Field
              label={t("Platform management token fallback")}
              type="password"
              value={form.service_token}
              onChange={(v) =>
                setForm({
                  ...form,
                  service_token: v,
                })
              }
            />
            <Field
              label={t("Tenant automation token fallback")}
              type="password"
              value={form.automation_token}
              onChange={(v) =>
                setForm({
                  ...form,
                  automation_token: v,
                })
              }
            />
          </div>
        )}
        {step.id === "dependencies" && !isBasaltPassDeploy && (
          <div className="form-grid">
            {form.application_type !== "monorepo" && (
              <p className="muted">
                {t(
                  "Managed dependency components are currently available for monorepo applications.",
                )}
              </p>
            )}
            {form.application_type === "monorepo" && (
              <>
                <div className="section-head">
                  <div>
                    <h3>{t("Dependency components")}</h3>
                    <p className="muted">
                      {t(
                        "Definitions drive config, outputs, and env presets for application components.",
                      )}
                    </p>
                  </div>
                  <Button
                    type="button"
                    onClick={addDependency}
                    disabled={!dependencyDefinitions.length}
                  >
                    <Plus size={15} /> {t("Add dependency")}
                  </Button>
                </div>
                <div className="dependency-list">
                  {analysis?.source === "beancs_spec" && (
                    <p className="muted">
                      {t(
                        "Dependencies are declared by repo config and will be created from the spec during deploy.",
                      )}
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
                          <b>{dependency.name || t("dependency")}</b>
                          <Button
                            type="button"
                            onClick={() => deleteDependency(index)}
                            title={t("Remove dependency")}
                            variant="danger"
                          >
                            <Trash2 size={15} />
                          </Button>
                        </div>
                        <div className="component-grid">
                          <Field
                            label={t("Name")}
                            value={dependency.name}
                            onChange={(v) =>
                              updateDependency(index, {
                                ...dependency,
                                name: slugify(v),
                              })
                            }
                            required
                          />
                          <label>{t("Source")}</label>
                          <Select
                            value={dependency.source || "new"}
                            onChange={(event) => {
                              const source = event.target.value;
                              if (source === "existing") {
                                const match = (reusableDependencies || []).find(
                                  (item) => item.type === dependency.type,
                                );
                                updateDependency(index, {
                                  source,
                                  existing_dependency_id: match?.id || "",
                                  name: match?.name || dependency.name,
                                  type: match?.type || dependency.type,
                                  credentials: match?.credentials || [],
                                });
                                return;
                              }
                              updateDependency(index, {
                                source,
                                existing_dependency_id: "",
                                credentials: [],
                              });
                            }}
                          >
                            <option value="new">{t("New")}</option>
                            <option value="existing">{t("Existing")}</option>
                          </Select>
                          <label>{t("Type")}</label>
                          <Select
                            value={dependency.type}
                            disabled={(dependency.source || "new") === "existing"}
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
                          {(dependency.source || "new") === "existing" && (
                            <>
                              <label>{t("Dependency")}</label>
                              <Select
                                value={dependency.existing_dependency_id || ""}
                                onChange={(event) => {
                                  const match = (reusableDependencies || []).find(
                                    (item) =>
                                      String(item.id) === event.target.value,
                                  );
                                  updateDependency(index, {
                                    existing_dependency_id: match?.id || "",
                                    name: match?.name || dependency.name,
                                    type: match?.type || dependency.type,
                                    credentials: match?.credentials || [],
                                  });
                                }}
                              >
                                <option value="">{t("Choose dependency")}</option>
                                {(reusableDependencies || [])
                                  .filter(
                                    (item) => item.type === dependency.type,
                                  )
                                  .map((item) => (
                                    <option key={item.id} value={item.id}>
                                      {item.name} ·{" "}
                                      {item.external ? t("external") : t("managed")}
                                    </option>
                                  ))}
                              </Select>
                            </>
                          )}
                          <label>{t("Deploy method")}</label>
                          <Select
                            value={
                              dependency.deploy_method ||
                              definition?.default_deploy_method ||
                              "helm"
                            }
                            onChange={(event) =>
                              updateDependency(index, {
                                deploy_method: event.target.value,
                                external: event.target.value === "external",
                              })
                            }
                            disabled={(dependency.source || "new") === "existing"}
                          >
                            {(
                              definition?.supported_deploy_methods || ["helm"]
                            ).map((method) => (
                              <option key={method} value={method}>
                                {t(method)}
                              </option>
                            ))}
                          </Select>
                          <Field
                            label={t("Version")}
                            value={dependency.version || ""}
                            onChange={(v) =>
                              updateDependency(index, {
                                version: v.trim(),
                              })
                            }
                          />
                        </div>
                        {(dependency.source || "new") !== "existing" && (
                          <>
                            {(dependency.deploy_method === "external" ||
                              dependency.external) && (
                              <label className="checkbox-label">
                                <Input
                                  type="checkbox"
                                  checked={Boolean(dependency.controlled)}
                                  onChange={(event) =>
                                    updateDependency(index, {
                                      controlled: event.target.checked,
                                    })
                                  }
                                />
                                <span>{t("BeanCS can create credentials")}</span>
                              </label>
                            )}
                            <DependencyConfigEditor
                              definition={definition}
                              value={dependency.config || {}}
                              onChange={(config) =>
                                updateDependency(index, {
                                  config,
                                })
                              }
                            />
                          </>
                        )}
                      </div>
                    );
                  })}
                  {(form.dependencies || []).length === 0 && (
                    <div className="empty">
                      {t("No managed dependencies selected.")}
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
            <label>{t("Namespace")}</label>
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
              {t("Leave empty to create {name} automatically.", {
                name: form.name ? `proj-${form.name}` : t("a project namespace"),
              })}
            </p>
          </div>
        )}
        {step.id === "ingress" && (
          <div className="form-grid">
            {form.application_type === "monorepo" ? (
              <>
                <p className="muted">
                  {t(
                    "Traffic mode is configured per component in the parameters step.",
                  )}
                </p>
                <div className="signal-list">
                  {(form.components || [])
                    .filter((component) => component.enabled !== false)
                    .map((component) => (
                      <span key={component.project_name}>
                        {component.project_name}:{" "}
                        {component.port
                          ? component.exposure_mode
                          : t("internal-only")}
                      </span>
                    ))}
                </div>
              </>
            ) : (
              <>
                <label>{t("Traffic")}</label>
                <Select
                  value={form.exposure_mode}
                  onChange={(event) =>
                    setForm({
                      ...form,
                      exposure_mode: event.target.value,
                    })
                  }
                >
                  <option value="public">{t("Traefik public ingress")}</option>
                  <option value="private">{t("Tailscale private ingress")}</option>
                  <option value="internal-only">{t("Cluster internal only")}</option>
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
                  <label>{t("Cloudflare credential")}</label>
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
                    <option value="">{t("Choose Cloudflare zone")}</option>
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
                    {t(
                      "Public component hostnames use the component project name under the selected zone.",
                    )}
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
                    {t("No public DNS zone is required for the selected components.")}
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
                            {t(component.exposure_mode || "internal-only")}
                          </span>
                        </div>
                        {component.exposure_mode === "public" ? (
                          <div className="component-grid">
                            <Field
                              label={t("Subdomain")}
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
                              label={t("Tailscale host")}
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
                          <p className="muted">{t("Internal-only component.")}</p>
                        )}
                      </div>
                    ))}
                </div>
              </>
            )}
            {form.application_type !== "monorepo" &&
              form.exposure_mode === "public" && (
                <>
                  <label>{t("Cloudflare credential")}</label>
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
                    <option value="">{t("Choose Cloudflare zone")}</option>
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
                    label={t("Subdomain")}
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
                    {publicHost || t("Subdomain preview")}
                  </div>
                </>
              )}
            {form.application_type !== "monorepo" &&
              form.exposure_mode === "private" && (
                <Field
                  label={t("Tailscale host")}
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
                  {t("No domain is required for internal-only projects.")}
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
            title={t("Runtime environment")}
          />
        )}
        {step.id === "confirm" && isBasaltPassDeploy && (
          <div className="detail-list">
            <span>
              {t("Target")} <b>BasaltPass</b>
            </span>
            <span>
              {t("Source")}{" "}
              <b>
                {form.github_repo || "-"} @ {form.github_branch || "main"}
              </b>
            </span>
            <span>
              {t("Name")} <b>{form.name || "-"}</b>
            </span>
            <span>
              {t("Namespace")}{" "}
              <b>{form.namespace || (form.name ? `bp-${form.name}` : "-")}</b>
            </span>
            <span>
              {t("Base URL")} <b>{basaltPassBaseURL || "-"}</b>
            </span>
            <span>
              {t("Host")} <b>{basaltPassPublicHost || t("private ingress")}</b>
            </span>
            <span>
              {t("Backend image")} <b>{form.backend_image || "-"}</b>
            </span>
            <span>
              {t("Frontend image")} <b>{form.frontend_image || "-"}</b>
            </span>
            <span>
              {t("Tenant")} <b>{form.tenant_name || "-"}</b>
            </span>
            <span>
              {t("Tenant code")} <b>{form.tenant_code || "-"}</b>
            </span>
            <span>
              {t("Platform admin")} <b>{form.platform_admin_email || "-"}</b>
            </span>
            <span>
              {t("Tenant admin")} <b>{form.owner_email || "-"}</b>
            </span>
            <span>
              {t("Database")}{" "}
              <b>
                {selectedDatabaseDependency?.name || "-"} /{" "}
                {form.database_credential_mode === "new"
                  ? form.database_credential_name || t("new credential")
                  : selectedDatabaseCredentials.find(
                      (credential) =>
                        form.database_binding ===
                        `${selectedDatabaseDependency?.id}:${credential.id}`,
                    )?.name || "-"}
              </b>
            </span>
          </div>
        )}
        {step.id === "confirm" && !isBasaltPassDeploy && (
          <div className="detail-list">
            <span>
              {t("Install method")} <b>{sourceLabel(form.build_source)}</b>
            </span>
            <span>
              {t("Source")} <b>{sourceSummary(form)}</b>
            </span>
            <span>
              {form.application_type === "monorepo"
                ? t("Application")
                : t("Project")}{" "}
              <b>{form.name || "-"}</b>
            </span>
            <span>
              {t("Namespace")}{" "}
              <b>{form.namespace || (form.name ? `proj-${form.name}` : "-")}</b>
            </span>
            <span>
              {t("Ingress")} <b>{t(form.exposure_mode)}</b>
            </span>
            <span>
              {t("Domain")}{" "}
              <b>{publicHost || form.private_host || t("internal only")}</b>
            </span>
            {form.application_type === "monorepo" ? (
              <span>
                {t("Components")}{" "}
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
                {t("Port")} <b>{form.port}</b>
              </span>
            )}
            {form.application_type === "monorepo" && (
              <span>
                {t("Dependencies")} <b>{(form.dependencies || []).length}</b>
              </span>
            )}
            <span>
              {t("Runtime variables")}{" "}
              <b>
                {
                  (form.env_entries || []).filter((entry) => entry.key.trim())
                    .length
                }
              </b>
            </span>
            <span>
              {t("Update mode")}{" "}
              <b>
                {form.deploy_source === "registry"
                  ? t("Passive")
                  : form.update_mode === "argocd"
                    ? t("Argo CD")
                    : t("Passive")}
              </b>
            </span>
            {form.deploy_source === "gitops" &&
              form.update_mode === "argocd" && (
                <span>
                  {t("Future Harbor image")} <b>{harborPreview}</b>
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
                  isBasaltPassDeploy
                    ? basaltPassBuildBlockers.length > 0
                    : form.application_type === "monorepo"
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
            {!canContinue && stepBlockers.length > 0 && (
              <p className="warning-note span-2">
                {t("Missing:")} {stepBlockers.join(", ")}
              </p>
            )}
          </div>
        )}
      </form>
      {dependencyCreateOpen && (
        <Modal
          title={t("Deploy dependency")}
          subtitle={t("Create and deploy a managed dependency from this workflow.")}
          className="wide-modal"
          onClose={() => setDependencyCreateOpen(false)}
        >
          <DependencyCreateForm
            definitions={dependencyDefinitions}
            githubCredentials={credentials.github}
            onSubmit={onDeployDependency}
            onCancel={() => setDependencyCreateOpen(false)}
            requireGitOpsCredential
            submitLabel="Deploy dependency"
          />
        </Modal>
      )}
    </div>
  );
}

import React, { useEffect, useMemo, useRef, useState } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter, useLocation, useNavigate } from "react-router-dom";
import {
  filterNavItems,
  filterNavSections,
  shouldShowSkeleton,
  defaultDeployForm,
  buildProjectPayload,
  monorepoComponentsFromAnalysis,
  applicationSpecAnalysis,
  deployFormFromApplicationSpec,
  monorepoComponentDomainOverrides,
  buildMonorepoApplicationPayload,
  normalizeDependencyDefinition,
  imageName,
  profileFromBasalt,
  trimLiveLog,
  titleFor,
  subtitleFor,
  parseKeyValues,
  parseTaints,
  parseCSV,
  parsePermissionSubjects,
  parseServicePorts,
  localDateTimeToRFC3339,
  slugify,
  trimSlash,
  browserRedirectURI,
  randomString,
  codeChallenge,
} from "./utils/index";
import {
  makeAPI,
  consumeTextStream,
  publicJSON,
  finishLogin,
} from "./api/index";
import {
  SidebarNavGroup,
  PageHeading,
  SkeletonPage,
  RuntimeTable,
  CredentialManager,
  Button,
  Input,
} from "./components/index";
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
  ExternalLink,
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
  LogOut,
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
import "./style.css";
import { I18nProvider, useI18n, t } from "./i18n/index";
import { LanguageSwitcher } from "./i18n/LanguageSwitcher";
const API = "/v1/api";
const tokenKey = "beancs.accessToken";
const hasOAuthCallback = () =>
  new URLSearchParams(window.location.search).has("code");
const emptyRuntime = {
  namespaces: [],
  pods: [],
  nodes: [],
  deployments: [],
  services: [],
  ingresses: [],
};
const emptyStorage = {
  persistent_volume_claims: [],
  persistent_volumes: [],
  storage_classes: [],
};
const viewPaths = {
  dashboard: "/",
  applications: "/workloads/applications",
  projects: "/workloads/projects",
  deploy: "/workloads/deploy",
  dependencies: "/workloads/dependencies",
  progress: "/workloads/progress",
  deployments: "/workloads/deployments",
  pods: "/workloads/pods",
  services: "/workloads/services",
  ingresses: "/workloads/ingresses",
  workloadImage: "/workloads/images",
  nodes: "/infrastructure/nodes",
  namespaces: "/infrastructure/namespaces",
  networking: "/infrastructure/networking",
  storage: "/infrastructure/storage",
  github: "/integrations/github",
  cloudflare: "/integrations/cloudflare",
  domains: "/integrations/domains",
  registries: "/integrations/registries",
  apiKeys: "/security/api-keys",
  secrets: "/security/secrets",
  accessControl: "/security/access-control",
  alerts: "/observability/alerts",
  events: "/observability/events",
  logs: "/observability/logs",
  metrics: "/observability/metrics",
  settings: "/settings",
};
const viewsByPath = Object.fromEntries(
  Object.entries(viewPaths).map(([view, path]) => [path, view]),
);

function pathForView(view) {
  return viewPaths[view] || viewPaths.dashboard;
}

function viewForPath(pathname) {
  const normalized =
    pathname.length > 1 ? pathname.replace(/\/+$/, "") : pathname;
  return viewsByPath[normalized] || "dashboard";
}

function isKnownViewPath(pathname) {
  const normalized =
    pathname.length > 1 ? pathname.replace(/\/+$/, "") : pathname;
  return Boolean(viewsByPath[normalized]);
}

function progressPath(projectID = "", processID = "") {
  const params = new URLSearchParams();
  if (projectID) params.set("project", String(projectID));
  if (processID) params.set("process", String(processID));
  const query = params.toString();
  return `${pathForView("progress")}${query ? `?${query}` : ""}`;
}
const navOverview = {
  id: "dashboard",
  label: "Overview",
  icon: LayoutDashboard,
};
const navSections = [
  {
    id: "workloads",
    label: "Workloads",
    items: [
      {
        id: "applications",
        label: "Applications",
        icon: Layers3,
      },
      {
        id: "projects",
        label: "Projects",
        icon: Boxes,
      },
      {
        id: "deploy",
        label: "Deploy",
        icon: Rocket,
      },
      {
        id: "dependencies",
        label: "Dependencies",
        icon: Database,
      },
      {
        id: "progress",
        label: "Progress",
        icon: LoaderCircle,
      },
      {
        id: "deployments",
        label: "Deployments",
        icon: Box,
      },
      {
        id: "pods",
        label: "Pods",
        icon: Layers3,
      },
      {
        id: "services",
        label: "Services",
        icon: Database,
      },
      {
        id: "ingresses",
        label: "Ingresses",
        icon: Network,
      },
      {
        id: "workloadImage",
        label: "Image",
        icon: ImageIcon,
      },
    ],
  },
  {
    id: "infrastructure",
    label: "Infrastructure",
    items: [
      {
        id: "nodes",
        label: "Nodes",
        icon: Server,
      },
      {
        id: "namespaces",
        label: "Namespaces",
        icon: Layers3,
      },
      {
        id: "networking",
        label: "Networking",
        icon: Network,
      },
      {
        id: "storage",
        label: "Storage",
        icon: HardDrive,
      },
    ],
  },
  {
    id: "integrations",
    label: "Integrations",
    items: [
      {
        id: "github",
        label: "GitHub",
        icon: Github,
      },
      {
        id: "cloudflare",
        label: "Cloudflare",
        icon: Cloud,
      },
      {
        id: "domains",
        label: "Domains",
        icon: Globe2,
      },
      {
        id: "registries",
        label: "Image Registry",
        icon: Package,
      },
    ],
  },
  {
    id: "security",
    label: "Security",
    items: [
      {
        id: "apiKeys",
        label: "API Keys",
        icon: KeyRound,
      },
      {
        id: "secrets",
        label: "Secrets",
        icon: Lock,
      },
      {
        id: "accessControl",
        label: "Access Control",
        icon: ShieldCheck,
      },
    ],
  },
  {
    id: "observability",
    label: "Observability",
    items: [
      {
        id: "alerts",
        label: "Alerts",
        icon: Bell,
      },
      {
        id: "events",
        label: "Events",
        icon: ScrollText,
      },
      {
        id: "logs",
        label: "Logs",
        icon: FileText,
      },
      {
        id: "metrics",
        label: "Metrics",
        icon: LineChart,
      },
    ],
  },
  {
    id: "settings",
    label: "Settings",
    items: [
      {
        id: "settings",
        label: "Settings",
        icon: Settings,
      },
    ],
  },
];
import DeployView from "./views/DeployView";
import ProgressView from "./views/ProgressView";
import DashboardView from "./views/DashboardView";
import AlertsView from "./views/AlertsView";
import EventsView from "./views/EventsView";
import MetricsView from "./views/MetricsView";
import LogsView from "./views/LogsView";
import DeploymentsView from "./views/DeploymentsView";
import DependenciesView from "./views/DependenciesView";
import ProjectsView from "./views/ProjectsView";
import ApplicationsView from "./views/ApplicationsView";
import ProjectTrackingModal from "./views/ProjectTrackingModal";
import DeleteProjectModal from "./views/DeleteProjectModal";
import DeleteApplicationModal from "./views/DeleteApplicationModal";
import ContainerRegistriesView from "./views/ContainerRegistriesView";
import WorkloadImageView from "./views/WorkloadImageView";
import ComingSoonView from "./views/ComingSoonView";
import SettingsView from "./views/SettingsView";
import APIKeysView from "./views/APIKeysView";
import GitHubView from "./views/GitHubView";
import CloudflareView from "./views/CloudflareView";
import DomainsView from "./views/DomainsView";
import NetworkingView from "./views/NetworkingView";
import StorageView from "./views/StorageView";
import RuntimeDetailDrawer from "./views/RuntimeDetailDrawer";
import NodeDetailView from "./views/NodeDetailView";
import NamespaceDetailView from "./views/NamespaceDetailView";
import ProjectModal from "./views/ProjectModal";
function App() {
  const { t } = useI18n();
  const navigate = useNavigate();
  const routeLocation = useLocation();
  const [config, setConfig] = useState(null);
  const [token, setToken] = useState(localStorage.getItem(tokenKey) || "");
  const [loginPhase, setLoginPhase] = useState(() =>
    hasOAuthCallback() ? "completing" : "idle",
  );
  const [basaltProfile, setBasaltProfile] = useState(null);
  const view = viewForPath(routeLocation.pathname);
  const [notice, setNotice] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [reposLoading, setReposLoading] = useState(false);
  const [runtime, setRuntime] = useState(emptyRuntime);
  const [dashboard, setDashboard] = useState(null);
  const [network, setNetwork] = useState(null);
  const [storage, setStorage] = useState(emptyStorage);
  const [projects, setProjects] = useState([]);
  const [applications, setApplications] = useState([]);
  const [dependencyDefinitions, setDependencyDefinitions] = useState([]);
  const [reusableDependencies, setReusableDependencies] = useState([]);
  const [credentials, setCredentials] = useState({
    github: [],
    cloudflare: [],
    basaltpass: [],
  });
  const [apiKeys, setAPIKeys] = useState([]);
  const [apiKeyScopeCatalog, setAPIKeyScopeCatalog] = useState({
    scopes: [],
    presets: [],
  });
  const [registryPresets, setRegistryPresets] = useState([]);
  const [containerRegistries, setContainerRegistries] = useState([]);
  const [containerImages, setContainerImages] = useState([]);
  const [appVersion, setAppVersion] = useState("");
  const [createdAPIKey, setCreatedAPIKey] = useState(null);
  const [domains, setDomains] = useState([]);
  const [repos, setRepos] = useState([]);
  const [reposByCredential, setReposByCredential] = useState({});
  const [repoFilters, setRepoFilters] = useState({});
  const [selectedCloudflareID, setSelectedCloudflareID] = useState("");
  const [selectedCloudflareZoneID, setSelectedCloudflareZoneID] = useState("");
  const [dnsRecords, setDNSRecords] = useState([]);
  const [editingDNSRecord, setEditingDNSRecord] = useState(null);
  const [runtimeDetail, setRuntimeDetail] = useState(null);
  const [runtimeLogs, setRuntimeLogs] = useState("");
  const [selectedCredential, setSelectedCredential] = useState("");
  const [selectedRepo, setSelectedRepo] = useState("");
  const [analysis, setAnalysis] = useState(null);
  const [deployForm, setDeployForm] = useState(defaultDeployForm());
  const [editingProject, setEditingProject] = useState(null);
  const [deletingProject, setDeletingProject] = useState(null);
  const [deletingApplication, setDeletingApplication] = useState(null);
  const [trackingProject, setTrackingProject] = useState(null);
  const [projectTracking, setProjectTracking] = useState(null);
  const [trackingLoading, setTrackingLoading] = useState(false);
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [sidebarQuery, setSidebarQuery] = useState("");
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const userMenuRef = useRef(null);
  const [activeProgressProjectID, setActiveProgressProjectID] = useState("");
  const [activeProcessID, setActiveProcessID] = useState("");
  const [processRecords, setProcessRecords] = useState([]);
  const [projectProgress, setProjectProgress] = useState(null);
  const [installProgress, setInstallProgress] = useState(null);
  const [projectLogFollow, setProjectLogFollow] = useState(false);
  const [projectLiveLogs, setProjectLiveLogs] = useState("");
  const [projectLogStatus, setProjectLogStatus] = useState("");
  const [runtimeLogFollow, setRuntimeLogFollow] = useState(false);
  const [runtimeLogStatus, setRuntimeLogStatus] = useState("");
  const [runtimeLogContainer, setRuntimeLogContainer] = useState("");
  const [runtimeLogTail, setRuntimeLogTail] = useState(200);
  const [runtimeLogLoaded, setRuntimeLogLoaded] = useState(false);
  const [nodeJoinCommand, setNodeJoinCommand] = useState(null);
  const [nodeHealth, setNodeHealth] = useState(null);
  const projectLogController = useRef(null);
  const runtimeLogController = useRef(null);
  const workspaceLoadingRef = useRef(false);
  const dashboardLoadingRef = useRef(false);
  const networkLoadingRef = useRef(false);
  const storageLoadingRef = useRef(false);
  const progressLoadingRef = useRef(false);
  const nodeDetailLoadingRef = useRef(false);
  const registriesLoadingRef = useRef(false);
  const api = useMemo(() => makeAPI(token, logout), [token]);
  const userProfile = useMemo(
    () => profileFromBasalt(basaltProfile, token),
    [basaltProfile, token],
  );
  const configuredNavSections = useMemo(() => {
    const argocdURL = String(config?.argocd_url || "").trim();
    if (!argocdURL) return navSections;
    return navSections.map((section) =>
      section.id === "integrations"
        ? {
            ...section,
            items: [
              ...section.items,
              {
                id: "argocd",
                label: "Argo CD",
                icon: ExternalLink,
                externalUrl: argocdURL,
              },
            ],
          }
        : section,
    );
  }, [config?.argocd_url]);
  const filteredNavSections = useMemo(
    () => filterNavSections(configuredNavSections, sidebarQuery),
    [configuredNavSections, sidebarQuery],
  );
  const filteredOverview = useMemo(
    () => filterNavItems([navOverview], sidebarQuery),
    [sidebarQuery],
  );

  useEffect(() => {
    if (!["progress", "logs"].includes(view)) return;
    const params = new URLSearchParams(routeLocation.search);
    const projectID = params.get("project") || "";
    const processID = params.get("process") || "";
    if (projectID !== activeProgressProjectID) {
      setActiveProgressProjectID(projectID);
      setProjectProgress(null);
    }
    if (processID !== activeProcessID) setActiveProcessID(processID);
  }, [view, routeLocation.search]);

  useEffect(() => {
    const query = new URLSearchParams(routeLocation.search);
    if (
      !token ||
      isKnownViewPath(routeLocation.pathname) ||
      query.has("code") ||
      query.get("github_app") === "connected" ||
      query.has("cloudflare_app")
    )
      return;
    navigate(pathForView("dashboard"), { replace: true });
  }, [token, routeLocation.pathname, routeLocation.search, navigate]);

  useEffect(() => {
    if (!userMenuOpen) return undefined;
    function handlePointerDown(event) {
      if (userMenuRef.current?.contains(event.target)) return;
      setUserMenuOpen(false);
    }
    function handleKeyDown(event) {
      if (event.key === "Escape") setUserMenuOpen(false);
    }
    document.addEventListener("mousedown", handlePointerDown);
    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("mousedown", handlePointerDown);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [userMenuOpen]);
  useEffect(() => {
    boot();
  }, []);
  useEffect(() => {
    if (token) {
      loadWorkspace();
      loadUserProfile();
    }
  }, [token]);
  useEffect(() => {
    if (!token || !["dashboard", "alerts", "events", "metrics"].includes(view))
      return;
    loadDashboard();
    const timer = setInterval(() => {
      if (!document.hidden) loadDashboard();
    }, 15000);
    return () => clearInterval(timer);
  }, [token, view]);
  useEffect(() => {
    if (!token || view !== "networking") return;
    loadNetwork();
    const timer = setInterval(() => {
      if (!document.hidden) loadNetwork();
    }, 30000);
    return () => clearInterval(timer);
  }, [token, view]);
  useEffect(() => {
    if (!token || view !== "storage") return;
    loadStorage();
    const timer = setInterval(() => {
      if (!document.hidden) loadStorage();
    }, 30000);
    return () => clearInterval(timer);
  }, [token, view]);
  useEffect(() => {
    if (!token || view !== "nodes") return;
    loadNodeJoinCommand("agent");
  }, [token, view]);
  useEffect(() => {
    if (!token || !["progress", "logs"].includes(view)) return;
    loadProcesses();
    if (view === "progress" && !activeProgressProjectID) {
      setProjectProgress(null);
    } else {
      loadProjectProgress();
    }
    const timer = setInterval(() => {
      if (!document.hidden) {
        loadProcesses();
        if (activeProgressProjectID) loadProjectProgress();
      }
    }, 5000);
    return () => clearInterval(timer);
  }, [
    token,
    view,
    activeProgressProjectID,
    activeProcessID,
    projects.length,
    projectLogFollow,
  ]);
  useEffect(() => {
    if (!token || runtimeDetail?.kind !== "node") return;
    const nodeName =
      runtimeDetail.row?.summary?.name || runtimeDetail.row?.name;
    if (!nodeName) return;
    const timer = setInterval(() => {
      if (!document.hidden)
        loadNodeDetail(
          {
            name: nodeName,
          },
          false,
        );
    }, 15000);
    return () => clearInterval(timer);
  }, [
    token,
    runtimeDetail?.kind,
    runtimeDetail?.row?.summary?.name,
    runtimeDetail?.row?.name,
  ]);
  useEffect(() => {
    if (!token || view !== "settings") return;
    publicJSON(`${API}/version`)
      .then((d) => setAppVersion(d.version || ""))
      .catch(() => setAppVersion(""));
  }, [token, view]);
  useEffect(() => {
    if (!token || view !== "apiKeys") return;
    loadAPIKeys();
  }, [token, view]);
  useEffect(() => {
    if (!token || !["deploy", "registries", "workloadImage"].includes(view))
      return;
    loadRegistriesPage();
    const timer = setInterval(() => {
      if (!document.hidden) loadContainerImages();
    }, 120000);
    return () => clearInterval(timer);
  }, [token, view]);
  useEffect(() => {
    return () => {
      projectLogController.current?.abort();
      runtimeLogController.current?.abort();
    };
  }, []);
  async function boot() {
    try {
      const cfg = await publicJSON(`${API}/ui/config`);
      setConfig(cfg);
      if (location.search.includes("code=")) {
        setLoginPhase("completing");
        const accessToken = await finishLogin(cfg);
        setLoginPhase("authenticated");
        localStorage.setItem(tokenKey, accessToken);
        setToken(accessToken);
        navigate(pathForView("dashboard"), { replace: true });
      } else if (location.search.includes("github_app=connected")) {
        setLoginPhase("idle");
        navigate(pathForView("github"), { replace: true });
        setNotice(t("GitHub App connected."));
      } else if (location.search.includes("cloudflare_app=connected")) {
        setLoginPhase("idle");
        navigate(pathForView("cloudflare"), { replace: true });
        setNotice(t("Cloudflare account connected."));
      } else if (location.search.includes("cloudflare_app=error")) {
        const params = new URLSearchParams(location.search);
        setLoginPhase("idle");
        navigate(pathForView("cloudflare"), { replace: true });
        setError(params.get("message") || t("Cloudflare authorization failed."));
      } else {
        setLoginPhase("idle");
      }
    } catch (err) {
      setLoginPhase("idle");
      setError(err.message);
    }
  }
  async function loadWorkspace() {
    if (workspaceLoadingRef.current) return;
    workspaceLoadingRef.current = true;
    setLoading(true);
    setError("");
    try {
      const [
        runtimeData,
        projectData,
        applicationData,
        dependencyDefinitionData,
        dependencyData,
        apiKeyData,
        githubData,
        cloudflareData,
        domainsData,
        basaltpassData,
        processData,
      ] = await Promise.all([
        api.get("/runtime/overview"),
        api.get("/projects"),
        api.get("/applications"),
        api.get("/dependency-definitions"),
        api.get("/dependencies"),
        api.get("/api-keys"),
        api.get("/credentials/github/"),
        api.get("/credentials/cloudflare/"),
        api.get("/credentials/cloudflare/domains"),
        api.get("/credentials/basaltpass/"),
        api.get("/processes"),
      ]);
      setRuntime(runtimeData.data || emptyRuntime);
      setProjects(projectData.data || []);
      setApplications(applicationData.data || []);
      const definitionSummaries = dependencyDefinitionData.data || [];
      const definitions = await Promise.all(
        definitionSummaries.map((definition) =>
          api.get(`/dependency-definitions/${definition.name}`),
        ),
      );
      setDependencyDefinitions(definitions.map(normalizeDependencyDefinition));
      const dependencyRows = dependencyData.data || [];
      const dependenciesWithCredentials = await Promise.all(
        dependencyRows.map(async (dependency) => {
          try {
            const credentialData = await api.get(
              `/dependencies/${dependency.id}/credentials`,
            );
            return {
              ...dependency,
              credentials: credentialData.data || [],
            };
          } catch {
            return {
              ...dependency,
              credentials: [],
            };
          }
        }),
      );
      setReusableDependencies(dependenciesWithCredentials);
      setAPIKeys(apiKeyData.data || []);
      setCredentials({
        github: githubData.data || [],
        cloudflare: cloudflareData.data || [],
        basaltpass: basaltpassData.data || [],
      });
      setProcessRecords(processData.data || []);
      setDomains(domainsData.data || []);
    } catch (err) {
      setError(err.message);
    } finally {
      workspaceLoadingRef.current = false;
      setLoading(false);
    }
  }
  async function loadProcesses() {
    try {
      const data = await api.get("/processes");
      const rows = data.data || [];
      setProcessRecords(rows);
      if (
        activeProcessID &&
        !rows.some((row) => String(row.id) === String(activeProcessID))
      ) {
        setActiveProcessID("");
      }
    } catch (err) {
      setError(err.message);
    }
  }
  async function loadUserProfile() {
    try {
      const data = await api.get("/me");
      setBasaltProfile(data || null);
    } catch {
      setBasaltProfile(null);
    }
  }
  async function loadDashboard() {
    if (dashboardLoadingRef.current) return;
    dashboardLoadingRef.current = true;
    try {
      const data = await api.get("/runtime/dashboard");
      setDashboard(data.data || null);
    } catch (err) {
      setError(err.message);
    } finally {
      dashboardLoadingRef.current = false;
    }
  }
  async function loadNetwork() {
    if (networkLoadingRef.current) return;
    networkLoadingRef.current = true;
    try {
      const data = await api.get("/runtime/network/overview");
      setNetwork(data.data || null);
    } catch (err) {
      setError(err.message);
    } finally {
      networkLoadingRef.current = false;
    }
  }
  async function loadStorage() {
    if (storageLoadingRef.current) return;
    storageLoadingRef.current = true;
    try {
      const data = await api.get("/runtime/storage/overview");
      setStorage(data.data || emptyStorage);
    } catch (err) {
      setError(err.message);
    } finally {
      storageLoadingRef.current = false;
    }
  }
  async function startLogin() {
    if (!config || loginPhase !== "idle") return;
    setLoginPhase("redirecting");
    setError("");
    const verifier = randomString(64);
    const challenge = await codeChallenge(verifier);
    const authState = randomString(32);
    const nonce = randomString(32);
    sessionStorage.setItem("beancs.pkceVerifier", verifier);
    sessionStorage.setItem("beancs.oauthState", authState);
    sessionStorage.setItem("beancs.oauthNonce", nonce);
    const params = new URLSearchParams({
      response_type: "code",
      client_id: config.client_id,
      redirect_uri: browserRedirectURI(),
      scope: "openid profile email",
      state: authState,
      nonce,
      code_challenge: challenge,
      code_challenge_method: "S256",
    });
    location.href =
      trimSlash(config.auth_url) + "/oauth/authorize?" + params.toString();
  }
  function logout() {
    localStorage.removeItem(tokenKey);
    setToken("");
    setLoginPhase("idle");
    setBasaltProfile(null);
    setRuntime(emptyRuntime);
    setProjects([]);
    setApplications([]);
    setDependencyDefinitions([]);
    setReusableDependencies([]);
    navigate(pathForView("dashboard"), { replace: true });
  }
  function openProgress(projectID = "", processID = "", replace = false) {
    const nextProjectID = String(projectID || "");
    const nextProcessID = String(processID || "");
    setActiveProgressProjectID(nextProjectID);
    setActiveProcessID(nextProcessID);
    setProjectProgress(null);
    navigate(progressPath(nextProjectID, nextProcessID), { replace });
  }
  async function createDependency(event) {
    event.preventDefault();
    setError("");
    setNotice("");
    const form = event.currentTarget;
    const data = new FormData(form);
    const type = String(data.get("type") || "").trim();
    const deployMethod = String(data.get("deploy_method") || "helm").trim();
    const external =
      data.get("external") === "true" || deployMethod === "external";
    const controlled = data.get("controlled") === "on";
    let config = {};
    if (external) {
      config = {
        host: String(data.get("host") || "").trim(),
        port: String(data.get("port") || "").trim(),
      };
      if (type === "rabbitmq") {
        config.management_port = String(
          data.get("management_port") || "",
        ).trim();
      }
      if (controlled) {
        config.admin_username = String(
          data.get("admin_username") || "",
        ).trim();
        config.admin_password = String(data.get("admin_password") || "");
      }
    } else {
      try {
        config = JSON.parse(String(data.get("config_json") || "{}"));
      } catch {
        config = {};
      }
    }
    const body = {
      name: String(data.get("name") || "").trim(),
      display_name: String(data.get("display_name") || "").trim(),
      application_name: String(data.get("application_name") || "").trim(),
      namespace: String(data.get("namespace") || "").trim(),
      type,
      version: String(data.get("version") || "").trim(),
      deploy_method: deployMethod,
      external,
      controlled: external ? controlled : true,
      shared: data.get("shared") === "on" || external,
      config,
    };
    const githubCredentialID = Number(data.get("github_credential_id") || 0);
    if (!external && githubCredentialID) {
      body.github_credential_id = githubCredentialID;
    }
    Object.keys(body).forEach((key) => {
      if (body[key] === "") delete body[key];
    });
    Object.keys(body.config).forEach((key) => {
      if (body.config[key] === "") delete body.config[key];
    });
    try {
      await api.post("/dependencies", body);
      form.reset();
      setNotice(t("Dependency added."));
      await loadWorkspace();
      return true;
    } catch (err) {
      setError(err.message);
      return false;
    }
  }
  async function deployDependency(event) {
    const created = await createDependency(event);
    if (created) setNotice(t("Dependency deployment submitted."));
    return created;
  }
  async function createDependencyCredential(dependencyID, event) {
    event.preventDefault();
    setError("");
    setNotice("");
    const form = event.currentTarget;
    const data = new FormData(form);
    const body = {
      name: String(data.get("name") || "").trim(),
      description: String(data.get("description") || "").trim(),
      config: {
        database: String(data.get("database") || "").trim(),
        username: String(data.get("username") || "").trim(),
        password: String(data.get("password") || ""),
      },
    };
    Object.keys(body).forEach((key) => {
      if (body[key] === "") delete body[key];
    });
    Object.keys(body.config).forEach((key) => {
      if (body.config[key] === "") delete body.config[key];
    });
    try {
      await api.post(`/dependencies/${dependencyID}/credentials`, body);
      form.reset();
      setNotice(t("Credential added."));
      await loadWorkspace();
      return true;
    } catch (err) {
      setError(err.message);
      return false;
    }
  }
  async function connectGitHubApp(event, gitopsRepo) {
    event?.preventDefault();
    setError("");
    const body = {};
    if (gitopsRepo) body.gitops_repo = gitopsRepo.trim();
    const data = await api.post("/credentials/github/app/start", body);
    location.href = data.install_url;
  }
  async function connectCloudflareApp(event) {
    event?.preventDefault();
    setError("");
    const data = await api.post("/credentials/cloudflare/app/start", {});
    location.href = data.auth_url;
  }
  async function updateGitHubCredential(id, updates) {
    try {
      await api.patch(`/credentials/github/${id}`, updates);
      await loadWorkspace();
      setNotice(t("GitHub credential updated."));
    } catch (err) {
      setError(err.message);
    }
  }
  async function createCredential(kind, event) {
    event.preventDefault();
    const body = Object.fromEntries(
      new FormData(event.currentTarget).entries(),
    );
    Object.keys(body).forEach((key) => {
      if (typeof body[key] === "string") body[key] = body[key].trim();
      if (body[key] === "") delete body[key];
    });
    try {
      await api.post(`/credentials/${kind}/`, body);
      event.currentTarget.reset();
      await loadWorkspace();
      return true;
    } catch (err) {
      setError(err.message);
      return false;
    }
  }
  async function deleteCredential(kind, id) {
    if (!confirm(t("Delete this {kind} credential?", { kind }))) return;
    try {
      await api.delete(`/credentials/${kind}/${id}`);
      await loadWorkspace();
    } catch (err) {
      setError(err.message);
    }
  }
  async function refreshCloudflareDomains(id) {
    if (!id) return;
    setError("");
    try {
      const data = await api.post(`/credentials/cloudflare/${id}/domains/refresh`, {});
      const refreshed = data.data || [];
      setDomains((current) => [
        ...current.filter(
          (domain) => String(domain.credential_id) !== String(id),
        ),
        ...refreshed,
      ]);
      setNotice(t("Cloudflare domains refreshed."));
    } catch (err) {
      setError(err.message);
    }
  }
  async function createAPIKey(event) {
    event.preventDefault();
    setError("");
    setNotice("");
    const form = event.currentTarget;
    const data = new FormData(form);
    const scopes = data
      .getAll("scopes")
      .map((scope) => String(scope || "").trim())
      .filter(Boolean);
    const body = {
      name: String(data.get("name") || "").trim(),
      preset: String(data.get("preset") || "").trim(),
      scopes,
      expires_at: localDateTimeToRFC3339(data.get("expires_at")),
    };
    try {
      const out = await api.post("/api-keys", body);
      setCreatedAPIKey(out);
      form.reset();
      await loadAPIKeys();
      return true;
    } catch (err) {
      setError(err.message);
      return false;
    }
  }
  async function loadAPIKeys() {
    const [keyData, scopeData] = await Promise.all([
      api.get("/api-keys"),
      api.get("/api-keys/scopes"),
    ]);
    setAPIKeys(keyData.data || []);
    setAPIKeyScopeCatalog(
      scopeData || {
        scopes: [],
        presets: [],
      },
    );
  }
  async function loadRegistriesPage() {
    if (registriesLoadingRef.current) return;
    registriesLoadingRef.current = true;
    setError("");
    try {
      const [presetData, regData, imgData] = await Promise.all([
        api.get("/container-registries/presets"),
        api.get("/container-registries"),
        api.get("/container-images"),
      ]);
      setRegistryPresets(presetData.data || []);
      setContainerRegistries(regData.data || []);
      setContainerImages(imgData.data || []);
    } catch (err) {
      setError(err.message);
    } finally {
      registriesLoadingRef.current = false;
    }
  }
  async function loadContainerImages() {
    try {
      const imgData = await api.get("/container-images");
      setContainerImages(imgData.data || []);
    } catch (err) {
      setError(err.message);
    }
  }
  async function createContainerRegistry(event) {
    event.preventDefault();
    setError("");
    const form = event.currentTarget;
    const data = new FormData(form);
    const kind = String(data.get("kind") || "").trim();
    const host = String(data.get("host") || "").trim();
    const body = {
      kind,
      name: String(data.get("name") || "").trim(),
      host,
      username: String(data.get("username") || "").trim(),
      password: String(data.get("password") || ""),
      insecure_tls: data.get("insecure_tls") === "on",
    };
    try {
      await api.post("/container-registries", body);
      form.reset();
      await loadRegistriesPage();
    } catch (err) {
      setError(err.message);
    }
  }
  async function deleteContainerRegistry(row) {
    if (
      !confirm(
        t('Delete image registry "{name}"? Linked image tracking will also be removed.', {
          name: row.name,
        }),
      )
    )
      return;
    try {
      await api.delete(`/container-registries/${row.id}`);
      setNotice(t("Image registry deleted."));
      await loadRegistriesPage();
    } catch (err) {
      setError(err.message);
    }
  }
  async function createTrackedImage(event) {
    event.preventDefault();
    setError("");
    const form = event.currentTarget;
    const data = new FormData(form);
    const body = {
      registry_id: Number(data.get("registry_id")),
      repository: String(data.get("repository") || "").trim(),
    };
    if (!body.registry_id || !body.repository) {
      setError(t("Select a registry and enter the repository path."));
      return;
    }
    try {
      await api.post("/container-images", body);
      form.reset();
      await loadRegistriesPage();
    } catch (err) {
      setError(err.message);
    }
  }
  async function createTrackedImageFromDeploy(body) {
    const created = await api.post("/container-images", body);
    await loadRegistriesPage();
    return created;
  }
  async function refreshTrackedImage(id) {
    setError("");
    try {
      await api.post(`/container-images/${id}/refresh`, {});
      await loadContainerImages();
    } catch (err) {
      setError(err.message);
    }
  }
  async function deleteTrackedImage(row) {
    if (
      !confirm(
        t('Remove "{repository}" from the list?', { repository: row.repository }),
      )
    )
      return;
    try {
      await api.delete(`/container-images/${row.id}`);
      await loadContainerImages();
    } catch (err) {
      setError(err.message);
    }
  }
  async function syncAllTrackedImages() {
    setError("");
    let list = [];
    try {
      const imgData = await api.get("/container-images");
      list = imgData.data || [];
    } catch (err) {
      setError(err.message);
      return;
    }
    for (const im of list) {
      try {
        await api.post(`/container-images/${im.id}/refresh`, {});
      } catch (err) {
        setError(err.message);
        break;
      }
    }
    await loadContainerImages();
  }
  async function revokeAPIKey(key) {
    if (
      !confirm(
        t("Revoke API key {name}? Existing clients using it will stop working.", {
          name: key.name,
        }),
      )
    )
      return;
    try {
      await api.delete(`/api-keys/${key.id}`);
      setNotice(`${key.name} revoked.`);
      await loadAPIKeys();
    } catch (err) {
      setError(err.message);
    }
  }
  async function loadRepos(credentialID = selectedCredential) {
    if (!credentialID) return;
    setSelectedCredential(String(credentialID));
    setAnalysis(null);
    setRepos([]);
    setReposLoading(true);
    try {
      const data = await api.get(
        `/credentials/github/${credentialID}/repositories`,
      );
      setRepos(data.data || []);
      setReposByCredential((current) => ({
        ...current,
        [credentialID]: data.data || [],
      }));
    } catch (err) {
      setError(err.message);
    } finally {
      setReposLoading(false);
    }
  }
  async function loadDNSRecords(
    credentialID = selectedCloudflareID,
    zoneID = selectedCloudflareZoneID,
  ) {
    if (!credentialID) return;
    setSelectedCloudflareID(String(credentialID));
    setSelectedCloudflareZoneID(String(zoneID || ""));
    setLoading(true);
    setError("");
    try {
      const qs = zoneID ? `?zone_id=${encodeURIComponent(zoneID)}` : "";
      const data = await api.get(
        `/credentials/cloudflare/${credentialID}/dns-records${qs}`,
      );
      setDNSRecords(data.data || []);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }
  async function saveDNSRecord(event) {
    event.preventDefault();
    if (!selectedCloudflareID) return;
    const body = Object.fromEntries(
      new FormData(event.currentTarget).entries(),
    );
    body.ttl = Number(body.ttl || 1);
    body.proxied = Boolean(body.proxied);
    try {
      const qs = selectedCloudflareZoneID
        ? `?zone_id=${encodeURIComponent(selectedCloudflareZoneID)}`
        : "";
      if (editingDNSRecord?.id) {
        await api.put(
          `/credentials/cloudflare/${selectedCloudflareID}/dns-records/${editingDNSRecord.id}${qs}`,
          body,
        );
      } else {
        await api.post(
          `/credentials/cloudflare/${selectedCloudflareID}/dns-records${qs}`,
          body,
        );
      }
      event.currentTarget.reset();
      setEditingDNSRecord(null);
      await loadDNSRecords(selectedCloudflareID);
    } catch (err) {
      setError(err.message);
    }
  }
  async function deleteDNSRecord(record) {
    if (
      !selectedCloudflareID ||
      !confirm(t("Delete DNS record {name}?", { name: record.name }))
    )
      return;
    try {
      const qs = selectedCloudflareZoneID
        ? `?zone_id=${encodeURIComponent(selectedCloudflareZoneID)}`
        : "";
      await api.delete(
        `/credentials/cloudflare/${selectedCloudflareID}/dns-records/${record.id}${qs}`,
      );
      await loadDNSRecords(selectedCloudflareID);
    } catch (err) {
      setError(err.message);
    }
  }
  async function createNamespace(event) {
    event.preventDefault();
    const body = Object.fromEntries(
      new FormData(event.currentTarget).entries(),
    );
    body.labels = parseKeyValues(body.labels);
    delete body.labels_raw;
    try {
      await api.post("/runtime/namespaces", body);
      event.currentTarget.reset();
      await loadWorkspace();
    } catch (err) {
      setError(err.message);
    }
  }
  async function patchNamespaceLabels(namespace, labelsText) {
    try {
      await api.patch(`/runtime/namespaces/${namespace}`, {
        labels: parseKeyValues(labelsText),
      });
      await loadWorkspace();
    } catch (err) {
      setError(err.message);
    }
  }
  async function loadNamespaceDetail(namespace) {
    setRuntimeDetail({
      kind: "namespace-detail",
      row: {
        name: namespace,
      },
      loading: true,
    });
    try {
      const data = await api.get(
        `/runtime/namespaces/${encodeURIComponent(namespace)}`,
      );
      setRuntimeDetail({
        kind: "namespace-detail",
        row: data.data || {
          name: namespace,
        },
        loading: false,
      });
    } catch (err) {
      setRuntimeDetail({
        kind: "namespace-detail",
        row: {
          name: namespace,
        },
        loading: false,
        error: err.message,
      });
    }
  }
  async function saveResourceQuota(namespace, event) {
    event.preventDefault();
    const body = Object.fromEntries(
      new FormData(event.currentTarget).entries(),
    );
    body.hard = parseKeyValues(body.hard);
    try {
      await api.put(
        `/runtime/namespaces/${encodeURIComponent(namespace)}/resource-quotas`,
        body,
      );
      await loadNamespaceDetail(namespace);
      await loadWorkspace();
    } catch (err) {
      setError(err.message);
    }
  }
  async function deleteResourceQuota(namespace, name) {
    if (!confirm(t("Delete ResourceQuota {name}?", { name }))) return;
    try {
      await api.delete(
        `/runtime/namespaces/${encodeURIComponent(namespace)}/resource-quotas/${encodeURIComponent(name)}`,
      );
      await loadNamespaceDetail(namespace);
    } catch (err) {
      setError(err.message);
    }
  }
  async function saveLimitRange(namespace, event) {
    event.preventDefault();
    const body = Object.fromEntries(
      new FormData(event.currentTarget).entries(),
    );
    body.default = parseKeyValues(body.default);
    body.default_request = parseKeyValues(body.default_request);
    body.min = parseKeyValues(body.min);
    body.max = parseKeyValues(body.max);
    try {
      await api.put(
        `/runtime/namespaces/${encodeURIComponent(namespace)}/limit-ranges`,
        body,
      );
      await loadNamespaceDetail(namespace);
    } catch (err) {
      setError(err.message);
    }
  }
  async function deleteLimitRange(namespace, name) {
    if (!confirm(t("Delete LimitRange {name}?", { name }))) return;
    try {
      await api.delete(
        `/runtime/namespaces/${encodeURIComponent(namespace)}/limit-ranges/${encodeURIComponent(name)}`,
      );
      await loadNamespaceDetail(namespace);
    } catch (err) {
      setError(err.message);
    }
  }
  async function saveNamespacePermission(namespace, event) {
    event.preventDefault();
    const body = Object.fromEntries(
      new FormData(event.currentTarget).entries(),
    );
    body.verbs = parseCSV(body.verbs);
    body.resources = parseCSV(body.resources);
    body.api_groups = parseCSV(body.api_groups);
    body.subjects = parsePermissionSubjects(body.subjects, namespace);
    try {
      await api.put(
        `/runtime/namespaces/${encodeURIComponent(namespace)}/permissions`,
        body,
      );
      await loadNamespaceDetail(namespace);
    } catch (err) {
      setError(err.message);
    }
  }
  async function deleteNamespacePermission(namespace, name) {
    if (!confirm(t("Delete namespace permission {name}?", { name }))) return;
    try {
      await api.delete(
        `/runtime/namespaces/${encodeURIComponent(namespace)}/permissions/${encodeURIComponent(name)}`,
      );
      await loadNamespaceDetail(namespace);
    } catch (err) {
      setError(err.message);
    }
  }
  async function saveNamespaceIsolation(namespace, event) {
    event.preventDefault();
    const form = event.currentTarget;
    try {
      await api.put(
        `/runtime/namespaces/${encodeURIComponent(namespace)}/isolation`,
        {
          enabled: Boolean(form.enabled.checked),
          allow_same_namespace: Boolean(form.allow_same_namespace.checked),
          allow_dns: Boolean(form.allow_dns.checked),
        },
      );
      await loadNamespaceDetail(namespace);
      await loadNetwork();
    } catch (err) {
      setError(err.message);
    }
  }
  async function deleteNamespace(namespace) {
    if (
      !confirm(
        t("Delete namespace {namespace}? This removes resources inside it.", {
          namespace,
        }),
      )
    )
      return;
    try {
      await api.delete(`/runtime/namespaces/${namespace}`);
      await loadWorkspace();
    } catch (err) {
      setError(err.message);
    }
  }
  async function deletePod(pod) {
    if (!confirm(t("Delete pod {name}? Kubernetes may recreate it.", { name: pod.name })))
      return;
    try {
      await api.delete(`/runtime/pods/${pod.namespace}/${pod.name}`);
      await loadWorkspace();
    } catch (err) {
      setError(err.message);
    }
  }
  async function loadNodeDetail(node, showModal = true) {
    if (nodeDetailLoadingRef.current) return;
    nodeDetailLoadingRef.current = true;
    if (showModal)
      setRuntimeDetail({
        kind: "node",
        row: node,
        loading: true,
      });
    if (showModal) setNodeHealth(null);
    try {
      const data = await api.get(
        `/runtime/nodes/${encodeURIComponent(node.name)}`,
      );
      setRuntimeDetail({
        kind: "node",
        row: data.data || node,
        loading: false,
      });
    } catch (err) {
      setRuntimeDetail({
        kind: "node",
        row: node,
        loading: false,
        error: err.message,
      });
    } finally {
      nodeDetailLoadingRef.current = false;
    }
  }
  async function loadNodeHealth(nodeName) {
    try {
      const data = await api.get(
        `/runtime/nodes/${encodeURIComponent(nodeName)}/health`,
      );
      setNodeHealth(data.data || null);
    } catch (err) {
      setError(err.message);
    }
  }
  async function loadNodeJoinCommand(role = "agent") {
    try {
      const data = await api.get(
        `/runtime/nodes/join-command?role=${encodeURIComponent(role)}`,
      );
      setNodeJoinCommand(data.data || null);
    } catch (err) {
      setError(err.message);
    }
  }
  async function saveNodeLabels(nodeName, labelsText) {
    try {
      await api.patch(`/runtime/nodes/${encodeURIComponent(nodeName)}/labels`, {
        labels: parseKeyValues(labelsText),
      });
      setNotice(t("{name} labels updated.", { name: nodeName }));
      await loadWorkspace();
      await loadNodeDetail(
        {
          name: nodeName,
        },
        false,
      );
    } catch (err) {
      setError(err.message);
    }
  }
  async function saveNodeTaints(nodeName, taintsText) {
    try {
      await api.put(`/runtime/nodes/${encodeURIComponent(nodeName)}/taints`, {
        taints: parseTaints(taintsText),
      });
      setNotice(t("{name} taints updated.", { name: nodeName }));
      await loadWorkspace();
      await loadNodeDetail(
        {
          name: nodeName,
        },
        false,
      );
    } catch (err) {
      setError(err.message);
    }
  }
  async function cordonNode(nodeName, schedulable) {
    try {
      await api.post(
        `/runtime/nodes/${encodeURIComponent(nodeName)}/${schedulable ? "uncordon" : "cordon"}`,
        {},
      );
      setNotice(
        t("{name} {state}.", {
          name: nodeName,
          state: schedulable ? t("uncordoned") : t("cordoned"),
        }),
      );
      await loadWorkspace();
      await loadNodeDetail(
        {
          name: nodeName,
        },
        false,
      );
    } catch (err) {
      setError(err.message);
    }
  }
  async function drainNode(nodeName, options) {
    if (
      !confirm(
        t("Drain node {nodeName}? Workloads will be evicted from this node.", {
          nodeName,
        }),
      )
    )
      return;
    try {
      const data = await api.post(
        `/runtime/nodes/${encodeURIComponent(nodeName)}/drain`,
        options,
      );
      setNotice(
        t("Drain started for {name}: {evicted} pods evicted, {skipped} skipped.", {
          name: nodeName,
          evicted: (data.data?.evicted_pods || []).length,
          skipped: (data.data?.skipped_pods || []).length,
        }),
      );
      await loadWorkspace();
      await loadNodeDetail(
        {
          name: nodeName,
        },
        false,
      );
    } catch (err) {
      setError(err.message);
    }
  }
  async function deleteNode(nodeName) {
    try {
      await api.delete(`/runtime/nodes/${encodeURIComponent(nodeName)}`);
      setRuntimeDetail(null);
      setNotice(t("{name} deleted from the cluster.", { name: nodeName }));
      await loadWorkspace();
    } catch (err) {
      setError(err.message);
    }
  }
  async function loadPodLogs(pod) {
    stopRuntimeLogFollow();
    setRuntimeDetail({
      kind: "pod",
      row: pod,
    });
    setRuntimeLogs("");
    setRuntimeLogContainer("");
    setRuntimeLogTail(200);
    setRuntimeLogLoaded(false);
    setRuntimeLogStatus(
      "Choose a container to load logs. Logs are loaded lazily to keep the browser responsive.",
    );
  }
  async function loadRuntimeContainerLogs(
    pod,
    container = runtimeLogContainer,
    tail = runtimeLogTail,
  ) {
    stopRuntimeLogFollow();
    if (!container) {
      setRuntimeLogStatus("Choose a container first.");
      return;
    }
    setRuntimeLogContainer(container);
    setRuntimeLogTail(Number(tail || 200));
    setRuntimeLogStatus("Loading recent logs...");
    setRuntimeLogs("");
    setRuntimeLogLoaded(true);
    try {
      const namespace = encodeURIComponent(pod.namespace);
      const name = encodeURIComponent(pod.name);
      const selected = encodeURIComponent(container);
      const data = await api.get(
        `/runtime/pods/${namespace}/${name}/logs?tail=${Number(tail || 200)}&container=${selected}`,
      );
      setRuntimeLogs(trimLiveLog(data.logs || ""));
      setRuntimeLogStatus(
        `Loaded last ${Number(tail || 200)} lines from ${container}.`,
      );
    } catch (err) {
      setRuntimeLogs("");
      setRuntimeLogStatus(err.message);
    }
  }
  async function startRuntimeLogFollow(
    pod,
    container = runtimeLogContainer,
    tail = runtimeLogTail,
  ) {
    if (!container) {
      setRuntimeLogStatus("Choose a container first.");
      return;
    }
    runtimeLogController.current?.abort();
    const controller = new AbortController();
    runtimeLogController.current = controller;
    setRuntimeLogContainer(container);
    setRuntimeLogTail(Number(tail || 200));
    setRuntimeLogFollow(true);
    setRuntimeLogStatus("Connecting...");
    setRuntimeLogs("");
    setRuntimeLogLoaded(true);
    try {
      const namespace = encodeURIComponent(pod.namespace);
      const name = encodeURIComponent(pod.name);
      const selected = encodeURIComponent(container);
      const res = await api.stream(
        `/runtime/pods/${namespace}/${name}/logs?tail=${Number(tail || 200)}&container=${selected}&follow=true`,
        {
          signal: controller.signal,
        },
      );
      setRuntimeLogStatus(`Following live logs for ${container}`);
      await consumeTextStream(res, (chunk) =>
        setRuntimeLogs((current) => trimLiveLog(current + chunk)),
      );
      setRuntimeLogStatus("Log stream ended");
    } catch (err) {
      if (err.name !== "AbortError") setRuntimeLogStatus(err.message);
    } finally {
      if (runtimeLogController.current === controller) {
        runtimeLogController.current = null;
        setRuntimeLogFollow(false);
      }
    }
  }
  function stopRuntimeLogFollow() {
    runtimeLogController.current?.abort();
    runtimeLogController.current = null;
    setRuntimeLogFollow(false);
  }
  async function saveService(event, existing = null) {
    event.preventDefault();
    const body = Object.fromEntries(
      new FormData(event.currentTarget).entries(),
    );
    body.selector = parseKeyValues(body.selector);
    body.labels = parseKeyValues(body.labels);
    body.ports = parseServicePorts(body.ports);
    body.external_ips = parseCSV(body.external_ips);
    try {
      if (existing) {
        await api.put(
          `/runtime/services/${existing.namespace}/${existing.name}`,
          body,
        );
      } else {
        await api.post("/runtime/services", body);
        event.currentTarget.reset();
      }
      setRuntimeDetail(null);
      await loadWorkspace();
    } catch (err) {
      setError(err.message);
    }
  }
  async function deleteService(service) {
    if (!confirm(t("Delete service {name}?", { name: service.name }))) return;
    try {
      await api.delete(
        `/runtime/services/${service.namespace}/${service.name}`,
      );
      await loadWorkspace();
    } catch (err) {
      setError(err.message);
    }
  }
  async function saveIngress(event, existing = null) {
    event.preventDefault();
    const body = Object.fromEntries(
      new FormData(event.currentTarget).entries(),
    );
    body.service_port = Number(body.service_port || 80);
    body.annotations = parseKeyValues(body.annotations);
    body.labels = parseKeyValues(body.labels);
    try {
      if (existing) {
        await api.put(
          `/runtime/ingresses/${existing.namespace}/${existing.name}`,
          body,
        );
      } else {
        await api.post("/runtime/ingresses", body);
        event.currentTarget.reset();
      }
      setNotice(t("Ingress {name} saved.", { name: body.name || existing?.name }));
      await loadWorkspace();
    } catch (err) {
      setError(err.message);
    }
  }
  async function deleteIngress(ingress) {
    if (
      !confirm(
        t("Delete ingress {namespace}/{name}?", {
          namespace: ingress.namespace,
          name: ingress.name,
        }),
      )
    )
      return;
    try {
      await api.delete(
        `/runtime/ingresses/${ingress.namespace}/${ingress.name}`,
      );
      await loadWorkspace();
    } catch (err) {
      setError(err.message);
    }
  }
  async function saveNetworkPolicy(event, existing = null) {
    event.preventDefault();
    const body = Object.fromEntries(
      new FormData(event.currentTarget).entries(),
    );
    body.pod_selector = parseKeyValues(body.pod_selector);
    body.labels = parseKeyValues(body.labels);
    body.policy_types = Array.from(
      event.currentTarget.querySelectorAll(
        "input[name='policy_types']:checked",
      ),
    ).map((input) => input.value);
    body.allow_same_namespace = Boolean(body.allow_same_namespace);
    body.allow_dns = Boolean(body.allow_dns);
    try {
      if (existing) {
        await api.put(
          `/runtime/network-policies/${existing.namespace}/${existing.name}`,
          body,
        );
      } else {
        await api.post("/runtime/network-policies", body);
        event.currentTarget.reset();
      }
      setNotice(
        t("NetworkPolicy {name} saved.", { name: body.name || existing?.name }),
      );
      await loadWorkspace();
    } catch (err) {
      setError(err.message);
    }
  }
  async function deleteNetworkPolicy(policy) {
    if (
      !confirm(
        t("Delete NetworkPolicy {namespace}/{name}?", {
          namespace: policy.namespace,
          name: policy.name,
        }),
      )
    )
      return;
    try {
      await api.delete(
        `/runtime/network-policies/${policy.namespace}/${policy.name}`,
      );
      await loadWorkspace();
    } catch (err) {
      setError(err.message);
    }
  }
  async function analyzeRepo(repoFullName = selectedRepo, branchOverride = "") {
    if (!selectedCredential || !repoFullName) return;
    setError("");
    setNotice("");
    const branch = branchOverride || deployForm.github_branch || "main";
    try {
      const specAnalysis = await analyzeApplicationSpec(repoFullName, branch);
      if (specAnalysis) return specAnalysis;
      const endpoint =
        deployForm.application_type === "monorepo"
          ? "/repositories/analyze-monorepo"
          : "/projects/analyze";
      const data = await api.post(endpoint, {
        github_credential_id: Number(selectedCredential),
        github_repo: repoFullName,
        github_branch: branch,
      });
      setAnalysis(data);
      setSelectedRepo(repoFullName);
      setDeployForm((current) => ({
        ...current,
        name:
          current.name || slugify(repoFullName.split("/")[1] || repoFullName),
        github_repo: repoFullName,
        github_branch: branch,
        dockerfile_path: data.dockerfile_path || current.dockerfile_path || "",
        port: data.default_port || current.port || 8080,
        components: data.components?.length
          ? monorepoComponentsFromAnalysis(
              slugify(repoFullName.split("/")[1] || repoFullName),
              data.components,
            )
          : current.components,
      }));
      return data;
    } catch (err) {
      setAnalysis(null);
      setError(`Repository analysis failed: ${err.message}`);
      return null;
    }
  }
  async function analyzeApplicationSpec(repoFullName, branch) {
    try {
      const data = await api.post("/application-specs/plan", {
        github_credential_id: Number(selectedCredential),
        github_repo: repoFullName,
        github_branch: branch,
      });
      const specAnalysis = applicationSpecAnalysis(data);
      setAnalysis(specAnalysis);
      setSelectedRepo(repoFullName);
      setDeployForm((current) =>
        deployFormFromApplicationSpec(current, repoFullName, branch, data),
      );
      return specAnalysis;
    } catch (err) {
      const message = String(err.message || "");
      if (message.includes("application spec config not found")) return false;
      const specAnalysis = {
        source: "beancs_spec",
        is_monorepo: false,
        deployable: false,
        spec_error: message,
        warnings: [message],
      };
      setAnalysis(specAnalysis);
      setDeployForm((current) => ({
        ...current,
        application_type: "monorepo",
        github_repo: repoFullName,
        github_branch: branch,
      }));
      setError(`Application spec check failed: ${message}`);
      return specAnalysis;
    }
  }
  function checkInstallSource(nextForm = deployForm) {
    const source = nextForm.deploy_source === "gitops" ? "github" : "registry";
    setError("");
    setNotice("");
    if (source === "github") {
      return analyzeRepo(
        nextForm.github_repo || selectedRepo,
        nextForm.github_branch,
      );
    }
    const image = (nextForm.image_reference || "").trim();
    if (!image) {
      setError(t("Image reference is required for registry deployments."));
      return false;
    }
    setAnalysis({
      deployable: true,
      containerized: source !== "source-upload",
      scaffoldable: false,
      default_port: nextForm.port || 8080,
      ports: [Number(nextForm.port || 8080)],
      signals: [`Registry image: ${image}`, "Update mode: passive"],
      warnings: [],
    });
    if (!nextForm.name) {
      setDeployForm((current) => ({
        ...current,
        name: slugify(imageName(image)),
      }));
    }
    return true;
  }
  async function deployProject(event) {
    event.preventDefault();
    if (deployForm.application_type === "monorepo") {
      if (
        !analysis?.is_monorepo ||
        analysis?.deployable === false ||
        !(deployForm.components || []).some((component) => component.enabled)
      )
        return;
      return deployMonorepoApplication();
    }
    if (!analysis?.deployable) return;
    const payload = buildProjectPayload(deployForm, selectedCredential, {
      ...credentials,
      domains,
    });
    setLoading(true);
    setError("");
    setInstallProgress({
      project: payload.name,
      started_at: new Date().toISOString(),
      logs: [
        `Starting deploy for ${payload.name}`,
        `Source: ${payload.github_repo || payload.image_reference || payload.build_source}`,
        `Namespace: ${payload.namespace || `proj-${payload.name}`}`,
      ],
      steps: [
        {
          label: "Validate install source",
          state: "done",
        },
        {
          label: "Create project resources",
          state: "running",
        },
        {
          label: "Apply service and ingress",
          state: "pending",
        },
        {
          label: "Apply deployment or GitOps manifests",
          state: "pending",
        },
      ],
    });
    openProgress();
    try {
      const created = await api.post("/projects", {
        ...payload,
        auto_deploy: false,
      });
      const deploymentResult = await api.post(
        `/projects/${created.id}/deployments`,
        {
          tag: payload.image_reference || "",
          commit_sha: payload.github_branch || "",
        },
      );
      if (deploymentResult.process?.id)
        setActiveProcessID(String(deploymentResult.process.id));
      setNotice(t("Project created. Deployment process queued."));
      setActiveProgressProjectID(String(created.id));
      openProgress(created.id, deploymentResult.process?.id, true);
      setInstallProgress((current) =>
        current
          ? {
              ...current,
              logs: [
                ...(current.logs || []),
                `Project created with id ${created.id}`,
                "Deployment process submitted to the executor.",
                "Waiting for process jobs to write logs.",
              ],
              steps: current.steps.map((step) => {
                if (
                  step.label === "Validate install source" ||
                  step.label === "Create project resources"
                )
                  return {
                    ...step,
                    state: "done",
                  };
                if (step.label === "Apply service and ingress")
                  return {
                    ...step,
                    state: "running",
                    log: "Waiting for Services, Ingresses, and Kubernetes events to confirm the route.",
                  };
                return {
                  ...step,
                  state: "pending",
                  log: "Waiting for a direct Deployment, GitHub Actions build, or Argo CD sync to create workload resources.",
                };
              }),
            }
          : null,
      );
      setDeployForm(defaultDeployForm());
      setAnalysis(null);
      setSelectedRepo("");
      await loadWorkspace();
      await loadProcesses();
      await loadProjectProgress(String(created.id));
    } catch (err) {
      setError(err.message);
      setInstallProgress((current) =>
        current
          ? {
              ...current,
              logs: [...(current.logs || []), `ERROR: ${err.message}`],
              steps: current.steps.map((step) =>
                step.state === "running"
                  ? {
                      ...step,
                      state: "failed",
                      log: err.message,
                    }
                  : step,
              ),
            }
          : null,
      );
    } finally {
      setLoading(false);
    }
  }
  async function deployBasaltPass(event) {
    event.preventDefault();
    let dependencyID = String(deployForm.database_dependency_id || "");
    let credentialID = "";
    if (deployForm.database_credential_mode !== "new") {
      const binding = String(deployForm.database_binding || "").split(":");
      dependencyID = binding[0] || dependencyID;
      credentialID = binding[1] || "";
    }
    const selectedCF =
      (domains || []).find(
        (domain) =>
          String(domain.credential_id) ===
            String(deployForm.cloudflare_credential_id) &&
          String(domain.zone_id) === String(deployForm.cloudflare_zone_id),
      ) ||
      credentials.cloudflare.find(
        (cred) =>
          String(cred.id) === String(deployForm.cloudflare_credential_id),
      );
    const publicHost =
      deployForm.exposure_mode === "public" && selectedCF?.domain
        ? `${deployForm.subdomain}.${selectedCF.domain}`
        : deployForm.public_host || "";
    const baseURL = publicHost
      ? `https://${publicHost.replace(/^https?:\/\//, "")}`
      : deployForm.base_url;
    const body = {
      name: deployForm.name,
      base_url: baseURL,
      tenant_name: deployForm.tenant_name,
      tenant_code: deployForm.tenant_code,
      namespace: deployForm.namespace || undefined,
      backend_image: deployForm.backend_image,
      frontend_image: deployForm.frontend_image,
      github_credential_id: selectedCredential
        ? Number(selectedCredential)
        : undefined,
      github_repo: deployForm.github_repo || selectedRepo || undefined,
      github_branch: deployForm.github_branch || "main",
      public_host: publicHost || undefined,
      exposure_mode: deployForm.exposure_mode || "public",
      platform_admin_email: deployForm.platform_admin_email,
      platform_admin_username: deployForm.platform_admin_username,
      platform_admin_password: deployForm.platform_admin_password,
      cloudflare_credential_id: deployForm.cloudflare_credential_id
        ? Number(deployForm.cloudflare_credential_id)
        : undefined,
      cloudflare_zone_id: deployForm.cloudflare_zone_id || undefined,
      database_dependency_id: Number(dependencyID || 0),
      database_credential_id: Number(credentialID || 0),
      owner_email: deployForm.owner_email,
      owner_username: deployForm.owner_username,
      owner_password: deployForm.owner_password,
      description: deployForm.description || undefined,
      max_apps: deployForm.max_apps ? Number(deployForm.max_apps) : undefined,
      max_users: deployForm.max_users ? Number(deployForm.max_users) : undefined,
      max_tokens_per_hour: deployForm.max_tokens_per_hour
        ? Number(deployForm.max_tokens_per_hour)
        : undefined,
      service_token: deployForm.service_token,
      automation_token: deployForm.automation_token,
      jwt_secret: deployForm.jwt_secret || undefined,
      cors_allow_origins: deployForm.cors_allow_origins || undefined,
    };
    Object.keys(body).forEach((key) => {
      if (body[key] === "" || body[key] === undefined || body[key] === 0)
        delete body[key];
    });
    setLoading(true);
    setError("");
    setInstallProgress({
      project: body.name,
      started_at: new Date().toISOString(),
      logs: [
        `Starting BasaltPass deploy for ${body.name}`,
        `Source: ${deployForm.github_repo || "-"}`,
        `Namespace: ${body.namespace || `bp-${body.name}`}`,
      ],
      steps: [
        { label: "Apply BasaltPass runtime", state: "running" },
        { label: "Wait for health", state: "pending" },
        { label: "Create tenant", state: "pending" },
        { label: "Store tenant credentials", state: "pending" },
      ],
    });
    try {
      if (deployForm.database_credential_mode === "new") {
        const credential = await api.post(
          `/dependencies/${dependencyID}/credentials`,
          {
            name: deployForm.database_credential_name,
            description:
              deployForm.database_credential_description || undefined,
            config: {
              database: deployForm.database_name,
              username: deployForm.database_username,
              password: deployForm.database_password,
            },
          },
        );
        credentialID = String(credential.id || "");
        body.database_credential_id = Number(credentialID || 0);
      }
      const result = await api.post("/credentials/basaltpass/deployments", body);
      if (result.process?.id) setActiveProcessID(String(result.process.id));
      setNotice(t("BasaltPass deployment process started."));
      setDeployForm(defaultDeployForm());
      await loadWorkspace();
      await loadProcesses();
      openProgress("", result.process?.id);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }
  async function deployMonorepoApplication() {
    if (analysis?.source === "beancs_spec") {
      return deployApplicationFromRepoConfig();
    }
    const payload = buildMonorepoApplicationPayload(
      deployForm,
      selectedCredential,
      {
        ...credentials,
        domains,
      },
    );
    if (!payload.components.length) {
      setError(t("Select at least one monorepo component."));
      return;
    }
    setLoading(true);
    setError("");
    setInstallProgress({
      project: payload.name,
      started_at: new Date().toISOString(),
      logs: [
        `Starting ${analysis?.source === "beancs_spec" ? ".beancs spec" : "monorepo"} deploy for ${payload.name}`,
        `Repository: ${payload.github_repo} @ ${payload.github_branch}`,
        `Dependencies: ${(payload.dependencies || []).map((dep) => dep.name).join(", ") || "none"}`,
        `Components: ${payload.components.map((component) => component.project_name).join(", ")}`,
      ],
      steps: [
        {
          label: "Create application record",
          state: "running",
        },
        {
          label: "Create dependency components",
          state: (payload.dependencies || []).length ? "pending" : "done",
        },
        {
          label: "Create component projects",
          state: "pending",
        },
        {
          label: "Queue component deployments",
          state: "pending",
        },
        {
          label: "Wait for GitOps and runtime status",
          state: "pending",
        },
      ],
    });
    openProgress();
    try {
      const created = await api.post("/applications/monorepo", payload);
      const projects = created.projects || created.data?.projects || [];
      const dependencies =
        created.dependencies || created.data?.dependencies || [];
      const firstProject = projects[0];
      setNotice(
        t(
          "Application {name} created with {count} components and {deps} dependencies.",
          {
            name: payload.name,
            count: projects.length,
            deps: dependencies.length,
          },
        ),
      );
      if (firstProject?.id) openProgress(firstProject.id);
      setInstallProgress((current) =>
        current
          ? {
              ...current,
              logs: [
                ...(current.logs || []),
                `Application created with id ${created.id || created.data?.id || "-"}`,
                `${dependencies.length} dependency component(s) created.`,
                `${projects.length} component project(s) created.`,
              ],
              steps: current.steps.map((step) => ({
                ...step,
                state:
                  step.state === "running" || step.state === "pending"
                    ? "done"
                    : step.state,
              })),
            }
          : null,
      );
      setDeployForm(defaultDeployForm());
      setAnalysis(null);
      setSelectedRepo("");
      await loadWorkspace();
      await loadProcesses();
      if (firstProject?.id) await loadProjectProgress(String(firstProject.id));
    } catch (err) {
      setError(err.message);
      setInstallProgress((current) =>
        current
          ? {
              ...current,
              logs: [...(current.logs || []), `ERROR: ${err.message}`],
              steps: current.steps.map((step) =>
                step.state === "running"
                  ? {
                      ...step,
                      state: "failed",
                      log: err.message,
                    }
                  : step,
              ),
            }
          : null,
      );
    } finally {
      setLoading(false);
    }
  }
  async function deployApplicationFromRepoConfig() {
    const payload = {
      github_credential_id: Number(selectedCredential),
      github_repo: deployForm.github_repo,
      github_branch: deployForm.github_branch || "main",
      config_path: analysis?.config_path || ".beancs/app.yaml",
      basaltpass_instance_id: deployForm.basaltpass_instance_id
        ? Number(deployForm.basaltpass_instance_id)
        : undefined,
      cloudflare_credential_id: deployForm.cloudflare_credential_id
        ? Number(deployForm.cloudflare_credential_id)
        : undefined,
      cloudflare_zone_id: deployForm.cloudflare_zone_id || undefined,
      component_domains: monorepoComponentDomainOverrides(deployForm, {
        ...credentials,
        domains,
      }),
    };
    setLoading(true);
    setError("");
    setInstallProgress({
      project:
        deployForm.name ||
        analysis?.plan?.application?.name ||
        payload.github_repo,
      started_at: new Date().toISOString(),
      logs: [
        `Starting .beancs application deploy for ${deployForm.name || analysis?.plan?.application?.name || payload.github_repo}`,
        `Repository: ${payload.github_repo} @ ${payload.github_branch}`,
        `Config: ${payload.config_path}`,
      ],
      steps: [
        {
          label: "Read .beancs/app.yaml",
          state: "done",
        },
        {
          label: "Validate application spec",
          state: "done",
        },
        {
          label: "Apply application plan",
          state: "running",
        },
        {
          label: "Create dependencies and projects",
          state: "pending",
        },
        {
          label: "Wait for GitOps and runtime status",
          state: "pending",
        },
      ],
    });
    openProgress();
    try {
      const created = await api.post("/applications/from-repo-config", payload);
      const app = created.application || {};
      const projects = app.projects || [];
      const dependencies = app.dependencies || [];
      const firstProject = projects[0];
      setNotice(
        t("Application {name} applied from {path}.", {
          name: app.name || deployForm.name,
          path: payload.config_path,
        }),
      );
      if (firstProject?.id) openProgress(firstProject.id);
      setInstallProgress((current) =>
        current
          ? {
              ...current,
              logs: [
                ...(current.logs || []),
                `Application applied with id ${app.id || "-"}.`,
                `${dependencies.length} dependency component(s) created.`,
                `${projects.length} project component(s) created.`,
              ],
              steps: current.steps.map((step) => ({
                ...step,
                state:
                  step.state === "running" || step.state === "pending"
                    ? "done"
                    : step.state,
              })),
            }
          : null,
      );
      setDeployForm(defaultDeployForm());
      setAnalysis(null);
      setSelectedRepo("");
      await loadWorkspace();
      await loadProcesses();
      if (firstProject?.id) await loadProjectProgress(String(firstProject.id));
    } catch (err) {
      setError(err.message);
      setInstallProgress((current) =>
        current
          ? {
              ...current,
              logs: [...(current.logs || []), `ERROR: ${err.message}`],
              steps: current.steps.map((step) =>
                step.state === "running"
                  ? {
                      ...step,
                      state: "failed",
                      log: err.message,
                    }
                  : step,
              ),
            }
          : null,
      );
    } finally {
      setLoading(false);
    }
  }
  async function loadProjectProgress(projectID = activeProgressProjectID) {
    if (progressLoadingRef.current) return;
    progressLoadingRef.current = true;
    let selected = projectID
      ? projects.find((project) => String(project.id) === String(projectID))
      : projects[0];
    if (!selected) {
      if (!projectID) {
        setProjectProgress(null);
        progressLoadingRef.current = false;
        return;
      }
      try {
        selected = await api.get(`/projects/${projectID}`);
      } catch (err) {
        setProjectProgress(null);
        progressLoadingRef.current = false;
        return;
      }
    }
    setActiveProgressProjectID(String(selected.id));
    try {
      const logRequest = projectLogFollow
        ? Promise.resolve({
            logs: projectProgress?.logs || "",
          })
        : api.get(`/projects/${selected.id}/logs?tail=160`);
      const [status, deployments, logData] = await Promise.all([
        api.get(`/projects/${selected.id}/status`),
        api.get(`/projects/${selected.id}/deployments`),
        logRequest,
      ]);
      const deploymentRows = deployments.data || [];
      let workflowLogs = "";
      const latestWorkflow = deploymentRows.find(
        (deployment) =>
          deployment.workflow_run_id ||
          deployment.workflow_url ||
          deployment.failure_reason,
      );
      if (!projectLogFollow && latestWorkflow?.id) {
        try {
          const workflowLogData = await api.get(
            `/projects/${selected.id}/deployments/${latestWorkflow.id}/logs`,
          );
          workflowLogs = workflowLogData.logs || "";
        } catch (err) {
          workflowLogs = `GitHub Actions/deployment logs unavailable: ${err.message}\n`;
        }
      }
      setProjectProgress({
        project: selected,
        pods: status.pods || [],
        deployment: status.deployment || null,
        services: status.services || [],
        ingresses: status.ingresses || [],
        events: status.events || [],
        deployments: deploymentRows,
        logs: [workflowLogs, logData.logs || ""].filter(Boolean).join("\n"),
        checked_at: new Date().toISOString(),
      });
    } catch (err) {
      setProjectProgress({
        project: selected,
        pods: [],
        deployments: [],
        error: err.message,
        checked_at: new Date().toISOString(),
      });
    } finally {
      progressLoadingRef.current = false;
    }
  }
  async function startProjectLogFollow(projectID = activeProgressProjectID) {
    let selected = projectID
      ? projects.find((project) => String(project.id) === String(projectID))
      : projectProgress?.project || projects[0];
    if (!selected && projectID) {
      try {
        selected = await api.get(`/projects/${projectID}`);
      } catch (err) {
        setProjectLogStatus(err.message);
        return;
      }
    }
    if (!selected) {
      setProjectLogStatus(t("Choose a project before following logs."));
      return;
    }
    projectLogController.current?.abort();
    const controller = new AbortController();
    projectLogController.current = controller;
    setActiveProgressProjectID(String(selected.id));
    setProjectLogFollow(true);
    setProjectLiveLogs("");
    setProjectLogStatus("Connecting...");
    try {
      const res = await api.stream(
        `/projects/${selected.id}/logs?tail=160&follow=true`,
        {
          signal: controller.signal,
        },
      );
      setProjectLogStatus("Following live logs");
      await consumeTextStream(res, (chunk) =>
        setProjectLiveLogs((current) => trimLiveLog(current + chunk)),
      );
      setProjectLogStatus("Log stream ended");
    } catch (err) {
      if (err.name !== "AbortError") setProjectLogStatus(err.message);
    } finally {
      if (projectLogController.current === controller) {
        projectLogController.current = null;
        setProjectLogFollow(false);
      }
    }
  }
  function stopProjectLogFollow() {
    projectLogController.current?.abort();
    projectLogController.current = null;
    setProjectLogFollow(false);
    setProjectLogStatus("");
  }
  async function loadProjectEnv(project) {
    const data = await api.get(`/projects/${project.id}/env`);
    return data.data || {};
  }
  async function loadProjectVolumes(project) {
    const data = await api.get(`/projects/${project.id}/volumes`);
    return data.data?.items || [];
  }
  async function loadAvailablePVCs(project) {
    const data = await api.get(`/projects/${project.id}/available-pvcs`);
    return data.data || [];
  }
  async function updateProject(event, envData, volumes) {
    event.preventDefault();
    const body = Object.fromEntries(
      new FormData(event.currentTarget).entries(),
    );
    body.replicas = Number(body.replicas || 1);
    body.auto_deploy = body.auto_deploy === "on";
    await api.patch(`/projects/${editingProject.id}`, body);
    await api.put(`/projects/${editingProject.id}/volumes`, { items: volumes });
    if (envData) {
      await api.put(`/projects/${editingProject.id}/env`, envData);
      setNotice(t("{name} updated and restarted.", { name: editingProject.name }));
    }
    setEditingProject(null);
    await loadWorkspace();
  }
  async function deleteProject(project) {
    setDeletingProject(project);
  }
  async function confirmDeleteProject() {
    if (!deletingProject) return;
    setLoading(true);
    setError("");
    try {
      await api.delete(`/projects/${deletingProject.id}`);
      setNotice(t("{name} deleted.", { name: deletingProject.name }));
      setDeletingProject(null);
      if (String(activeProgressProjectID) === String(deletingProject.id)) {
        setActiveProgressProjectID("");
        setProjectProgress(null);
      }
      await loadWorkspace();
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }
  async function deleteApplication(application) {
    setDeletingApplication(application);
  }
  async function confirmDeleteApplication() {
    if (!deletingApplication) return;
    setLoading(true);
    setError("");
    try {
      await api.delete(`/applications/${deletingApplication.id}`);
      setNotice(t("{name} deleted.", { name: deletingApplication.name }));
      setDeletingApplication(null);
      await loadWorkspace();
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }
  async function scaleProject(project, replicas) {
    await api.post(`/projects/${project.id}/scale`, {
      replicas,
    });
    await loadWorkspace();
  }
  async function restartProject(project) {
    await api.post(`/projects/${project.id}/restart`, {});
    setNotice(`${project.name} restarted.`);
  }
  async function buildProject(project) {
    try {
      const result = await api.post(`/projects/${project.id}/deployments`, {
        tag: project.image_reference || "github-actions",
        commit_sha: project.github_branch || "",
      });
      setNotice(t("{name} build started.", { name: project.name }));
      openProgress(project.id, result.process?.id);
      await loadProcesses();
      await loadProjectProgress(String(project.id));
    } catch (err) {
      setError(err.message);
    }
  }
  async function openProjectTracking(project) {
    setTrackingProject(project);
    setProjectTracking(null);
    setTrackingLoading(true);
    setError("");
    try {
      const data = await api.get(`/projects/${project.id}/tracking?limit=100`);
      setProjectTracking(data);
    } catch (err) {
      setError(err.message);
    } finally {
      setTrackingLoading(false);
    }
  }
  function selectNav(item) {
    if (item.externalUrl) {
      setSidebarOpen(false);
      window.open(item.externalUrl, "_blank", "noopener,noreferrer");
      return;
    }
    if (item.id === "progress") {
      setActiveProgressProjectID("");
      setActiveProcessID("");
      setProjectProgress(null);
      stopProjectLogFollow();
    }
    setSidebarOpen(false);
    navigate(pathForView(item.id));
  }
  if (!token) {
    const loginBusy = loginPhase !== "idle";
    const callbackBusy =
      loginPhase === "completing" || loginPhase === "authenticated";
    return (
      <main className="login-screen">
        <section className="login-copy">
          <h1>{t("BeanCS")}</h1>
          <p>
            {t(
              "Operate k3s projects, GitHub App deployments, DNS, and traffic routes from one console.",
            )}
          </p>
          {loginBusy ? (
            <div className="login-status" role="status" aria-live="polite">
              {callbackBusy ? (
                <CheckCircle2 size={18} />
              ) : (
                <LoaderCircle className="spin" size={18} />
              )}
              <span>
                {callbackBusy
                  ? t("Login successful. Redirecting...")
                  : t("Communicating with the identity server...")}
              </span>
            </div>
          ) : (
            <Button onClick={startLogin} variant="primary" disabled={!config}>
              <Lock size={18} /> {t("Sign in with BasaltPass")}
            </Button>
          )}
          {error && <p className="error-text">{error}</p>}
        </section>
      </main>
    );
  }
  return (
    <div className="app-shell">
      <div
        className={sidebarOpen ? "sidebar-overlay active" : "sidebar-overlay"}
        onClick={() => setSidebarOpen(false)}
      />
      <aside className={sidebarOpen ? "sidebar open" : "sidebar"}>
        <div className="sidebar-product">
          <span className="brand-orb">
            <Coffee size={16} />
          </span>
          <b>{t("BeanCS")}</b>
        </div>
        <label className="sidebar-search">
          <Search size={19} />
          <Input
            value={sidebarQuery}
            onChange={(event) => setSidebarQuery(event.target.value)}
            placeholder={t("Find...")}
          />
          <kbd>F</kbd>
        </label>
        <div className="sidebar-nav">
          {filteredOverview.length > 0 && (
            <SidebarNavGroup
              items={filteredOverview}
              view={view}
              onSelect={selectNav}
            />
          )}
          {filteredNavSections.map((section) => (
            <SidebarNavGroup
              key={section.id}
              label={section.label}
              items={section.items}
              view={view}
              onSelect={selectNav}
            />
          ))}
          {sidebarQuery &&
            filteredOverview.length === 0 &&
            filteredNavSections.length === 0 && (
              <div className="nav-empty">{t("No matches")}</div>
            )}
        </div>
        <div className="sidebar-user">
          <div className="user-avatar">
            {userProfile.avatar ? (
              <img
                src={userProfile.avatar}
                alt={userProfile.name || "User avatar"}
              />
            ) : (
              userProfile.initial
            )}
          </div>
          <div className="user-copy">
            <b>{userProfile.name}</b>
            <span>{userProfile.detail}</span>
          </div>
          <div className="sidebar-user-menu-anchor" ref={userMenuRef}>
            <Button
              type="button"
              aria-label={t("More account actions")}
              aria-expanded={userMenuOpen}
              aria-haspopup="menu"
              variant="icon"
              onClick={() => setUserMenuOpen((open) => !open)}
            >
              <MoreHorizontal size={16} />
            </Button>
            {userMenuOpen && (
              <div className="sidebar-user-menu" role="menu">
                <div className="sidebar-user-menu-section" role="none">
                  <span className="sidebar-user-menu-label">{t("Language")}</span>
                  <LanguageSwitcher />
                </div>
                <button
                  type="button"
                  className="sidebar-user-menu-item"
                  role="menuitem"
                  onClick={() => {
                    setUserMenuOpen(false);
                    logout();
                  }}
                >
                  <LogOut size={16} />
                  {t("Sign out")}
                </button>
              </div>
            )}
          </div>
        </div>
      </aside>
      <main className="workspace">
        <div className="mobile-topbar">
          <Button
            type="button"
            aria-label={t("Open navigation")}
            onClick={() => setSidebarOpen(true)}
            variant="icon"
          >
            <Menu size={18} />
          </Button>
          <span className="mobile-brand">{t("BeanCS")}</span>
        </div>
        <PageHeading
          title={
            view === "dashboard"
              ? dashboard?.cluster_name || t("Overview")
              : titleFor(view)
          }
          topLabel={view === "dashboard" ? t("Overview") : undefined}
          subtitle={
            view === "dashboard"
              ? `Kubernetes ${dashboard?.kubernetes_version || "-"}${dashboard?.k3s_version ? ` · K3s ${dashboard.k3s_version}` : ""}`
              : subtitleFor(view, runtime, projects)
          }
          actions={
            view === "dashboard" ? null : (
              <Button onClick={loadWorkspace} disabled={loading}>
                <RefreshCw size={15} /> Refresh
              </Button>
            )
          }
        />
        {notice && <div className="notice">{notice}</div>}
        {error && <div className="alert">{error}</div>}
        {shouldShowSkeleton(view, dashboard, network) ? (
          <SkeletonPage />
        ) : (
          <>
            {view === "dashboard" && (
              <DashboardView dashboard={dashboard} />
            )}
            {view === "deploy" && (
              <DeployView
                config={config}
                credentials={credentials}
                domains={domains}
                namespaces={runtime.namespaces || []}
                selectedCredential={selectedCredential}
                setSelectedCredential={setSelectedCredential}
                repos={repos}
                selectedRepo={selectedRepo}
                analysis={analysis}
                setAnalysis={setAnalysis}
                form={deployForm}
                setForm={setDeployForm}
                loadRepos={loadRepos}
                analyzeRepo={analyzeRepo}
                checkInstallSource={checkInstallSource}
                deployProject={deployProject}
                containerRegistries={containerRegistries}
                containerImages={containerImages}
                dependencyDefinitions={dependencyDefinitions}
                reusableDependencies={reusableDependencies}
                createTrackedImageFromDeploy={createTrackedImageFromDeploy}
                deployBasaltPass={deployBasaltPass}
                onDeployDependency={deployDependency}
                onConnectGitHub={connectGitHubApp}
                reposLoading={reposLoading}
              />
            )}
            {view === "dependencies" && (
              <DependenciesView
                definitions={dependencyDefinitions}
                dependencies={reusableDependencies}
                githubCredentials={credentials.github}
                onCreateDependency={createDependency}
                onCreateCredential={createDependencyCredential}
              />
            )}
            {view === "progress" && (
              <ProgressView
                projects={projects}
                processes={processRecords}
                activeProcessID={activeProcessID}
                setActiveProcessID={setActiveProcessID}
                activeProjectID={activeProgressProjectID}
                setActiveProjectID={setActiveProgressProjectID}
                progress={projectProgress}
                installProgress={installProgress}
                refresh={loadProjectProgress}
                refreshList={loadProcesses}
                logFollow={projectLogFollow}
                liveLogs={projectLiveLogs}
                logStatus={projectLogStatus}
                onStartLogFollow={startProjectLogFollow}
                onStopLogFollow={stopProjectLogFollow}
              />
            )}
            {view === "projects" && (
              <ProjectsView
                projects={projects}
                onEdit={setEditingProject}
                onDelete={deleteProject}
                onScale={scaleProject}
                onRestart={restartProject}
                onBuild={buildProject}
                onTracking={openProjectTracking}
                onProgress={(project) => {
                  openProgress(project.id);
                }}
              />
            )}
            {view === "applications" && (
              <ApplicationsView
                applications={applications}
                onDeleteApplication={deleteApplication}
              />
            )}
            {view === "deployments" && (
              <DeploymentsView
                projects={projects}
                processes={processRecords}
                runtimeDeployments={runtime.deployments || []}
                refresh={loadWorkspace}
                onOpenProcess={(process) => {
                  openProgress(process.project_id, process.id);
                }}
              />
            )}
            {view === "apiKeys" && (
              <APIKeysView
                keys={apiKeys}
                scopeCatalog={apiKeyScopeCatalog}
                createdKey={createdAPIKey}
                onDismissCreated={() => setCreatedAPIKey(null)}
                onCreate={createAPIKey}
                onRevoke={revokeAPIKey}
                onRefresh={loadAPIKeys}
              />
            )}
            {view === "registries" && (
              <ContainerRegistriesView
                presets={registryPresets}
                registries={containerRegistries}
                images={containerImages}
                onAddRegistry={createContainerRegistry}
                onDeleteRegistry={deleteContainerRegistry}
                onAddImage={createTrackedImage}
                onRefreshImage={refreshTrackedImage}
                onDeleteImage={deleteTrackedImage}
                onSyncAll={syncAllTrackedImages}
                onRefresh={loadRegistriesPage}
              />
            )}
            {view === "workloadImage" && (
              <WorkloadImageView
                images={containerImages}
                onRefresh={loadRegistriesPage}
                onOpenRegistry={() => navigate(pathForView("registries"))}
                onRefreshImage={refreshTrackedImage}
                onDeleteImage={deleteTrackedImage}
              />
            )}
            {view === "storage" && (
              <StorageView storage={storage} refresh={loadStorage} />
            )}
            {view === "secrets" && (
              <ComingSoonView
                title="Secrets"
                description="Kubernetes Secret inspection and rotation workflows are not wired in this console yet. Use kubectl or your GitOps pipeline for now."
              />
            )}
            {view === "alerts" && (
              <AlertsView dashboard={dashboard} refresh={loadDashboard} />
            )}
            {view === "events" && (
              <EventsView dashboard={dashboard} refresh={loadDashboard} />
            )}
            {view === "logs" && (
              <LogsView
                projects={projects}
                activeProjectID={activeProgressProjectID}
                setActiveProjectID={setActiveProgressProjectID}
                progress={projectProgress}
                refresh={loadProjectProgress}
                logFollow={projectLogFollow}
                liveLogs={projectLiveLogs}
                logStatus={projectLogStatus}
                onStartLogFollow={startProjectLogFollow}
                onStopLogFollow={stopProjectLogFollow}
                onOpenPods={() => navigate(pathForView("pods"))}
              />
            )}
            {view === "metrics" && (
              <MetricsView
                dashboard={dashboard}
                runtime={runtime}
                refresh={loadDashboard}
              />
            )}
            {view === "settings" && <SettingsView version={appVersion} />}
            {view === "github" && (
              <GitHubView
                credentials={credentials.github}
                onConnect={connectGitHubApp}
                onUpdate={updateGitHubCredential}
                onRepos={loadRepos}
                onDelete={(id) => deleteCredential("github", id)}
                reposByCredential={reposByCredential}
                repoFilters={repoFilters}
                setRepoFilters={setRepoFilters}
              />
            )}
            {view === "domains" && <DomainsView domains={domains} />}
            {view === "networking" && (
              <NetworkingView
                network={network}
                refresh={loadNetwork}
                onSaveService={saveService}
                onDeleteService={deleteService}
                onSaveIngress={saveIngress}
                onDeleteIngress={deleteIngress}
                onSaveNetworkPolicy={saveNetworkPolicy}
                onDeleteNetworkPolicy={deleteNetworkPolicy}
                onDetail={setRuntimeDetail}
              />
            )}
            {view === "cloudflare" && (
              <CloudflareView
                credentials={credentials.cloudflare}
                domains={domains}
                selectedID={selectedCloudflareID}
                selectedZoneID={selectedCloudflareZoneID}
                setSelectedID={setSelectedCloudflareID}
                setSelectedZoneID={setSelectedCloudflareZoneID}
                dnsRecords={dnsRecords}
                editingRecord={editingDNSRecord}
                setEditingRecord={setEditingDNSRecord}
                onConnectApp={connectCloudflareApp}
                onCreate={createCredential}
                onDelete={(id) => deleteCredential("cloudflare", id)}
                onRefreshDomains={refreshCloudflareDomains}
                onLoadDNS={loadDNSRecords}
                onSaveDNS={saveDNSRecord}
                onDeleteDNS={deleteDNSRecord}
              />
            )}
            {view === "accessControl" && (
              <CredentialManager
                kind="basaltpass"
                rows={credentials.basaltpass}
                onCreate={createCredential}
                onDelete={deleteCredential}
              />
            )}
            {["namespaces", "pods", "nodes", "ingresses", "services"].includes(
              view,
            ) && (
              <RuntimeTable
                kind={view}
                rows={runtime[view] || []}
                nodeJoinCommand={nodeJoinCommand}
                onLoadNodeJoinCommand={loadNodeJoinCommand}
                onCreateNamespace={createNamespace}
                onPatchNamespace={patchNamespaceLabels}
                onNamespaceDetail={loadNamespaceDetail}
                onDeleteNamespace={deleteNamespace}
                onDeletePod={deletePod}
                onNodeDetail={loadNodeDetail}
                onPodLogs={loadPodLogs}
                onSaveService={saveService}
                onDeleteService={deleteService}
                onDetail={setRuntimeDetail}
              />
            )}
          </>
        )}
      </main>
      {editingProject && (
        <ProjectModal
          project={editingProject}
          onClose={() => setEditingProject(null)}
          onSubmit={updateProject}
          onLoadEnv={loadProjectEnv}
          onLoadVolumes={loadProjectVolumes}
          onLoadAvailablePVCs={loadAvailablePVCs}
        />
      )}
      {deletingProject && (
        <DeleteProjectModal
          project={deletingProject}
          busy={loading}
          onClose={() => setDeletingProject(null)}
          onDelete={confirmDeleteProject}
        />
      )}
      {deletingApplication && (
        <DeleteApplicationModal
          application={deletingApplication}
          busy={loading}
          onClose={() => setDeletingApplication(null)}
          onDelete={confirmDeleteApplication}
        />
      )}
      {trackingProject && (
        <ProjectTrackingModal
          project={trackingProject}
          tracking={projectTracking}
          loading={trackingLoading}
          onRefresh={() => openProjectTracking(trackingProject)}
          onClose={() => {
            setTrackingProject(null);
            setProjectTracking(null);
          }}
        />
      )}
      {runtimeDetail && (
        <RuntimeDetailDrawer
          detail={runtimeDetail}
          logs={runtimeLogs}
          logFollow={runtimeLogFollow}
          logStatus={runtimeLogStatus}
          selectedLogContainer={runtimeLogContainer}
          logTail={runtimeLogTail}
          logLoaded={runtimeLogLoaded}
          nodeHealth={nodeHealth}
          onLoadNodeHealth={loadNodeHealth}
          onSaveNodeLabels={saveNodeLabels}
          onSaveNodeTaints={saveNodeTaints}
          onCordonNode={cordonNode}
          onDrainNode={drainNode}
          onDeleteNode={deleteNode}
          onSaveResourceQuota={saveResourceQuota}
          onDeleteResourceQuota={deleteResourceQuota}
          onSaveLimitRange={saveLimitRange}
          onDeleteLimitRange={deleteLimitRange}
          onSaveNamespacePermission={saveNamespacePermission}
          onDeleteNamespacePermission={deleteNamespacePermission}
          onSaveNamespaceIsolation={saveNamespaceIsolation}
          onSelectLogContainer={setRuntimeLogContainer}
          onSetLogTail={setRuntimeLogTail}
          onLoadContainerLogs={loadRuntimeContainerLogs}
          onFollowPodLogs={startRuntimeLogFollow}
          onStopPodLogs={stopRuntimeLogFollow}
          onClose={() => {
            stopRuntimeLogFollow();
            setRuntimeDetail(null);
            setRuntimeLogs("");
            setRuntimeLogContainer("");
            setRuntimeLogLoaded(false);
            setRuntimeLogStatus("");
            setNodeHealth(null);
          }}
          onSaveService={saveService}
          onPatchNamespace={patchNamespaceLabels}
        />
      )}
    </div>
  );
}
createRoot(document.getElementById("root")).render(
  <I18nProvider>
    <BrowserRouter>
      <App />
    </BrowserRouter>
  </I18nProvider>,
);

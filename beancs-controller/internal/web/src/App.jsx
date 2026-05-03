import React, {useEffect, useMemo, useRef, useState} from "react";
import {createRoot} from "react-dom/client";
import {
  Activity,
  AlertTriangle,
  Bell,
  Boxes,
  Box,
  CheckCircle2,
  Cloud,
  Code2,
  Container,
  Cpu,
  Database,
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
  MemoryStick,
  Network,
  Package,
  Play,
  Plus,
  RefreshCw,
  Rocket,
  ScrollText,
  Server,
  Settings,
  Shield,
  ShieldCheck,
  Trash2,
  Upload,
} from "lucide-react";
import "./style.css";

const API = "/v1/api";
const tokenKey = "beancs.accessToken";

const emptyRuntime = {
  namespaces: [],
  pods: [],
  nodes: [],
  deployments: [],
  services: [],
  ingresses: [],
};

const navOverview = {id: "dashboard", label: "Overview", icon: LayoutDashboard};

const navSections = [
  {
    id: "workloads",
    label: "Workloads",
    items: [
      {id: "projects", label: "Projects", icon: Boxes},
      {id: "deploy", label: "Deploy", icon: Rocket},
      {id: "progress", label: "Progress", icon: LoaderCircle},
      {id: "deployments", label: "Deployments", icon: Box},
      {id: "pods", label: "Pods", icon: Layers3},
      {id: "services", label: "Services", icon: Database},
      {id: "ingresses", label: "Ingresses", icon: Network},
      {id: "workloadImage", label: "Image", icon: ImageIcon},
    ],
  },
  {
    id: "infrastructure",
    label: "Infrastructure",
    items: [
      {id: "nodes", label: "Nodes", icon: Server},
      {id: "namespaces", label: "Namespaces", icon: Layers3},
      {id: "networking", label: "Networking", icon: Network},
      {id: "storage", label: "Storage", icon: HardDrive},
    ],
  },
  {
    id: "integrations",
    label: "Integrations",
    items: [
      {id: "github", label: "GitHub", icon: Github},
      {id: "cloudflare", label: "Cloudflare", icon: Cloud},
      {id: "domains", label: "Domains", icon: Globe2},
      {id: "registries", label: "Image Registry", icon: Package},
    ],
  },
  {
    id: "security",
    label: "Security",
    items: [
      {id: "apiKeys", label: "API Keys", icon: KeyRound},
      {id: "secrets", label: "Secrets", icon: Lock},
      {id: "accessControl", label: "Access Control", icon: ShieldCheck},
    ],
  },
  {
    id: "observability",
    label: "Observability",
    items: [
      {id: "alerts", label: "Alerts", icon: Bell},
      {id: "events", label: "Events", icon: ScrollText},
      {id: "logs", label: "Logs", icon: FileText},
      {id: "metrics", label: "Metrics", icon: LineChart},
    ],
  },
  {
    id: "settings",
    label: "Settings",
    items: [{id: "settings", label: "Settings", icon: Settings}],
  },
];

function App() {
  const [config, setConfig] = useState(null);
  const [token, setToken] = useState(localStorage.getItem(tokenKey) || "");
  const [view, setView] = useState("dashboard");
  const [notice, setNotice] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [runtime, setRuntime] = useState(emptyRuntime);
  const [dashboard, setDashboard] = useState(null);
  const [network, setNetwork] = useState(null);
  const [projects, setProjects] = useState([]);
  const [credentials, setCredentials] = useState({github: [], cloudflare: [], basaltpass: []});
  const [apiKeys, setAPIKeys] = useState([]);
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
  const [activeProgressProjectID, setActiveProgressProjectID] = useState("");
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
  const progressLoadingRef = useRef(false);
  const nodeDetailLoadingRef = useRef(false);
  const registriesLoadingRef = useRef(false);

  const api = useMemo(() => makeAPI(token, logout), [token]);
  const userProfile = useMemo(() => profileFromToken(token), [token]);

  useEffect(() => {
    boot();
  }, []);

  useEffect(() => {
    if (token) loadWorkspace();
  }, [token]);

  useEffect(() => {
    if (!token || !["dashboard", "alerts", "events", "metrics"].includes(view)) return;
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
    if (!token || view !== "nodes") return;
    loadNodeJoinCommand("agent");
  }, [token, view]);

  useEffect(() => {
    if (!token || !["progress", "logs"].includes(view)) return;
    loadProjectProgress();
    const timer = setInterval(() => {
      if (!document.hidden) loadProjectProgress();
    }, 10000);
    return () => clearInterval(timer);
  }, [token, view, activeProgressProjectID, projects.length, projectLogFollow]);

  useEffect(() => {
    if (!token || runtimeDetail?.kind !== "node") return;
    const nodeName = runtimeDetail.row?.summary?.name || runtimeDetail.row?.name;
    if (!nodeName) return;
    const timer = setInterval(() => {
      if (!document.hidden) loadNodeDetail({name: nodeName}, false);
    }, 15000);
    return () => clearInterval(timer);
  }, [token, runtimeDetail?.kind, runtimeDetail?.row?.summary?.name, runtimeDetail?.row?.name]);

  useEffect(() => {
    if (!token || view !== "settings") return;
    publicJSON(`${API}/version`)
      .then((d) => setAppVersion(d.version || ""))
      .catch(() => setAppVersion(""));
  }, [token, view]);

  useEffect(() => {
    if (!token || !["deploy", "registries", "workloadImage"].includes(view)) return;
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
        const accessToken = await finishLogin(cfg);
        localStorage.setItem(tokenKey, accessToken);
        setToken(accessToken);
        history.replaceState({}, "", location.pathname);
      } else if (location.search.includes("github_app=connected")) {
        setView("github");
        setNotice("GitHub App connected.");
        history.replaceState({}, "", location.pathname);
      }
    } catch (err) {
      setError(err.message);
    }
  }

  async function loadWorkspace() {
    if (workspaceLoadingRef.current) return;
    workspaceLoadingRef.current = true;
    setLoading(true);
    setError("");
    try {
      const [runtimeData, projectData, apiKeyData, githubData, cloudflareData, domainsData, basaltpassData] = await Promise.all([
        api.get("/runtime/overview"),
        api.get("/projects"),
        api.get("/api-keys"),
        api.get("/credentials/github/"),
        api.get("/credentials/cloudflare/"),
        api.get("/credentials/cloudflare/domains"),
        api.get("/credentials/basaltpass/"),
      ]);
      setRuntime(runtimeData.data || emptyRuntime);
      setProjects(projectData.data || []);
      setAPIKeys(apiKeyData.data || []);
      setCredentials({
        github: githubData.data || [],
        cloudflare: cloudflareData.data || [],
        basaltpass: basaltpassData.data || [],
      });
      setDomains(domainsData.data || []);
    } catch (err) {
      setError(err.message);
    } finally {
      workspaceLoadingRef.current = false;
      setLoading(false);
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

  async function startLogin() {
    if (!config) return;
    const verifier = randomString(64);
    const challenge = await codeChallenge(verifier);
    const authState = randomString(32);
    sessionStorage.setItem("beancs.pkceVerifier", verifier);
    sessionStorage.setItem("beancs.oauthState", authState);
    const params = new URLSearchParams({
      response_type: "code",
      client_id: config.client_id,
      redirect_uri: browserRedirectURI(),
      scope: "openid profile email",
      state: authState,
      code_challenge: challenge,
      code_challenge_method: "S256",
    });
    location.href = trimSlash(config.auth_url) + "/oauth/authorize?" + params.toString();
  }

  function logout() {
    localStorage.removeItem(tokenKey);
    setToken("");
    setRuntime(emptyRuntime);
    setProjects([]);
  }

  async function connectGitHubApp(event) {
    event.preventDefault();
    setError("");
    const data = await api.post("/credentials/github/app/start", {});
    location.href = data.install_url;
  }

  async function createCredential(kind, event) {
    event.preventDefault();
    const body = Object.fromEntries(new FormData(event.currentTarget).entries());
    Object.keys(body).forEach((key) => {
      if (typeof body[key] === "string") body[key] = body[key].trim();
      if (body[key] === "") delete body[key];
    });
    try {
      await api.post(`/credentials/${kind}/`, body);
      event.currentTarget.reset();
      await loadWorkspace();
    } catch (err) {
      setError(err.message);
    }
  }

  async function deleteCredential(kind, id) {
    if (!confirm(`Delete this ${kind} credential?`)) return;
    try {
      await api.delete(`/credentials/${kind}/${id}`);
      await loadWorkspace();
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
    const scopes = ["beancs.api"];
    if (data.get("admin_scope") === "on") scopes.push("beancs.admin");
    const body = {
      name: String(data.get("name") || "").trim(),
      scopes,
      expires_at: localDateTimeToRFC3339(data.get("expires_at")),
    };
    try {
      const out = await api.post("/api-keys", body);
      setCreatedAPIKey(out);
      form.reset();
      await loadAPIKeys();
    } catch (err) {
      setError(err.message);
    }
  }

  async function loadAPIKeys() {
    const data = await api.get("/api-keys");
    setAPIKeys(data.data || []);
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
    if (!confirm(`删除镜像源「${row.name}」？关联的镜像跟踪也会删除。`)) return;
    try {
      await api.delete(`/container-registries/${row.id}`);
      setNotice("镜像源已删除。");
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
      setError("请选择镜像源并填写仓库路径。");
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
    if (!confirm(`从列表中移除「${row.repository}」？`)) return;
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
    if (!confirm(`Revoke API key ${key.name}? Existing clients using it will stop working.`)) return;
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
    setLoading(true);
    try {
      const data = await api.get(`/credentials/github/${credentialID}/repositories`);
      setRepos(data.data || []);
      setReposByCredential((current) => ({...current, [credentialID]: data.data || []}));
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }

  async function loadDNSRecords(credentialID = selectedCloudflareID) {
    if (!credentialID) return;
    setSelectedCloudflareID(String(credentialID));
    setLoading(true);
    setError("");
    try {
      const data = await api.get(`/credentials/cloudflare/${credentialID}/dns-records`);
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
    const body = Object.fromEntries(new FormData(event.currentTarget).entries());
    body.ttl = Number(body.ttl || 1);
    body.proxied = Boolean(body.proxied);
    try {
      if (editingDNSRecord?.id) {
        await api.put(`/credentials/cloudflare/${selectedCloudflareID}/dns-records/${editingDNSRecord.id}`, body);
      } else {
        await api.post(`/credentials/cloudflare/${selectedCloudflareID}/dns-records`, body);
      }
      event.currentTarget.reset();
      setEditingDNSRecord(null);
      await loadDNSRecords(selectedCloudflareID);
    } catch (err) {
      setError(err.message);
    }
  }

  async function deleteDNSRecord(record) {
    if (!selectedCloudflareID || !confirm(`Delete DNS record ${record.name}?`)) return;
    try {
      await api.delete(`/credentials/cloudflare/${selectedCloudflareID}/dns-records/${record.id}`);
      await loadDNSRecords(selectedCloudflareID);
    } catch (err) {
      setError(err.message);
    }
  }

  async function createNamespace(event) {
    event.preventDefault();
    const body = Object.fromEntries(new FormData(event.currentTarget).entries());
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
      await api.patch(`/runtime/namespaces/${namespace}`, {labels: parseKeyValues(labelsText)});
      await loadWorkspace();
    } catch (err) {
      setError(err.message);
    }
  }

  async function loadNamespaceDetail(namespace) {
    setRuntimeDetail({kind: "namespace-detail", row: {name: namespace}, loading: true});
    try {
      const data = await api.get(`/runtime/namespaces/${encodeURIComponent(namespace)}`);
      setRuntimeDetail({kind: "namespace-detail", row: data.data || {name: namespace}, loading: false});
    } catch (err) {
      setRuntimeDetail({kind: "namespace-detail", row: {name: namespace}, loading: false, error: err.message});
    }
  }

  async function saveResourceQuota(namespace, event) {
    event.preventDefault();
    const body = Object.fromEntries(new FormData(event.currentTarget).entries());
    body.hard = parseKeyValues(body.hard);
    try {
      await api.put(`/runtime/namespaces/${encodeURIComponent(namespace)}/resource-quotas`, body);
      await loadNamespaceDetail(namespace);
      await loadWorkspace();
    } catch (err) {
      setError(err.message);
    }
  }

  async function deleteResourceQuota(namespace, name) {
    if (!confirm(`Delete ResourceQuota ${name}?`)) return;
    try {
      await api.delete(`/runtime/namespaces/${encodeURIComponent(namespace)}/resource-quotas/${encodeURIComponent(name)}`);
      await loadNamespaceDetail(namespace);
    } catch (err) {
      setError(err.message);
    }
  }

  async function saveLimitRange(namespace, event) {
    event.preventDefault();
    const body = Object.fromEntries(new FormData(event.currentTarget).entries());
    body.default = parseKeyValues(body.default);
    body.default_request = parseKeyValues(body.default_request);
    body.min = parseKeyValues(body.min);
    body.max = parseKeyValues(body.max);
    try {
      await api.put(`/runtime/namespaces/${encodeURIComponent(namespace)}/limit-ranges`, body);
      await loadNamespaceDetail(namespace);
    } catch (err) {
      setError(err.message);
    }
  }

  async function deleteLimitRange(namespace, name) {
    if (!confirm(`Delete LimitRange ${name}?`)) return;
    try {
      await api.delete(`/runtime/namespaces/${encodeURIComponent(namespace)}/limit-ranges/${encodeURIComponent(name)}`);
      await loadNamespaceDetail(namespace);
    } catch (err) {
      setError(err.message);
    }
  }

  async function saveNamespacePermission(namespace, event) {
    event.preventDefault();
    const body = Object.fromEntries(new FormData(event.currentTarget).entries());
    body.verbs = parseCSV(body.verbs);
    body.resources = parseCSV(body.resources);
    body.api_groups = parseCSV(body.api_groups);
    body.subjects = parsePermissionSubjects(body.subjects, namespace);
    try {
      await api.put(`/runtime/namespaces/${encodeURIComponent(namespace)}/permissions`, body);
      await loadNamespaceDetail(namespace);
    } catch (err) {
      setError(err.message);
    }
  }

  async function deleteNamespacePermission(namespace, name) {
    if (!confirm(`Delete namespace permission ${name}?`)) return;
    try {
      await api.delete(`/runtime/namespaces/${encodeURIComponent(namespace)}/permissions/${encodeURIComponent(name)}`);
      await loadNamespaceDetail(namespace);
    } catch (err) {
      setError(err.message);
    }
  }

  async function saveNamespaceIsolation(namespace, event) {
    event.preventDefault();
    const form = event.currentTarget;
    try {
      await api.put(`/runtime/namespaces/${encodeURIComponent(namespace)}/isolation`, {
        enabled: Boolean(form.enabled.checked),
        allow_same_namespace: Boolean(form.allow_same_namespace.checked),
        allow_dns: Boolean(form.allow_dns.checked),
      });
      await loadNamespaceDetail(namespace);
      await loadNetwork();
    } catch (err) {
      setError(err.message);
    }
  }

  async function deleteNamespace(namespace) {
    if (!confirm(`Delete namespace ${namespace}? This removes resources inside it.`)) return;
    try {
      await api.delete(`/runtime/namespaces/${namespace}`);
      await loadWorkspace();
    } catch (err) {
      setError(err.message);
    }
  }

  async function deletePod(pod) {
    if (!confirm(`Delete pod ${pod.name}? Kubernetes may recreate it.`)) return;
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
    if (showModal) setRuntimeDetail({kind: "node", row: node, loading: true});
    if (showModal) setNodeHealth(null);
    try {
      const data = await api.get(`/runtime/nodes/${encodeURIComponent(node.name)}`);
      setRuntimeDetail({kind: "node", row: data.data || node, loading: false});
    } catch (err) {
      setRuntimeDetail({kind: "node", row: node, loading: false, error: err.message});
    } finally {
      nodeDetailLoadingRef.current = false;
    }
  }

  async function loadNodeHealth(nodeName) {
    try {
      const data = await api.get(`/runtime/nodes/${encodeURIComponent(nodeName)}/health`);
      setNodeHealth(data.data || null);
    } catch (err) {
      setError(err.message);
    }
  }

  async function loadNodeJoinCommand(role = "agent") {
    try {
      const data = await api.get(`/runtime/nodes/join-command?role=${encodeURIComponent(role)}`);
      setNodeJoinCommand(data.data || null);
    } catch (err) {
      setError(err.message);
    }
  }

  async function saveNodeLabels(nodeName, labelsText) {
    try {
      await api.patch(`/runtime/nodes/${encodeURIComponent(nodeName)}/labels`, {labels: parseKeyValues(labelsText)});
      setNotice(`${nodeName} labels updated.`);
      await loadWorkspace();
      await loadNodeDetail({name: nodeName}, false);
    } catch (err) {
      setError(err.message);
    }
  }

  async function saveNodeTaints(nodeName, taintsText) {
    try {
      await api.put(`/runtime/nodes/${encodeURIComponent(nodeName)}/taints`, {taints: parseTaints(taintsText)});
      setNotice(`${nodeName} taints updated.`);
      await loadWorkspace();
      await loadNodeDetail({name: nodeName}, false);
    } catch (err) {
      setError(err.message);
    }
  }

  async function cordonNode(nodeName, schedulable) {
    try {
      await api.post(`/runtime/nodes/${encodeURIComponent(nodeName)}/${schedulable ? "uncordon" : "cordon"}`, {});
      setNotice(`${nodeName} ${schedulable ? "uncordoned" : "cordoned"}.`);
      await loadWorkspace();
      await loadNodeDetail({name: nodeName}, false);
    } catch (err) {
      setError(err.message);
    }
  }

  async function drainNode(nodeName, options) {
    if (!confirm(`Drain node ${nodeName}? Workloads will be evicted from this node.`)) return;
    try {
      const data = await api.post(`/runtime/nodes/${encodeURIComponent(nodeName)}/drain`, options);
      setNotice(`Drain started for ${nodeName}: ${(data.data?.evicted_pods || []).length} pods evicted, ${(data.data?.skipped_pods || []).length} skipped.`);
      await loadWorkspace();
      await loadNodeDetail({name: nodeName}, false);
    } catch (err) {
      setError(err.message);
    }
  }

  async function deleteNode(nodeName) {
    if (!confirm(`Delete node ${nodeName}? Make sure it has been drained and removed from the K3s host first.`)) return;
    try {
      await api.delete(`/runtime/nodes/${encodeURIComponent(nodeName)}`);
      setRuntimeDetail(null);
      setNotice(`${nodeName} deleted from the cluster.`);
      await loadWorkspace();
    } catch (err) {
      setError(err.message);
    }
  }

  async function loadPodLogs(pod) {
    stopRuntimeLogFollow();
    setRuntimeDetail({kind: "pod", row: pod});
    setRuntimeLogs("");
    setRuntimeLogContainer("");
    setRuntimeLogTail(200);
    setRuntimeLogLoaded(false);
    setRuntimeLogStatus("Choose a container to load logs. Logs are loaded lazily to keep the browser responsive.");
  }

  async function loadRuntimeContainerLogs(pod, container = runtimeLogContainer, tail = runtimeLogTail) {
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
      const data = await api.get(`/runtime/pods/${namespace}/${name}/logs?tail=${Number(tail || 200)}&container=${selected}`);
      setRuntimeLogs(trimLiveLog(data.logs || ""));
      setRuntimeLogStatus(`Loaded last ${Number(tail || 200)} lines from ${container}.`);
    } catch (err) {
      setRuntimeLogs("");
      setRuntimeLogStatus(err.message);
    }
  }

  async function startRuntimeLogFollow(pod, container = runtimeLogContainer, tail = runtimeLogTail) {
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
      const res = await api.stream(`/runtime/pods/${namespace}/${name}/logs?tail=${Number(tail || 200)}&container=${selected}&follow=true`, {signal: controller.signal});
      setRuntimeLogStatus(`Following live logs for ${container}`);
      await consumeTextStream(res, (chunk) => setRuntimeLogs((current) => trimLiveLog(current + chunk)));
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
    const body = Object.fromEntries(new FormData(event.currentTarget).entries());
    body.selector = parseKeyValues(body.selector);
    body.labels = parseKeyValues(body.labels);
    body.ports = parseServicePorts(body.ports);
    body.external_ips = parseCSV(body.external_ips);
    try {
      if (existing) {
        await api.put(`/runtime/services/${existing.namespace}/${existing.name}`, body);
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
    if (!confirm(`Delete service ${service.name}?`)) return;
    try {
      await api.delete(`/runtime/services/${service.namespace}/${service.name}`);
      await loadWorkspace();
    } catch (err) {
      setError(err.message);
    }
  }

  async function saveIngress(event, existing = null) {
    event.preventDefault();
    const body = Object.fromEntries(new FormData(event.currentTarget).entries());
    body.service_port = Number(body.service_port || 80);
    body.annotations = parseKeyValues(body.annotations);
    body.labels = parseKeyValues(body.labels);
    try {
      if (existing) {
        await api.put(`/runtime/ingresses/${existing.namespace}/${existing.name}`, body);
      } else {
        await api.post("/runtime/ingresses", body);
        event.currentTarget.reset();
      }
      setNotice(`Ingress ${body.name || existing?.name} saved.`);
      await loadWorkspace();
    } catch (err) {
      setError(err.message);
    }
  }

  async function deleteIngress(ingress) {
    if (!confirm(`Delete ingress ${ingress.namespace}/${ingress.name}?`)) return;
    try {
      await api.delete(`/runtime/ingresses/${ingress.namespace}/${ingress.name}`);
      await loadWorkspace();
    } catch (err) {
      setError(err.message);
    }
  }

  async function saveNetworkPolicy(event, existing = null) {
    event.preventDefault();
    const body = Object.fromEntries(new FormData(event.currentTarget).entries());
    body.pod_selector = parseKeyValues(body.pod_selector);
    body.labels = parseKeyValues(body.labels);
    body.policy_types = Array.from(event.currentTarget.querySelectorAll("input[name='policy_types']:checked")).map((input) => input.value);
    body.allow_same_namespace = Boolean(body.allow_same_namespace);
    body.allow_dns = Boolean(body.allow_dns);
    try {
      if (existing) {
        await api.put(`/runtime/network-policies/${existing.namespace}/${existing.name}`, body);
      } else {
        await api.post("/runtime/network-policies", body);
        event.currentTarget.reset();
      }
      setNotice(`NetworkPolicy ${body.name || existing?.name} saved.`);
      await loadWorkspace();
    } catch (err) {
      setError(err.message);
    }
  }

  async function deleteNetworkPolicy(policy) {
    if (!confirm(`Delete NetworkPolicy ${policy.namespace}/${policy.name}?`)) return;
    try {
      await api.delete(`/runtime/network-policies/${policy.namespace}/${policy.name}`);
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
	const data = await api.post("/projects/analyze", {
	  github_credential_id: Number(selectedCredential),
	  github_repo: repoFullName,
	  github_branch: branch,
	});
    setAnalysis(data);
    setSelectedRepo(repoFullName);
    setDeployForm((current) => ({
	  ...current,
	  name: slugify(repoFullName.split("/")[1] || repoFullName),
	  github_repo: repoFullName,
	  github_branch: branch,
	  dockerfile_path: data.dockerfile_path || "",
	  port: data.default_port || 8080,
	}));
  }

  function checkInstallSource(nextForm = deployForm) {
    const source = nextForm.deploy_source === "gitops" ? "github" : "registry";
    setError("");
    setNotice("");
    if (source === "github") {
      return analyzeRepo(nextForm.github_repo || selectedRepo, nextForm.github_branch);
    }
    const image = (nextForm.image_reference || "").trim();
    if (!image) {
      setError("Image reference is required for registry deployments.");
      return;
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
      setDeployForm((current) => ({...current, name: slugify(imageName(image))}));
    }
  }

  async function deployProject(event) {
    event.preventDefault();
    if (!analysis?.deployable) return;
    const payload = buildProjectPayload(deployForm, selectedCredential, credentials);
    setLoading(true);
    setError("");
    setInstallProgress({
      project: payload.name,
      started_at: new Date().toISOString(),
      steps: [
        {label: "Validate install source", state: "done"},
        {label: "Create namespace and secrets", state: "running"},
        {label: "Apply service and ingress", state: "pending"},
        {label: "Apply deployment or GitOps manifests", state: "pending"},
      ],
    });
    setView("progress");
    try {
      const created = await api.post("/projects", payload);
      setNotice("Project created. BeanCS is preparing GitOps manifests and traffic routes.");
      setActiveProgressProjectID(String(created.id));
      setInstallProgress((current) => current ? {
        ...current,
        steps: current.steps.map((step) => ({...step, state: "done"})),
      } : null);
      setDeployForm(defaultDeployForm());
      setAnalysis(null);
      setSelectedRepo("");
      await loadWorkspace();
      await loadProjectProgress(String(created.id));
    } catch (err) {
      setError(err.message);
      setInstallProgress((current) => current ? {
        ...current,
        steps: current.steps.map((step) => step.state === "running" ? {...step, state: "failed"} : step),
      } : null);
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
      const logRequest = projectLogFollow ? Promise.resolve({logs: projectProgress?.logs || ""}) : api.get(`/projects/${selected.id}/logs?tail=160`);
      const [status, deployments, logData] = await Promise.all([
        api.get(`/projects/${selected.id}/status`),
        api.get(`/projects/${selected.id}/deployments`),
        logRequest,
      ]);
      setProjectProgress({
        project: selected,
        pods: status.pods || [],
        deployment: status.deployment || null,
        services: status.services || [],
        ingresses: status.ingresses || [],
        events: status.events || [],
        deployments: deployments.data || [],
        logs: logData.logs || "",
        checked_at: new Date().toISOString(),
      });
    } catch (err) {
      setProjectProgress({project: selected, pods: [], deployments: [], error: err.message, checked_at: new Date().toISOString()});
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
      setProjectLogStatus("Choose a project before following logs.");
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
      const res = await api.stream(`/projects/${selected.id}/logs?tail=160&follow=true`, {signal: controller.signal});
      setProjectLogStatus("Following live logs");
      await consumeTextStream(res, (chunk) => setProjectLiveLogs((current) => trimLiveLog(current + chunk)));
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

  async function updateProject(event) {
    event.preventDefault();
    const body = Object.fromEntries(new FormData(event.currentTarget).entries());
    body.replicas = Number(body.replicas || 1);
    body.auto_deploy = body.auto_deploy === "on";
    await api.patch(`/projects/${editingProject.id}`, body);
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
      setNotice(`${deletingProject.name} deleted.`);
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

  async function scaleProject(project, replicas) {
    await api.post(`/projects/${project.id}/scale`, {replicas});
    await loadWorkspace();
  }

  async function restartProject(project) {
    await api.post(`/projects/${project.id}/restart`, {});
    setNotice(`${project.name} restarted.`);
  }

  async function buildProject(project) {
    try {
      await api.post(`/projects/${project.id}/deployments`, {tag: "github-actions", commit_sha: project.github_branch || ""});
      setNotice(`${project.name} build started.`);
      setActiveProgressProjectID(String(project.id));
      setView("progress");
      await loadProjectProgress(String(project.id));
    } catch (err) {
      setError(err.message);
    }
  }

  if (!token) {
    return (
      <main className="login-screen">
        <section className="login-copy">
          <h1>BeanCS</h1>
          <p>Operate k3s projects, GitHub App deployments, DNS, and traffic routes from one console.</p>
          <button className="primary" onClick={startLogin}>
            <Lock size={18} /> Sign in with BasaltPass
          </button>
          {error && <p className="error-text">{error}</p>}
        </section>
      </main>
    );
  }

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div className="brand">BeanCS</div>
        <div className="sidebar-nav">
          {(() => {
            const OverviewIcon = navOverview.icon;
            return (
              <button type="button" className={view === navOverview.id ? "nav nav-featured active" : "nav nav-featured"} onClick={() => setView(navOverview.id)}>
                <OverviewIcon size={18} /> {navOverview.label}
              </button>
            );
          })()}
          {navSections.map((section) => (
            <div className="nav-section" key={section.id}>
              <div className="nav-section-label">{section.label}</div>
              {section.items.map((item) => {
                const Icon = item.icon;
                return (
                  <button key={item.id} type="button" className={view === item.id ? "nav active" : "nav"} onClick={() => setView(item.id)}>
                    <Icon size={16} /> {item.label}
                  </button>
                );
              })}
            </div>
          ))}
        </div>
        <div className="sidebar-user">
          <div className="user-avatar">{userProfile.initial}</div>
          <div className="user-copy">
            <b>{userProfile.name}</b>
            <span>{userProfile.detail}</span>
          </div>
          <button className="signout-button" onClick={logout}>Sign out</button>
        </div>
      </aside>
      <main className="workspace">
        <header className="topbar">
          <div>
            <h1>{titleFor(view)}</h1>
            <p>{subtitleFor(view, runtime, projects)}</p>
          </div>
          <div className="top-actions">
            <button onClick={loadWorkspace} disabled={loading}><RefreshCw size={16} /> Refresh</button>
          </div>
        </header>
        {notice && <div className="notice">{notice}</div>}
        {error && <div className="alert">{error}</div>}
        {view === "dashboard" && <DashboardView dashboard={dashboard} refresh={loadDashboard} />}
        {view === "deploy" && (
          <DeployView
            credentials={credentials}
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
            createTrackedImageFromDeploy={createTrackedImageFromDeploy}
          />
        )}
        {view === "progress" && (
          <ProgressView
            projects={projects}
            activeProjectID={activeProgressProjectID}
            setActiveProjectID={setActiveProgressProjectID}
            progress={projectProgress}
            installProgress={installProgress}
            refresh={loadProjectProgress}
            logFollow={projectLogFollow}
            liveLogs={projectLiveLogs}
            logStatus={projectLogStatus}
            onStartLogFollow={startProjectLogFollow}
            onStopLogFollow={stopProjectLogFollow}
          />
        )}
        {view === "projects" && (
          <ProjectsView projects={projects} onEdit={setEditingProject} onDelete={deleteProject} onScale={scaleProject} onRestart={restartProject} onBuild={buildProject} onProgress={(project) => { setActiveProgressProjectID(String(project.id)); setView("progress"); }} />
        )}
        {view === "apiKeys" && <APIKeysView keys={apiKeys} createdKey={createdAPIKey} onDismissCreated={() => setCreatedAPIKey(null)} onCreate={createAPIKey} onRevoke={revokeAPIKey} onRefresh={loadAPIKeys} isAdmin={userProfile.scopes.includes("beancs.admin")} />}
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
            onOpenRegistry={() => setView("registries")}
            onRefreshImage={refreshTrackedImage}
            onDeleteImage={deleteTrackedImage}
          />
        )}
        {view === "storage" && (
          <ComingSoonView
            title="Storage"
            description="PersistentVolumeClaims, PersistentVolumes, and StorageClasses will be manageable here in a future release."
          />
        )}
        {view === "secrets" && (
          <ComingSoonView
            title="Secrets"
            description="Kubernetes Secret inspection and rotation workflows are not wired in this console yet. Use kubectl or your GitOps pipeline for now."
          />
        )}
        {view === "alerts" && <AlertsView dashboard={dashboard} refresh={loadDashboard} />}
        {view === "events" && <EventsView dashboard={dashboard} refresh={loadDashboard} />}
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
            onOpenPods={() => setView("pods")}
          />
        )}
        {view === "metrics" && <MetricsView dashboard={dashboard} runtime={runtime} refresh={loadDashboard} />}
        {view === "settings" && <SettingsView version={appVersion} />}
        {view === "github" && (
          <GitHubView credentials={credentials.github} onConnect={connectGitHubApp} onRepos={loadRepos} onDelete={(id) => deleteCredential("github", id)} reposByCredential={reposByCredential} repoFilters={repoFilters} setRepoFilters={setRepoFilters} />
        )}
        {view === "domains" && <DomainsView domains={domains} />}
        {view === "networking" && <NetworkingView network={network} refresh={loadNetwork} onSaveService={saveService} onDeleteService={deleteService} onSaveIngress={saveIngress} onDeleteIngress={deleteIngress} onSaveNetworkPolicy={saveNetworkPolicy} onDeleteNetworkPolicy={deleteNetworkPolicy} />}
        {view === "cloudflare" && <CloudflareView credentials={credentials.cloudflare} domains={domains} selectedID={selectedCloudflareID} setSelectedID={setSelectedCloudflareID} dnsRecords={dnsRecords} editingRecord={editingDNSRecord} setEditingRecord={setEditingDNSRecord} onCreate={createCredential} onDelete={(id) => deleteCredential("cloudflare", id)} onLoadDNS={loadDNSRecords} onSaveDNS={saveDNSRecord} onDeleteDNS={deleteDNSRecord} />}
        {view === "accessControl" && <CredentialManager kind="basaltpass" rows={credentials.basaltpass} onCreate={createCredential} onDelete={deleteCredential} />}
        {["namespaces", "pods", "nodes", "ingresses", "services", "deployments"].includes(view) && <RuntimeTable kind={view} rows={runtime[view] || []} nodeJoinCommand={nodeJoinCommand} onLoadNodeJoinCommand={loadNodeJoinCommand} onCreateNamespace={createNamespace} onPatchNamespace={patchNamespaceLabels} onNamespaceDetail={loadNamespaceDetail} onDeleteNamespace={deleteNamespace} onDeletePod={deletePod} onNodeDetail={loadNodeDetail} onPodLogs={loadPodLogs} onSaveService={saveService} onDeleteService={deleteService} onDetail={setRuntimeDetail} />}
      </main>
      {editingProject && <ProjectModal project={editingProject} onClose={() => setEditingProject(null)} onSubmit={updateProject} />}
      {deletingProject && <DeleteProjectModal project={deletingProject} busy={loading} onClose={() => setDeletingProject(null)} onDelete={confirmDeleteProject} />}
      {runtimeDetail && <RuntimeDetailModal detail={runtimeDetail} logs={runtimeLogs} logFollow={runtimeLogFollow} logStatus={runtimeLogStatus} selectedLogContainer={runtimeLogContainer} logTail={runtimeLogTail} logLoaded={runtimeLogLoaded} nodeHealth={nodeHealth} onLoadNodeHealth={loadNodeHealth} onSaveNodeLabels={saveNodeLabels} onSaveNodeTaints={saveNodeTaints} onCordonNode={cordonNode} onDrainNode={drainNode} onDeleteNode={deleteNode} onSaveResourceQuota={saveResourceQuota} onDeleteResourceQuota={deleteResourceQuota} onSaveLimitRange={saveLimitRange} onDeleteLimitRange={deleteLimitRange} onSaveNamespacePermission={saveNamespacePermission} onDeleteNamespacePermission={deleteNamespacePermission} onSaveNamespaceIsolation={saveNamespaceIsolation} onSelectLogContainer={setRuntimeLogContainer} onSetLogTail={setRuntimeLogTail} onLoadContainerLogs={loadRuntimeContainerLogs} onFollowPodLogs={startRuntimeLogFollow} onStopPodLogs={stopRuntimeLogFollow} onClose={() => { stopRuntimeLogFollow(); setRuntimeDetail(null); setRuntimeLogs(""); setRuntimeLogContainer(""); setRuntimeLogLoaded(false); setRuntimeLogStatus(""); setNodeHealth(null); }} onSaveService={saveService} onPatchNamespace={patchNamespaceLabels} />}
    </div>
  );
}

function DeployView({credentials, namespaces, selectedCredential, setSelectedCredential, repos, selectedRepo, analysis, setAnalysis, form, setForm, loadRepos, analyzeRepo, checkInstallSource, deployProject, containerRegistries, containerImages, createTrackedImageFromDeploy}) {
  const [stepIndex, setStepIndex] = useState(0);
  const [creatingImage, setCreatingImage] = useState(false);
  const selectedCloudflare = credentials.cloudflare.find((cred) => String(cred.id) === String(form.cloudflare_credential_id));
  const publicHost = form.subdomain && selectedCloudflare ? `${form.subdomain}.${selectedCloudflare.domain}` : "";
  const step = deploySteps[stepIndex];
  const canContinue = canContinueDeployStep(step.id, form, selectedCredential, analysis);
  const ghcrPreview = form.github_repo ? `ghcr.io/${form.github_repo.toLowerCase()}:beancs-<build>` : "ghcr.io/<owner>/<repo>:beancs-<build>";
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
  };
  const updateSourceForm = (nextForm) => {
    setAnalysis(null);
    setForm(nextForm);
  };
  const setRepoType = (repoType) => {
    setAnalysis(null);
    setForm({...form, repo_type: repoType, github_repo: "", git_url: "", update_mode: repoType === "github" ? form.update_mode || "argocd" : "passive"});
  };
  const setUpdateMode = (updateMode) => {
    setForm({...form, update_mode: form.deploy_source === "registry" ? "passive" : updateMode, auto_deploy: updateMode === "argocd"});
  };
  const selectTrackedImage = (image, tag = "") => {
    const ref = imageReferenceFromTrackedImage(image, tag);
    updateSourceForm({...form, selected_image_id: String(image.id), image_reference: ref, name: form.name || slugify(imageName(ref))});
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
  const next = () => {
    if (step.id === "check") checkInstallSource(form);
    if (stepIndex < deploySteps.length - 1) setStepIndex(stepIndex + 1);
  };
  const back = () => setStepIndex(Math.max(0, stepIndex - 1));
  return (
    <div className="deploy-wizard">
      <section className="panel wizard-progress-panel">
        <div className="wizard-progress-head">
          <span>{step.label}</span>
          <b>{stepIndex + 1} / {deploySteps.length}</b>
        </div>
        <div className="wizard-progress-track">
          <span style={{width: `${((stepIndex + 1) / deploySteps.length) * 100}%`}} />
        </div>
        <div className="wizard-step-labels">
          {deploySteps.map((item, index) => (
            <span key={item.id} className={index === stepIndex ? "active" : index < stepIndex ? "done" : ""}>{item.label}</span>
          ))}
        </div>
      </section>
      <form className="panel deploy-form wizard-panel" onSubmit={deployProject}>
        <h2><Rocket size={18} /> {step.title}</h2>
        {step.id === "method" && (
          <div className="method-grid">
            {deploySourceOptions.map((method) => {
              const Icon = method.icon;
              return (
                <button key={method.id} type="button" className={form.deploy_source === method.id ? "method-card active" : "method-card"} onClick={() => setDeploySource(method.id)}>
                  <Icon size={22} />
                  <b>{method.label}</b>
                  <span>{method.description}</span>
                </button>
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
                  <button type="button" className={form.repo_type === "github" ? "active" : ""} onClick={() => setRepoType("github")}><Github size={15} /> GitHub</button>
                  <button type="button" className={form.repo_type === "git-url" ? "active" : ""} onClick={() => setRepoType("git-url")}><GitBranch size={15} /> Git link</button>
                </div>
                {form.repo_type === "github" && (
                  <>
                    <label>GitHub credential</label>
                    <select value={selectedCredential} onChange={(event) => { setAnalysis(null); setSelectedCredential(event.target.value); loadRepos(event.target.value); }}>
                      <option value="">Choose credential</option>
                      {credentials.github.map((cred) => <option key={cred.id} value={cred.id}>{cred.name} ({cred.account_login || cred.auth_type})</option>)}
                    </select>
                    <Field label="Repository" value={form.github_repo} onChange={(v) => updateSourceForm({...form, github_repo: v.trim()})} required />
                    <Field label="Branch" value={form.github_branch} onChange={(v) => updateSourceForm({...form, github_branch: v})} />
                    <div className="repo-list compact-repos">
                      {repos.map((repo) => (
                        <button key={repo.full_name} type="button" className={selectedRepo === repo.full_name ? "repo active" : "repo"} onClick={() => { setForm({...form, github_repo: repo.full_name, github_branch: repo.default_branch || "main", name: slugify(repo.name || repo.full_name.split("/")[1])}); analyzeRepo(repo.full_name, repo.default_branch); }}>
                          <Code2 size={15} />
                          <span>{repo.full_name}</span>
                          <small>{repo.private ? "Private" : "Public"} · {repo.default_branch}</small>
                        </button>
                      ))}
                    </div>
                  </>
                )}
                {form.repo_type === "git-url" && (
                  <>
                    <Field label="Git URL" value={form.git_url} onChange={(v) => updateSourceForm({...form, git_url: v.trim()})} required />
                    <p className="warning-note">当前部署模式展示不支持直接的 git 链接。请改用已连接的 GitHub 仓库继续部署。</p>
                  </>
                )}
              </>
            )}
            {form.deploy_source === "registry" && (
              <>
                <label>Image object</label>
                <div className="segmented-control">
                  <button type="button" className={form.image_choice === "existing" ? "active" : ""} onClick={() => updateSourceForm({...form, image_choice: "existing", image_reference: ""})}><Package size={15} /> Existing</button>
                  <button type="button" className={form.image_choice === "new" ? "active" : ""} onClick={() => updateSourceForm({...form, image_choice: "new", selected_image_id: "", image_reference: ""})}><Plus size={15} /> New object</button>
                </div>
                {form.image_choice === "existing" && (
                  <>
                    <div className="image-picker">
                      {containerImages.map((image) => (
                        <button key={image.id} type="button" className={String(form.selected_image_id) === String(image.id) ? "image-option active" : "image-option"} onClick={() => selectTrackedImage(image, (image.tags || [])[0] || "")}>
                          <b>{image.repository}</b>
                          <span>{image.registry?.name || `registry #${image.registry_id}`}</span>
                          <small>{(image.tags || []).length ? `${(image.tags || []).length} tags cached` : "No cached tags"}</small>
                        </button>
                      ))}
                      {containerImages.length === 0 && <div className="empty">No image objects yet. Create one below or open Image Registry.</div>}
                    </div>
                    {form.selected_image_id && (
                      <>
                        <label>Tag</label>
                        <select value={imageTagFromReference(form.image_reference)} onChange={(event) => {
                          const image = containerImages.find((item) => String(item.id) === String(form.selected_image_id));
                          if (image) selectTrackedImage(image, event.target.value);
                        }}>
                          {(containerImages.find((item) => String(item.id) === String(form.selected_image_id))?.tags || ["latest"]).map((tag) => <option key={tag} value={tag}>{tag}</option>)}
                        </select>
                      </>
                    )}
                    <Field label="Image reference" value={form.image_reference} onChange={(v) => updateSourceForm({...form, image_reference: v.trim(), name: form.name || slugify(imageName(v))})} required />
                  </>
                )}
                {form.image_choice === "new" && (
                  <>
                    <label>Registry</label>
                    <select value={form.new_image_registry_id} onChange={(event) => updateSourceForm({...form, new_image_registry_id: event.target.value})} required>
                      <option value="">Choose registry</option>
                      {containerRegistries.map((registry) => <option key={registry.id} value={registry.id}>{registry.name} ({registry.kind})</option>)}
                    </select>
                    <Field label="Repository path" value={form.new_image_repository} onChange={(v) => updateSourceForm({...form, new_image_repository: v.trim()})} required />
                    <button type="button" className="primary inline-primary" disabled={creatingImage || !form.new_image_registry_id || !form.new_image_repository} onClick={createImage}><Plus size={15} /> Create image object</button>
                    <p className="muted">保存对象后会回到对象选择，并使用该镜像进行被动更新部署。</p>
                  </>
                )}
              </>
            )}
          </div>
        )}
        {step.id === "update" && (
          <div className="method-grid two-up">
            {form.deploy_source === "gitops" && updateModeOptions.map((mode) => {
              const Icon = mode.icon;
              return (
                <button key={mode.id} type="button" className={form.update_mode === mode.id ? "method-card active" : "method-card"} onClick={() => setUpdateMode(mode.id)}>
                  <Icon size={22} />
                  <b>{mode.label}</b>
                  <span>{mode.description}</span>
                </button>
              );
            })}
            {form.deploy_source === "registry" && (
              <button type="button" className="method-card active" onClick={() => setUpdateMode("passive")}>
                <RefreshCw size={22} />
                <b>Passive update</b>
                <span>Registry deployments only support passive updates in the current flow.</span>
              </button>
            )}
          </div>
        )}
        {step.id === "check" && (
          <div className="readiness-card">
            <button type="button" className="primary" onClick={() => checkInstallSource(form)}><Shield size={16} /> Check installability</button>
            {!analysis && <p className="muted">BeanCS will verify repository signals or image/source inputs before continuing.</p>}
            {analysis && (
              <>
                <div className={analysis.deployable ? "status good" : "status bad"}>{analysis.containerized ? "Deployable" : analysis.scaffoldable ? "Source detected" : "Needs containerization"}</div>
                <div className="signal-list">
                  {(analysis.signals || []).map((signal) => <span key={signal}>{signal}</span>)}
                  {analysis.compose_path && <span>Compose: {analysis.compose_path}</span>}
                  {analysis.ports?.length > 0 && <span>Ports: {analysis.ports.join(", ")}</span>}
                  {(analysis.warnings || []).map((warning) => <span className="warning" key={warning}>{warning}</span>)}
                </div>
              </>
            )}
          </div>
        )}
        {step.id === "params" && (
          <div className="form-grid">
            <Field label="Project name" value={form.name} onChange={(v) => setForm({...form, name: slugify(v)})} required />
            <Field label="Port" type="number" value={form.port} onChange={(v) => setForm({...form, port: Number(v)})} />
            <Field label="Replicas" type="number" value={form.replicas} onChange={(v) => setForm({...form, replicas: Number(v)})} />
            <label>Resource preset</label>
            <select value={form.resource_preset} onChange={(event) => setForm({...form, resource_preset: event.target.value})}>
              <option value="nano">Nano</option>
              <option value="small">Small</option>
              <option value="medium">Medium</option>
              <option value="large">Large</option>
            </select>
            <label>BasaltPass optional</label>
            <select value={form.basaltpass_instance_id} onChange={(event) => setForm({...form, basaltpass_instance_id: event.target.value})}>
              <option value="">Do not register OAuth app</option>
              {credentials.basaltpass.map((cred) => <option key={cred.id} value={cred.id}>{cred.name}</option>)}
            </select>
          </div>
        )}
        {step.id === "namespace" && (
          <div className="form-grid">
            <label>Namespace</label>
            <input list="namespace-options" value={form.namespace} placeholder={form.name ? `proj-${form.name}` : "proj-my-app"} onChange={(event) => setForm({...form, namespace: slugify(event.target.value)})} />
            <datalist id="namespace-options">
              {namespaces.map((ns) => <option key={ns.name} value={ns.name} />)}
            </datalist>
            <p className="muted">Leave empty to create {form.name ? <b>proj-{form.name}</b> : "a project namespace"} automatically.</p>
          </div>
        )}
        {step.id === "ingress" && (
          <div className="form-grid">
            <label>Traffic</label>
            <select value={form.exposure_mode} onChange={(event) => setForm({...form, exposure_mode: event.target.value})}>
              <option value="public">Traefik public ingress</option>
              <option value="private">Tailscale private ingress</option>
              <option value="internal-only">Cluster internal only</option>
            </select>
          </div>
        )}
        {step.id === "domain" && (
          <div className="form-grid">
            {form.exposure_mode === "public" && (
              <>
                <label>Cloudflare credential</label>
                <select value={form.cloudflare_credential_id} onChange={(event) => setForm({...form, cloudflare_credential_id: event.target.value})} required>
                  <option value="">Choose Cloudflare zone</option>
                  {credentials.cloudflare.map((cred) => <option key={cred.id} value={cred.id}>{cred.name} · {cred.domain}</option>)}
                </select>
                <Field label="Subdomain" value={form.subdomain} onChange={(v) => setForm({...form, subdomain: slugify(v)})} required />
                <div className="computed-host">{publicHost || "Subdomain preview"}</div>
              </>
            )}
            {form.exposure_mode === "private" && <Field label="Tailscale host" value={form.private_host} onChange={(v) => setForm({...form, private_host: v.trim().toLowerCase()})} required />}
            {form.exposure_mode === "internal-only" && <p className="muted">No domain is required for internal-only projects.</p>}
          </div>
        )}
        {step.id === "confirm" && (
          <div className="detail-list">
            <span>Install method <b>{sourceLabel(form.build_source)}</b></span>
            <span>Source <b>{sourceSummary(form)}</b></span>
            <span>Project <b>{form.name || "-"}</b></span>
            <span>Namespace <b>{form.namespace || (form.name ? `proj-${form.name}` : "-")}</b></span>
            <span>Ingress <b>{form.exposure_mode}</b></span>
            <span>Domain <b>{publicHost || form.private_host || "internal only"}</b></span>
            <span>Port <b>{form.port}</b></span>
            <span>Update mode <b>{form.deploy_source === "registry" ? "Passive" : form.update_mode === "argocd" ? "Argo CD" : "Passive"}</b></span>
            {form.deploy_source === "gitops" && form.update_mode === "argocd" && <span>Future GHCR image <b>{ghcrPreview}</b></span>}
          </div>
        )}
        <div className="wizard-actions">
          <button type="button" onClick={back} disabled={stepIndex === 0}>Back</button>
          {step.id === "confirm" ? (
            <button className="primary" disabled={!analysis?.deployable} type="submit"><Play size={16} /> Build</button>
          ) : (
            <button className="primary" type="button" disabled={!canContinue} onClick={next}>Next</button>
          )}
        </div>
      </form>
    </div>
  );
}

function ProgressView({projects, activeProjectID, setActiveProjectID, progress, installProgress, refresh, logFollow, liveLogs, logStatus, onStartLogFollow, onStopLogFollow}) {
  const pods = progress?.pods || [];
  const events = progress?.events || [];
  const deployments = progress?.deployments || [];
  const readyPods = pods.filter((pod) => Number(pod.ready_containers || 0) > 0 && Number(pod.ready_containers) === Number(pod.total_containers || 0)).length;
  const desiredReplicas = progress?.deployment?.replicas ?? progress?.project?.replicas ?? 0;
  const readyReplicas = progress?.deployment?.ready_replicas ?? 0;
  const logs = logFollow ? liveLogs : progress?.logs;
  return (
    <div className="stack">
      <section className="panel action-panel">
        <div>
          <h2><LoaderCircle size={18} /> Installation progress</h2>
          <p>Track project creation, GitOps activity, and live pod readiness.</p>
        </div>
        <div className="progress-controls">
          <select value={activeProjectID} onChange={(event) => setActiveProjectID(event.target.value)}>
            <option value="">Choose project</option>
            {projects.map((project) => <option key={project.id} value={project.id}>{project.display_name || project.name}</option>)}
          </select>
          <button onClick={() => refresh()}><RefreshCw size={15} /> Refresh</button>
        </div>
      </section>
      {installProgress && (
        <section className="panel">
          <h2><Rocket size={18} /> Current install</h2>
          <div className="step-list">
            {installProgress.steps.map((step) => <ProgressStep key={step.label} step={step} />)}
          </div>
        </section>
      )}
      {progress ? (
        <div className="progress-grid">
          <section className="panel">
            <h2><Boxes size={18} /> Project</h2>
            <div className="detail-list">
              <span>Name <b>{progress.project.display_name || progress.project.name}</b></span>
              <span>Namespace <b>{progress.project.namespace}</b></span>
              <span>Route <b>{progress.project.domain || progress.project.exposure_mode}</b></span>
              <span>Status <b>{progress.project.status}</b></span>
            <span>Deploy mode <b>{progress.project.auto_deploy ? "Auto GitOps" : "Manual only"}</b></span>
            <span>Image <b>{progress.project.image_reference || "-"}</b></span>
              <span>Last checked <b>{formatTime(progress.checked_at)}</b></span>
            </div>
            {progress.error && <p className="error-inline">{progress.error}</p>}
          </section>
          <section className="panel">
            <h2><Server size={18} /> Runtime</h2>
            <div className="runtime-summary">
              <strong>{readyReplicas}/{desiredReplicas}</strong>
              <span>replicas ready · {readyPods}/{pods.length} pods</span>
            </div>
            {progress.deployment && (
              <div className="detail-list compact-details">
                <span>Updated <b>{progress.deployment.updated_replicas}</b></span>
                <span>Available <b>{progress.deployment.available_replicas}</b></span>
              </div>
            )}
            <div className="mini-table">
              {pods.map((pod) => (
                <div key={pod.name || pod.pod || JSON.stringify(pod)}>
                  <span>
                    {pod.name || "pod"}
                    {pod.reason && <small>{pod.reason}</small>}
                    {pod.containers?.length > 0 && <small>{pod.containers.join(" · ")}</small>}
                  </span>
                  <b>{pod.ready_containers}/{pod.total_containers} · {pod.status || "-"}</b>
                </div>
              ))}
              {pods.length === 0 && <div className="empty">No pods reported yet.</div>}
            </div>
          </section>
          <section className="panel">
            <h2><GitBranch size={18} /> Deployments</h2>
            <div className="timeline">
              {deployments.map((deployment) => (
                <div className="timeline-item" key={deployment.id}>
                  <span className={deployment.status === "failed" ? "dot failed" : ["deployed", "provisioned"].includes(deployment.status) ? "dot done" : "dot running"} />
                  <div>
                    <b>{deployment.status || "pending"}</b>
                    <small>{deployment.image_ref || deployment.tag || deployment.commit_sha || "manual"} · {formatTime(deployment.created_at)}</small>
                    {deployment.workflow_url && <small><a href={deployment.workflow_url} target="_blank" rel="noreferrer">GitHub Actions run</a></small>}
                    {deployment.commit_sha && <small>Commit: {deployment.commit_sha}</small>}
                    {deployment.failure_reason && <small className="warning">{deployment.failure_reason}</small>}
                  </div>
                </div>
              ))}
              {deployments.length === 0 && <div className="empty">No deployment events yet.</div>}
            </div>
          </section>
          <section className="panel">
            <h2><ListRestart size={18} /> Kubernetes events</h2>
            <div className="timeline">
              {events.map((event, index) => (
                <div className="timeline-item" key={`${event.object}-${event.reason}-${index}`}>
                  <span className={event.type === "Warning" ? "dot failed" : "dot done"} />
                  <div>
                    <b>{event.reason || event.type}</b>
                    <small>{event.object} · {event.count || 1}x · {formatTime(event.last_seen)}</small>
                    <p>{event.message}</p>
                  </div>
                </div>
              ))}
              {events.length === 0 && <div className="empty">No Kubernetes events yet.</div>}
            </div>
          </section>
          <section className="panel log-panel">
            <div className="log-header">
              <h2><Code2 size={18} /> Container logs</h2>
              <div className="row-actions">
                <button onClick={() => refresh()} disabled={logFollow}><RefreshCw size={15} /> Snapshot</button>
                {logFollow ? (
                  <button onClick={onStopLogFollow}>Stop follow</button>
                ) : (
                  <button className="primary" onClick={() => onStartLogFollow(progress.project.id)}>Follow live</button>
                )}
              </div>
            </div>
            {logStatus && <p className="log-status">{logStatus}</p>}
            <pre>{logs || "No application logs yet."}</pre>
          </section>
        </div>
      ) : (
        <section className="panel"><div className="empty">Choose a project to view progress.</div></section>
      )}
    </div>
  );
}

function DashboardView({dashboard, refresh}) {
  if (!dashboard) {
    return <section className="panel"><div className="empty">Loading cluster dashboard...</div></section>;
  }
  const resources = dashboard.resources || {};
  const pods = dashboard.pods || {};
  const nodes = dashboard.nodes || {};
  const alerts = dashboard.alerts || [];
  const events = dashboard.events || [];
  return (
    <div className="dashboard-shell">
      <section className="dashboard-hero">
        <div>
          <span className="eyebrow">Cluster Operations</span>
          <h2>{dashboard.cluster_name}</h2>
          <p>Kubernetes {dashboard.kubernetes_version || "-"}{dashboard.k3s_version ? ` · K3s ${dashboard.k3s_version}` : ""}</p>
        </div>
        <div className={dashboard.healthy ? "health-badge good" : "health-badge bad"}>
          <span>{dashboard.status || "Unknown"}</span>
          <b>{dashboard.healthy ? "Ready" : "NotReady"}</b>
        </div>
        <button onClick={refresh}><RefreshCw size={15} /> Refresh</button>
      </section>

      <section className="dashboard-kpis">
        <MetricCard icon={Server} label="Nodes" value={nodes.total || 0} detail={`${nodes.server || 0} Server · ${nodes.agent || 0} Agent · ${nodes.not_ready || 0} NotReady`} />
        <MetricCard icon={Boxes} label="Pods" value={`${pods.running || 0} / ${pods.total || 0}`} detail={`${pods.abnormal || 0} abnormal · ${pods.pending || 0} pending`} />
        <MetricCard icon={Cpu} label="CPU" value={`${formatPercent(resources.cpu_percent)}%`} detail={`${resources.cpu_used_millis || 0}m / ${resources.cpu_total_millis || 0}m`} />
        <MetricCard icon={MemoryStick} label="Memory" value={`${formatPercent(resources.memory_percent)}%`} detail={`${formatBytes(resources.memory_used_bytes)} / ${formatBytes(resources.memory_total_bytes)}`} />
        <MetricCard icon={HardDrive} label="Disk" value={`${formatPercent(resources.disk_percent)}%`} detail={`${formatBytes(resources.disk_used_bytes)} / ${formatBytes(resources.disk_total_bytes)}`} />
        <MetricCard icon={AlertTriangle} label="Alerts" value={alerts.length} detail={`${events.length} recent warning events`} tone={alerts.length > 0 ? "warning" : "good"} />
      </section>

      <section className="dashboard-grid">
        <div className="panel dashboard-panel">
          <h2><Activity size={18} /> Live Resource Utilization</h2>
          <div className="industrial-meters">
            <IndustrialMeter label="CPU" value={resources.cpu_percent} detail={`${resources.cpu_used_millis || 0}m / ${resources.cpu_total_millis || 0}m`} />
            <IndustrialMeter label="Memory" value={resources.memory_percent} detail={`${formatBytes(resources.memory_used_bytes)} / ${formatBytes(resources.memory_total_bytes)}`} />
            <IndustrialMeter label="Disk" value={resources.disk_percent} detail={`${formatBytes(resources.disk_used_bytes)} / ${formatBytes(resources.disk_total_bytes)}`} />
          </div>
          {!dashboard.metrics_available && <p className="muted">Metrics partially unavailable: {dashboard.metrics_error || "metrics-server or node stats endpoint did not return data."}</p>}
        </div>
        <div className="panel dashboard-panel">
          <h2><Server size={18} /> Cluster Runtime</h2>
          <div className="detail-list">
            <span>Status <b>{dashboard.status}</b></span>
            <span>Ready nodes <b>{nodes.ready || 0}/{nodes.total || 0}</b></span>
            <span>Running pods <b>{pods.running || 0}/{pods.total || 0}</b></span>
            <span>Uptime <b>{formatDuration(dashboard.uptime_seconds)}</b></span>
            <span>Last check <b>{formatTime(dashboard.checked_at)}</b></span>
          </div>
        </div>
      </section>

      <section className="dashboard-grid">
        <div className="panel dashboard-panel">
          <h2><AlertTriangle size={18} /> Recent Alerts</h2>
          <AlertList rows={alerts} empty="No active alerts reported." />
        </div>
        <div className="panel dashboard-panel">
          <h2><ListRestart size={18} /> Events and Error Signals</h2>
          <div className="timeline">
            {events.map((event, index) => (
              <div className="timeline-item" key={`${event.object}-${event.reason}-${index}`}>
                <span className="dot failed" />
                <div>
                  <b>{event.reason || event.type}</b>
                  <small>{event.object} · {event.count || 1}x · {formatTime(event.last_seen)}</small>
                  <p>{event.message}</p>
                </div>
              </div>
            ))}
            {events.length === 0 && <div className="empty">No warning events in the latest cluster feed.</div>}
          </div>
        </div>
      </section>
    </div>
  );
}

function AlertsView({dashboard, refresh}) {
  if (!dashboard) {
    return <section className="panel"><div className="empty">Loading alerts...</div></section>;
  }
  const alerts = dashboard.alerts || [];
  const critical = alerts.filter((row) => ["critical", "error", "failed"].includes(String(row.severity || "").toLowerCase())).length;
  const warnings = alerts.length - critical;
  return (
    <div className="stack observability-page">
      <section className="panel action-panel">
        <div>
          <h2><AlertTriangle size={18} /> Alerts</h2>
          <p>Active cluster health signals generated from abnormal pods, warning events, and node readiness.</p>
        </div>
        <button onClick={refresh}><RefreshCw size={15} /> Refresh</button>
      </section>
      <section className="dashboard-kpis">
        <MetricCard icon={AlertTriangle} label="Active" value={alerts.length} detail={`${critical} critical · ${warnings} warning`} tone={alerts.length > 0 ? "warning" : "good"} />
        <MetricCard icon={Server} label="Nodes" value={`${dashboard.nodes?.ready || 0}/${dashboard.nodes?.total || 0}`} detail={`${dashboard.nodes?.not_ready || 0} not ready`} tone={dashboard.nodes?.not_ready ? "warning" : "good"} />
        <MetricCard icon={Boxes} label="Pods" value={dashboard.pods?.abnormal || 0} detail={`${dashboard.pods?.pending || 0} pending · ${dashboard.pods?.failed || 0} failed`} tone={dashboard.pods?.abnormal ? "warning" : "good"} />
        <MetricCard icon={Activity} label="Status" value={dashboard.status || "-"} detail={`Last check ${formatTime(dashboard.checked_at)}`} tone={dashboard.healthy ? "good" : "warning"} />
      </section>
      <section className="panel">
        <h2><Shield size={18} /> Alert feed</h2>
        <AlertList rows={alerts} empty="No active alerts reported." />
      </section>
    </div>
  );
}

function EventsView({dashboard, refresh}) {
  if (!dashboard) {
    return <section className="panel"><div className="empty">Loading events...</div></section>;
  }
  const events = dashboard.events || [];
  const byReason = events.reduce((acc, event) => {
    const key = event.reason || event.type || "Unknown";
    acc[key] = (acc[key] || 0) + Number(event.count || 1);
    return acc;
  }, {});
  return (
    <div className="stack observability-page">
      <section className="panel action-panel">
        <div>
          <h2><ListRestart size={18} /> Events</h2>
          <p>Recent warning events from the Kubernetes event stream, grouped by object, reason, and last seen time.</p>
        </div>
        <button onClick={refresh}><RefreshCw size={15} /> Refresh</button>
      </section>
      <section className="dashboard-kpis">
        <MetricCard icon={ListRestart} label="Warning events" value={events.length} detail={`${Object.keys(byReason).length} reasons`} tone={events.length > 0 ? "warning" : "good"} />
        <MetricCard icon={AlertTriangle} label="Event count" value={events.reduce((sum, event) => sum + Number(event.count || 1), 0)} detail="Summed Kubernetes count values" />
        <MetricCard icon={Activity} label="Cluster" value={dashboard.status || "-"} detail={`Checked ${formatTime(dashboard.checked_at)}`} tone={dashboard.healthy ? "good" : "warning"} />
      </section>
      <section className="dashboard-grid">
        <div className="panel">
          <h2><Database size={18} /> Reasons</h2>
          <div className="mini-table">
            {Object.entries(byReason).map(([reason, count]) => (
              <div key={reason}><span>{reason}</span><b>{count}</b></div>
            ))}
            {Object.keys(byReason).length === 0 && <div className="empty">No warning reasons in the latest feed.</div>}
          </div>
        </div>
        <div className="panel">
          <h2><ScrollText size={18} /> Event stream</h2>
          <EventTimeline events={events} />
        </div>
      </section>
    </div>
  );
}

function MetricsView({dashboard, runtime, refresh}) {
  if (!dashboard) {
    return <section className="panel"><div className="empty">Loading metrics...</div></section>;
  }
  const resources = dashboard.resources || {};
  const nodes = runtime.nodes || [];
  return (
    <div className="stack observability-page">
      <section className="panel action-panel">
        <div>
          <h2><LineChart size={18} /> Metrics</h2>
          <p>Cluster capacity, utilization, and node-level resource readings from metrics-server and node stats.</p>
        </div>
        <button onClick={refresh}><RefreshCw size={15} /> Refresh</button>
      </section>
      <section className="dashboard-kpis">
        <MetricCard icon={Cpu} label="CPU" value={`${formatPercent(resources.cpu_percent)}%`} detail={`${resources.cpu_used_millis || 0}m / ${resources.cpu_total_millis || 0}m`} />
        <MetricCard icon={MemoryStick} label="Memory" value={`${formatPercent(resources.memory_percent)}%`} detail={`${formatBytes(resources.memory_used_bytes)} / ${formatBytes(resources.memory_total_bytes)}`} />
        <MetricCard icon={HardDrive} label="Disk" value={`${formatPercent(resources.disk_percent)}%`} detail={`${formatBytes(resources.disk_used_bytes)} / ${formatBytes(resources.disk_total_bytes)}`} />
        <MetricCard icon={Activity} label="Metrics source" value={dashboard.metrics_available ? "Live" : "Partial"} detail={dashboard.metrics_error || `Checked ${formatTime(dashboard.checked_at)}`} tone={dashboard.metrics_available ? "good" : "warning"} />
      </section>
      <section className="dashboard-grid">
        <div className="panel dashboard-panel">
          <h2><Activity size={18} /> Utilization</h2>
          <div className="industrial-meters">
            <IndustrialMeter label="CPU" value={resources.cpu_percent} detail={`${resources.cpu_used_millis || 0}m / ${resources.cpu_total_millis || 0}m`} />
            <IndustrialMeter label="Memory" value={resources.memory_percent} detail={`${formatBytes(resources.memory_used_bytes)} / ${formatBytes(resources.memory_total_bytes)}`} />
            <IndustrialMeter label="Disk" value={resources.disk_percent} detail={`${formatBytes(resources.disk_used_bytes)} / ${formatBytes(resources.disk_total_bytes)}`} />
          </div>
        </div>
        <div className="panel">
          <h2><Server size={18} /> Node readings</h2>
          <div className="mini-table">
            {nodes.map((node) => (
              <div key={node.name}>
                <span>{node.name}<small>{node.status || "-"} · {node.version || "-"}</small></span>
                <b>{node.cpu || node.cpu_percent || "-"} / {node.memory || node.memory_percent || "-"}</b>
              </div>
            ))}
            {nodes.length === 0 && <div className="empty">Node runtime data is not loaded yet.</div>}
          </div>
        </div>
      </section>
    </div>
  );
}

function LogsView({projects, activeProjectID, setActiveProjectID, progress, refresh, logFollow, liveLogs, logStatus, onStartLogFollow, onStopLogFollow, onOpenPods}) {
  const logs = logFollow ? liveLogs : progress?.logs;
  return (
    <div className="stack observability-page">
      <section className="panel action-panel">
        <div>
          <h2><ScrollText size={18} /> Logs</h2>
          <p>Project container log snapshots and live follow without leaving the observability section.</p>
        </div>
        <div className="progress-controls">
          <select value={activeProjectID} onChange={(event) => setActiveProjectID(event.target.value)}>
            <option value="">Choose project</option>
            {projects.map((project) => <option key={project.id} value={project.id}>{project.display_name || project.name}</option>)}
          </select>
          <button onClick={() => refresh()} disabled={logFollow}><RefreshCw size={15} /> Snapshot</button>
          {logFollow ? <button onClick={onStopLogFollow}>Stop follow</button> : <button className="primary" onClick={() => onStartLogFollow(activeProjectID)} disabled={!activeProjectID}>Follow live</button>}
        </div>
      </section>
      <section className="dashboard-kpis">
        <MetricCard icon={Boxes} label="Project" value={progress?.project?.display_name || progress?.project?.name || "-"} detail={progress?.project?.namespace || "No project selected"} />
        <MetricCard icon={Layers3} label="Pods" value={(progress?.pods || []).length} detail={`${(progress?.pods || []).filter((pod) => pod.status === "Running").length} running`} />
        <MetricCard icon={GitBranch} label="Deployments" value={(progress?.deployments || []).length} detail={(progress?.deployments || [])[0]?.status || "No deployment events"} />
      </section>
      <section className="panel log-panel observability-log-panel">
        <div className="log-header">
          <h2><Code2 size={18} /> Container logs</h2>
          <div className="row-actions">
            <button type="button" onClick={onOpenPods}><Layers3 size={15} /> Pod detail</button>
          </div>
        </div>
        {logStatus && <p className="log-status">{logStatus}</p>}
        <pre>{logs || "Choose a project to load recent logs."}</pre>
      </section>
    </div>
  );
}

function EventTimeline({events}) {
  return (
    <div className="timeline">
      {events.map((event, index) => (
        <div className="timeline-item" key={`${event.object}-${event.reason}-${index}`}>
          <span className={event.type === "Warning" ? "dot failed" : "dot done"} />
          <div>
            <b>{event.reason || event.type}</b>
            <small>{event.object} · {event.count || 1}x · {formatTime(event.last_seen)}</small>
            <p>{event.message}</p>
          </div>
        </div>
      ))}
      {events.length === 0 && <div className="empty">No warning events in the latest cluster feed.</div>}
    </div>
  );
}

function MetricCard({icon: Icon, label, value, detail, tone = "neutral"}) {
  return (
    <div className={`metric-card ${tone}`}>
      <div><Icon size={18} /><span>{label}</span></div>
      <strong>{value}</strong>
      <small>{detail}</small>
    </div>
  );
}

function IndustrialMeter({label, value, detail}) {
  const normalized = Math.max(0, Math.min(100, Number(value || 0)));
  return (
    <div className="industrial-meter">
      <div><b>{label}</b><span>{formatPercent(normalized)}%</span></div>
      <progress value={normalized} max="100" />
      <small>{detail}</small>
    </div>
  );
}

function AlertList({rows, empty}) {
  return (
    <div className="alert-list">
      {rows.map((row, index) => (
        <div className={`alert-row ${row.severity || "warning"}`} key={`${row.object}-${row.reason}-${index}`}>
          <b>{row.reason || "Warning"}</b>
          <span>{row.object}{row.namespace ? ` · ${row.namespace}` : ""}</span>
          <p>{row.message}</p>
          <small>{formatTime(row.last_seen)}</small>
        </div>
      ))}
      {rows.length === 0 && <div className="empty">{empty}</div>}
    </div>
  );
}

function ProgressStep({step}) {
  const Icon = step.state === "done" ? CheckCircle2 : step.state === "running" ? LoaderCircle : step.state === "failed" ? Trash2 : Play;
  return (
    <div className={`step ${step.state}`}>
      <Icon size={16} />
      <span>{step.label}</span>
      <b>{step.state}</b>
    </div>
  );
}

const deploySteps = [
  {id: "method", label: "Source type", title: "Choose deployment source"},
  {id: "source", label: "Source", title: "Choose deployment source details"},
  {id: "update", label: "Update", title: "Choose update mode"},
  {id: "check", label: "Check", title: "Check installability"},
  {id: "params", label: "Params", title: "Configure parameters"},
  {id: "namespace", label: "Namespace", title: "Choose namespace"},
  {id: "ingress", label: "Ingress", title: "Choose ingress mode"},
  {id: "domain", label: "Domain", title: "Choose domain"},
  {id: "confirm", label: "Confirm", title: "Confirm and build"},
];

const deploySourceOptions = [
  {id: "gitops", label: "GitOps repository", icon: GitBranch, description: "Use a GitHub repository as source and publish runtime images to GHCR."},
  {id: "registry", label: "Container registry", icon: Package, description: "Deploy an existing or newly tracked container image object."},
];

const updateModeOptions = [
  {id: "argocd", label: "Argo CD", icon: GitBranch, description: "Create GitOps manifests, register an Argo CD app, and let GitHub Actions build the first GHCR image."},
  {id: "passive", label: "Passive update", icon: RefreshCw, description: "Create the project without automatic GitHub push deployment."},
];

function canContinueDeployStep(stepID, form, selectedCredential, analysis) {
  if (stepID === "method") return Boolean(form.deploy_source);
  if (stepID === "source") {
    if (form.deploy_source === "gitops") {
      if (form.repo_type === "git-url") return false;
      return Boolean(selectedCredential && form.github_repo);
    }
    if (form.image_choice === "new") return Boolean(form.selected_image_id && form.image_reference);
    return Boolean(form.image_reference);
  }
  if (stepID === "update") return form.deploy_source === "registry" || Boolean(form.update_mode);
  if (stepID === "check") return Boolean(analysis?.deployable);
  if (stepID === "params") return Boolean(form.name && Number(form.port || 0) > 0 && Number(form.replicas || 0) > 0);
  if (stepID === "domain") {
    if (form.exposure_mode === "public") return Boolean(form.cloudflare_credential_id && form.subdomain);
    if (form.exposure_mode === "private") return Boolean(form.private_host);
  }
  return true;
}

function sourceLabel(source) {
  return ({github: "GitHub", dockerhub: "Docker Hub", ghcr: "Container registry", registry: "Container registry", "source-upload": "Source upload"}[source || "github"] || source);
}

function sourceSummary(form) {
  if (form.deploy_source === "gitops") return form.repo_type === "git-url" ? form.git_url || "-" : `${form.github_repo || "-"} @ ${form.github_branch || "main"}`;
  return form.image_reference || "-";
}

function ProjectsView({projects, onEdit, onDelete, onScale, onRestart, onBuild, onProgress}) {
  return (
    <section className="panel">
      <div className="table">
        <div className="tr head"><span>Name</span><span>Repo</span><span>Route</span><span>Status</span><span>Scale</span><span>Actions</span></div>
        {projects.map((project) => (
          <div className="tr" key={project.id}>
            <span className="strong">{project.display_name || project.name}</span>
            <span>{project.github_repo || project.image_reference || project.source_archive_name || project.build_source}</span>
            <span>{project.domain || project.exposure_mode}</span>
            <span>{project.status}</span>
            <span>
              <button onClick={() => onScale(project, Math.max(0, Number(project.replicas || 1) - 1))}>-</button>
              <b>{project.replicas}</b>
              <button onClick={() => onScale(project, Number(project.replicas || 1) + 1)}>+</button>
            </span>
            <span className="row-actions">
              <button onClick={() => onProgress(project)} title="Progress"><LoaderCircle size={15} /> Progress</button>
              <button onClick={() => onBuild(project)} title="Build"><Play size={15} /> Build</button>
              <button onClick={() => onRestart(project)} title="Restart"><ListRestart size={15} /></button>
              <button onClick={() => onEdit(project)} title="Edit"><Plus size={15} /></button>
              <button className="danger-button" onClick={() => onDelete(project)} title="Delete"><Trash2 size={15} /> Delete</button>
            </span>
          </div>
        ))}
        {projects.length === 0 && <div className="empty">No projects yet.</div>}
      </div>
    </section>
  );
}

function DeleteProjectModal({project, busy, onClose, onDelete}) {
  return (
    <div className="modal-backdrop">
      <div className="modal">
        <h2>Delete {project.name}</h2>
        <p className="muted">This removes the project record, namespace, DNS records, and managed OAuth app where applicable.</p>
        <div className="delete-summary">
          <span>Namespace <b>{project.namespace}</b></span>
          <span>Route <b>{project.domain || project.exposure_mode}</b></span>
        </div>
        <div className="modal-actions">
          <button type="button" onClick={onClose} disabled={busy}>Cancel</button>
          <button className="danger-button filled" type="button" onClick={onDelete} disabled={busy}><Trash2 size={15} /> Delete</button>
        </div>
      </div>
    </div>
  );
}

function ContainerRegistriesView({presets, registries, images, onAddRegistry, onDeleteRegistry, onAddImage, onRefreshImage, onDeleteImage, onSyncAll, onRefresh}) {
  const presetByKind = useMemo(() => Object.fromEntries((presets || []).map((p) => [p.kind, p])), [presets]);
  const [previewKind, setPreviewKind] = useState("ghcr");

  return (
    <div className="stack registry-page">
      <section className="panel action-panel">
        <div>
          <h2><Package size={18} /> 镜像源</h2>
          <p>基于 Docker Registry HTTP API V2 列出标签；Docker Hub 会使用 registry-1.docker.io；私有仓库请填写凭据。</p>
        </div>
        <button type="button" onClick={onRefresh}><RefreshCw size={15} /> 刷新</button>
      </section>

      <section className="panel">
        <h2><Plus size={18} /> 添加镜像源</h2>
        <form className="form-grid registry-form" onSubmit={onAddRegistry}>
          <label>
            类型
            <select name="kind" value={previewKind} onChange={(e) => setPreviewKind(e.target.value)}>
              {(presets || []).map((p) => (
                <option key={p.kind} value={p.kind}>{p.label}</option>
              ))}
            </select>
          </label>
          <label>
            显示名称（可选）
            <input name="name" placeholder={`例如 ${presetByKind[previewKind]?.label || ""}`} />
          </label>
          <label className="span-2">
            镜像源地址
            <input name="host" required placeholder={presetByKind[previewKind]?.example_host || "registry.example.com"} />
          </label>
          <label>
            用户名（可选）
            <input name="username" autoComplete="off" placeholder="私有仓库 / PAT 用户名" />
          </label>
          <label>
            密码或 Token（可选）
            <input name="password" type="password" autoComplete="new-password" placeholder="不会明文存储" />
          </label>
          <label className="checkbox-row span-2">
            <input name="insecure_tls" type="checkbox" />
            跳过 TLS 校验（仅可信内网）
          </label>
          {presetByKind[previewKind]?.hint && (
            <p className="muted span-2">{presetByKind[previewKind].hint}</p>
          )}
          <button className="primary" type="submit"><Plus size={15} /> 保存镜像源</button>
        </form>
      </section>

      <section className="panel">
        <h2><Database size={18} /> 已保存的镜像源</h2>
        <div className="table compact-table registry-table">
          <div className="tr head"><span>名称</span><span>类型</span><span>API 根</span><span>鉴权</span><span /></div>
          {(registries || []).map((r) => (
            <div className="tr" key={r.id}>
              <span className="strong">{r.name}</span>
              <span>{r.kind}</span>
              <span className="mono">{r.api_base}</span>
              <span>{r.has_auth ? "已配置" : "匿名"}</span>
              <span className="row-actions">
                <button type="button" className="danger-button" onClick={() => onDeleteRegistry(r)}><Trash2 size={15} /> 删除</button>
              </span>
            </div>
          ))}
          {(registries || []).length === 0 && <div className="empty">尚未添加镜像源。</div>}
        </div>
      </section>

      <section className="panel">
        <div className="panel-heading-inline">
          <h2><Boxes size={18} /> 镜像与标签</h2>
          <button type="button" className="ghost" onClick={onSyncAll} disabled={!(images || []).length}>
            <RefreshCw size={15} /> 同步全部远程标签
          </button>
        </div>
        <p className="muted">仓库路径需与 Registry API 一致（例如 Docker Hub 官方 nginx：<span className="mono">library/nginx</span>；GHCR：<span className="mono">owner/repo</span>）。保存后会立即拉取标签；页面每 2 分钟刷新本地缓存列表。</p>
        <form className="form-grid registry-form" onSubmit={onAddImage}>
          <label>
            镜像源
            <select name="registry_id" required>
              <option value="">选择...</option>
              {(registries || []).map((r) => (
                <option key={r.id} value={r.id}>{r.name} ({r.kind})</option>
              ))}
            </select>
          </label>
          <label className="span-2">
            仓库路径（repository）
            <input name="repository" required placeholder="namespace/name" />
          </label>
          <button className="primary" type="submit"><Plus size={15} /> 添加并同步标签</button>
        </form>

        {(images || []).map((im) => (
          <div className="registry-image-card" key={im.id}>
            <div className="registry-image-head">
              <div>
                <div className="mono strong">{im.repository}</div>
                <small className="muted">来源：{im.registry?.name || `registry #${im.registry_id}`} · 更新 {formatTime(im.refreshed_at)}</small>
              </div>
              <div className="row-actions">
                <button type="button" onClick={() => onRefreshImage(im.id)}><RefreshCw size={15} /> 同步标签</button>
                <button type="button" className="danger-button" onClick={() => onDeleteImage(im)}><Trash2 size={15} /> 移除</button>
              </div>
            </div>
            <div className="tag-chip-grid">
              {(im.tags || []).slice(0, 200).map((t) => (
                <span className="tag-chip" key={t}>{t}</span>
              ))}
              {(im.tags || []).length > 200 && (
                <span className="muted">… 共 {(im.tags || []).length} 个标签，仅显示前 200 个</span>
              )}
              {(im.tags || []).length === 0 && <span className="muted">暂无标签或未同步成功。</span>}
            </div>
          </div>
        ))}
        {(images || []).length === 0 && <div className="empty">尚未添加镜像仓库跟踪。</div>}
      </section>
    </div>
  );
}

function WorkloadImageView({images, onRefresh, onOpenRegistry, onRefreshImage, onDeleteImage}) {
  return (
    <div className="stack">
      <section className="panel action-panel">
        <div>
          <h2><ImageIcon size={18} /> Image</h2>
          <p>Running workload images are visible on Pods and Deployments. Tracked registry tags and sync use <b>Integrations → Image Registry</b>.</p>
        </div>
        <button type="button" onClick={onRefresh}><RefreshCw size={15} /> Refresh</button>
      </section>
      <section className="panel">
        <h2><Package size={18} /> Tracked image tags</h2>
        <p className="muted">Mirrors and tag lists you have registered. To add registries or repositories, open Image Registry.</p>
        <div className="row-actions" style={{marginBottom: 12}}>
          <button type="button" className="primary" onClick={onOpenRegistry}><Package size={15} /> Open Image Registry</button>
        </div>
        {(images || []).map((im) => (
          <div className="registry-image-card" key={im.id}>
            <div className="registry-image-head">
              <div>
                <div className="mono strong">{im.repository}</div>
                <small className="muted">{im.registry?.name || `registry #${im.registry_id}`} · {formatTime(im.refreshed_at)}</small>
              </div>
              <div className="row-actions">
                <button type="button" onClick={() => onRefreshImage(im.id)}><RefreshCw size={15} /> Sync</button>
                <button type="button" className="danger-button" onClick={() => onDeleteImage(im)}><Trash2 size={15} /> Remove</button>
              </div>
            </div>
            <div className="tag-chip-grid">
              {(im.tags || []).slice(0, 120).map((t) => (
                <span className="tag-chip" key={t}>{t}</span>
              ))}
              {(im.tags || []).length > 120 && <span className="muted">… {(im.tags || []).length} tags</span>}
              {(im.tags || []).length === 0 && <span className="muted">No tags cached yet.</span>}
            </div>
          </div>
        ))}
        {(images || []).length === 0 && <div className="empty">No tracked images. Configure mirrors under Image Registry.</div>}
      </section>
    </div>
  );
}

function ComingSoonView({title, description, actionLabel, onAction}) {
  return (
    <div className="stack">
      <section className="panel">
        <h2>{title}</h2>
        <p className="muted">{description}</p>
        {actionLabel && onAction && (
          <div style={{marginTop: 14}}>
            <button type="button" className="primary" onClick={onAction}>{actionLabel}</button>
          </div>
        )}
      </section>
    </div>
  );
}

function SettingsView({version}) {
  return (
    <div className="stack">
      <section className="panel">
        <h2><Settings size={18} /> Settings</h2>
        <p className="muted">Controller API version: <code className="mono">{version || "—"}</code></p>
        <p className="muted">Authentication uses BasaltPass. Manage identity provider connections under <b>Security → Access Control</b>.</p>
      </section>
    </div>
  );
}

function APIKeysView({keys, createdKey, onDismissCreated, onCreate, onRevoke, onRefresh, isAdmin}) {
  return (
    <div className="stack">
      <section className="panel action-panel">
        <div>
          <h2><KeyRound size={18} /> API keys</h2>
          <p>Create keys for beanctl, scripts, and external systems that need to manage BeanCS through the API.</p>
        </div>
        <button onClick={onRefresh}><RefreshCw size={15} /> Refresh</button>
      </section>
      {createdKey && (
        <section className="panel api-key-created">
          <h2><Shield size={18} /> Save this API key now</h2>
          <p className="muted">BeanCS stores only a hash. This full key will not be shown again.</p>
          <pre>{createdKey.key}</pre>
          <div className="modal-actions"><button onClick={onDismissCreated}>I saved it</button></div>
        </section>
      )}
      <section className="panel">
        <h2><Plus size={18} /> Create API key</h2>
        <form className="form-grid api-key-form" onSubmit={onCreate}>
          <input name="name" placeholder="Key name, e.g. local beanctl" required />
          <input name="expires_at" type="datetime-local" />
          <label className="checkbox-row">
            <input name="admin_scope" type="checkbox" disabled={!isAdmin} />
            Include beancs.admin scope {isAdmin ? "" : "(admin session required)"}
          </label>
          <button className="primary" type="submit"><KeyRound size={15} /> Create key</button>
        </form>
      </section>
      <section className="panel">
        <h2><KeyRound size={18} /> Issued keys</h2>
        <div className="table api-key-table">
          <div className="tr head"><span>Name</span><span>Prefix</span><span>Scopes</span><span>Last used</span><span>Expires</span><span>Actions</span></div>
          {keys.map((key) => (
            <div className="tr" key={key.id}>
              <span className="strong">{key.name}</span>
              <span>{key.prefix}</span>
              <span>{(key.scopes || []).join(", ") || "-"}</span>
              <span>{formatTime(key.last_used_at)}</span>
              <span>{key.revoked_at ? `Revoked ${formatTime(key.revoked_at)}` : formatTime(key.expires_at)}</span>
              <span className="row-actions">
                <button className="danger-button" disabled={Boolean(key.revoked_at)} onClick={() => onRevoke(key)}><Trash2 size={15} /> Revoke</button>
              </span>
            </div>
          ))}
          {keys.length === 0 && <div className="empty">No API keys issued yet.</div>}
        </div>
      </section>
    </div>
  );
}

function GitHubView({credentials, onConnect, onRepos, onDelete, reposByCredential, repoFilters, setRepoFilters}) {
  return (
    <div className="stack">
      <section className="panel action-panel">
        <div>
          <h2><Github size={18} /> GitHub App</h2>
          <p>Authorize repositories directly. BeanCS will name the credential from the GitHub account.</p>
        </div>
        <form onSubmit={onConnect}><button className="primary"><Github size={16} /> Connect GitHub App</button></form>
      </section>
      {credentials.map((cred) => {
        const repos = reposByCredential[cred.id] || [];
        const filter = repoFilters[cred.id] || "";
        const visible = repos.filter((repo) => repo.full_name.toLowerCase().includes(filter.toLowerCase()));
        return (
          <section className="panel" key={cred.id}>
            <div className="account-header">
              <div className="account-cell">{cred.avatar_url ? <img src={cred.avatar_url} alt="" /> : <Github size={18} />}<div><b>{cred.name}</b><small>{cred.account_login || cred.org || "-"} · {cred.auth_type || "pat"} · GitOps {cred.gitops_repo || "-"}</small></div></div>
              <div className="row-actions">
                <button onClick={() => onRepos(cred.id)}><RefreshCw size={15} /> Load repos</button>
                <button onClick={() => onDelete(cred.id)}><Trash2 size={15} /></button>
              </div>
            </div>
            <div className="repo-toolbar">
              <input placeholder="Search repositories" value={filter} onChange={(event) => setRepoFilters((current) => ({...current, [cred.id]: event.target.value}))} />
              <span>{visible.length}/{repos.length} repos</span>
            </div>
            <div className="repo-grid">
              {visible.map((repo) => (
                <a key={repo.full_name} className="repo-card" href={repo.html_url} target="_blank" rel="noreferrer">
                  <b>{repo.full_name}</b>
                  <span>{repo.private ? "Private" : "Public"} · {repo.default_branch || "main"}</span>
                </a>
              ))}
              {repos.length === 0 && <div className="empty">Click Load repos to inspect this account.</div>}
            </div>
          </section>
        );
      })}
    </div>
  );
}

function CloudflareView({credentials, domains, selectedID, setSelectedID, dnsRecords, editingRecord, setEditingRecord, onCreate, onDelete, onLoadDNS, onSaveDNS, onDeleteDNS}) {
  const selected = credentials.find((cred) => String(cred.id) === String(selectedID));
  return (
    <div className="stack">
      <CredentialManager kind="cloudflare" rows={credentials} onCreate={onCreate} onDelete={(_, id) => onDelete(id)} />
      <section className="panel">
        <h2><Globe2 size={18} /> Zones and DNS tools</h2>
        <div className="domain-grid">
          {domains.map((domain) => (
            <button type="button" className={String(selectedID) === String(domain.credential_id) ? "domain-tile active" : "domain-tile"} key={`${domain.credential_id}-${domain.zone_id}`} onClick={() => { setSelectedID(String(domain.credential_id)); onLoadDNS(domain.credential_id); }}>
              <Globe2 size={20} />
              <div>
                <b>{domain.domain}</b>
                <span>{domain.credential}</span>
                <small>{domain.zone_id}</small>
              </div>
              <em>{domain.active ? "Active" : "Inactive"}</em>
            </button>
          ))}
          {domains.length === 0 && <div className="empty">No Cloudflare domains linked yet.</div>}
        </div>
      </section>
      <section className="panel">
        <div className="account-header">
          <h2><Network size={18} /> DNS records {selected ? `for ${selected.domain || selected.name}` : ""}</h2>
          <button disabled={!selectedID} onClick={() => onLoadDNS(selectedID)}><RefreshCw size={15} /> Refresh DNS</button>
        </div>
        <form className="form-grid dns-form" onSubmit={onSaveDNS} key={editingRecord?.id || "new-dns"}>
          <select name="type" defaultValue={editingRecord?.type || "A"}><option>A</option><option>AAAA</option><option>CNAME</option><option>TXT</option><option>MX</option></select>
          <input name="name" placeholder="app.example.com" defaultValue={editingRecord?.name || ""} required />
          <input name="content" placeholder="Target content" defaultValue={editingRecord?.content || ""} required />
          <input name="ttl" type="number" min="1" defaultValue={editingRecord?.ttl || 1} />
          <label className="check-row"><input name="proxied" type="checkbox" defaultChecked={Boolean(editingRecord?.proxied)} /> Proxied</label>
          <input name="comment" placeholder="Comment" defaultValue={editingRecord?.comment || ""} />
          <button className="primary" disabled={!selectedID} type="submit">{editingRecord ? "Save DNS" : "Add DNS"}</button>
          {editingRecord && <button type="button" onClick={() => setEditingRecord(null)}>Cancel</button>}
        </form>
        <div className="table dns-table">
          <div className="tr head"><span>Type</span><span>Name</span><span>Content</span><span>TTL</span><span>Proxy</span><span>Actions</span></div>
          {dnsRecords.map((record) => (
            <div className="tr" key={record.id}>
              <span>{record.type}</span><span>{record.name}</span><span>{record.content}</span><span>{record.ttl}</span><span>{record.proxied ? "Yes" : "No"}</span>
              <span className="row-actions"><button onClick={() => setEditingRecord(record)}>Edit</button><button className="danger-button" onClick={() => onDeleteDNS(record)}><Trash2 size={15} /></button></span>
            </div>
          ))}
          {dnsRecords.length === 0 && <div className="empty">{selectedID ? "No DNS records loaded." : "Choose a zone to view DNS records."}</div>}
        </div>
      </section>
    </div>
  );
}

function DomainsView({domains}) {
  return (
    <section className="panel">
      <h2><Globe2 size={18} /> Cloudflare domains</h2>
      <div className="domain-grid">
        {domains.map((domain) => (
          <div className="domain-tile" key={`${domain.credential_id}-${domain.zone_id}`}>
            <Globe2 size={20} />
            <div>
              <b>{domain.domain}</b>
              <span>{domain.credential}</span>
              <small>{domain.zone_id}</small>
            </div>
            <em>{domain.active ? "Active" : "Inactive"}</em>
          </div>
        ))}
        {domains.length === 0 && <div className="empty">No Cloudflare domains linked yet.</div>}
      </div>
    </section>
  );
}

function NetworkingView({network, refresh, onSaveService, onDeleteService, onSaveIngress, onDeleteIngress, onSaveNetworkPolicy, onDeleteNetworkPolicy}) {
  const data = network || {services: [], ingresses: [], endpoints: [], network_policies: [], access: [], controllers: {}};
  const controllers = data.controllers || {};
  return (
    <div className="stack network-page">
      <section className="panel network-overview">
        <div className="action-panel">
          <div>
            <h2><Network size={18} /> Service and network management</h2>
            <p>Manage Service, Ingress, Endpoint, NetworkPolicy, Traefik, Tailscale and TLS bindings from one operational view.</p>
          </div>
          <button onClick={refresh}><RefreshCw size={15} /> Refresh</button>
        </div>
        <div className="dashboard-kpis">
          <MetricCard icon={Database} label="Services" value={data.services.length} detail="ClusterIP / NodePort / LoadBalancer" />
          <MetricCard icon={Network} label="Ingresses" value={data.ingresses.length} detail={`${controllers.traefik_ingresses || 0} Traefik · ${controllers.tailscale_ingresses || 0} Tailscale`} />
          <MetricCard icon={Shield} label="TLS" value={controllers.tls_ingresses || 0} detail="Ingress TLS bindings" />
          <MetricCard icon={Layers3} label="Endpoints" value={data.endpoints.length} detail="Resolved backend addresses" />
          <MetricCard icon={Lock} label="Policies" value={data.network_policies.length} detail="NetworkPolicy rules" />
          <MetricCard icon={Globe2} label="Access URLs" value={data.access.length} detail="Service access entries" />
        </div>
        <div className="detail-list compact-details">
          <span>Traefik namespaces <b>{(controllers.traefik_namespaces || []).join(", ") || "-"}</b></span>
          <span>Tailscale namespaces <b>{(controllers.tailscale_namespaces || []).join(", ") || "-"}</b></span>
          <span>Checked <b>{formatTime(data.checked_at)}</b></span>
        </div>
      </section>

      <section className="panel">
        <h2><Database size={18} /> Service, LoadBalancer and NodePort</h2>
        <ServiceForm onSubmit={(event) => onSaveService(event)} />
        <SimpleTable rows={data.services} columns={["namespace", "name", "type", "cluster_ip", "external_ip", "ports"]} actions={(row) => <button className="danger-button" onClick={() => onDeleteService(row)}><Trash2 size={15} /></button>} />
      </section>

      <section className="panel">
        <h2><Network size={18} /> Ingress, domain and TLS binding</h2>
        <IngressForm onSubmit={(event) => onSaveIngress(event)} />
        <SimpleTable rows={data.ingresses} columns={["namespace", "name", "class", "hosts", "services", "tls", "address"]} actions={(row) => <button className="danger-button" onClick={() => onDeleteIngress(row)}><Trash2 size={15} /></button>} />
      </section>

      <section className="panel">
        <h2><Lock size={18} /> NetworkPolicy</h2>
        <NetworkPolicyForm onSubmit={(event) => onSaveNetworkPolicy(event)} />
        <SimpleTable rows={data.network_policies} columns={["namespace", "name", "pod_selector", "policy_types", "ingress_rules", "egress_rules"]} actions={(row) => <button className="danger-button" onClick={() => onDeleteNetworkPolicy(row)}><Trash2 size={15} /></button>} />
      </section>

      <section className="dashboard-grid">
        <div className="panel dashboard-panel">
          <h2><Globe2 size={18} /> Service access addresses</h2>
          <div className="mini-table">
            {(data.access || []).map((item) => (
              <div key={`${item.namespace}/${item.service}/${item.ingress || item.type}`}>
                <span>{item.namespace}/{item.service}<small>{[item.type, item.class, item.tls ? "TLS" : "", item.load_balancer].filter(Boolean).join(" · ")}</small></span>
                <b>{[...(item.urls || []), ...(item.node_ports || []).map((port) => `NodePort ${port}`)].join(" · ") || "-"}</b>
              </div>
            ))}
            {(data.access || []).length === 0 && <div className="empty">No service access addresses reported.</div>}
          </div>
        </div>
        <div className="panel dashboard-panel">
          <h2><Layers3 size={18} /> Endpoints</h2>
          <SimpleTable rows={data.endpoints || []} columns={["namespace", "name", "addresses", "ports"]} compact />
        </div>
      </section>
    </div>
  );
}

function IngressForm({onSubmit}) {
  return (
    <form className="form-grid ingress-form" onSubmit={onSubmit}>
      <input name="namespace" placeholder="namespace" required />
      <input name="name" placeholder="ingress-name" required />
      <select name="class_name" defaultValue="traefik">
        <option value="traefik">Traefik public</option>
        <option value="tailscale">Tailscale private</option>
        <option value="nginx">nginx</option>
      </select>
      <input name="host" placeholder="app.example.com or app.tailnet.ts.net" required />
      <input name="path" placeholder="path, default /" />
      <input name="service_name" placeholder="service name" required />
      <input name="service_port" type="number" min="1" max="65535" placeholder="service port" required />
      <input name="tls_secret_name" placeholder="TLS secret, e.g. app-tls" />
      <input name="annotations" placeholder="annotations: cert-manager.io/cluster-issuer=letsencrypt-prod" />
      <input name="labels" placeholder="labels: app=my-app" />
      <button className="primary" type="submit">Save ingress</button>
    </form>
  );
}

function NetworkPolicyForm({onSubmit}) {
  return (
    <form className="form-grid network-policy-form" onSubmit={onSubmit}>
      <input name="namespace" placeholder="namespace" required />
      <input name="name" placeholder="policy-name" required />
      <input name="pod_selector" placeholder="pod selector: app=my-app" />
      <label className="check-row"><input name="policy_types" type="checkbox" value="Ingress" defaultChecked /> Ingress</label>
      <label className="check-row"><input name="policy_types" type="checkbox" value="Egress" /> Egress</label>
      <label className="check-row"><input name="allow_same_namespace" type="checkbox" /> Allow same namespace</label>
      <label className="check-row"><input name="allow_dns" type="checkbox" /> Allow DNS egress</label>
      <input name="labels" placeholder="labels: managed-by=beancs" />
      <button className="primary" type="submit">Save policy</button>
    </form>
  );
}

function SimpleTable({rows, columns, actions, compact = false}) {
  return (
    <div className={compact ? "table compact-table" : "table network-table"}>
      <div className="tr head">{columns.map((column) => <span key={column}>{column.replaceAll("_", " ")}</span>)}{actions && <span>Actions</span>}</div>
      {(rows || []).map((row, index) => (
        <div className="tr" key={`${row.namespace || ""}-${row.name || row.service || index}`}>
          {columns.map((column) => <span key={column}>{formatCell(row[column])}</span>)}
          {actions && <span className="row-actions">{actions(row)}</span>}
        </div>
      ))}
      {(!rows || rows.length === 0) && <div className="empty">No records found.</div>}
    </div>
  );
}

function RuntimeTable({kind, rows, nodeJoinCommand, onLoadNodeJoinCommand, onCreateNamespace, onPatchNamespace, onNamespaceDetail, onDeleteNamespace, onDeletePod, onNodeDetail, onPodLogs, onSaveService, onDeleteService, onDetail}) {
  const keys = rows[0] ? Object.keys(rows[0]).slice(0, 7) : [];
  return (
    <div className="stack">
      {kind === "namespaces" && (
        <section className="panel">
          <h2><Layers3 size={18} /> Create namespace</h2>
          <form className="form-grid inline-form" onSubmit={onCreateNamespace}>
            <input name="name" placeholder="namespace-name" required />
            <input name="labels" placeholder="labels: env=dev,team=platform" />
            <button className="primary" type="submit"><Plus size={15} /> Create</button>
          </form>
        </section>
      )}
      {kind === "services" && (
        <section className="panel">
          <h2><Database size={18} /> Create service</h2>
          <ServiceForm onSubmit={(event) => onSaveService(event)} namespaces={[]} />
        </section>
      )}
      {kind === "nodes" && <NodeJoinPanel command={nodeJoinCommand} onLoad={onLoadNodeJoinCommand} />}
      <section className="panel">
        <div className="table runtime-table">
          <div className="tr head">{keys.map((key) => <span key={key}>{key.replaceAll("_", " ")}</span>)}<span>Actions</span></div>
          {rows.map((row, index) => (
            <div className="tr" key={`${kind}-${row.namespace || ""}-${row.name || index}`}>
              {keys.map((key) => <span key={key}>{formatCell(row[key])}</span>)}
              <span className="row-actions">
                <button onClick={() => kind === "nodes" ? onNodeDetail(row) : kind === "namespaces" ? onNamespaceDetail(row.name) : onDetail({kind, row})}>Details</button>
                {kind === "namespaces" && <button onClick={() => onDeleteNamespace(row.name)} className="danger-button"><Trash2 size={15} /></button>}
                {kind === "pods" && <><button onClick={() => onPodLogs(row)}>Logs</button><button onClick={() => onDeletePod(row)} className="danger-button"><Trash2 size={15} /></button></>}
                {kind === "services" && <><button onClick={() => onDetail({kind: "service-edit", row})}>Edit</button><button onClick={() => onDeleteService(row)} className="danger-button"><Trash2 size={15} /></button></>}
              </span>
            </div>
          ))}
          {rows.length === 0 && <div className="empty">No {kind} found.</div>}
        </div>
      </section>
    </div>
  );
}

function NodeJoinPanel({command, onLoad}) {
  return (
    <section className="panel node-ops-panel">
      <div className="action-panel">
        <div>
          <h2><Server size={18} /> K3s node join</h2>
          <p>Generate an agent or server join command from the configured K3s server URL and node token.</p>
        </div>
        <div className="row-actions">
          <button onClick={() => onLoad("agent")}>Agent command</button>
          <button onClick={() => onLoad("server")}>Server command</button>
        </div>
      </div>
      {command?.configured ? (
        <pre className="command-box">{command.command}</pre>
      ) : (
        <p className="muted">{command?.message || "Loading join command configuration..."}</p>
      )}
    </section>
  );
}

function RuntimeDetailModal({detail, logs, logFollow, logStatus, selectedLogContainer, logTail, logLoaded, nodeHealth, onLoadNodeHealth, onSaveNodeLabels, onSaveNodeTaints, onCordonNode, onDrainNode, onDeleteNode, onSaveResourceQuota, onDeleteResourceQuota, onSaveLimitRange, onDeleteLimitRange, onSaveNamespacePermission, onDeleteNamespacePermission, onSaveNamespaceIsolation, onSelectLogContainer, onSetLogTail, onLoadContainerLogs, onFollowPodLogs, onStopPodLogs, onClose, onSaveService, onPatchNamespace}) {
  const row = detail.row || {};
  return (
    <div className="modal-backdrop">
      <div className="modal wide-modal">
        <h2>{detail.kind} · {row.namespace ? `${row.namespace}/` : ""}{row.name}</h2>
        {detail.kind === "service-edit" ? (
          <ServiceForm existing={row} onSubmit={(event) => onSaveService(event, row)} />
        ) : detail.kind === "namespaces" ? (
          <form className="form-grid" onSubmit={(event) => { event.preventDefault(); onPatchNamespace(row.name, event.currentTarget.labels.value); onClose(); }}>
            <label>Labels</label>
            <textarea name="labels" defaultValue={formatKeyValues(row.labels)} />
            <button className="primary">Save labels</button>
          </form>
        ) : detail.kind === "pod" ? (
          <>
            <ContainerLogViewer
              pod={row}
              logs={logs}
              logFollow={logFollow}
              logStatus={logStatus}
              selectedContainer={selectedLogContainer}
              tail={logTail}
              loaded={logLoaded}
              onSelectContainer={onSelectLogContainer}
              onSetTail={onSetLogTail}
              onLoad={() => onLoadContainerLogs(row, selectedLogContainer, logTail)}
              onFollow={() => onFollowPodLogs(row, selectedLogContainer, logTail)}
              onStop={onStopPodLogs}
            />
          </>
        ) : detail.kind === "node" ? (
          <NodeDetailView detail={detail} health={nodeHealth} onLoadHealth={onLoadNodeHealth} onSaveLabels={onSaveNodeLabels} onSaveTaints={onSaveNodeTaints} onCordon={onCordonNode} onDrain={onDrainNode} onDelete={onDeleteNode} />
        ) : detail.kind === "namespace-detail" ? (
          <NamespaceDetailView detail={detail} onPatchNamespace={onPatchNamespace} onSaveResourceQuota={onSaveResourceQuota} onDeleteResourceQuota={onDeleteResourceQuota} onSaveLimitRange={onSaveLimitRange} onDeleteLimitRange={onDeleteLimitRange} onSavePermission={onSaveNamespacePermission} onDeletePermission={onDeleteNamespacePermission} onSaveIsolation={onSaveNamespaceIsolation} />
        ) : (
          <div className="detail-list">{Object.entries(row).map(([key, value]) => <span key={key}>{key.replaceAll("_", " ")} <b>{formatCell(value)}</b></span>)}</div>
        )}
        <div className="modal-actions"><button type="button" onClick={onClose}>Close</button></div>
      </div>
    </div>
  );
}

function ContainerLogViewer({pod, logs, logFollow, logStatus, selectedContainer, tail, loaded, onSelectContainer, onSetTail, onLoad, onFollow, onStop}) {
  const containers = podContainers(pod);
  const canRead = Boolean(selectedContainer);
  return (
    <div className="container-log-viewer">
      <div className="log-header">
        <span className="muted">{logStatus || "Choose a container to load logs."}</span>
        <div className="row-actions">
          <select className="compact-select" value={tail} disabled={logFollow} onChange={(event) => onSetTail(Number(event.target.value))}>
            <option value={100}>Last 100 lines</option>
            <option value={200}>Last 200 lines</option>
            <option value={500}>Last 500 lines</option>
            <option value={1000}>Last 1000 lines</option>
          </select>
          <button disabled={!canRead || logFollow} onClick={onLoad}><RefreshCw size={15} /> Load</button>
          {logFollow ? (
            <button onClick={onStop}>Stop follow</button>
          ) : (
            <button className="primary" disabled={!canRead} onClick={onFollow}>Follow live</button>
          )}
        </div>
      </div>
      <div className="container-picker">
        {containers.map((container) => (
          <button
            key={container.name}
            className={selectedContainer === container.name ? "container-chip active" : "container-chip"}
            onClick={() => onSelectContainer(container.name)}
            type="button"
            disabled={logFollow}
          >
            <b>{container.name}</b>
            {container.image && <small>{container.image}</small>}
          </button>
        ))}
        {containers.length === 0 && <div className="empty">No containers reported for this pod.</div>}
      </div>
      <pre className="modal-log">{loaded ? (logs || "No logs returned for this container.") : "Logs are not loaded yet. Select a container, then click Load or Follow live."}</pre>
    </div>
  );
}

function podContainers(pod) {
  return (pod.containers || []).map((value) => {
    const text = String(value || "");
    const [name, ...rest] = text.split(":");
    return {name: name || text, image: rest.join(":")};
  }).filter((container) => container.name);
}

function NodeDetailView({detail, health, onLoadHealth, onSaveLabels, onSaveTaints, onCordon, onDrain, onDelete}) {
  const row = detail.row || {};
  const summary = row.summary || row;
  const usage = row.usage || {};
  const disk = row.disk || {};
  const network = row.network || {};
  const pods = row.pods || [];
  const conditions = row.conditions || [];
  const nodeName = summary.name || row.name;
  return (
    <div className="node-detail">
      {detail.loading && <p className="muted">Loading live node status...</p>}
      {detail.error && <p className="error-inline">{detail.error}</p>}
      <section className="node-section node-actions">
        <div className="row-actions">
          <button onClick={() => onLoadHealth(nodeName)}><CheckCircle2 size={15} /> Health check</button>
          <button onClick={() => onCordon(nodeName, false)}>Cordon</button>
          <button onClick={() => onCordon(nodeName, true)}>Uncordon</button>
          <button onClick={() => onDrain(nodeName, {force: false, ignore_daemonsets: true, delete_emptydir_data: false, grace_period_seconds: 30})}>Drain safe</button>
          <button className="danger-button" onClick={() => onDelete(nodeName)}><Trash2 size={15} /> Delete node</button>
        </div>
        {health && (
          <div className={health.healthy ? "health-card good" : "health-card warning"}>
            <b>{health.status}</b>
            <span>{(health.checks || []).length} checks · {(health.abnormal_pods || []).length} abnormal pods · {formatTime(health.checked_at)}</span>
            {(health.checks || []).map((check) => <small key={`${check.name}-${check.message}`}>{check.name}: {check.status}{check.message ? ` · ${check.message}` : ""}</small>)}
          </div>
        )}
      </section>
      <div className="node-status-grid">
        <div className="runtime-summary">
          <strong>{summary.status || "-"}</strong>
          <span>{summary.name} · {summary.version || "-"}</span>
        </div>
        <div className="detail-list compact-details">
          <span>Internal IP <b>{summary.internal_ip || row.addresses?.InternalIP || "-"}</b></span>
          <span>Roles <b>{(summary.roles || []).join(", ") || "-"}</b></span>
          <span>Scheduling <b>{summary.schedulable === false ? "Cordoned" : "Schedulable"}</b></span>
          <span>Pods <b>{pods.length}/{row.allocatable?.pods || "-"}</b></span>
          <span>Checked <b>{formatTime(row.checked_at)}</b></span>
        </div>
      </div>
      <section className="node-section">
        <h3>Live resources</h3>
        {(row.metrics_available || row.disk || row.network) ? (
          <div className="resource-grid">
            <ResourceMeter label="CPU allocatable" value={usage.cpu_allocatable_percent} detail={row.metrics_available ? `${usage.cpu_millis || 0}m / ${row.allocatable?.cpu_millis || 0}m` : "metrics-server unavailable"} />
            <ResourceMeter label="Memory allocatable" value={usage.memory_allocatable_percent} detail={row.metrics_available ? `${formatBytes(usage.memory_bytes)} / ${formatBytes(row.allocatable?.memory_bytes)}` : "metrics-server unavailable"} />
            <ResourceMeter label="CPU capacity" value={usage.cpu_capacity_percent} detail={row.metrics_available ? `${usage.cpu || "-"} / ${row.capacity?.cpu || "-"}` : "metrics-server unavailable"} />
            <ResourceMeter label="Memory capacity" value={usage.memory_capacity_percent} detail={row.metrics_available ? `${usage.memory || "-"} / ${row.capacity?.memory || "-"}` : "metrics-server unavailable"} />
            <ResourceMeter label="Disk" value={disk.used_percent} detail={`${formatBytes(disk.used_bytes)} / ${formatBytes(disk.capacity_bytes)}`} />
            <ResourceMeter label="Network" value={0} detail={`RX ${formatBytes(network.rx_bytes)} · TX ${formatBytes(network.tx_bytes)}`} />
          </div>
        ) : (
          <p className="muted">Metrics unavailable{row.metrics_error ? `: ${row.metrics_error}` : ". Install metrics-server to show live CPU and memory usage."}</p>
        )}
      </section>
      <section className="node-section">
        <h3>Conditions</h3>
        <div className="condition-grid">
          {conditions.map((condition) => (
            <div className={condition.status === "True" && condition.type === "Ready" ? "condition good" : condition.status === "True" ? "condition warning" : "condition"} key={condition.type}>
              <b>{condition.type}: {condition.status}</b>
              <small>{condition.reason || "-"} · {formatTime(condition.last_transition_time)}</small>
              {condition.message && <p>{condition.message}</p>}
            </div>
          ))}
        </div>
      </section>
      <section className="node-section">
        <h3>System</h3>
        <div className="detail-list compact-details">
          {Object.entries(row.system_info || {}).map(([key, value]) => <span key={key}>{key.replaceAll("_", " ")} <b>{value || "-"}</b></span>)}
        </div>
      </section>
      <section className="node-section">
        <h3>Pods on this node</h3>
        <div className="mini-table">
          {pods.map((pod) => (
            <div key={`${pod.namespace}/${pod.name}`}>
              <span>{pod.namespace}/{pod.name}<small>{(pod.containers || []).join(" · ")}</small></span>
              <b>{pod.ready_containers}/{pod.total_containers} · {pod.status}</b>
            </div>
          ))}
          {pods.length === 0 && <div className="empty">No pods scheduled on this node.</div>}
        </div>
      </section>
      <section className="node-section">
        <h3>Labels</h3>
        <form className="form-grid node-edit-form" onSubmit={(event) => { event.preventDefault(); onSaveLabels(nodeName, event.currentTarget.labels.value); }}>
          <textarea name="labels" defaultValue={formatKeyValues(row.labels)} />
          <button className="primary">Save labels</button>
        </form>
        <div className="label-cloud">
          {Object.entries(row.labels || {}).map(([key, value]) => <span key={key}>{key}={value}</span>)}
        </div>
      </section>
      <section className="node-section">
        <h3>Taints</h3>
        <form className="form-grid node-edit-form" onSubmit={(event) => { event.preventDefault(); onSaveTaints(nodeName, event.currentTarget.taints.value); }}>
          <textarea name="taints" placeholder="key=value:NoSchedule, dedicated=gpu:NoExecute" defaultValue={taintsToForm(row.taints || [])} />
          <button className="primary">Save taints</button>
        </form>
        <div className="signal-list">
          {(row.taints || []).map((taint) => <span key={taint}>{taint}</span>)}
          {(row.taints || []).length === 0 && <span>No taints</span>}
        </div>
      </section>
    </div>
  );
}

function NamespaceDetailView({detail, onPatchNamespace, onSaveResourceQuota, onDeleteResourceQuota, onSaveLimitRange, onDeleteLimitRange, onSavePermission, onDeletePermission, onSaveIsolation}) {
  const row = detail.row || {};
  const summary = row.summary || row;
  const namespace = summary.name || row.name;
  const stats = row.stats || {};
  const isolation = row.isolation || {};
  return (
    <div className="namespace-detail">
      {detail.loading && <p className="muted">Loading namespace detail...</p>}
      {detail.error && <p className="error-inline">{detail.error}</p>}
      <div className="dashboard-kpis">
        <MetricCard icon={Boxes} label="Pods" value={stats.pods || 0} detail={`${stats.running_pods || 0} running · ${stats.abnormal_pods || 0} abnormal`} />
        <MetricCard icon={Database} label="Services" value={stats.services || 0} detail={`${stats.deployments || 0} deployments`} />
        <MetricCard icon={Network} label="Ingresses" value={stats.ingresses || 0} detail={`${stats.network_policies || 0} policies`} />
        <MetricCard icon={KeyRound} label="Secrets" value={stats.secrets || 0} detail={`${stats.config_maps || 0} configmaps`} />
        <MetricCard icon={Shield} label="Isolation" value={isolation.enabled ? "On" : "Off"} detail={isolation.policy_name || "No default isolation"} tone={isolation.enabled ? "good" : "warning"} />
        <MetricCard icon={ListRestart} label="Checked" value={formatTime(row.checked_at)} detail={summary.status || "-"} />
      </div>

      <section className="node-section">
        <h3>Namespace labels</h3>
        <form className="form-grid node-edit-form" onSubmit={(event) => { event.preventDefault(); onPatchNamespace(namespace, event.currentTarget.labels.value); }}>
          <textarea name="labels" defaultValue={formatKeyValues(summary.labels)} />
          <button className="primary">Save labels</button>
        </form>
      </section>

      <section className="node-section">
        <h3>ResourceQuota</h3>
        <form className="form-grid quota-form" onSubmit={(event) => onSaveResourceQuota(namespace, event)}>
          <input name="name" placeholder="quota name" defaultValue="default-quota" required />
          <input name="hard" placeholder="requests.cpu=4,requests.memory=8Gi,limits.cpu=8,pods=40" required />
          <button className="primary">Save quota</button>
        </form>
        <SimpleTable rows={row.resource_quotas || []} columns={["name", "hard", "used"]} actions={(quota) => <button className="danger-button" onClick={() => onDeleteResourceQuota(namespace, quota.name)}><Trash2 size={15} /></button>} compact />
      </section>

      <section className="node-section">
        <h3>LimitRange</h3>
        <form className="form-grid limit-form" onSubmit={(event) => onSaveLimitRange(namespace, event)}>
          <input name="name" placeholder="limit range name" defaultValue="default-limits" required />
          <select name="type" defaultValue="Container"><option>Container</option><option>Pod</option><option>PersistentVolumeClaim</option></select>
          <input name="default_request" placeholder="default request: cpu=100m,memory=128Mi" />
          <input name="default" placeholder="default limit: cpu=500m,memory=512Mi" />
          <input name="min" placeholder="min: cpu=50m,memory=64Mi" />
          <input name="max" placeholder="max: cpu=2,memory=2Gi" />
          <button className="primary">Save limits</button>
        </form>
        <SimpleTable rows={row.limit_ranges || []} columns={["name", "types"]} actions={(limit) => <button className="danger-button" onClick={() => onDeleteLimitRange(namespace, limit.name)}><Trash2 size={15} /></button>} compact />
      </section>

      <section className="node-section">
        <h3>Namespace permissions</h3>
        <form className="form-grid permission-form" onSubmit={(event) => onSavePermission(namespace, event)}>
          <input name="name" placeholder="permission name" defaultValue="namespace-admin" required />
          <input name="subjects" placeholder="subjects: User:alice,Group:platform,ServiceAccount:builder" required />
          <input name="verbs" placeholder="verbs: get,list,watch,create,update,delete" defaultValue="get,list,watch" required />
          <input name="resources" placeholder="resources: pods,services,deployments" defaultValue="pods,services" required />
          <input name="api_groups" placeholder="api groups: ,apps,networking.k8s.io" />
          <button className="primary">Save permission</button>
        </form>
        <SimpleTable rows={row.role_bindings || []} columns={["name", "role_ref", "subjects"]} actions={(binding) => <button className="danger-button" onClick={() => onDeletePermission(namespace, binding.name)}><Trash2 size={15} /></button>} compact />
      </section>

      <section className="node-section">
        <h3>Namespace isolation</h3>
        <form className="form-grid isolation-form" onSubmit={(event) => onSaveIsolation(namespace, event)}>
          <label className="check-row"><input name="enabled" type="checkbox" defaultChecked={Boolean(isolation.enabled)} /> Enable default deny isolation</label>
          <label className="check-row"><input name="allow_same_namespace" type="checkbox" defaultChecked={Boolean(isolation.allow_same_namespace)} /> Allow same namespace traffic</label>
          <label className="check-row"><input name="allow_dns" type="checkbox" defaultChecked={Boolean(isolation.allow_dns)} /> Allow DNS egress</label>
          <button className="primary">Save isolation</button>
        </form>
      </section>
    </div>
  );
}

function ResourceMeter({label, value, detail}) {
  const normalized = Math.max(0, Math.min(100, Number(value || 0)));
  return (
    <div className="resource-meter">
      <div><b>{label}</b><span>{normalized.toFixed(1)}%</span></div>
      <progress value={normalized} max="100" />
      <small>{detail}</small>
    </div>
  );
}

function ServiceForm({existing, onSubmit}) {
  return (
    <form className="form-grid inline-form service-form" onSubmit={onSubmit}>
      {!existing && <input name="namespace" placeholder="namespace" required />}
      {!existing && <input name="name" placeholder="service-name" required />}
      <select name="type" defaultValue={existing?.type || "ClusterIP"}><option>ClusterIP</option><option>NodePort</option><option>LoadBalancer</option></select>
      <input name="selector" placeholder="selector: app=my-app,managed-by=beancs" defaultValue={formatKeyValues(existing?.selector)} />
      <input name="ports" placeholder="ports: http:80:8080:30080/TCP,https:443:8443/TCP" defaultValue={portsToForm(existing?.ports)} required />
      <input name="labels" placeholder="labels: app=my-app" defaultValue={formatKeyValues(existing?.labels)} />
      <input name="load_balancer_ip" placeholder="LoadBalancer IP, optional" />
      <input name="external_ips" placeholder="External IPs: 1.2.3.4,5.6.7.8" />
      <select name="external_traffic_policy" defaultValue="">
        <option value="">Traffic policy</option>
        <option value="Cluster">Cluster</option>
        <option value="Local">Local</option>
      </select>
      <button className="primary" type="submit">{existing ? "Save service" : "Create service"}</button>
    </form>
  );
}

function CredentialManager({kind, rows, onCreate, onDelete}) {
  const isCloudflare = kind === "cloudflare";
  const title = isCloudflare ? "Cloudflare credentials" : "BasaltPass instances";
  const columns = isCloudflare ? ["name", "domain", "zone_id", "account_id"] : ["name", "base_url", "client_id"];
  return (
    <div className="stack">
      <section className="panel">
        <h2><KeyRound size={18} /> Add {isCloudflare ? "Cloudflare key" : "BasaltPass instance"}</h2>
        <form className="form-grid" onSubmit={(event) => onCreate(kind, event)}>
          <input name="name" placeholder="Name" required />
          {isCloudflare ? (
            <>
              <input name="account_id" placeholder="Account ID, optional" />
              <input name="api_token" type="password" placeholder="Cloudflare API token" required />
            </>
          ) : (
            <>
              <input name="base_url" placeholder="https://auth.example.com" required />
              <input name="client_id" placeholder="Management client ID" required />
              <input name="client_secret" type="password" placeholder="Management client secret" required />
              <input name="service_token" type="password" placeholder="Service token, optional" />
            </>
          )}
          <button className="primary" type="submit"><Plus size={16} /> Add</button>
        </form>
      </section>
      <section className="panel">
        <h2><KeyRound size={18} /> {title}</h2>
        <div className="table compact">
          <div className="tr head">{columns.map((column) => <span key={column}>{column.replaceAll("_", " ")}</span>)}<span>Actions</span></div>
          {rows.map((row) => (
            <div className="tr" key={row.id}>
              {columns.map((column) => <span key={column}>{row[column] || "-"}</span>)}
              <span><button onClick={() => onDelete(kind, row.id)}><Trash2 size={15} /></button></span>
            </div>
          ))}
          {rows.length === 0 && <div className="empty">No credentials found.</div>}
        </div>
      </section>
    </div>
  );
}

function ProjectModal({project, onClose, onSubmit}) {
  return (
    <div className="modal-backdrop">
      <form className="modal" onSubmit={onSubmit}>
        <h2>Edit {project.name}</h2>
        <label>Display name</label>
        <input name="display_name" defaultValue={project.display_name || ""} />
        <label>Description</label>
        <textarea name="description" defaultValue={project.description || ""} />
        <label>Replicas</label>
        <input name="replicas" type="number" min="0" max="20" defaultValue={project.replicas || 1} />
        <label>Status</label>
        <select name="status" defaultValue={project.status || "active"}>
          <option value="active">Active</option>
          <option value="suspended">Suspended</option>
          <option value="deleted">Deleted</option>
        </select>
        {project.build_source === "github" && (
          <label className="checkbox-row">
            <input name="auto_deploy" type="checkbox" defaultChecked={project.auto_deploy !== false} />
            Auto build and deploy on GitHub push
          </label>
        )}
        <div className="modal-actions">
          <button type="button" onClick={onClose}>Cancel</button>
          <button className="primary" type="submit">Save</button>
        </div>
      </form>
    </div>
  );
}

function Field({label, value, onChange, type = "text", required = false}) {
  return (
    <>
      <label>{label}</label>
      <input type={type} value={value ?? ""} required={required} onChange={(event) => onChange(event.target.value)} />
    </>
  );
}

function defaultDeployForm() {
  return {
    deploy_source: "gitops",
    build_source: "github",
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
    auto_deploy: true,
    image_reference: "",
    source_archive_name: "",
    basaltpass_instance_id: "",
    cloudflare_credential_id: "",
    exposure_mode: "private",
    subdomain: "",
    private_host: "",
    port: 8080,
    replicas: 1,
    resource_preset: "small",
  };
}

function buildProjectPayload(form, githubCredentialID, credentials) {
  const exposure = form.exposure_mode;
  const selectedCF = credentials.cloudflare.find((cred) => String(cred.id) === String(form.cloudflare_credential_id));
  const domain = exposure === "public" && selectedCF ? `${form.subdomain}.${selectedCF.domain}` : exposure === "private" ? form.private_host : "";
  const source = form.deploy_source === "registry" ? "registry" : "github";
  return {
    build_source: source,
    name: form.name,
    namespace: form.namespace || undefined,
    image_reference: form.image_reference || undefined,
    source_archive_name: form.source_archive_name || undefined,
    github_credential_id: source === "github" ? Number(githubCredentialID) : undefined,
    github_repo: source === "github" ? form.github_repo : undefined,
    github_branch: form.github_branch || "main",
    dockerfile_path: form.dockerfile_path || "Dockerfile",
    auto_deploy: source === "github" ? form.update_mode === "argocd" : false,
    basaltpass_instance_id: form.basaltpass_instance_id ? Number(form.basaltpass_instance_id) : undefined,
    cloudflare_credential_id: exposure === "public" ? Number(form.cloudflare_credential_id) : undefined,
    exposure_mode: exposure,
    subdomain: form.subdomain || undefined,
    resource_preset: form.resource_preset || "small",
    port: Number(form.port || 8080),
    replicas: Number(form.replicas || 1),
    ports: [{name: "http", port: Number(form.port || 8080), protocol: "http", exposure, domain}],
    env: {},
  };
}

function imageName(image) {
  const withoutDigest = String(image || "").split("@")[0];
  const slash = withoutDigest.lastIndexOf("/");
  const colon = withoutDigest.lastIndexOf(":");
  const value = colon > slash ? withoutDigest.slice(0, colon) : withoutDigest;
  return value.split("/").filter(Boolean).pop() || "app";
}

function imageReferenceFromTrackedImage(image, tag = "") {
  if (!image) return "";
  const registry = registryHostFromAPIBase(image.registry?.api_base || "");
  const repository = String(image.repository || "").replace(/^\/+/, "");
  const normalizedTag = tag || (image.tags || [])[0] || "latest";
  return `${registry ? `${registry}/` : ""}${repository}:${normalizedTag}`;
}

function registryHostFromAPIBase(apiBase) {
  const value = String(apiBase || "").trim();
  if (!value) return "";
  try {
    const url = new URL(value);
    return url.host;
  } catch {
    return value.replace(/^https?:\/\//, "").replace(/\/v2\/?$/, "").replace(/\/+$/, "");
  }
}

function imageTagFromReference(image) {
  const value = String(image || "");
  const withoutDigest = value.split("@")[0];
  const slash = withoutDigest.lastIndexOf("/");
  const colon = withoutDigest.lastIndexOf(":");
  return colon > slash ? withoutDigest.slice(colon + 1) : "latest";
}

function profileFromToken(token) {
  const fallback = {name: "Signed in user", detail: "BeanCS session", initial: "U", scopes: []};
  if (!token || !token.includes(".")) return fallback;
  try {
    const payload = JSON.parse(base64URLDecode(token.split(".")[1]));
    const name = payload.name || payload.preferred_username || payload.email || payload.sub || fallback.name;
    const detail = payload.email && payload.email !== name ? payload.email : payload.preferred_username && payload.preferred_username !== name ? payload.preferred_username : "BeanCS session";
    return {name, detail, initial: String(name).trim().slice(0, 1).toUpperCase() || "U", scopes: String(payload.scope || "").split(/\s+/).filter(Boolean)};
  } catch {
    return fallback;
  }
}

function base64URLDecode(value) {
  const normalized = String(value || "").replace(/-/g, "+").replace(/_/g, "/");
  const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, "=");
  return decodeURIComponent(Array.from(atob(padded), (char) => `%${char.charCodeAt(0).toString(16).padStart(2, "0")}`).join(""));
}

function makeAPI(token, onUnauthorized) {
  async function request(path, options = {}) {
    const res = await fetch(API + path, {
      ...options,
      headers: {
        ...(options.body ? {"Content-Type": "application/json"} : {}),
        Authorization: `Bearer ${token}`,
        ...(options.headers || {}),
      },
    });
    const data = await res.json().catch(() => ({}));
    if (res.status === 401 && isSessionAuthError(data)) onUnauthorized();
    if (!res.ok) throw new Error(data.error || data.error_description || "Request failed");
    return data;
  }
  async function stream(path, options = {}) {
    const res = await fetch(API + path, {
      ...options,
      headers: {
        Authorization: `Bearer ${token}`,
        ...(options.headers || {}),
      },
    });
    if (!res.ok) {
      const data = await res.json().catch(() => ({}));
      if (res.status === 401 && isSessionAuthError(data)) onUnauthorized();
      throw new Error(data.error || data.error_description || "Request failed");
    }
    return res;
  }
  return {
    get: (path) => request(path),
    post: (path, body) => request(path, {method: "POST", body: JSON.stringify(body)}),
    put: (path, body) => request(path, {method: "PUT", body: JSON.stringify(body)}),
    patch: (path, body) => request(path, {method: "PATCH", body: JSON.stringify(body)}),
    delete: (path) => request(path, {method: "DELETE"}),
    stream,
  };
}

function isSessionAuthError(data) {
  const error = String(data?.error || data?.error_description || "").toLowerCase();
  return error === "missing token" || error === "invalid token" || error === "invalid api key";
}

async function consumeTextStream(res, onChunk) {
  const reader = res.body?.getReader();
  if (!reader) throw new Error("Streaming logs are not supported by this browser.");
  const decoder = new TextDecoder();
  while (true) {
    const {value, done} = await reader.read();
    if (done) break;
    onChunk(decoder.decode(value, {stream: true}));
  }
  const remaining = decoder.decode();
  if (remaining) onChunk(remaining);
}

function trimLiveLog(value) {
  const maxLength = 200000;
  if (value.length <= maxLength) return value;
  return value.slice(value.length - maxLength);
}

async function publicJSON(path, options = {}) {
  const res = await fetch(path, options);
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || "Request failed");
  return data;
}

async function finishLogin(config) {
  const params = new URLSearchParams(location.search);
  const code = params.get("code");
  const returnedState = params.get("state");
  const expectedState = sessionStorage.getItem("beancs.oauthState");
  const verifier = sessionStorage.getItem("beancs.pkceVerifier");
  if (!code || !verifier || returnedState !== expectedState) throw new Error("Login callback was incomplete.");
  const data = await publicJSON(`${API}/ui/oauth/token`, {
    method: "POST",
    headers: {"Content-Type": "application/json"},
    body: JSON.stringify({code, redirect_uri: browserRedirectURI(), code_verifier: verifier}),
  });
  return data.access_token;
}

function titleFor(view) {
  const map = {
    dashboard: "Overview",
    deploy: "Deploy",
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

function subtitleFor(view, runtime, projects) {
  if (view === "dashboard") return "Real-time cluster health and operating signals";
  if (view === "networking") return "Service, Ingress, Endpoint, NetworkPolicy, Traefik and Tailscale operations";
  if (view === "projects") return `${projects.length} managed projects`;
  if (view === "progress") return "Watch installs and runtime readiness";
  if (view === "registries") return "Register OCI mirrors and sync image tags for this account";
  if (view === "workloadImage") return "Tracked registry tags; use Image Registry to add mirrors";
  if (view === "apiKeys") return "Issue and revoke API keys for automation";
  if (view === "accessControl") return "BasaltPass and access integrations";
  if (view === "settings") return "Workspace and version information";
  if (view === "storage" || view === "secrets") return "Planned console capabilities";
  if (view === "alerts") return "Active warning signals and degraded runtime objects";
  if (view === "events") return "Recent Kubernetes warning events and reason groups";
  if (view === "metrics") return "Cluster resource utilization and node readings";
  if (view === "logs") return "Project log snapshots and live streaming";
  if (runtime[view]) return `${(runtime[view] || []).length} cluster resources`;
  return "Operate k3s, GitHub, DNS, and traffic from one console.";
}

function formatTime(value) {
  if (!value) return "-";
  return new Date(value).toLocaleString();
}

function formatBytes(value) {
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

function formatPercent(value) {
  return Number(value || 0).toFixed(0);
}

function formatDuration(seconds) {
  const value = Number(seconds || 0);
  if (!value) return "-";
  const days = Math.floor(value / 86400);
  const hours = Math.floor((value % 86400) / 3600);
  const minutes = Math.floor((value % 3600) / 60);
  if (days > 0) return `${days}d ${hours}h`;
  if (hours > 0) return `${hours}h ${minutes}m`;
  return `${minutes}m`;
}

function formatCell(value) {
  if (Array.isArray(value)) return value.join(", ") || "-";
  if (typeof value === "object" && value !== null) return formatKeyValues(value);
  if (typeof value === "boolean") return value ? "Yes" : "No";
  if (value === null || value === undefined || value === "") return "-";
  return String(value);
}

function parseKeyValues(value) {
  if (!value) return {};
  return String(value).split(",").map((item) => item.trim()).filter(Boolean).reduce((out, item) => {
    const [key, ...rest] = item.split("=");
    if (key?.trim()) out[key.trim()] = rest.join("=").trim();
    return out;
  }, {});
}

function formatKeyValues(value) {
  if (!value || typeof value !== "object") return "";
  return Object.entries(value).map(([key, val]) => `${key}=${val}`).join(",");
}

function parseTaints(value) {
  return String(value || "").split(",").map((item) => item.trim()).filter(Boolean).map((item) => {
    const [left, effect = "NoSchedule"] = item.split(":");
    const [key, ...valueParts] = left.split("=");
    return {key: key.trim(), value: valueParts.join("=").trim(), effect: effect.trim() || "NoSchedule"};
  }).filter((taint) => taint.key);
}

function parseCSV(value) {
  return String(value || "").split(",").map((item) => item.trim()).filter(Boolean);
}

function parsePermissionSubjects(value, namespace) {
  return parseCSV(value).map((item) => {
    const [kind = "User", name = item, subjectNamespace = ""] = item.split(":");
    return {
      kind: kind.trim(),
      name: name.trim(),
      namespace: subjectNamespace.trim() || (kind.trim() === "ServiceAccount" ? namespace : ""),
    };
  }).filter((subject) => subject.name);
}

function taintsToForm(taints) {
  return (taints || []).join(",");
}

function parseServicePorts(value) {
  return String(value || "").split(",").map((item) => item.trim()).filter(Boolean).map((item) => {
    const [left, protocol = "TCP"] = item.split("/");
    const parts = left.split(":");
    const hasName = parts.length > 1 && Number.isNaN(Number(parts[0]));
    const port = hasName ? Number(parts[1]) : Number(parts[0]);
    return {
      name: hasName ? parts[0] : "",
      port,
      target_port: Number(hasName ? (parts[2] || parts[1]) : (parts[1] || parts[0])),
      node_port: Number(hasName ? (parts[3] || 0) : (parts[2] || 0)),
      protocol: protocol || "TCP",
    };
  });
}

function portsToForm(ports) {
  if (!Array.isArray(ports)) return "";
  return ports.map((port) => String(port)).join(",");
}

function localDateTimeToRFC3339(value) {
  if (!value) return "";
  return new Date(value).toISOString();
}

function slugify(value) {
  return String(value || "").toLowerCase().replace(/[^a-z0-9-]+/g, "-").replace(/^-+|-+$/g, "").slice(0, 63);
}

function trimSlash(value) {
  return String(value || "").replace(/\/+$/, "");
}

function browserRedirectURI() {
  return `${location.origin}/api/v1/ui/oauth/callback`;
}

function randomString(length) {
  const bytes = new Uint8Array(length);
  crypto.getRandomValues(bytes);
  return Array.from(bytes, (byte) => "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~"[byte % 66]).join("");
}

async function codeChallenge(verifier) {
  const encoded = new TextEncoder().encode(verifier);
  const digest = await crypto.subtle.digest("SHA-256", encoded);
  return btoa(String.fromCharCode(...new Uint8Array(digest))).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

createRoot(document.getElementById("root")).render(<App />);

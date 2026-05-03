import React, {useEffect, useMemo, useRef, useState} from "react";
import {createRoot} from "react-dom/client";
import {
  Boxes,
  CheckCircle2,
  Cloud,
  Code2,
  Database,
  GitBranch,
  Github,
  Globe2,
  KeyRound,
  Layers3,
  ListRestart,
  LoaderCircle,
  Lock,
  Network,
  Package,
  Play,
  Plus,
  RefreshCw,
  Rocket,
  Server,
  Shield,
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

const nav = [
  {id: "deploy", label: "Deploy", icon: Rocket},
  {id: "progress", label: "Progress", icon: LoaderCircle},
  {id: "projects", label: "Projects", icon: Boxes},
  {id: "apiKeys", label: "API Keys", icon: KeyRound},
  {id: "github", label: "GitHub", icon: Github},
  {id: "domains", label: "Domains", icon: Globe2},
  {id: "cloudflare", label: "Cloudflare", icon: Cloud},
  {id: "basaltpass", label: "BasaltPass", icon: Shield},
  {id: "namespaces", label: "Namespaces", icon: Layers3},
  {id: "pods", label: "Pods", icon: Layers3},
  {id: "nodes", label: "Nodes", icon: Server},
  {id: "ingresses", label: "Ingresses", icon: Network},
  {id: "services", label: "Services", icon: Database},
];

function App() {
  const [config, setConfig] = useState(null);
  const [token, setToken] = useState(localStorage.getItem(tokenKey) || "");
  const [view, setView] = useState("deploy");
  const [notice, setNotice] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [runtime, setRuntime] = useState(emptyRuntime);
  const [projects, setProjects] = useState([]);
  const [credentials, setCredentials] = useState({github: [], cloudflare: [], basaltpass: []});
  const [apiKeys, setAPIKeys] = useState([]);
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
  const projectLogController = useRef(null);
  const runtimeLogController = useRef(null);

  const api = useMemo(() => makeAPI(token, logout), [token]);
  const userProfile = useMemo(() => profileFromToken(token), [token]);

  useEffect(() => {
    boot();
  }, []);

  useEffect(() => {
    if (token) loadWorkspace();
  }, [token]);

  useEffect(() => {
    if (!token || view !== "progress") return;
    loadProjectProgress();
    const timer = setInterval(loadProjectProgress, 3000);
    return () => clearInterval(timer);
  }, [token, view, activeProgressProjectID, projects.length, projectLogFollow]);

  useEffect(() => {
    if (!token || runtimeDetail?.kind !== "node") return;
    const nodeName = runtimeDetail.row?.summary?.name || runtimeDetail.row?.name;
    if (!nodeName) return;
    const timer = setInterval(() => loadNodeDetail({name: nodeName}, false), 5000);
    return () => clearInterval(timer);
  }, [token, runtimeDetail?.kind, runtimeDetail?.row?.summary?.name, runtimeDetail?.row?.name]);

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
      setLoading(false);
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
    if (showModal) setRuntimeDetail({kind: "node", row: node, loading: true});
    try {
      const data = await api.get(`/runtime/nodes/${encodeURIComponent(node.name)}`);
      setRuntimeDetail({kind: "node", row: data.data || node, loading: false});
    } catch (err) {
      setRuntimeDetail({kind: "node", row: node, loading: false, error: err.message});
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
    const source = nextForm.build_source || "github";
    setError("");
    setNotice("");
    if (source === "github") {
      return analyzeRepo(nextForm.github_repo || selectedRepo, nextForm.github_branch);
    }
    const image = (nextForm.image_reference || "").trim();
    if (!image) {
      setError("Image reference is required for this install method.");
      return;
    }
    if (source === "ghcr" && !image.toLowerCase().startsWith("ghcr.io/")) {
      setError("GHCR image references must start with ghcr.io/.");
      return;
    }
    if (source === "source-upload" && !nextForm.source_archive_name) {
      setError("Choose a source archive before checking installability.");
      return;
    }
    const label = sourceLabel(source);
    setAnalysis({
      deployable: true,
      containerized: source !== "source-upload",
      scaffoldable: source === "source-upload",
      default_port: nextForm.port || 8080,
      ports: [Number(nextForm.port || 8080)],
      signals: source === "source-upload" ? [`Source archive: ${nextForm.source_archive_name}`, `Target image: ${image}`] : [`${label} image: ${image}`],
      warnings: source === "source-upload" ? ["Source upload is recorded with a target image. Make sure your build runner publishes that image before rollout."] : [],
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
    let selected = projectID
      ? projects.find((project) => String(project.id) === String(projectID))
      : projects[0];
    if (!selected) {
      if (!projectID) {
        setProjectProgress(null);
        return;
      }
      try {
        selected = await api.get(`/projects/${projectID}`);
      } catch (err) {
        setProjectProgress(null);
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
        {nav.map((item) => {
          const Icon = item.icon;
          return (
            <button key={item.id} className={view === item.id ? "nav active" : "nav"} onClick={() => setView(item.id)}>
              <Icon size={17} /> {item.label}
            </button>
          );
        })}
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
        {view === "github" && (
          <GitHubView credentials={credentials.github} onConnect={connectGitHubApp} onRepos={loadRepos} onDelete={(id) => deleteCredential("github", id)} reposByCredential={reposByCredential} repoFilters={repoFilters} setRepoFilters={setRepoFilters} />
        )}
        {view === "domains" && <DomainsView domains={domains} />}
        {view === "cloudflare" && <CloudflareView credentials={credentials.cloudflare} domains={domains} selectedID={selectedCloudflareID} setSelectedID={setSelectedCloudflareID} dnsRecords={dnsRecords} editingRecord={editingDNSRecord} setEditingRecord={setEditingDNSRecord} onCreate={createCredential} onDelete={(id) => deleteCredential("cloudflare", id)} onLoadDNS={loadDNSRecords} onSaveDNS={saveDNSRecord} onDeleteDNS={deleteDNSRecord} />}
        {view === "basaltpass" && <CredentialManager kind="basaltpass" rows={credentials.basaltpass} onCreate={createCredential} onDelete={deleteCredential} />}
        {["namespaces", "pods", "nodes", "ingresses", "services"].includes(view) && <RuntimeTable kind={view} rows={runtime[view] || []} onCreateNamespace={createNamespace} onPatchNamespace={patchNamespaceLabels} onDeleteNamespace={deleteNamespace} onDeletePod={deletePod} onNodeDetail={loadNodeDetail} onPodLogs={loadPodLogs} onSaveService={saveService} onDeleteService={deleteService} onDetail={setRuntimeDetail} />}
      </main>
      {editingProject && <ProjectModal project={editingProject} onClose={() => setEditingProject(null)} onSubmit={updateProject} />}
      {deletingProject && <DeleteProjectModal project={deletingProject} busy={loading} onClose={() => setDeletingProject(null)} onDelete={confirmDeleteProject} />}
      {runtimeDetail && <RuntimeDetailModal detail={runtimeDetail} logs={runtimeLogs} logFollow={runtimeLogFollow} logStatus={runtimeLogStatus} selectedLogContainer={runtimeLogContainer} logTail={runtimeLogTail} logLoaded={runtimeLogLoaded} onSelectLogContainer={setRuntimeLogContainer} onSetLogTail={setRuntimeLogTail} onLoadContainerLogs={loadRuntimeContainerLogs} onFollowPodLogs={startRuntimeLogFollow} onStopPodLogs={stopRuntimeLogFollow} onClose={() => { stopRuntimeLogFollow(); setRuntimeDetail(null); setRuntimeLogs(""); setRuntimeLogContainer(""); setRuntimeLogLoaded(false); setRuntimeLogStatus(""); }} onSaveService={saveService} onPatchNamespace={patchNamespaceLabels} />}
    </div>
  );
}

function DeployView({credentials, namespaces, selectedCredential, setSelectedCredential, repos, selectedRepo, analysis, setAnalysis, form, setForm, loadRepos, analyzeRepo, checkInstallSource, deployProject}) {
  const [stepIndex, setStepIndex] = useState(0);
  const selectedCloudflare = credentials.cloudflare.find((cred) => String(cred.id) === String(form.cloudflare_credential_id));
  const publicHost = form.subdomain && selectedCloudflare ? `${form.subdomain}.${selectedCloudflare.domain}` : "";
  const step = deploySteps[stepIndex];
  const canContinue = canContinueDeployStep(step.id, form, selectedCredential, analysis);
  const setSource = (buildSource) => {
    setAnalysis(null);
    setForm({...defaultDeployForm(), build_source: buildSource, github_branch: form.github_branch || "main", port: form.port || 8080});
  };
  const updateSourceForm = (nextForm) => {
    setAnalysis(null);
    setForm(nextForm);
  };
  const next = () => {
    if (step.id === "check") checkInstallSource(form);
    if (stepIndex < deploySteps.length - 1) setStepIndex(stepIndex + 1);
  };
  const back = () => setStepIndex(Math.max(0, stepIndex - 1));
  return (
    <div className="deploy-wizard">
      <section className="panel">
        <div className="wizard-steps">
          {deploySteps.map((item, index) => (
            <button key={item.id} type="button" className={index === stepIndex ? "wizard-step active" : index < stepIndex ? "wizard-step done" : "wizard-step"} onClick={() => setStepIndex(index)}>
              <b>{index + 1}</b>
              <span>{item.label}</span>
            </button>
          ))}
        </div>
      </section>
      <form className="panel deploy-form wizard-panel" onSubmit={deployProject}>
        <h2><Rocket size={18} /> {step.title}</h2>
        {step.id === "method" && (
          <div className="method-grid">
            {installMethods.map((method) => {
              const Icon = method.icon;
              return (
                <button key={method.id} type="button" className={form.build_source === method.id ? "method-card active" : "method-card"} onClick={() => setSource(method.id)}>
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
            {form.build_source === "github" && (
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
            {form.build_source === "dockerhub" && (
              <>
                <Field label="Docker Hub image" value={form.image_reference} onChange={(v) => updateSourceForm({...form, image_reference: v.trim(), name: form.name || slugify(imageName(v))})} required />
                <p className="muted">Example: nginx:1.27 or your-org/your-app:latest.</p>
              </>
            )}
            {form.build_source === "ghcr" && (
              <>
                <Field label="GHCR image" value={form.image_reference} onChange={(v) => updateSourceForm({...form, image_reference: v.trim(), name: form.name || slugify(imageName(v))})} required />
                <p className="muted">Example: ghcr.io/owner/repo:latest.</p>
              </>
            )}
            {form.build_source === "source-upload" && (
              <>
                <label>Source archive</label>
                <input type="file" accept=".zip,.tar,.tgz,.tar.gz" onChange={(event) => updateSourceForm({...form, source_archive_name: event.target.files?.[0]?.name || ""})} required />
                <Field label="Target image after build" value={form.image_reference} onChange={(v) => updateSourceForm({...form, image_reference: v.trim(), name: form.name || slugify(imageName(v))})} required />
                <Field label="Dockerfile path" value={form.dockerfile_path} onChange={(v) => updateSourceForm({...form, dockerfile_path: v})} />
              </>
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
            {form.build_source === "github" && (
              <label className="checkbox-row">
                <input type="checkbox" checked={form.auto_deploy !== false} onChange={(event) => setForm({...form, auto_deploy: event.target.checked})} />
                Auto build and deploy on GitHub push
              </label>
            )}
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
            {form.build_source === "github" && <span>Deploy mode <b>{form.auto_deploy !== false ? "Auto GitOps tracking" : "Manual deployments only"}</b></span>}
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
  {id: "method", label: "Method", title: "Choose install method"},
  {id: "source", label: "Source", title: "Choose or enter source"},
  {id: "check", label: "Check", title: "Check installability"},
  {id: "params", label: "Params", title: "Configure parameters"},
  {id: "namespace", label: "Namespace", title: "Choose namespace"},
  {id: "ingress", label: "Ingress", title: "Choose ingress mode"},
  {id: "domain", label: "Domain", title: "Choose domain"},
  {id: "confirm", label: "Confirm", title: "Confirm and build"},
];

const installMethods = [
  {id: "github", label: "GitHub", icon: Github, description: "Analyze a GitHub repository and deploy the matching GHCR image."},
  {id: "dockerhub", label: "Docker Hub", icon: Package, description: "Deploy an existing Docker Hub image."},
  {id: "ghcr", label: "GHCR", icon: Github, description: "Deploy an existing GitHub Container Registry image."},
  {id: "source-upload", label: "Upload source", icon: Upload, description: "Upload a source archive and record the target image to build."},
];

function canContinueDeployStep(stepID, form, selectedCredential, analysis) {
  if (stepID === "method") return Boolean(form.build_source);
  if (stepID === "source") {
    if (form.build_source === "github") return Boolean(selectedCredential && form.github_repo);
    if (form.build_source === "source-upload") return Boolean(form.source_archive_name && form.image_reference);
    return Boolean(form.image_reference);
  }
  if (stepID === "check") return Boolean(analysis?.deployable);
  if (stepID === "params") return Boolean(form.name && Number(form.port || 0) > 0 && Number(form.replicas || 0) > 0);
  if (stepID === "domain") {
    if (form.exposure_mode === "public") return Boolean(form.cloudflare_credential_id && form.subdomain);
    if (form.exposure_mode === "private") return Boolean(form.private_host);
  }
  return true;
}

function sourceLabel(source) {
  return ({github: "GitHub", dockerhub: "Docker Hub", ghcr: "GHCR", "source-upload": "Source upload"}[source || "github"] || source);
}

function sourceSummary(form) {
  if (form.build_source === "github") return `${form.github_repo || "-"} @ ${form.github_branch || "main"}`;
  if (form.build_source === "source-upload") return `${form.source_archive_name || "-"} -> ${form.image_reference || "-"}`;
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

function RuntimeTable({kind, rows, onCreateNamespace, onPatchNamespace, onDeleteNamespace, onDeletePod, onNodeDetail, onPodLogs, onSaveService, onDeleteService, onDetail}) {
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
      <section className="panel">
        <div className="table runtime-table">
          <div className="tr head">{keys.map((key) => <span key={key}>{key.replaceAll("_", " ")}</span>)}<span>Actions</span></div>
          {rows.map((row, index) => (
            <div className="tr" key={`${kind}-${row.namespace || ""}-${row.name || index}`}>
              {keys.map((key) => <span key={key}>{formatCell(row[key])}</span>)}
              <span className="row-actions">
                <button onClick={() => kind === "nodes" ? onNodeDetail(row) : onDetail({kind, row})}>Details</button>
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

function RuntimeDetailModal({detail, logs, logFollow, logStatus, selectedLogContainer, logTail, logLoaded, onSelectLogContainer, onSetLogTail, onLoadContainerLogs, onFollowPodLogs, onStopPodLogs, onClose, onSaveService, onPatchNamespace}) {
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
          <NodeDetailView detail={detail} />
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

function NodeDetailView({detail}) {
  const row = detail.row || {};
  const summary = row.summary || row;
  const usage = row.usage || {};
  const pods = row.pods || [];
  const conditions = row.conditions || [];
  return (
    <div className="node-detail">
      {detail.loading && <p className="muted">Loading live node status...</p>}
      {detail.error && <p className="error-inline">{detail.error}</p>}
      <div className="node-status-grid">
        <div className="runtime-summary">
          <strong>{summary.status || "-"}</strong>
          <span>{summary.name} · {summary.version || "-"}</span>
        </div>
        <div className="detail-list compact-details">
          <span>Internal IP <b>{summary.internal_ip || row.addresses?.InternalIP || "-"}</b></span>
          <span>Roles <b>{(summary.roles || []).join(", ") || "-"}</b></span>
          <span>Pods <b>{pods.length}/{row.allocatable?.pods || "-"}</b></span>
          <span>Checked <b>{formatTime(row.checked_at)}</b></span>
        </div>
      </div>
      <section className="node-section">
        <h3>Live resources</h3>
        {row.metrics_available ? (
          <div className="resource-grid">
            <ResourceMeter label="CPU allocatable" value={usage.cpu_allocatable_percent} detail={`${usage.cpu_millis || 0}m / ${row.allocatable?.cpu_millis || 0}m`} />
            <ResourceMeter label="Memory allocatable" value={usage.memory_allocatable_percent} detail={`${formatBytes(usage.memory_bytes)} / ${formatBytes(row.allocatable?.memory_bytes)}`} />
            <ResourceMeter label="CPU capacity" value={usage.cpu_capacity_percent} detail={`${usage.cpu || "-"} / ${row.capacity?.cpu || "-"}`} />
            <ResourceMeter label="Memory capacity" value={usage.memory_capacity_percent} detail={`${usage.memory || "-"} / ${row.capacity?.memory || "-"}`} />
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
        <h3>Taints and labels</h3>
        <div className="signal-list">
          {(row.taints || []).map((taint) => <span key={taint}>{taint}</span>)}
          {(row.taints || []).length === 0 && <span>No taints</span>}
        </div>
        <div className="label-cloud">
          {Object.entries(row.labels || {}).map(([key, value]) => <span key={key}>{key}={value}</span>)}
        </div>
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
      <input name="ports" placeholder="ports: http:80:8080/TCP,https:443:8443/TCP" defaultValue={portsToForm(existing?.ports)} required />
      <input name="labels" placeholder="labels: app=my-app" defaultValue={formatKeyValues(existing?.labels)} />
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
    build_source: "github",
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
  const source = form.build_source || "github";
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
    auto_deploy: source === "github" ? form.auto_deploy !== false : false,
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
    if (res.status === 401) onUnauthorized();
    const data = await res.json().catch(() => ({}));
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
    if (res.status === 401) onUnauthorized();
    if (!res.ok) {
      const data = await res.json().catch(() => ({}));
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
  return ({deploy: "Deploy project", progress: "Progress", projects: "Projects", apiKeys: "API Keys", github: "GitHub", domains: "Domains", cloudflare: "Cloudflare", basaltpass: "BasaltPass", namespaces: "Namespaces", pods: "Pods", nodes: "Nodes", ingresses: "Ingresses", services: "Services"}[view] || "BeanCS");
}

function subtitleFor(view, runtime, projects) {
  if (view === "projects") return `${projects.length} managed projects`;
  if (view === "progress") return "Watch installs and runtime readiness";
  if (view === "apiKeys") return "Issue and revoke API keys for automation";
  if (runtime[view]) return `${runtime[view].length} cluster resources`;
  return "Select a repository, verify containerization, and publish traffic.";
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

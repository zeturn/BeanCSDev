import React, {useEffect, useMemo, useState} from "react";
import {createRoot} from "react-dom/client";
import {
  Boxes,
  Cloud,
  Code2,
  Database,
  GitBranch,
  Github,
  Globe2,
  KeyRound,
  Layers3,
  ListRestart,
  Lock,
  Network,
  Play,
  Plus,
  RefreshCw,
  Rocket,
  Server,
  Shield,
  Trash2,
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
  {id: "projects", label: "Projects", icon: Boxes},
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
  const [domains, setDomains] = useState([]);
  const [repos, setRepos] = useState([]);
  const [selectedCredential, setSelectedCredential] = useState("");
  const [selectedRepo, setSelectedRepo] = useState("");
  const [analysis, setAnalysis] = useState(null);
  const [deployForm, setDeployForm] = useState(defaultDeployForm());
  const [editingProject, setEditingProject] = useState(null);

  const api = useMemo(() => makeAPI(token, logout), [token]);

  useEffect(() => {
    boot();
  }, []);

  useEffect(() => {
    if (token) loadWorkspace();
  }, [token]);

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
      const [runtimeData, projectData, githubData, cloudflareData, domainsData, basaltpassData] = await Promise.all([
        api.get("/runtime/overview"),
        api.get("/projects"),
        api.get("/credentials/github/"),
        api.get("/credentials/cloudflare/"),
        api.get("/credentials/cloudflare/domains"),
        api.get("/credentials/basaltpass/"),
      ]);
      setRuntime(runtimeData.data || emptyRuntime);
      setProjects(projectData.data || []);
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

  async function loadRepos(credentialID = selectedCredential) {
    if (!credentialID) return;
    setSelectedCredential(String(credentialID));
    setAnalysis(null);
    setRepos([]);
    setLoading(true);
    try {
      const data = await api.get(`/credentials/github/${credentialID}/repositories`);
      setRepos(data.data || []);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
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

  async function deployProject(event) {
    event.preventDefault();
    if (!analysis?.deployable) return;
    const payload = buildProjectPayload(deployForm, selectedCredential, credentials);
    setLoading(true);
    setError("");
    try {
      await api.post("/projects", payload);
      setNotice("Project created. BeanCS is preparing GitOps manifests and traffic routes.");
      setDeployForm(defaultDeployForm());
      setAnalysis(null);
      setSelectedRepo("");
      await loadWorkspace();
      setView("projects");
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }

  async function updateProject(event) {
    event.preventDefault();
    const body = Object.fromEntries(new FormData(event.currentTarget).entries());
    body.replicas = Number(body.replicas || 1);
    await api.patch(`/projects/${editingProject.id}`, body);
    setEditingProject(null);
    await loadWorkspace();
  }

  async function deleteProject(project) {
    if (!confirm(`Delete ${project.name}?`)) return;
    await api.delete(`/projects/${project.id}`);
    await loadWorkspace();
  }

  async function scaleProject(project, replicas) {
    await api.post(`/projects/${project.id}/scale`, {replicas});
    await loadWorkspace();
  }

  async function restartProject(project) {
    await api.post(`/projects/${project.id}/restart`, {});
    setNotice(`${project.name} restarted.`);
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
      </aside>
      <main className="workspace">
        <header className="topbar">
          <div>
            <h1>{titleFor(view)}</h1>
            <p>{subtitleFor(view, runtime, projects)}</p>
          </div>
          <div className="top-actions">
            <button onClick={loadWorkspace} disabled={loading}><RefreshCw size={16} /> Refresh</button>
            <button onClick={logout}>Sign out</button>
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
            form={deployForm}
            setForm={setDeployForm}
            loadRepos={loadRepos}
            analyzeRepo={analyzeRepo}
            deployProject={deployProject}
          />
        )}
        {view === "projects" && (
          <ProjectsView projects={projects} onEdit={setEditingProject} onDelete={deleteProject} onScale={scaleProject} onRestart={restartProject} />
        )}
        {view === "github" && (
          <GitHubView credentials={credentials.github} onConnect={connectGitHubApp} onRepos={loadRepos} onDelete={(id) => deleteCredential("github", id)} repos={repos} />
        )}
        {view === "domains" && <DomainsView domains={domains} />}
        {view === "cloudflare" && <CredentialManager kind="cloudflare" rows={credentials.cloudflare} onCreate={createCredential} onDelete={deleteCredential} />}
        {view === "basaltpass" && <CredentialManager kind="basaltpass" rows={credentials.basaltpass} onCreate={createCredential} onDelete={deleteCredential} />}
        {["namespaces", "pods", "nodes", "ingresses", "services"].includes(view) && <RuntimeTable kind={view} rows={runtime[view] || []} />}
      </main>
      {editingProject && <ProjectModal project={editingProject} onClose={() => setEditingProject(null)} onSubmit={updateProject} />}
    </div>
  );
}

function DeployView({credentials, namespaces, selectedCredential, setSelectedCredential, repos, selectedRepo, analysis, form, setForm, loadRepos, analyzeRepo, deployProject}) {
  const selectedCloudflare = credentials.cloudflare.find((cred) => String(cred.id) === String(form.cloudflare_credential_id));
  const publicHost = form.subdomain && selectedCloudflare ? `${form.subdomain}.${selectedCloudflare.domain}` : "";
  return (
    <div className="deploy-grid">
      <section className="panel">
        <h2><Github size={18} /> Repository</h2>
        <label>GitHub credential</label>
        <select value={selectedCredential} onChange={(event) => { setSelectedCredential(event.target.value); loadRepos(event.target.value); }}>
          <option value="">Choose credential</option>
          {credentials.github.map((cred) => <option key={cred.id} value={cred.id}>{cred.name} ({cred.account_login || cred.auth_type})</option>)}
        </select>
        <div className="repo-list">
          {repos.map((repo) => (
            <button key={repo.full_name} className={selectedRepo === repo.full_name ? "repo active" : "repo"} onClick={() => analyzeRepo(repo.full_name, repo.default_branch)}>
              <Code2 size={15} />
              <span>{repo.full_name}</span>
              <small>{repo.private ? "Private" : "Public"} · {repo.default_branch}</small>
            </button>
          ))}
        </div>
      </section>
      <section className="panel">
        <h2><Shield size={18} /> Readiness</h2>
        {!analysis && <p className="muted">Select a repository to check whether BeanCS can deploy it.</p>}
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
      </section>
      <form className="panel deploy-form" onSubmit={deployProject}>
        <h2><Rocket size={18} /> Deployment</h2>
        <div className="form-grid">
          <Field label="Project name" value={form.name} onChange={(v) => setForm({...form, name: slugify(v)})} required />
          <label>Namespace</label>
          <input
            list="namespace-options"
            value={form.namespace}
            placeholder={form.name ? `proj-${form.name}` : "proj-my-app"}
            onChange={(event) => setForm({...form, namespace: slugify(event.target.value)})}
          />
          <datalist id="namespace-options">
            {namespaces.map((ns) => <option key={ns.name} value={ns.name} />)}
          </datalist>
          <Field label="Branch" value={form.github_branch} onChange={(v) => setForm({...form, github_branch: v})} />
          <Field label="Dockerfile path" value={form.dockerfile_path} onChange={(v) => setForm({...form, dockerfile_path: v})} />
          <Field label="Port" type="number" value={form.port} onChange={(v) => setForm({...form, port: Number(v)})} />
          <label>BasaltPass optional</label>
          <select value={form.basaltpass_instance_id} onChange={(event) => setForm({...form, basaltpass_instance_id: event.target.value})}>
            <option value="">Do not register OAuth app</option>
            {credentials.basaltpass.map((cred) => <option key={cred.id} value={cred.id}>{cred.name}</option>)}
          </select>
          <label>Traffic</label>
          <select value={form.exposure_mode} onChange={(event) => setForm({...form, exposure_mode: event.target.value})}>
            <option value="public">Traefik public ingress</option>
            <option value="private">Tailscale private ingress</option>
            <option value="internal-only">Cluster internal only</option>
          </select>
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
        </div>
        <button className="primary" disabled={!analysis?.deployable} type="submit"><Play size={16} /> Deploy</button>
      </form>
    </div>
  );
}

function ProjectsView({projects, onEdit, onDelete, onScale, onRestart}) {
  return (
    <section className="panel">
      <div className="table">
        <div className="tr head"><span>Name</span><span>Repo</span><span>Route</span><span>Status</span><span>Scale</span><span>Actions</span></div>
        {projects.map((project) => (
          <div className="tr" key={project.id}>
            <span className="strong">{project.display_name || project.name}</span>
            <span>{project.github_repo}</span>
            <span>{project.domain || project.exposure_mode}</span>
            <span>{project.status}</span>
            <span>
              <button onClick={() => onScale(project, Math.max(0, Number(project.replicas || 1) - 1))}>-</button>
              <b>{project.replicas}</b>
              <button onClick={() => onScale(project, Number(project.replicas || 1) + 1)}>+</button>
            </span>
            <span className="row-actions">
              <button onClick={() => onRestart(project)} title="Restart"><ListRestart size={15} /></button>
              <button onClick={() => onEdit(project)} title="Edit"><Plus size={15} /></button>
              <button onClick={() => onDelete(project)} title="Delete"><Trash2 size={15} /></button>
            </span>
          </div>
        ))}
        {projects.length === 0 && <div className="empty">No projects yet.</div>}
      </div>
    </section>
  );
}

function GitHubView({credentials, onConnect, onRepos, onDelete, repos}) {
  return (
    <div className="stack">
      <section className="panel action-panel">
        <div>
          <h2><Github size={18} /> GitHub App</h2>
          <p>Authorize repositories directly. BeanCS will name the credential from the GitHub account.</p>
        </div>
        <form onSubmit={onConnect}><button className="primary"><Github size={16} /> Connect GitHub App</button></form>
      </section>
      <section className="panel">
        <div className="table compact">
          <div className="tr head"><span>Name</span><span>Account</span><span>Type</span><span>GitOps repo</span><span>Actions</span></div>
          {credentials.map((cred) => (
            <div className="tr" key={cred.id}>
              <span className="account-cell">{cred.avatar_url ? <img src={cred.avatar_url} alt="" /> : <Github size={18} />}{cred.name}</span>
              <span>{cred.account_login || cred.org || "-"}</span><span>{cred.auth_type || "pat"}</span><span>{cred.gitops_repo || "-"}</span>
              <span className="row-actions">
                <button onClick={() => onRepos(cred.id)}><GitBranch size={15} /> Repos</button>
                <button onClick={() => onDelete(cred.id)}><Trash2 size={15} /></button>
              </span>
            </div>
          ))}
        </div>
      </section>
      {repos.length > 0 && <section className="panel repo-cloud">{repos.slice(0, 80).map((repo) => <span key={repo.full_name}>{repo.full_name}</span>)}</section>}
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

function RuntimeTable({kind, rows}) {
  const keys = rows[0] ? Object.keys(rows[0]).slice(0, 7) : [];
  return (
    <section className="panel">
      <div className="table compact">
        <div className="tr head">{keys.map((key) => <span key={key}>{key.replaceAll("_", " ")}</span>)}</div>
        {rows.map((row, index) => <div className="tr" key={`${kind}-${index}`}>{keys.map((key) => <span key={key}>{formatCell(row[key])}</span>)}</div>)}
        {rows.length === 0 && <div className="empty">No {kind} found.</div>}
      </div>
    </section>
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
    name: "",
    namespace: "",
    github_branch: "main",
    dockerfile_path: "Dockerfile",
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
  return {
    name: form.name,
    namespace: form.namespace || undefined,
    github_credential_id: Number(githubCredentialID),
    github_repo: form.github_repo,
    github_branch: form.github_branch || "main",
    dockerfile_path: form.dockerfile_path || "Dockerfile",
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
  return {
    get: (path) => request(path),
    post: (path, body) => request(path, {method: "POST", body: JSON.stringify(body)}),
    patch: (path, body) => request(path, {method: "PATCH", body: JSON.stringify(body)}),
    delete: (path) => request(path, {method: "DELETE"}),
  };
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
  return ({deploy: "Deploy project", projects: "Projects", github: "GitHub", domains: "Domains", cloudflare: "Cloudflare", basaltpass: "BasaltPass", namespaces: "Namespaces", pods: "Pods", nodes: "Nodes", ingresses: "Ingresses", services: "Services"}[view] || "BeanCS");
}

function subtitleFor(view, runtime, projects) {
  if (view === "projects") return `${projects.length} managed projects`;
  if (runtime[view]) return `${runtime[view].length} cluster resources`;
  return "Select a repository, verify containerization, and publish traffic.";
}

function formatCell(value) {
  if (Array.isArray(value)) return value.join(", ") || "-";
  if (typeof value === "boolean") return value ? "Yes" : "No";
  if (value === null || value === undefined || value === "") return "-";
  return String(value);
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

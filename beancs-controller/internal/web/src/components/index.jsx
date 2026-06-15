import React, { useEffect, useMemo, useRef, useState } from "react";
import * as LucideIcons from "lucide-react";
export * from "./ui";
import { Button, Input, Select, Textarea, Checkbox, Modal, Drawer } from "./ui";
import {
  filterLogLines,
  podContainers,
  definitionForDependency,
  firstEnvPreset,
  labelize,
  parseDotEnv,
  formatTime,
  normalizeProcessStatus,
  truncateMiddle,
  formatPercent,
  formatCell,
  formatKeyValues,
  portsToForm,
} from "../utils/index";
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
export function SidebarNavGroup({ label, items, view, onSelect }) {
  if (!items?.length) return null;
  return (
    <div className="nav-group">
      {label && <div className="nav-group-label">{label}</div>}
      {items.map((item) => {
        const Icon = item.icon;
        return (
          <Button
            key={item.id}
            type="button"
            className={view === item.id ? "nav active" : "nav"}
            onClick={() => onSelect(item)}
          >
            <Icon size={18} />
            <span>{item.label}</span>
            {item.badge && <em>{item.badge}</em>}
          </Button>
        );
      })}
    </div>
  );
}

export function PageHeading({ title, topLabel, subtitle, actions }) {
  return (
    <section className="page-heading">
      <div>
        {topLabel && <span className="page-heading-top-label">{topLabel}</span>}
        <h1>{title}</h1>
        {subtitle && <p>{subtitle}</p>}
      </div>
      {actions && <div className="top-actions">{actions}</div>}
    </section>
  );
}

export function SkeletonPage() {
  return (
    <div className="skeleton-page">
      <div className="skeleton-header">
        <div className="skeleton-line w-40" />
        <div className="skeleton-line w-60" />
      </div>
      <div className="skeleton-grid">
        {Array.from({ length: 6 }).map((_, index) => (
          <div className="skeleton-card" key={`skeleton-card-${index}`}>
            <div className="skeleton-line w-70" />
            <div className="skeleton-line w-50" />
            <div className="skeleton-line w-80" />
          </div>
        ))}
      </div>
      <div className="skeleton-table">
        <div className="skeleton-row">
          {Array.from({ length: 6 }).map((_, index) => (
            <div
              className="skeleton-line w-60"
              key={`skeleton-head-${index}`}
            />
          ))}
        </div>
        {Array.from({ length: 4 }).map((_, index) => (
          <div className="skeleton-row" key={`skeleton-row-${index}`}>
            {Array.from({ length: 6 }).map((_, cellIndex) => (
              <div
                className="skeleton-line w-80"
                key={`skeleton-cell-${index}-${cellIndex}`}
              />
            ))}
          </div>
        ))}
      </div>
    </div>
  );
}

export function RepoListSkeleton() {
  return (
    <>
      {Array.from({ length: 4 }).map((_, index) => (
        <div
          className="import-repo-row repo-skeleton-row"
          key={`repo-skeleton-${index}`}
        >
          <div>
            <div className="skeleton-dot" />
            <span className="skeleton-line w-40" />
            <small className="skeleton-line w-20" />
          </div>
          <div className="skeleton-button" />
        </div>
      ))}
    </>
  );
}

export function ApplicationSpecPlanSummary({ analysis }) {
  const plan = analysis?.plan || {};
  const projects = plan.willCreateProjects || [];
  const dependencies = plan.willCreateDependencies || [];
  const injections = plan.willInjectEnv || [];
  return (
    <div className="spec-plan-summary">
      <span>
        <FileText size={15} /> Application{" "}
        <b>{plan.application?.name || "-"}</b>
      </span>
      <span>
        <Layers3 size={15} /> Projects <b>{projects.length}</b>
      </span>
      <span>
        <Database size={15} /> Dependencies <b>{dependencies.length}</b>
      </span>
      <span>
        <KeyRound size={15} /> Env injections <b>{injections.length}</b>
      </span>
    </div>
  );
}

export function DependencyConfigEditor({ definition, value, onChange }) {
  const properties = definition?.config_schema?.properties || {};
  const keys = Object.keys(properties);
  if (!definition || keys.length === 0) return null;
  const update = (path, nextValue) => {
    const next = { ...value };
    if (path.length === 1) {
      next[path[0]] = nextValue;
    } else {
      const [head, tail] = path;
      next[head] = { ...(next[head] || {}), [tail]: nextValue };
    }
    onChange(next);
  };
  return (
    <div className="dependency-config-grid">
      {keys.map((key) => {
        const schema = properties[key] || {};
        if (schema.type === "object") {
          const nested = schema.properties || {};
          return (
            <div className="dependency-config-group" key={key}>
              <b>{labelize(key)}</b>
              <div className="component-grid">
                {Object.keys(nested).map((nestedKey) => (
                  <DependencyConfigField
                    key={`${key}.${nestedKey}`}
                    label={labelize(nestedKey)}
                    schema={nested[nestedKey] || {}}
                    value={(value[key] || {})[nestedKey]}
                    onChange={(nextValue) =>
                      update([key, nestedKey], nextValue)
                    }
                  />
                ))}
              </div>
            </div>
          );
        }
        return (
          <DependencyConfigField
            key={key}
            label={labelize(key)}
            schema={schema}
            value={value[key]}
            onChange={(nextValue) => update([key], nextValue)}
          />
        );
      })}
    </div>
  );
}

export function DependencyConfigField({ label, schema, value, onChange }) {
  if (schema.secret && schema.generate) {
    return (
      <label className="checkbox-label dependency-secret-toggle">
        <Input
          type="checkbox"
          checked={value !== ""}
          onChange={(event) =>
            onChange(event.target.checked ? String(value || "") : "")
          }
        />
        <span>{label}: auto generated</span>
      </label>
    );
  }
  if (schema.type === "boolean") {
    return (
      <label className="checkbox-label dependency-secret-toggle">
        <Input
          type="checkbox"
          checked={Boolean(value)}
          onChange={(event) => onChange(event.target.checked)}
        />
        <span>{label}</span>
      </label>
    );
  }
  if (Array.isArray(schema.enum)) {
    return (
      <>
        <label>{label}</label>
        <Select
          value={value ?? schema.default ?? ""}
          onChange={(event) => onChange(event.target.value)}
        >
          {schema.enum.map((item) => (
            <option key={item} value={item}>
              {item}
            </option>
          ))}
        </Select>
      </>
    );
  }
  return (
    <Field
      label={label}
      value={value ?? schema.default ?? ""}
      onChange={(nextValue) => onChange(nextValue)}
    />
  );
}

export function DependencyLinksEditor({
  component,
  dependencies,
  definitions,
  onChange,
}) {
  const links = component.dependency_links || [];
  const toggle = (dependency, checked) => {
    if (!checked) {
      onChange(links.filter((link) => link.dependency !== dependency.name));
      return;
    }
    const definition = definitionForDependency(definitions, dependency.type);
    const credential = (dependency.credentials || [])[0];
    onChange([
      ...links,
      {
        dependency: dependency.name,
        preset: firstEnvPreset(definition),
        credential: credential?.name || "",
        credential_id: credential?.id || "",
      },
    ]);
  };
  const updatePreset = (dependencyName, preset) => {
    onChange(
      links.map((link) =>
        link.dependency === dependencyName ? { ...link, preset } : link,
      ),
    );
  };
  const updateCredential = (dependencyName, credentialID) => {
    onChange(
      links.map((link) => {
        if (link.dependency !== dependencyName) return link;
        const dependency = dependencies.find(
          (item) => item.name === dependencyName,
        );
        const credential = (dependency?.credentials || []).find(
          (item) => String(item.id) === String(credentialID),
        );
        return {
          ...link,
          credential_id: credential?.id || "",
          credential: credential?.name || "",
        };
      }),
    );
  };
  return (
    <div className="component-card">
      <div className="component-card-head">
        <b>{component.project_name || component.name}</b>
        <span>{links.length} linked</span>
      </div>
      <div className="dependency-link-grid">
        {dependencies.map((dependency) => {
          const link = links.find(
            (item) => item.dependency === dependency.name,
          );
          const definition = definitionForDependency(
            definitions,
            dependency.type,
          );
          const presets = Object.keys(definition?.env_presets || {});
          return (
            <div
              className="dependency-link-row"
              key={`${component.project_name}-${dependency.name}`}
            >
              <label className="checkbox-label">
                <Input
                  type="checkbox"
                  checked={Boolean(link)}
                  onChange={(event) => toggle(dependency, event.target.checked)}
                />
                <span>{dependency.name}</span>
              </label>
              <Select
                value={link?.preset || firstEnvPreset(definition)}
                onChange={(event) =>
                  updatePreset(dependency.name, event.target.value)
                }
                disabled={!link}
              >
                {presets.map((preset) => (
                  <option key={preset} value={preset}>
                    {preset}
                  </option>
                ))}
              </Select>
              <Select
                value={link?.credential_id || ""}
                onChange={(event) =>
                  updateCredential(dependency.name, event.target.value)
                }
                disabled={!link || !(dependency.credentials || []).length}
              >
                <option value="">default</option>
                {(dependency.credentials || []).map((credential) => (
                  <option key={credential.id} value={credential.id}>
                    {credential.name}
                  </option>
                ))}
              </Select>
            </div>
          );
        })}
      </div>
    </div>
  );
}

export function ProgressEvidence({
  activeTab,
  detailQuery,
  progress,
  installProgress,
  selectedProcess,
  jobs,
  deployments,
  events,
  logs,
  logFollow,
  logStatus,
  onRefresh,
  onStartLogFollow,
  onStopLogFollow,
}) {
  const installLogs = (installProgress?.logs || []).join("\n");
  const deploymentText = deployments.length
    ? deployments
        .slice(0, 12)
        .map((deployment) =>
          [
            `#${deployment.id || "-"} ${deployment.status || "pending"}`,
            `image=${deployment.image_ref || deployment.tag || "-"}`,
            `commit=${deployment.commit_sha || "-"}`,
            deployment.workflow_url
              ? `workflow=${deployment.workflow_url}`
              : "",
            deployment.failure_reason
              ? `error=${deployment.failure_reason}`
              : "",
          ]
            .filter(Boolean)
            .join("\n"),
        )
        .join("\n\n")
    : "No deployment records yet.";
  const eventText = events.length
    ? events
        .slice(0, 20)
        .map((event) =>
          [
            `${event.type || "Event"} ${event.reason || ""}`.trim(),
            `object=${event.object || "-"}`,
            `count=${event.count || 1}`,
            `last_seen=${formatTime(event.last_seen)}`,
            event.message || "",
          ]
            .filter(Boolean)
            .join("\n"),
        )
        .join("\n\n")
    : "No Kubernetes events reported for this project.";
  const runText = [
    `process=${selectedProcess?.id ? `#${selectedProcess.id}` : "-"}`,
    `title=${selectedProcess?.title || progress?.project?.display_name || progress?.project?.name || "-"}`,
    `status=${selectedProcess?.status || progress?.deployment?.status || "-"}`,
    `project=${selectedProcess?.project?.name || progress?.project?.name || "-"}`,
    `namespace=${selectedProcess?.project?.namespace || progress?.project?.namespace || "-"}`,
    `jobs=${jobs?.length || 0}`,
    `deployments=${deployments.length}`,
    `events=${events.length}`,
    selectedProcess?.created_at
      ? `created=${formatTime(selectedProcess.created_at)}`
      : "",
    selectedProcess?.updated_at
      ? `updated=${formatTime(selectedProcess.updated_at)}`
      : progress?.checked_at
        ? `checked=${formatTime(progress.checked_at)}`
        : "",
  ]
    .filter(Boolean)
    .join("\n");
  const filteredRunText = filterLogLines(runText, detailQuery);
  const filteredInstallLogs = filterLogLines(
    installLogs || progress?.error || "",
    detailQuery,
  );
  const filteredDeploymentText = filterLogLines(deploymentText, detailQuery);
  const filteredEventText = filterLogLines(eventText, detailQuery);
  return (
    <div className="process-detail-panel">
      {activeTab === "run" && (
        <section className="process-evidence-card">
          <h3>Run details</h3>
          <pre>{filteredRunText || "No run details matched the search."}</pre>
        </section>
      )}
      {activeTab === "install" && (
        <section className="process-evidence-card">
          <h3>Install log</h3>
          <pre>
            {filteredInstallLogs || "No active install log for this project."}
          </pre>
        </section>
      )}
      {activeTab === "deployments" && (
        <section className="process-evidence-card">
          <h3>Deployment records</h3>
          <pre>
            {filteredDeploymentText ||
              "No deployment records matched the search."}
          </pre>
        </section>
      )}
      {activeTab === "events" && (
        <section className="process-evidence-card">
          <h3>Kubernetes events</h3>
          <pre>
            {filteredEventText || "No Kubernetes events matched the search."}
          </pre>
        </section>
      )}
      {activeTab === "runtime" && (
        <section className="process-evidence-card">
          <div className="process-evidence-head">
            <h3>Runtime logs</h3>
            <div className="row-actions process-log-actions">
              <Button type="button" onClick={onRefresh} disabled={logFollow}>
                <RefreshCw size={15} /> Snapshot
              </Button>
              {logFollow ? (
                <Button type="button" onClick={onStopLogFollow}>
                  Stop follow
                </Button>
              ) : (
                <Button
                  type="button"
                  className="primary"
                  onClick={onStartLogFollow}
                  disabled={!progress?.project?.id}
                >
                  Follow live
                </Button>
              )}
            </div>
          </div>
          {logStatus && <p className="log-status">{logStatus}</p>}
          <pre>
            {logs ||
              "No container logs yet. If the workload has not created pods, Kubernetes events above are the source of truth."}
          </pre>
        </section>
      )}
    </div>
  );
}

export function ProgressStatusIcon({ status }) {
  const normalizedStatus = normalizeProcessStatus(status);
  const normalized =
    normalizedStatus === "done" ||
    normalizedStatus === "deployed" ||
    normalizedStatus === "provisioned"
      ? "done"
      : normalizedStatus === "failed"
        ? "failed"
        : normalizedStatus === "running"
          ? "running"
          : "pending";
  const Icon =
    normalized === "done"
      ? CheckCircle2
      : normalized === "failed"
        ? AlertTriangle
        : normalized === "running"
          ? LoaderCircle
          : Play;
  return <Icon className={`process-status ${normalized}`} size={16} />;
}

export function EventTimeline({ events }) {
  return (
    <div className="timeline">
      {events.map((event, index) => (
        <div
          className="timeline-item"
          key={`${event.object}-${event.reason}-${index}`}
        >
          <span
            className={event.type === "Warning" ? "dot failed" : "dot done"}
          />
          <div>
            <b>{event.reason || event.type}</b>
            <small>
              {event.object} · {event.count || 1}x ·{" "}
              {formatTime(event.last_seen)}
            </small>
            <p>{event.message}</p>
          </div>
        </div>
      ))}
      {events.length === 0 && (
        <div className="empty">
          No warning events in the latest cluster feed.
        </div>
      )}
    </div>
  );
}

export function MetricCard({
  icon: Icon,
  label,
  value,
  detail,
  tone = "neutral",
}) {
  return (
    <div className={`metric-card ${tone}`}>
      <div>
        <Icon size={18} />
        <span>{label}</span>
      </div>
      <strong>{value}</strong>
      <small>{detail}</small>
    </div>
  );
}

export function IndustrialMeter({ label, value, detail }) {
  const normalized = Math.max(0, Math.min(100, Number(value || 0)));
  return (
    <div className="industrial-meter">
      <div>
        <b>{label}</b>
        <span>{formatPercent(normalized)}%</span>
      </div>
      <progress value={normalized} max="100" />
      <small>{detail}</small>
    </div>
  );
}

export function AlertList({ rows, empty }) {
  return (
    <div className="alert-list">
      {rows.map((row, index) => (
        <div
          className={`alert-row ${row.severity || "warning"}`}
          key={`${row.object}-${row.reason}-${index}`}
        >
          <b>{row.reason || "Warning"}</b>
          <span>
            {row.object}
            {row.namespace ? ` · ${row.namespace}` : ""}
          </span>
          <p>{row.message}</p>
          <small>{formatTime(row.last_seen)}</small>
        </div>
      ))}
      {rows.length === 0 && <div className="empty">{empty}</div>}
    </div>
  );
}

export function ProgressStep({ step }) {
  const Icon =
    step.state === "done"
      ? CheckCircle2
      : step.state === "running"
        ? LoaderCircle
        : step.state === "failed"
          ? Trash2
          : Play;
  return (
    <div className={`step ${step.state}`}>
      <Icon size={16} />
      <span>{step.label}</span>
      <b>{step.state}</b>
    </div>
  );
}

export function GitOpsRepoEditor({ cred, onUpdate }) {
  const [editing, setEditing] = useState(false);
  const [value, setValue] = useState(cred.gitops_repo || "");
  const save = () => {
    onUpdate(cred.id, { gitops_repo: value.trim() || null });
    setEditing(false);
  };
  useEffect(() => setValue(cred.gitops_repo || ""), [cred.gitops_repo]);
  return (
    <div className="gitops-repo-editor">
      <span
        style={{
          fontSize: "0.8rem",
          opacity: 0.7,
          display: "flex",
          alignItems: "center",
          gap: "0.35rem",
        }}
      >
        <GitBranch size={14} /> GitOps Repo
      </span>
      {editing ? (
        <div style={{ display: "flex", gap: "0.4rem", alignItems: "center" }}>
          <Input
            value={value}
            onChange={(e) => setValue(e.target.value)}
            placeholder="owner/gitops-manifests"
            style={{ flex: 1, minWidth: "200px" }}
          />
          <Button
            variant="primary"
            onClick={save}
            style={{ padding: "0.3rem 0.7rem", fontSize: "0.8rem" }}
          >
            Save
          </Button>
          <Button
            onClick={() => {
              setValue(cred.gitops_repo || "");
              setEditing(false);
            }}
            style={{ padding: "0.3rem 0.7rem", fontSize: "0.8rem" }}
          >
            Cancel
          </Button>
        </div>
      ) : (
        <div style={{ display: "flex", gap: "0.4rem", alignItems: "center" }}>
          <span style={{ fontFamily: "monospace", fontSize: "0.85rem" }}>
            {cred.gitops_repo || (
              <em style={{ opacity: 0.5 }}>Not configured</em>
            )}
          </span>
          <Button
            onClick={() => setEditing(true)}
            style={{ padding: "0.2rem 0.5rem", fontSize: "0.75rem" }}
          >
            <Edit3 size={13} /> Edit
          </Button>
        </div>
      )}
    </div>
  );
}

export function IngressForm({ onSubmit }) {
  return (
    <form className="form-grid ingress-form" onSubmit={onSubmit}>
      <Input name="namespace" placeholder="namespace" required />
      <Input name="name" placeholder="ingress-name" required />
      <Select name="class_name" defaultValue="traefik">
        <option value="traefik">Traefik public</option>
        <option value="tailscale">Tailscale private</option>
        <option value="nginx">nginx</option>
      </Select>
      <Input
        name="host"
        placeholder="app.example.com or app.tailnet.ts.net"
        required
      />
      <Input name="path" placeholder="path, default /" />
      <Input name="service_name" placeholder="service name" required />
      <Input
        name="service_port"
        type="number"
        min="1"
        max="65535"
        placeholder="service port"
        required
      />
      <Input name="tls_secret_name" placeholder="TLS secret, e.g. app-tls" />
      <Input
        name="annotations"
        placeholder="annotations: cert-manager.io/cluster-issuer=letsencrypt-prod"
      />
      <Input name="labels" placeholder="labels: app=my-app" />
      <Button variant="primary" type="submit">
        Save ingress
      </Button>
    </form>
  );
}

export function NetworkPolicyForm({ onSubmit }) {
  return (
    <form className="form-grid network-policy-form" onSubmit={onSubmit}>
      <Input name="namespace" placeholder="namespace" required />
      <Input name="name" placeholder="policy-name" required />
      <Input name="pod_selector" placeholder="pod selector: app=my-app" />
      <label className="check-row">
        <Input
          name="policy_types"
          type="checkbox"
          value="Ingress"
          defaultChecked
        />{" "}
        Ingress
      </label>
      <label className="check-row">
        <Input name="policy_types" type="checkbox" value="Egress" /> Egress
      </label>
      <label className="check-row">
        <Input name="allow_same_namespace" type="checkbox" /> Allow same
        namespace
      </label>
      <label className="check-row">
        <Input name="allow_dns" type="checkbox" /> Allow DNS egress
      </label>
      <Input name="labels" placeholder="labels: managed-by=beancs" />
      <Button variant="primary" type="submit">
        Save policy
      </Button>
    </form>
  );
}

export function ExpandableCell({ value, className = "", max = 36 }) {
  const [expanded, setExpanded] = useState(false);
  const text = formatCell(value);
  const isLong = text.length > max;
  if (!isLong) {
    return <span className={className}>{text}</span>;
  }
  return (
    <Button
      type="button"
      className={`expandable-cell ${expanded ? "expanded" : ""} ${className}`.trim()}
      title={expanded ? "Collapse value" : text}
      onClick={() => setExpanded((current) => !current)}
    >
      {expanded ? text : truncateMiddle(text, max)}
    </Button>
  );
}

export function PaginationBar({
  page,
  pageSize,
  total,
  onPageChange,
  label = "items",
}) {
  const totalItems = Number(total || 0);
  const totalPages = Math.max(1, Math.ceil(totalItems / pageSize));
  const safePage = Math.max(1, Math.min(page, totalPages));
  const start = totalItems === 0 ? 0 : (safePage - 1) * pageSize + 1;
  const end = Math.min(totalItems, safePage * pageSize);
  return (
    <div className="pagination-bar">
      <small className="muted">
        {totalItems === 0
          ? `No ${label}`
          : `${start}-${end} / ${totalItems} ${label}`}
      </small>
      <div className="row-actions">
        <Button
          type="button"
          onClick={() => onPageChange(safePage - 1)}
          disabled={safePage <= 1}
        >
          Prev
        </Button>
        <small className="muted">
          Page {safePage}/{totalPages}
        </small>
        <Button
          type="button"
          onClick={() => onPageChange(safePage + 1)}
          disabled={safePage >= totalPages}
        >
          Next
        </Button>
      </div>
    </div>
  );
}

export function SimpleTable({
  rows,
  columns,
  actions,
  compact = false,
  pageSize = 12,
}) {
  const safeRows = rows || [];
  const [page, setPage] = useState(1);
  const totalPages = Math.max(1, Math.ceil(safeRows.length / pageSize));
  const safePage = Math.min(page, totalPages);
  const pagedRows = safeRows.slice((safePage - 1) * pageSize, safePage * pageSize);
  useEffect(() => {
    setPage(1);
  }, [safeRows.length, pageSize]);
  return (
    <>
      <div className={compact ? "table compact-table" : "table network-table"}>
        <div className="tr head">
          {columns.map((column) => (
            <span key={column}>{column.replaceAll("_", " ")}</span>
          ))}
          {actions && <span>Actions</span>}
        </div>
        {pagedRows.map((row, index) => (
          <div
            className="tr"
            key={`${row.namespace || ""}-${row.name || row.service || index}`}
          >
            {columns.map((column) => {
              const value = formatCell(row[column]);
              return <ExpandableCell key={column} value={value} max={36} />;
            })}
            {actions && <span className="row-actions">{actions(row)}</span>}
          </div>
        ))}
        {safeRows.length === 0 && <div className="empty">No records found.</div>}
      </div>
      <PaginationBar
        page={safePage}
        pageSize={pageSize}
        total={safeRows.length}
        onPageChange={setPage}
      />
    </>
  );
}

export function RuntimeTable({
  kind,
  rows,
  nodeJoinCommand,
  onLoadNodeJoinCommand,
  onCreateNamespace,
  onPatchNamespace,
  onNamespaceDetail,
  onDeleteNamespace,
  onDeletePod,
  onNodeDetail,
  onPodLogs,
  onSaveService,
  onDeleteService,
  onDetail,
}) {
  const keys = rows[0] ? Object.keys(rows[0]).slice(0, 7) : [];
  const [page, setPage] = useState(1);
  const [createOpen, setCreateOpen] = useState(false);
  const pageSize = 12;
  const totalPages = Math.max(1, Math.ceil(rows.length / pageSize));
  const safePage = Math.min(page, totalPages);
  const pagedRows = rows.slice((safePage - 1) * pageSize, safePage * pageSize);
  useEffect(() => {
    setPage(1);
  }, [kind, rows.length]);
  return (
    <div className="stack">
      {(kind === "namespaces" || kind === "services") && (
        <section className="panel action-panel">
          <div>
            <h2>
              {kind === "namespaces" ? (
                <>
                  <Layers3 size={18} /> Namespaces
                </>
              ) : (
                <>
                  <Database size={18} /> Services
                </>
              )}
            </h2>
            <p className="muted">创建入口已从列表中分离，避免页面过长。</p>
          </div>
          <Button variant="primary" type="button" onClick={() => setCreateOpen(true)}>
            <Plus size={15} /> {kind === "namespaces" ? "Create namespace" : "Create service"}
          </Button>
        </section>
      )}
      {kind === "nodes" && (
        <NodeJoinPanel
          command={nodeJoinCommand}
          onLoad={onLoadNodeJoinCommand}
        />
      )}
      <section className="panel">
        <div className="table runtime-table">
          <div className="tr head">
            {keys.map((key) => (
              <span key={key}>{key.replaceAll("_", " ")}</span>
            ))}
            <span>Actions</span>
          </div>
          {pagedRows.map((row, index) => (
            <div
              className="tr"
              key={`${kind}-${row.namespace || ""}-${row.name || index}`}
            >
              {keys.map((key) => {
                const value = formatCell(row[key]);
                const compactFields = new Set([
                  "status",
                  "ready_containers",
                  "total_containers",
                  "restarts",
                ]);
                return (
                  <ExpandableCell
                    key={key}
                    value={value}
                    max={compactFields.has(key) ? 16 : 30}
                  />
                );
              })}
              <span className="row-actions">
                <Button
                  onClick={() =>
                    kind === "nodes"
                      ? onNodeDetail(row)
                      : kind === "namespaces"
                        ? onNamespaceDetail(row.name)
                        : onDetail({ kind, row })
                  }
                >
                  Details
                </Button>
                {kind === "namespaces" && (
                  <Button
                    onClick={() => onDeleteNamespace(row.name)}
                    className="danger-button"
                  >
                    <Trash2 size={15} />
                  </Button>
                )}
                {kind === "pods" && (
                  <>
                    <Button onClick={() => onPodLogs(row)}>Logs</Button>
                    <Button
                      onClick={() => onDeletePod(row)}
                      className="danger-button"
                    >
                      <Trash2 size={15} />
                    </Button>
                  </>
                )}
                {kind === "services" && (
                  <>
                    <Button
                      onClick={() => onDetail({ kind: "service-edit", row })}
                    >
                      Edit
                    </Button>
                    <Button
                      onClick={() => onDeleteService(row)}
                      className="danger-button"
                    >
                      <Trash2 size={15} />
                    </Button>
                  </>
                )}
              </span>
            </div>
          ))}
          {rows.length === 0 && <div className="empty">No {kind} found.</div>}
        </div>
        <PaginationBar
          page={safePage}
          pageSize={pageSize}
          total={rows.length}
          onPageChange={setPage}
          label={kind}
        />
      </section>
      {createOpen && kind === "namespaces" && (
        <Modal
          title="Create namespace"
          subtitle="将创建表单放在弹窗中，便于聚焦输入。"
          onClose={() => setCreateOpen(false)}
        >
          <form
            className="form-grid inline-form"
            onSubmit={async (event) => {
              await onCreateNamespace(event);
              setCreateOpen(false);
            }}
          >
            <Input name="name" placeholder="namespace-name" required />
            <Input name="labels" placeholder="labels: env=dev,team=platform" />
            <div className="modal-actions">
              <Button type="button" onClick={() => setCreateOpen(false)}>
                Cancel
              </Button>
              <Button variant="primary" type="submit">
                <Plus size={15} /> Create
              </Button>
            </div>
          </form>
        </Modal>
      )}
      {createOpen && kind === "services" && (
        <Modal
          title="Create service"
          subtitle="填写服务参数后保存。"
          onClose={() => setCreateOpen(false)}
          className="wide-modal"
        >
          <ServiceForm
            onSubmit={async (event) => {
              await onSaveService(event);
              setCreateOpen(false);
            }}
            namespaces={[]}
          />
          <div className="modal-actions">
            <Button type="button" onClick={() => setCreateOpen(false)}>
              Close
            </Button>
          </div>
        </Modal>
      )}
    </div>
  );
}

export function NodeJoinPanel({ command, onLoad }) {
  return (
    <section className="panel node-ops-panel">
      <div className="action-panel">
        <div>
          <h2>
            <Server size={18} /> K3s node join
          </h2>
          <p>
            Generate an agent or server join command from the configured K3s
            server URL and node token.
          </p>
        </div>
        <div className="row-actions">
          <Button onClick={() => onLoad("agent")}>Agent command</Button>
          <Button onClick={() => onLoad("server")}>Server command</Button>
        </div>
      </div>
      {command?.configured ? (
        <pre className="command-box">{command.command}</pre>
      ) : (
        <p className="muted">
          {command?.message || "Loading join command configuration..."}
        </p>
      )}
    </section>
  );
}

export function ContainerLogViewer({
  pod,
  logs,
  logFollow,
  logStatus,
  selectedContainer,
  tail,
  loaded,
  onSelectContainer,
  onSetTail,
  onLoad,
  onFollow,
  onStop,
}) {
  const containers = podContainers(pod);
  const canRead = Boolean(selectedContainer);
  return (
    <div className="container-log-viewer">
      <div className="log-header">
        <span className="muted">
          {logStatus || "Choose a container to load logs."}
        </span>
        <div className="row-actions">
          <Select
            className="compact-select"
            value={tail}
            disabled={logFollow}
            onChange={(event) => onSetTail(Number(event.target.value))}
          >
            <option value={100}>Last 100 lines</option>
            <option value={200}>Last 200 lines</option>
            <option value={500}>Last 500 lines</option>
            <option value={1000}>Last 1000 lines</option>
          </Select>
          <Button disabled={!canRead || logFollow} onClick={onLoad}>
            <RefreshCw size={15} /> Load
          </Button>
          {logFollow ? (
            <Button onClick={onStop}>Stop follow</Button>
          ) : (
            <Button variant="primary" disabled={!canRead} onClick={onFollow}>
              Follow live
            </Button>
          )}
        </div>
      </div>
      <div className="container-picker">
        {containers.map((container) => (
          <Button
            key={container.name}
            className={
              selectedContainer === container.name
                ? "container-chip active"
                : "container-chip"
            }
            onClick={() => onSelectContainer(container.name)}
            type="button"
            disabled={logFollow}
          >
            <b>{container.name}</b>
            {container.image && <small>{container.image}</small>}
          </Button>
        ))}
        {containers.length === 0 && (
          <div className="empty">No containers reported for this pod.</div>
        )}
      </div>
      <pre className="modal-log">
        {loaded
          ? logs || "No logs returned for this container."
          : "Logs are not loaded yet. Select a container, then click Load or Follow live."}
      </pre>
    </div>
  );
}

export function ResourceMeter({ label, value, detail }) {
  const normalized = Math.max(0, Math.min(100, Number(value || 0)));
  return (
    <div className="resource-meter">
      <div>
        <b>{label}</b>
        <span>{normalized.toFixed(1)}%</span>
      </div>
      <progress value={normalized} max="100" />
      <small>{detail}</small>
    </div>
  );
}

export function ServiceForm({ existing, onSubmit }) {
  return (
    <form className="form-grid inline-form service-form" onSubmit={onSubmit}>
      {!existing && <Input name="namespace" placeholder="namespace" required />}
      {!existing && <Input name="name" placeholder="service-name" required />}
      <Select name="type" defaultValue={existing?.type || "ClusterIP"}>
        <option>ClusterIP</option>
        <option>NodePort</option>
        <option>LoadBalancer</option>
      </Select>
      <Input
        name="selector"
        placeholder="selector: app=my-app,managed-by=beancs"
        defaultValue={formatKeyValues(existing?.selector)}
      />
      <Input
        name="ports"
        placeholder="ports: http:80:8080:30080/TCP,https:443:8443/TCP"
        defaultValue={portsToForm(existing?.ports)}
        required
      />
      <Input
        name="labels"
        placeholder="labels: app=my-app"
        defaultValue={formatKeyValues(existing?.labels)}
      />
      <Input name="load_balancer_ip" placeholder="LoadBalancer IP, optional" />
      <Input name="external_ips" placeholder="External IPs: 1.2.3.4,5.6.7.8" />
      <Select name="external_traffic_policy" defaultValue="">
        <option value="">Traffic policy</option>
        <option value="Cluster">Cluster</option>
        <option value="Local">Local</option>
      </Select>
      <Button variant="primary" type="submit">
        {existing ? "Save service" : "Create service"}
      </Button>
    </form>
  );
}

export function CredentialManager({
  kind,
  rows,
  onCreate,
  onDelete,
  dependencies = [],
}) {
  const safeRows = rows || [];
  const isCloudflare = kind === "cloudflare";
  const [basaltDeployMode, setBasaltDeployMode] = useState("external");
  const [createOpen, setCreateOpen] = useState(false);
  const [page, setPage] = useState(1);
  const pageSize = 10;
  const totalPages = Math.max(1, Math.ceil(safeRows.length / pageSize));
  const safePage = Math.min(page, totalPages);
  const pagedRows = safeRows.slice(
    (safePage - 1) * pageSize,
    safePage * pageSize,
  );
  useEffect(() => {
    setPage(1);
  }, [safeRows.length]);
  const databaseDependencies = (dependencies || []).filter((dependency) =>
    ["mysql", "postgresql"].includes(dependency.type),
  );
  const title = isCloudflare ? "Cloudflare accounts" : "BasaltPass tenants";
  const columns = isCloudflare
    ? ["name", "account_id", "is_active"]
    : [
        "name",
        "tenant_code",
        "tenant_id",
        "deploy_mode",
        "base_url",
        "deploy_status",
        "is_active",
      ];
  function renderCreateForm() {
    return (
      <form
        className="form-grid"
        onSubmit={async (event) => {
          const ok = await onCreate(kind, event);
          if (ok) setCreateOpen(false);
        }}
      >
        <Input name="name" placeholder="Name" required />
        {isCloudflare ? (
          <>
            <Input name="account_id" placeholder="Account ID, optional" />
            <Input
              name="api_token"
              type="password"
              placeholder="Cloudflare API token"
              required
            />
          </>
        ) : (
          <>
            <Input
              name="base_url"
              placeholder="https://auth.example.com"
              required
            />
            <Select
              name="deploy_mode"
              value={basaltDeployMode}
              onChange={(event) => setBasaltDeployMode(event.target.value)}
            >
              <option value="external">Existing tenant</option>
              <option value="managed">BeanCS managed tenant</option>
            </Select>
            <Input name="tenant_code" placeholder="Tenant code" required />
            <Input name="tenant_id" placeholder="Tenant ID, optional" />
            {basaltDeployMode === "managed" && (
              <>
                <Input
                  name="owner_email"
                  type="email"
                  placeholder="Tenant owner email"
                  required
                />
                <Input name="namespace" placeholder="Namespace, optional" />
                <Input
                  name="backend_image"
                  placeholder="ghcr.io/owner/basaltpass-backend:tag"
                  required
                />
                <Input
                  name="frontend_image"
                  placeholder="ghcr.io/owner/basaltpass-frontend:tag"
                  required
                />
                <Input
                  name="public_host"
                  placeholder="auth.example.com, optional"
                />
                <Select name="exposure_mode" defaultValue="public">
                  <option value="public">Public ingress</option>
                  <option value="private">Private ingress</option>
                </Select>
                <Select name="database_binding" required>
                  <option value="">Database credential</option>
                  {databaseDependencies.flatMap((dependency) =>
                    (dependency.credentials || []).map((credential) => (
                      <option
                        key={`${dependency.id}:${credential.id}`}
                        value={`${dependency.id}:${credential.id}`}
                      >
                        {dependency.name} / {credential.name}
                      </option>
                    )),
                  )}
                </Select>
                <Input name="max_apps" type="number" placeholder="Max apps" />
                <Input name="max_users" type="number" placeholder="Max users" />
                <Input
                  name="jwt_secret"
                  type="password"
                  placeholder="JWT secret, generated if empty"
                />
                <Input
                  name="service_token"
                  type="password"
                  placeholder="Management service token"
                  required
                />
              </>
            )}
            <Input
              name="automation_token"
              type="password"
              placeholder="Tenant automation token bpk_..."
              required
            />
          </>
        )}
        <div className="modal-actions">
          <Button type="button" onClick={() => setCreateOpen(false)}>
            Cancel
          </Button>
          <Button variant="primary" type="submit">
            <Plus size={16} /> Add
          </Button>
        </div>
      </form>
    );
  }
  return (
    <div className="stack">
      <section className="panel action-panel">
        <div>
          <h2>
            <KeyRound size={18} /> {title}
          </h2>
          <p className="muted">创建表单已单独放入弹窗，列表区域更聚焦。</p>
        </div>
        <Button type="button" variant="primary" onClick={() => setCreateOpen(true)}>
          <Plus size={16} /> Add{" "}
          {isCloudflare ? "Cloudflare account" : "BasaltPass tenant"}
        </Button>
      </section>
      <section className="panel">
        <h2>
          <KeyRound size={18} /> {title}
        </h2>
        <div className={`table compact ${isCloudflare ? "" : "basaltpass-table"}`}>
          <div className="tr head">
            {columns.map((column) => (
              <span key={column}>{column.replaceAll("_", " ")}</span>
            ))}
            <span>Actions</span>
          </div>
          {pagedRows.map((row) => (
            <div className="tr" key={row.id}>
              {columns.map((column) => (
                <ExpandableCell
                  key={column}
                  value={row[column] || "-"}
                  max={34}
                />
              ))}
              <span>
                <Button onClick={() => onDelete(kind, row.id)}>
                  <Trash2 size={15} />
                </Button>
              </span>
            </div>
          ))}
          {safeRows.length === 0 && (
            <div className="empty">No credentials found.</div>
          )}
        </div>
        <PaginationBar
          page={safePage}
          pageSize={pageSize}
          total={safeRows.length}
          onPageChange={setPage}
          label="credentials"
        />
      </section>
      {createOpen && (
        <Modal
          title={
            isCloudflare ? "Add Cloudflare account" : "Add BasaltPass tenant"
          }
          subtitle="创建表单已与列表分离。"
          className="wide-modal"
          onClose={() => setCreateOpen(false)}
        >
          {renderCreateForm()}
        </Modal>
      )}
    </div>
  );
}

export function EnvEditor({
  entries,
  onChange,
  title = "Environment variables",
  masked = false,
}) {
  const [bulkText, setBulkText] = useState("");
  const setEntry = (index, patch) =>
    onChange(
      entries.map((entry, itemIndex) =>
        itemIndex === index ? { ...entry, ...patch } : entry,
      ),
    );
  const addEntry = () => onChange([...(entries || []), { key: "", value: "" }]);
  const removeEntry = (index) =>
    onChange(entries.filter((_, itemIndex) => itemIndex !== index));
  const importBulk = () => {
    const parsed = parseDotEnv(bulkText);
    if (!parsed.length) return;
    const byKey = new Map(
      (entries || [])
        .filter((entry) => entry.key)
        .map((entry) => [entry.key, entry]),
    );
    parsed.forEach((entry) => byKey.set(entry.key, entry));
    onChange(Array.from(byKey.values()));
    setBulkText("");
  };
  return (
    <div className="env-editor">
      <div className="section-head">
        <h3>{title}</h3>
        <Button type="button" onClick={addEntry}>
          <Plus size={15} /> Add variable
        </Button>
      </div>
      <div className="env-list">
        {(entries || []).map((entry, index) => (
          <div className="env-row" key={`${entry.key}-${index}`}>
            <Input
              value={entry.key}
              placeholder="DATABASE_URL"
              onChange={(event) =>
                setEntry(index, { key: event.target.value.trim() })
              }
            />
            <Input
              value={entry.value}
              type={masked && entry.value === "********" ? "password" : "text"}
              placeholder={masked ? "Keep existing secret" : "value"}
              onChange={(event) =>
                setEntry(index, { value: event.target.value })
              }
            />
            <Button type="button" onClick={() => removeEntry(index)}>
              <Trash2 size={15} />
            </Button>
          </div>
        ))}
        {(entries || []).length === 0 && (
          <div className="empty">No runtime variables configured.</div>
        )}
      </div>
      <label>Import .env</label>
      <Textarea
        value={bulkText}
        placeholder={"DATABASE_URL=postgres://...\nRABBITMQ_URL=amqp://..."}
        onChange={(event) => setBulkText(event.target.value)}
      />
      <Button type="button" onClick={importBulk} disabled={!bulkText.trim()}>
        <Upload size={15} /> Import variables
      </Button>
      <p className="muted">
        Values are stored in the Kubernetes app-env-vars Secret. Existing masked
        values stay unchanged unless replaced.
      </p>
    </div>
  );
}

export function Field({
  label,
  value,
  onChange,
  type = "text",
  required = false,
}) {
  return (
    <>
      <label>{label}</label>
      <Input
        type={type}
        value={value ?? ""}
        required={required}
        onChange={(event) => onChange(event.target.value)}
      />
    </>
  );
}

export function ChevronIcon({ open }) {
  return <span className={open ? "chevron open" : "chevron"}>⌄</span>;
}

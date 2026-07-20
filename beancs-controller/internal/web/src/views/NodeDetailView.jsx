import React, { useEffect, useMemo, useRef, useState } from "react";
import * as LucideIcons from "lucide-react";
import {
  formatTime,
  formatBytes,
  formatKeyValues,
  taintsToForm,
} from "../utils/index";
import { ResourceMeter, Button, Input, Textarea } from "../components/index";
import { t } from "../i18n/index";
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
export default function NodeDetailView({
  detail,
  health,
  onLoadHealth,
  onSaveLabels,
  onSaveTaints,
  onCordon,
  onDrain,
  onDelete,
}) {
  const [deleteStep, setDeleteStep] = useState("idle");
  const [deleteName, setDeleteName] = useState("");
  const row = detail.row || {};
  const summary = row.summary || row;
  const usage = row.usage || {};
  const disk = row.disk || {};
  const network = row.network || {};
  const pods = row.pods || [];
  const conditions = row.conditions || [];
  const nodeName = summary.name || row.name;
  const canConfirmDelete = Boolean(nodeName) && deleteName.trim() === nodeName;
  return (
    <div className="node-detail">
      {detail.loading && <p className="muted">{t("Loading live node status...")}</p>}
      {detail.error && <p className="error-inline">{detail.error}</p>}
      <section className="node-section node-actions">
        <div className="row-actions">
          <Button onClick={() => onLoadHealth(nodeName)}>
            <CheckCircle2 size={15} /> {t("Health check")}
          </Button>
          <Button onClick={() => onCordon(nodeName, false)}>{t("Cordon")}</Button>
          <Button onClick={() => onCordon(nodeName, true)}>{t("Uncordon")}</Button>
          <Button
            onClick={() =>
              onDrain(nodeName, {
                force: false,
                ignore_daemonsets: true,
                delete_emptydir_data: false,
                grace_period_seconds: 30,
              })
            }
          >
            {t("Drain safe")}
          </Button>
          <Button
            disabled={!nodeName}
            onClick={() => {
              setDeleteStep("warning");
              setDeleteName("");
            }}
            variant="danger"
          >
            <Trash2 size={15} /> {t("Delete node")}
          </Button>
        </div>
        {health && (
          <div
            className={
              health.healthy ? "health-card good" : "health-card warning"
            }
          >
            <b>{health.status}</b>
            <span>
              {t("{count} checks", { count: (health.checks || []).length })} ·{" "}
              {t("{count} abnormal pods", {
                count: (health.abnormal_pods || []).length,
              })}{" "}
              · {formatTime(health.checked_at)}
            </span>
            {(health.checks || []).map((check) => (
              <small key={`${check.name}-${check.message}`}>
                {check.name}: {check.status}
                {check.message ? ` · ${check.message}` : ""}
              </small>
            ))}
          </div>
        )}
      </section>
      {deleteStep !== "idle" && (
        <section className="node-section destructive-flow">
          <h3>
            <AlertTriangle size={15} /> {t("Dangerous node deletion")}
          </h3>
          {deleteStep === "warning" && (
            <>
              <p>
                {t(
                  "Deleting a node removes it from Kubernetes cluster state. Make sure workloads are drained and the machine is intentionally removed or ready to rejoin.",
                )}
              </p>
              <div className="row-actions">
                <Button type="button" onClick={() => setDeleteStep("name")}>
                  {t("Continue")}
                </Button>
                <Button type="button" onClick={() => setDeleteStep("idle")}>
                  {t("Cancel")}
                </Button>
              </div>
            </>
          )}
          {deleteStep === "name" && (
            <>
              <p>{t("Type the exact machine name to continue.")}</p>
              <Input
                value={deleteName}
                onChange={(event) => setDeleteName(event.target.value)}
                placeholder={nodeName}
              />
              <div className="row-actions">
                <Button
                  type="button"
                  disabled={!canConfirmDelete}
                  onClick={() => setDeleteStep("final")}
                >
                  {t("Continue")}
                </Button>
                <Button type="button" onClick={() => setDeleteStep("idle")}>
                  {t("Cancel")}
                </Button>
              </div>
            </>
          )}
          {deleteStep === "final" && (
            <>
              <p>
                <b>{t("Final warning.")}</b> {t("This action deletes node {name} from the cluster API. This is the last confirmation step.", { name: nodeName })}
              </p>
              <div className="row-actions">
                <Button
                  type="button"
                  className="filled"
                  onClick={() => onDelete(nodeName)}
                  variant="danger"
                >
                  <Trash2 size={15} /> {t("Delete {name}", { name: nodeName })}
                </Button>
                <Button type="button" onClick={() => setDeleteStep("idle")}>
                  {t("Cancel")}
                </Button>
              </div>
            </>
          )}
        </section>
      )}
      <div className="node-status-grid">
        <div className="runtime-summary">
          <strong>{summary.status || "-"}</strong>
          <span>
            {summary.name} · {summary.version || "-"}
          </span>
        </div>
        <div className="detail-list compact-details">
          <span>
            {t("Internal IP")}{" "}
            <b>{summary.internal_ip || row.addresses?.InternalIP || "-"}</b>
          </span>
          <span>
            {t("Roles")} <b>{(summary.roles || []).join(", ") || "-"}</b>
          </span>
          <span>
            {t("Scheduling")}{" "}
            <b>
              {summary.schedulable === false ? t("Cordoned") : t("Schedulable")}
            </b>
          </span>
          <span>
            {t("Pods")}{" "}
            <b>
              {pods.length}/{row.allocatable?.pods || "-"}
            </b>
          </span>
          <span>
            {t("Checked")} <b>{formatTime(row.checked_at)}</b>
          </span>
        </div>
      </div>
      <section className="node-section">
        <h3>{t("Live resources")}</h3>
        {row.metrics_available || row.disk || row.network ? (
          <div className="resource-grid">
            <ResourceMeter
              label={t("CPU allocatable")}
              value={usage.cpu_allocatable_percent}
              detail={
                row.metrics_available
                  ? `${usage.cpu_millis || 0}m / ${row.allocatable?.cpu_millis || 0}m`
                  : t("metrics-server unavailable")
              }
            />
            <ResourceMeter
              label={t("Memory allocatable")}
              value={usage.memory_allocatable_percent}
              detail={
                row.metrics_available
                  ? `${formatBytes(usage.memory_bytes)} / ${formatBytes(row.allocatable?.memory_bytes)}`
                  : t("metrics-server unavailable")
              }
            />
            <ResourceMeter
              label={t("CPU capacity")}
              value={usage.cpu_capacity_percent}
              detail={
                row.metrics_available
                  ? `${usage.cpu || "-"} / ${row.capacity?.cpu || "-"}`
                  : t("metrics-server unavailable")
              }
            />
            <ResourceMeter
              label={t("Memory capacity")}
              value={usage.memory_capacity_percent}
              detail={
                row.metrics_available
                  ? `${usage.memory || "-"} / ${row.capacity?.memory || "-"}`
                  : t("metrics-server unavailable")
              }
            />
            <ResourceMeter
              label={t("Disk")}
              value={disk.used_percent}
              detail={`${formatBytes(disk.used_bytes)} / ${formatBytes(disk.capacity_bytes)}`}
            />
            <ResourceMeter
              label={t("Network")}
              value={0}
              detail={`RX ${formatBytes(network.rx_bytes)} · TX ${formatBytes(network.tx_bytes)}`}
            />
          </div>
        ) : (
          <p className="muted">
            {t("Metrics unavailable")}
            {row.metrics_error
              ? `: ${row.metrics_error}`
              : t(". Install metrics-server to show live CPU and memory usage.")}
          </p>
        )}
      </section>
      <section className="node-section">
        <h3>{t("Conditions")}</h3>
        <div className="condition-grid">
          {conditions.map((condition) => (
            <div
              className={
                condition.status === "True" && condition.type === "Ready"
                  ? "condition good"
                  : condition.status === "True"
                    ? "condition warning"
                    : "condition"
              }
              key={condition.type}
            >
              <b>
                {condition.type}: {condition.status}
              </b>
              <small>
                {condition.reason || "-"} ·{" "}
                {formatTime(condition.last_transition_time)}
              </small>
              {condition.message && <p>{condition.message}</p>}
            </div>
          ))}
        </div>
      </section>
      <section className="node-section">
        <h3>{t("System")}</h3>
        <div className="detail-list compact-details">
          {Object.entries(row.system_info || {}).map(([key, value]) => (
            <span key={key}>
              {key.replaceAll("_", " ")} <b>{value || "-"}</b>
            </span>
          ))}
        </div>
      </section>
      <section className="node-section">
        <h3>{t("Pods on this node")}</h3>
        <div className="mini-table">
          {pods.map((pod) => (
            <div key={`${pod.namespace}/${pod.name}`}>
              <span>
                {pod.namespace}/{pod.name}
                <small>{(pod.containers || []).join(" · ")}</small>
              </span>
              <b>
                {pod.ready_containers}/{pod.total_containers} · {pod.status}
              </b>
            </div>
          ))}
          {pods.length === 0 && (
            <div className="empty">{t("No pods scheduled on this node.")}</div>
          )}
        </div>
      </section>
      <section className="node-section">
        <h3>{t("Labels")}</h3>
        <form
          className="form-grid node-edit-form"
          onSubmit={(event) => {
            event.preventDefault();
            onSaveLabels(nodeName, event.currentTarget.labels.value);
          }}
        >
          <Textarea name="labels" defaultValue={formatKeyValues(row.labels)} />
          <Button variant="primary">{t("Save labels")}</Button>
        </form>
        <div className="label-cloud">
          {Object.entries(row.labels || {}).map(([key, value]) => (
            <span key={key}>
              {key}={value}
            </span>
          ))}
        </div>
      </section>
      <section className="node-section">
        <h3>{t("Taints")}</h3>
        <form
          className="form-grid node-edit-form"
          onSubmit={(event) => {
            event.preventDefault();
            onSaveTaints(nodeName, event.currentTarget.taints.value);
          }}
        >
          <Textarea
            name="taints"
            placeholder="key=value:NoSchedule, dedicated=gpu:NoExecute"
            defaultValue={taintsToForm(row.taints || [])}
          />
          <Button variant="primary">{t("Save taints")}</Button>
        </form>
        <div className="signal-list">
          {(row.taints || []).map((taint) => (
            <span key={taint}>{taint}</span>
          ))}
          {(row.taints || []).length === 0 && <span>{t("No taints")}</span>}
        </div>
      </section>
    </div>
  );
}

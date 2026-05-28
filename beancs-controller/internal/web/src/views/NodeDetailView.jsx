import React, { useEffect, useMemo, useRef, useState } from "react";
import * as LucideIcons from "lucide-react";
import {
  formatTime,
  formatBytes,
  formatKeyValues,
  taintsToForm,
} from "../utils/index";
import { ResourceMeter, Button, Input, Textarea } from "../components/index";
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
      {detail.loading && <p className="muted">Loading live node status...</p>}
      {detail.error && <p className="error-inline">{detail.error}</p>}
      <section className="node-section node-actions">
        <div className="row-actions">
          <Button onClick={() => onLoadHealth(nodeName)}>
            <CheckCircle2 size={15} /> Health check
          </Button>
          <Button onClick={() => onCordon(nodeName, false)}>Cordon</Button>
          <Button onClick={() => onCordon(nodeName, true)}>Uncordon</Button>
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
            Drain safe
          </Button>
          <Button
            disabled={!nodeName}
            onClick={() => {
              setDeleteStep("warning");
              setDeleteName("");
            }}
            variant="danger"
          >
            <Trash2 size={15} /> Delete node
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
              {(health.checks || []).length} checks ·{" "}
              {(health.abnormal_pods || []).length} abnormal pods ·{" "}
              {formatTime(health.checked_at)}
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
            <AlertTriangle size={15} /> Dangerous node deletion
          </h3>
          {deleteStep === "warning" && (
            <>
              <p>
                Deleting a node removes it from Kubernetes cluster state. Make
                sure workloads are drained and the machine is intentionally
                removed or ready to rejoin.
              </p>
              <div className="row-actions">
                <Button type="button" onClick={() => setDeleteStep("name")}>
                  Continue
                </Button>
                <Button type="button" onClick={() => setDeleteStep("idle")}>
                  Cancel
                </Button>
              </div>
            </>
          )}
          {deleteStep === "name" && (
            <>
              <p>Type the exact machine name to continue.</p>
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
                  Continue
                </Button>
                <Button type="button" onClick={() => setDeleteStep("idle")}>
                  Cancel
                </Button>
              </div>
            </>
          )}
          {deleteStep === "final" && (
            <>
              <p>
                <b>Final warning.</b> This action deletes node{" "}
                <span className="mono">{nodeName}</span> from the cluster API.
                This is the last confirmation step.
              </p>
              <div className="row-actions">
                <Button
                  type="button"
                  className="filled"
                  onClick={() => onDelete(nodeName)}
                  variant="danger"
                >
                  <Trash2 size={15} /> Delete {nodeName}
                </Button>
                <Button type="button" onClick={() => setDeleteStep("idle")}>
                  Cancel
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
            Internal IP{" "}
            <b>{summary.internal_ip || row.addresses?.InternalIP || "-"}</b>
          </span>
          <span>
            Roles <b>{(summary.roles || []).join(", ") || "-"}</b>
          </span>
          <span>
            Scheduling{" "}
            <b>{summary.schedulable === false ? "Cordoned" : "Schedulable"}</b>
          </span>
          <span>
            Pods{" "}
            <b>
              {pods.length}/{row.allocatable?.pods || "-"}
            </b>
          </span>
          <span>
            Checked <b>{formatTime(row.checked_at)}</b>
          </span>
        </div>
      </div>
      <section className="node-section">
        <h3>Live resources</h3>
        {row.metrics_available || row.disk || row.network ? (
          <div className="resource-grid">
            <ResourceMeter
              label="CPU allocatable"
              value={usage.cpu_allocatable_percent}
              detail={
                row.metrics_available
                  ? `${usage.cpu_millis || 0}m / ${row.allocatable?.cpu_millis || 0}m`
                  : "metrics-server unavailable"
              }
            />
            <ResourceMeter
              label="Memory allocatable"
              value={usage.memory_allocatable_percent}
              detail={
                row.metrics_available
                  ? `${formatBytes(usage.memory_bytes)} / ${formatBytes(row.allocatable?.memory_bytes)}`
                  : "metrics-server unavailable"
              }
            />
            <ResourceMeter
              label="CPU capacity"
              value={usage.cpu_capacity_percent}
              detail={
                row.metrics_available
                  ? `${usage.cpu || "-"} / ${row.capacity?.cpu || "-"}`
                  : "metrics-server unavailable"
              }
            />
            <ResourceMeter
              label="Memory capacity"
              value={usage.memory_capacity_percent}
              detail={
                row.metrics_available
                  ? `${usage.memory || "-"} / ${row.capacity?.memory || "-"}`
                  : "metrics-server unavailable"
              }
            />
            <ResourceMeter
              label="Disk"
              value={disk.used_percent}
              detail={`${formatBytes(disk.used_bytes)} / ${formatBytes(disk.capacity_bytes)}`}
            />
            <ResourceMeter
              label="Network"
              value={0}
              detail={`RX ${formatBytes(network.rx_bytes)} · TX ${formatBytes(network.tx_bytes)}`}
            />
          </div>
        ) : (
          <p className="muted">
            Metrics unavailable
            {row.metrics_error
              ? `: ${row.metrics_error}`
              : ". Install metrics-server to show live CPU and memory usage."}
          </p>
        )}
      </section>
      <section className="node-section">
        <h3>Conditions</h3>
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
        <h3>System</h3>
        <div className="detail-list compact-details">
          {Object.entries(row.system_info || {}).map(([key, value]) => (
            <span key={key}>
              {key.replaceAll("_", " ")} <b>{value || "-"}</b>
            </span>
          ))}
        </div>
      </section>
      <section className="node-section">
        <h3>Pods on this node</h3>
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
            <div className="empty">No pods scheduled on this node.</div>
          )}
        </div>
      </section>
      <section className="node-section">
        <h3>Labels</h3>
        <form
          className="form-grid node-edit-form"
          onSubmit={(event) => {
            event.preventDefault();
            onSaveLabels(nodeName, event.currentTarget.labels.value);
          }}
        >
          <Textarea name="labels" defaultValue={formatKeyValues(row.labels)} />
          <Button variant="primary">Save labels</Button>
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
        <h3>Taints</h3>
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
          <Button variant="primary">Save taints</Button>
        </form>
        <div className="signal-list">
          {(row.taints || []).map((taint) => (
            <span key={taint}>{taint}</span>
          ))}
          {(row.taints || []).length === 0 && <span>No taints</span>}
        </div>
      </section>
    </div>
  );
}

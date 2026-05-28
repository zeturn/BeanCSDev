import React, { useEffect, useMemo, useRef, useState } from "react";
import * as LucideIcons from "lucide-react";
import {
  formatTime,
  formatBytes,
  formatPercent,
  formatDuration,
} from "../utils/index";
import {
  MetricCard,
  IndustrialMeter,
  AlertList,
  Button,
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
export default function DashboardView({ dashboard, refresh }) {
  if (!dashboard) {
    return (
      <section className="panel">
        <div className="empty">Loading cluster dashboard...</div>
      </section>
    );
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
          <p>
            Kubernetes {dashboard.kubernetes_version || "-"}
            {dashboard.k3s_version ? ` · K3s ${dashboard.k3s_version}` : ""}
          </p>
        </div>
        <div
          className={
            dashboard.healthy ? "health-badge good" : "health-badge bad"
          }
        >
          <span>{dashboard.status || "Unknown"}</span>
          <b>{dashboard.healthy ? "Ready" : "NotReady"}</b>
        </div>
        <Button onClick={refresh}>
          <RefreshCw size={15} /> Refresh
        </Button>
      </section>

      <section className="dashboard-kpis">
        <MetricCard
          icon={Server}
          label="Nodes"
          value={nodes.total || 0}
          detail={`${nodes.server || 0} Server · ${nodes.agent || 0} Agent · ${nodes.not_ready || 0} NotReady`}
        />
        <MetricCard
          icon={Boxes}
          label="Pods"
          value={`${pods.running || 0} / ${pods.total || 0}`}
          detail={`${pods.abnormal || 0} abnormal · ${pods.pending || 0} pending`}
        />
        <MetricCard
          icon={Cpu}
          label="CPU"
          value={`${formatPercent(resources.cpu_percent)}%`}
          detail={`${resources.cpu_used_millis || 0}m / ${resources.cpu_total_millis || 0}m`}
        />
        <MetricCard
          icon={MemoryStick}
          label="Memory"
          value={`${formatPercent(resources.memory_percent)}%`}
          detail={`${formatBytes(resources.memory_used_bytes)} / ${formatBytes(resources.memory_total_bytes)}`}
        />
        <MetricCard
          icon={HardDrive}
          label="Disk"
          value={`${formatPercent(resources.disk_percent)}%`}
          detail={`${formatBytes(resources.disk_used_bytes)} / ${formatBytes(resources.disk_total_bytes)}`}
        />
        <MetricCard
          icon={AlertTriangle}
          label="Alerts"
          value={alerts.length}
          detail={`${events.length} recent warning events`}
          tone={alerts.length > 0 ? "warning" : "good"}
        />
      </section>

      <section className="dashboard-grid">
        <div className="panel dashboard-panel">
          <h2>
            <Activity size={18} /> Live Resource Utilization
          </h2>
          <div className="industrial-meters">
            <IndustrialMeter
              label="CPU"
              value={resources.cpu_percent}
              detail={`${resources.cpu_used_millis || 0}m / ${resources.cpu_total_millis || 0}m`}
            />
            <IndustrialMeter
              label="Memory"
              value={resources.memory_percent}
              detail={`${formatBytes(resources.memory_used_bytes)} / ${formatBytes(resources.memory_total_bytes)}`}
            />
            <IndustrialMeter
              label="Disk"
              value={resources.disk_percent}
              detail={`${formatBytes(resources.disk_used_bytes)} / ${formatBytes(resources.disk_total_bytes)}`}
            />
          </div>
          {!dashboard.metrics_available && (
            <p className="muted">
              Metrics partially unavailable:{" "}
              {dashboard.metrics_error ||
                "metrics-server or node stats endpoint did not return data."}
            </p>
          )}
        </div>
        <div className="panel dashboard-panel">
          <h2>
            <Server size={18} /> Cluster Runtime
          </h2>
          <div className="detail-list">
            <span>
              Status <b>{dashboard.status}</b>
            </span>
            <span>
              Ready nodes{" "}
              <b>
                {nodes.ready || 0}/{nodes.total || 0}
              </b>
            </span>
            <span>
              Running pods{" "}
              <b>
                {pods.running || 0}/{pods.total || 0}
              </b>
            </span>
            <span>
              Uptime <b>{formatDuration(dashboard.uptime_seconds)}</b>
            </span>
            <span>
              Last check <b>{formatTime(dashboard.checked_at)}</b>
            </span>
          </div>
        </div>
      </section>

      <section className="dashboard-grid">
        <div className="panel dashboard-panel">
          <h2>
            <AlertTriangle size={18} /> Recent Alerts
          </h2>
          <AlertList rows={alerts} empty="No active alerts reported." />
        </div>
        <div className="panel dashboard-panel">
          <h2>
            <ListRestart size={18} /> Events and Error Signals
          </h2>
          <div className="timeline">
            {events.map((event, index) => (
              <div
                className="timeline-item"
                key={`${event.object}-${event.reason}-${index}`}
              >
                <span className="dot failed" />
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
        </div>
      </section>
    </div>
  );
}

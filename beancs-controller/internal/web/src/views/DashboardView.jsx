import React, { useEffect, useMemo, useRef, useState } from "react";
import * as LucideIcons from "lucide-react";
import { t } from "../i18n/index";
import {
  formatTime,
  formatBytes,
  formatPercent,
} from "../utils/index";
import {
  MetricCard,
  IndustrialMeter,
  AlertList,
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
export default function DashboardView({ dashboard }) {
  if (!dashboard) {
    return (
      <section className="panel">
        <div className="empty">{t("Loading cluster dashboard...")}</div>
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
      <section className="dashboard-kpis">
        <MetricCard
          icon={Server}
          label={t("Nodes")}
          value={nodes.total || 0}
          detail={t("{server} Server · {agent} Agent · {notReady} NotReady", {
            server: nodes.server || 0,
            agent: nodes.agent || 0,
            notReady: nodes.not_ready || 0,
          })}
        />
        <MetricCard
          icon={Boxes}
          label={t("Pods")}
          value={`${pods.running || 0} / ${pods.total || 0}`}
          detail={t("{abnormal} abnormal · {pending} pending", {
            abnormal: pods.abnormal || 0,
            pending: pods.pending || 0,
          })}
        />
        <MetricCard
          icon={Cpu}
          label={t("CPU")}
          value={`${formatPercent(resources.cpu_percent)}%`}
          detail={`${resources.cpu_used_millis || 0}m / ${resources.cpu_total_millis || 0}m`}
        />
        <MetricCard
          icon={MemoryStick}
          label={t("Memory")}
          value={`${formatPercent(resources.memory_percent)}%`}
          detail={`${formatBytes(resources.memory_used_bytes)} / ${formatBytes(resources.memory_total_bytes)}`}
        />
        <MetricCard
          icon={HardDrive}
          label={t("Disk")}
          value={`${formatPercent(resources.disk_percent)}%`}
          detail={`${formatBytes(resources.disk_used_bytes)} / ${formatBytes(resources.disk_total_bytes)}`}
        />
        <MetricCard
          icon={AlertTriangle}
          label={t("Alerts")}
          value={alerts.length}
          detail={t("{count} recent warning events", { count: events.length })}
          tone={alerts.length > 0 ? "warning" : "good"}
        />
      </section>

      <section className="dashboard-grid">
        <div className="panel dashboard-panel">
          <h2>
            <Activity size={18} /> {t("Live Resource Utilization")}
          </h2>
          <div className="industrial-meters">
            <IndustrialMeter
              label={t("CPU")}
              value={resources.cpu_percent}
              detail={`${resources.cpu_used_millis || 0}m / ${resources.cpu_total_millis || 0}m`}
            />
            <IndustrialMeter
              label={t("Memory")}
              value={resources.memory_percent}
              detail={`${formatBytes(resources.memory_used_bytes)} / ${formatBytes(resources.memory_total_bytes)}`}
            />
            <IndustrialMeter
              label={t("Disk")}
              value={resources.disk_percent}
              detail={`${formatBytes(resources.disk_used_bytes)} / ${formatBytes(resources.disk_total_bytes)}`}
            />
          </div>
          {!dashboard.metrics_available && (
            <p className="muted">
              {t("Metrics partially unavailable: ")}
              {dashboard.metrics_error ||
                t("metrics-server or node stats endpoint did not return data.")}
            </p>
          )}
        </div>
      </section>

      <section className="dashboard-grid">
        <div className="panel dashboard-panel">
          <h2>
            <AlertTriangle size={18} /> {t("Recent Alerts")}
          </h2>
          <AlertList rows={alerts} empty={t("No active alerts reported.")} />
        </div>
        <div className="panel dashboard-panel">
          <h2>
            <ListRestart size={18} /> {t("Events and Error Signals")}
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
                {t("No warning events in the latest cluster feed.")}
              </div>
            )}
          </div>
        </div>
      </section>
    </div>
  );
}

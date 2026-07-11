import React, { useEffect, useMemo, useRef, useState } from "react";
import * as LucideIcons from "lucide-react";
import { formatTime } from "../utils/index";
import { t } from "../i18n/index";
import { MetricCard, AlertList, Button } from "../components/index";
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
export default function AlertsView({ dashboard, refresh }) {
  if (!dashboard) {
    return (
      <section className="panel">
        <div className="empty">{t("Loading alerts...")}</div>
      </section>
    );
  }
  const alerts = dashboard.alerts || [];
  const critical = alerts.filter((row) =>
    ["critical", "error", "failed"].includes(
      String(row.severity || "").toLowerCase(),
    ),
  ).length;
  const warnings = alerts.length - critical;
  return (
    <div className="stack observability-page">
      <section className="panel action-panel">
        <div>
          <h2>
            <AlertTriangle size={18} /> {t("Alerts")}
          </h2>
          <p>
            {t(
              "Active cluster health signals generated from abnormal pods, warning events, and node readiness.",
            )}
          </p>
        </div>
        <Button onClick={refresh}>
          <RefreshCw size={15} /> {t("Refresh")}
        </Button>
      </section>
      <section className="dashboard-kpis">
        <MetricCard
          icon={AlertTriangle}
          label={t("Active")}
          value={alerts.length}
          detail={t("{critical} critical · {warnings} warning", {
            critical,
            warnings,
          })}
          tone={alerts.length > 0 ? "warning" : "good"}
        />
        <MetricCard
          icon={Server}
          label={t("Nodes")}
          value={`${dashboard.nodes?.ready || 0}/${dashboard.nodes?.total || 0}`}
          detail={t("{count} not ready", {
            count: dashboard.nodes?.not_ready || 0,
          })}
          tone={dashboard.nodes?.not_ready ? "warning" : "good"}
        />
        <MetricCard
          icon={Boxes}
          label={t("Pods")}
          value={dashboard.pods?.abnormal || 0}
          detail={t("{pending} pending · {failed} failed", {
            pending: dashboard.pods?.pending || 0,
            failed: dashboard.pods?.failed || 0,
          })}
          tone={dashboard.pods?.abnormal ? "warning" : "good"}
        />
        <MetricCard
          icon={Activity}
          label={t("Status")}
          value={dashboard.status || "-"}
          detail={t("Last check {time}", {
            time: formatTime(dashboard.checked_at),
          })}
          tone={dashboard.healthy ? "good" : "warning"}
        />
      </section>
      <section className="panel">
        <h2>
          <Shield size={18} /> {t("Alert feed")}
        </h2>
        <AlertList rows={alerts} empty={t("No active alerts reported.")} />
      </section>
    </div>
  );
}

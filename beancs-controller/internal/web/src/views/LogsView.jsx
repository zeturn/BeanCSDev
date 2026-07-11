import React, { useEffect, useMemo, useRef, useState } from "react";
import * as LucideIcons from "lucide-react";
import { MetricCard, Select, Button } from "../components/index";
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
export default function LogsView({
  projects,
  activeProjectID,
  setActiveProjectID,
  progress,
  refresh,
  logFollow,
  liveLogs,
  logStatus,
  onStartLogFollow,
  onStopLogFollow,
  onOpenPods,
}) {
  const logs = logFollow ? liveLogs : progress?.logs;
  return (
    <div className="stack observability-page">
      <section className="panel action-panel">
        <div>
          <h2>
            <ScrollText size={18} /> {t("Logs")}
          </h2>
          <p>
            {t(
              "Project container log snapshots and live follow without leaving the observability section.",
            )}
          </p>
        </div>
        <div className="progress-controls">
          <Select
            value={activeProjectID}
            onChange={(event) => setActiveProjectID(event.target.value)}
          >
            <option value="">{t("Choose project")}</option>
            {projects.map((project) => (
              <option key={project.id} value={project.id}>
                {project.display_name || project.name}
              </option>
            ))}
          </Select>
          <Button onClick={() => refresh()} disabled={logFollow}>
            <RefreshCw size={15} /> {t("Snapshot")}
          </Button>
          {logFollow ? (
            <Button onClick={onStopLogFollow}>{t("Stop follow")}</Button>
          ) : (
            <Button
              onClick={() => onStartLogFollow(activeProjectID)}
              disabled={!activeProjectID}
              variant="primary"
            >
              {t("Follow live")}
            </Button>
          )}
        </div>
      </section>
      <section className="dashboard-kpis">
        <MetricCard
          icon={Boxes}
          label={t("Project")}
          value={
            progress?.project?.display_name || progress?.project?.name || "-"
          }
          detail={progress?.project?.namespace || t("No project selected")}
        />
        <MetricCard
          icon={Layers3}
          label={t("Pods")}
          value={(progress?.pods || []).length}
          detail={t("{count} running", {
            count: (progress?.pods || []).filter((pod) => pod.status === "Running").length,
          })}
        />
        <MetricCard
          icon={GitBranch}
          label={t("Deployments")}
          value={(progress?.deployments || []).length}
          detail={
            (progress?.deployments || [])[0]?.status || t("No deployment events")
          }
        />
      </section>
      <section className="panel log-panel observability-log-panel">
        <div className="log-header">
          <h2>
            <Code2 size={18} /> {t("Container logs")}
          </h2>
          <div className="row-actions">
            <Button type="button" onClick={onOpenPods}>
              <Layers3 size={15} /> {t("Pod detail")}
            </Button>
          </div>
        </div>
        {logStatus && <p className="log-status">{logStatus}</p>}
        <pre>{logs || t("Choose a project to load recent logs.")}</pre>
      </section>
    </div>
  );
}

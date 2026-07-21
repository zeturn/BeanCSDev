import React, { useEffect, useMemo, useRef, useState } from "react";
import * as LucideIcons from "lucide-react";
import { formatTime } from "../utils/index";
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
export default function ProgressListView({
  processes,
  projects,
  onSelectProcess,
}) {
  return (
    <div className="stack progress-list-page">
      <section className="progress-list-panel">
        <div className="progress-list-head">
          <span>{t("Process")}</span>
          <span>{t("Project")}</span>
          <span>{t("Type")}</span>
          <span>{t("Status")}</span>
          <span />
        </div>
        {(processes || []).map((process) => (
          <button
            type="button"
            className="progress-list-row"
            key={process.id}
            onClick={() => onSelectProcess(process)}
          >
            <span>
              <b>
                #{process.id} {process.title || process.type}
              </b>
              <small>{formatTime(process.created_at)}</small>
            </span>
            <span>
              {process.project?.display_name ||
                process.project?.name ||
                (process.project_id ? `project #${process.project_id}` : "-")}
            </span>
            <span>{process.type || "-"}</span>
            <span>{process.status || "-"}</span>
            <span>{t("Open")}</span>
          </button>
        ))}
        {(processes || []).length === 0 && (
          <div className="empty">
            {(projects || []).length
              ? t("No deployment process records yet.")
              : t("No projects yet.")}
          </div>
        )}
      </section>
    </div>
  );
}

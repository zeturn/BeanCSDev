import React, { useEffect, useMemo, useRef, useState } from "react";
import * as LucideIcons from "lucide-react";
import { t } from "../i18n/index";
import { MetricCard } from "../components/index";
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
export default function SettingsView({ version }) {
  return (
    <div className="stack">
      <section className="panel">
        <h2>
          <Settings size={18} /> {t("Settings")}
        </h2>
        <div className="metric-row">
          <MetricCard
            icon={Rocket}
            label={t("Deployed version")}
            value={version || "-"}
            detail={t("Controller VERSION environment value")}
          />
        </div>
        <p className="muted">
          {t(
            "Authentication uses BasaltPass. Manage identity provider connections under",
          )}{" "}
          <b>{t("Security")} → {t("Access Control")}</b>.
        </p>
      </section>
    </div>
  );
}

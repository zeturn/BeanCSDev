import React, { useEffect, useMemo, useRef, useState } from "react";
import * as LucideIcons from "lucide-react";
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
          <Settings size={18} /> Settings
        </h2>
        <div className="metric-row">
          <MetricCard
            icon={Rocket}
            label="Deployed version"
            value={version || "-"}
            detail="Controller VERSION environment value"
          />
        </div>
        <p className="muted">
          Authentication uses BasaltPass. Manage identity provider connections
          under <b>Security → Access Control</b>.
        </p>
      </section>
    </div>
  );
}

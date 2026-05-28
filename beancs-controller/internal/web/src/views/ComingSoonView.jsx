import { Button } from "../components/ui";
import React, { useEffect, useMemo, useRef, useState } from "react";
import * as LucideIcons from "lucide-react";
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
import * as Utils from "../utils";
import * as API from "../api";
import * as Components from "../components";
export default function ComingSoonView({
  title,
  description,
  actionLabel,
  onAction,
}) {
  return (
    <div className="stack">
      <section className="panel">
        <h2>{title}</h2>
        <p className="muted">{description}</p>
        {actionLabel && onAction && (
          <div
            style={{
              marginTop: 14,
            }}
          >
            <Button type="button" onClick={onAction} variant="primary">
              {actionLabel}
            </Button>
          </div>
        )}
      </section>
    </div>
  );
}

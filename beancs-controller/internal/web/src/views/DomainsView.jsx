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
import {
  Button,
  Input,
  Select,
  Checkbox,
  Textarea,
  Modal,
  Drawer,
} from "../components";
export default function DomainsView({ domains }) {
  return (
    <section className="panel">
      <h2>
        <Globe2 size={18} /> Cloudflare domains
      </h2>
      <div className="domain-grid">
        {domains.map((domain) => (
          <div
            className="domain-tile"
            key={`${domain.credential_id}-${domain.zone_id}`}
          >
            <Globe2 size={20} />
            <div>
              <b>{domain.domain}</b>
              <span>{domain.credential}</span>
              <small>{domain.zone_id}</small>
            </div>
            <em>{domain.active ? "Active" : "Inactive"}</em>
          </div>
        ))}
        {domains.length === 0 && (
          <div className="empty">No Cloudflare domains linked yet.</div>
        )}
      </div>
    </section>
  );
}

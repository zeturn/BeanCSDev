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
import { Drawer, Input, Button } from "../components";
export default function CloudflareAccountDrawer({ onClose, onCreate }) {
  return (
    <Drawer
      onClose={onClose}
      title={
        <h2>
          <Cloud size={18} /> Add Cloudflare account
        </h2>
      }
      subtitle={
        <p>
          Use an API token with zone read access so BeanCS can cache available
          domains.
        </p>
      }
    >
      <form className="drawer-form" onSubmit={onCreate}>
        <label>
          Account name
          <Input
            name="name"
            placeholder="Production Cloudflare"
            required
            autoFocus
          />
        </label>
        <label>
          Account ID
          <Input
            name="account_id"
            placeholder="Optional, limits zone discovery to this account"
          />
        </label>
        <label>
          API token
          <Input
            name="api_token"
            type="password"
            placeholder="Cloudflare API token"
            required
            autoComplete="new-password"
          />
        </label>
        <div className="drawer-actions">
          <Button type="button" onClick={onClose}>
            Cancel
          </Button>
          <Button variant="primary" type="submit">
            <KeyRound size={15} /> Create link
          </Button>
        </div>
      </form>
    </Drawer>
  );
}

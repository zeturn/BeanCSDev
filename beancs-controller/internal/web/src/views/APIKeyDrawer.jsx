import React, { useEffect, useMemo, useRef, useState } from "react";
import {
  Button,
  Input,
  Checkbox,
  Select,
  Textarea,
  Drawer,
  Modal,
} from "../components";
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
export default function APIKeyDrawer({ presets, scopes, onClose, onCreate }) {
  const defaultPreset = presets[0]?.id || "";
  return (
    <Drawer
      onClose={onClose}
      className="api-key-drawer"
      title={
        <h2>
          <KeyRound size={18} /> Create API key
        </h2>
      }
      subtitle={
        <p>Choose a preset or select exact scopes for local automation.</p>
      }
    >
      <form className="drawer-form api-key-form" onSubmit={onCreate}>
        <label>
          Key name
          <Input name="name" placeholder="local beanctl" required autoFocus />
        </label>
        <label>
          Expires at
          <Input name="expires_at" type="datetime-local" />
        </label>
        <label>
          Permission preset
          <Select name="preset" defaultValue={defaultPreset}>
            <option value="">Custom scopes</option>
            {presets.map((preset) => (
              <option key={preset.id} value={preset.id}>
                {preset.label}
              </option>
            ))}
          </Select>
        </label>
        <div className="scope-picker">
          {scopes.map((scope) => (
            <label
              className="checkbox-row"
              key={scope.id}
              title={scope.description}
            >
              <Checkbox name="scopes" type="checkbox" value={scope.id} />
              <span>
                <b>{scope.label}</b>
                <small>{scope.id}</small>
              </span>
            </label>
          ))}
          {scopes.length === 0 && (
            <div className="empty">No scope options loaded.</div>
          )}
        </div>
        <div className="drawer-actions">
          <Button type="button" onClick={onClose}>
            Cancel
          </Button>
          <Button variant="primary" type="submit">
            <KeyRound size={15} /> Create key
          </Button>
        </div>
      </form>
    </Drawer>
  );
}

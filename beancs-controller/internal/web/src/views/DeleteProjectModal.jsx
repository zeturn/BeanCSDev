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
import { Modal, Button } from "../components/index";
export default function DeleteProjectModal({
  project,
  busy,
  onClose,
  onDelete,
}) {
  return (
    <Modal
      title={`Delete ${project.name}`}
      subtitle="This removes the project record, namespace, DNS records, and managed OAuth app where applicable."
      onClose={onClose}
    >
      <div className="delete-summary">
        <span>
          Namespace <b>{project.namespace}</b>
        </span>
        <span>
          Route <b>{project.domain || project.exposure_mode}</b>
        </span>
      </div>
      <div className="modal-actions">
        <Button type="button" onClick={onClose} disabled={busy}>
          Cancel
        </Button>
        <Button
          className="filled"
          type="button"
          onClick={onDelete}
          disabled={busy}
          variant="danger"
        >
          <Trash2 size={15} /> Delete
        </Button>
      </div>
    </Modal>
  );
}

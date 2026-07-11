import React, { useEffect, useMemo, useRef, useState } from "react";
import * as LucideIcons from "lucide-react";
import {
  formatDeploymentDate,
  normalizeDeploymentStatus,
  truncateMiddle,
} from "../utils/index";
import { MetricCard, Modal, Button } from "../components/index";
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
export default function ProjectTrackingModal({
  project,
  tracking,
  loading,
  onRefresh,
  onClose,
}) {
  const history = tracking?.history || [];
  const current = tracking?.running_deployment || tracking?.latest_deployment;
  return (
    <Modal className="wide-modal tracking-modal" onClose={onClose}>
      <div className="side-drawer-head">
        <div>
          <h2>
            <ScrollText size={18} /> {project.display_name || project.name}
          </h2>
          <p>
            {tracking?.github_repo ||
              project.github_repo ||
              tracking?.current_image ||
              project.image_reference ||
              t("Deployment tracking")}
          </p>
        </div>
        <Button variant="icon" type="button" onClick={onClose} title="Close">
          <X size={16} />
        </Button>
      </div>
      <div className="tracking-summary-grid">
        <MetricCard
          icon={Rocket}
          label={t("Current")}
          value={tracking?.current_version || current?.version || "-"}
          detail={tracking?.current_image || current?.image_ref || "-"}
          tone={current?.status === "failed" ? "warning" : "good"}
        />
        <MetricCard
          icon={Activity}
          label={t("Latest")}
          value={tracking?.latest_status || current?.status || "-"}
          detail={
            tracking?.latest_deployment
              ? formatDeploymentDate(tracking.latest_deployment.updated_at)
              : "-"
          }
        />
        <MetricCard
          icon={Box}
          label={t("History")}
          value={tracking?.summary?.total ?? history.length}
          detail={t("{failed} failed, {deploying} deploying", {
            failed: tracking?.summary?.failed || 0,
            deploying: tracking?.summary?.deploying || 0,
          })}
        />
      </div>
      <div className="modal-actions tracking-actions">
        <Button type="button" onClick={onRefresh} disabled={loading}>
          <RefreshCw size={15} /> {t("Refresh")}
        </Button>
      </div>
      <div className="tracking-history">
        {loading && <div className="empty">{t("Loading release history...")}</div>}
        {!loading &&
          history.map((item) => (
            <div className="tracking-row" key={item.id}>
              <span
                className={`deploy-state ${normalizeDeploymentStatus(item.status)}`}
              >
                <i />
                <b>{item.version || item.tag || t("Deployment {id}", { id: item.id })}</b>
                <small>
                  {t(item.status || "pending")}
                  {item.process_status
                    ? ` · ${t("process {status}", { status: item.process_status })}`
                    : ""}
                </small>
              </span>
              <span>
                <b>{truncateMiddle(item.image_ref || item.tag || "-", 54)}</b>
                <small>
                  {item.commit_sha
                    ? truncateMiddle(item.commit_sha, 18)
                    : t("No commit recorded")}
                </small>
              </span>
              <span>
                <b>{formatDeploymentDate(item.created_at)}</b>
                <small>{t(item.triggered_by || "system")}</small>
              </span>
              <span>
                {item.workflow_url ? (
                  <a href={item.workflow_url} target="_blank" rel="noreferrer">
                    {t("Workflow")}
                  </a>
                ) : (
                  <small>{t("No workflow link")}</small>
                )}
                {item.failure_reason && (
                  <small className="error-inline">{item.failure_reason}</small>
                )}
              </span>
            </div>
          ))}
        {!loading && history.length === 0 && (
          <div className="empty">
            {t("No release history recorded for this project.")}
          </div>
        )}
      </div>
    </Modal>
  );
}

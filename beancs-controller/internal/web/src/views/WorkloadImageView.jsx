import { Button } from "../components/index";
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
export default function WorkloadImageView({
  images,
  onRefresh,
  onOpenRegistry,
  onRefreshImage,
  onDeleteImage,
}) {
  return (
    <div className="stack">
      <section className="panel action-panel">
        <div>
          <h2>
            <ImageIcon size={18} /> {t("Image")}
          </h2>
          <p>
            {t(
              "Running workload images are visible on Pods and Deployments. Tracked registry tags and sync use",
            )}{" "}
            <b>Integrations → Image Registry</b>.
          </p>
        </div>
        <Button type="button" onClick={onRefresh}>
          <RefreshCw size={15} /> {t("Refresh")}
        </Button>
      </section>
      <section className="panel">
        <h2>
          <Package size={18} /> {t("Tracked image tags")}
        </h2>
        <p className="muted">
          {t(
            "Mirrors and tag lists you have registered. To add registries or repositories, open Image Registry.",
          )}
        </p>
        <div
          className="row-actions"
          style={{
            marginBottom: 12,
          }}
        >
          <Button type="button" onClick={onOpenRegistry} variant="primary">
            <Package size={15} /> {t("Open Image Registry")}
          </Button>
        </div>
        {(images || []).map((im) => (
          <div className="registry-image-card" key={im.id}>
            <div className="registry-image-head">
              <div>
                <div className="mono strong">{im.repository}</div>
                <small className="muted">
                  {im.registry?.name || `registry #${im.registry_id}`} ·{" "}
                  {formatTime(im.refreshed_at)}
                </small>
              </div>
              <div className="row-actions">
                <Button type="button" onClick={() => onRefreshImage(im.id)}>
                  <RefreshCw size={15} /> {t("Sync")}
                </Button>
                <Button
                  type="button"
                  onClick={() => onDeleteImage(im)}
                  variant="danger"
                >
                  <Trash2 size={15} /> {t("Remove")}
                </Button>
              </div>
            </div>
            <div className="tag-chip-grid">
              {(im.tags || []).slice(0, 120).map((tg) => (
                <span className="tag-chip" key={tg}>
                  {tg}
                </span>
              ))}
              {(im.tags || []).length > 120 && (
                <span className="muted">
                  … {t("{count} tags", { count: (im.tags || []).length })}
                </span>
              )}
              {(im.tags || []).length === 0 && (
                <span className="muted">{t("No tags cached yet.")}</span>
              )}
            </div>
          </div>
        ))}
        {(images || []).length === 0 && (
          <div className="empty">
            {t("No tracked images. Configure mirrors under Image Registry.")}
          </div>
        )}
      </section>
    </div>
  );
}

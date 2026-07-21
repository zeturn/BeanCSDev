import React, { useEffect, useMemo, useRef, useState } from "react";
import * as LucideIcons from "lucide-react";
import { t } from "../i18n/index";
import { ExpandableCell, Button, Input } from "../components/index";
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
export default function ProjectsView({
  projects,
  onOpen,
  onEdit,
  onDelete,
  onScale,
  onRestart,
  onBuild,
  onTracking,
  onProgress,
}) {
  const [projectSearch, setProjectSearch] = useState("");
  const visibleProjects = useMemo(() => {
    const needle = String(projectSearch || "")
      .trim()
      .toLowerCase();
    if (!needle) return projects;
    return projects.filter((project) => {
      const haystack = [
        project.display_name,
        project.name,
        project.github_repo,
        project.image_reference,
        project.source_archive_name,
        project.build_source,
        project.domain,
        project.exposure_mode,
        project.status,
      ]
        .filter(Boolean)
        .join(" ")
        .toLowerCase();
      return haystack.includes(needle);
    });
  }, [projects, projectSearch]);
  return (
    <div className="stack">
      <section className="panel">
        <div className="project-search">
          <Search size={18} />
          <Input
            value={projectSearch}
            onChange={(event) => setProjectSearch(event.target.value)}
            placeholder={t("Search projects")}
          />
        </div>
        <div className="table">
          <div className="tr head project-table-row">
            <span>{t("Name")}</span>
            <span>{t("Repo")}</span>
            <span>{t("Route")}</span>
            <span>{t("Deps")}</span>
            <span>{t("Status")}</span>
            <span>{t("Actions")}</span>
          </div>
          {visibleProjects.map((project) => (
            <div className="tr project-table-row" key={project.id}>
              <button
                type="button"
                className="project-name-link strong"
                onClick={() => onOpen?.(project)}
              >
                {project.display_name || project.name}
              </button>
              <ExpandableCell
                value={
                  project.github_repo ||
                  project.image_reference ||
                  project.source_archive_name ||
                  project.build_source
                }
                max={42}
              />
              <ExpandableCell
                value={project.domain || project.exposure_mode}
                max={36}
              />
              <ExpandableCell
                value={(project.depends_on || []).join(", ") || "-"}
                max={24}
              />
              <ExpandableCell value={project.status} max={24} />
              <span className="row-actions">
                <Button
                  onClick={() => onTracking(project)}
                  title={t("Release history")}
                >
                  <ScrollText size={15} /> {t("History")}
                </Button>
                <Button
                  onClick={() => onProgress(project)}
                  title={t("Progress")}
                >
                  <LoaderCircle size={15} /> {t("Progress")}
                </Button>
                <Button onClick={() => onBuild(project)} title={t("Rebuild")}>
                  <Play size={15} /> {t("Rebuild")}
                </Button>
                <Button onClick={() => onRestart(project)} title={t("Restart")}>
                  <RotateCcw size={15} />
                </Button>
                <Button onClick={() => onEdit(project)} title={t("Edit")}>
                  <Edit3 size={15} />
                </Button>
                <Button
                  onClick={() => onDelete(project)}
                  title={t("Delete")}
                  variant="danger"
                >
                  <Trash2 size={15} /> {t("Delete")}
                </Button>
              </span>
            </div>
          ))}
          {visibleProjects.length === 0 && (
            <div className="empty">
              {projectSearch
                ? t("No projects match this search.")
                : t("No projects yet.")}
            </div>
          )}
        </div>
      </section>
    </div>
  );
}

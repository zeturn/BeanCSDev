import React from "react";
import {
  Boxes,
  Edit3,
  GitBranch,
  Globe2,
  Layers3,
  LoaderCircle,
  Play,
  RotateCcw,
  ScrollText,
  Server,
  Trash2,
} from "lucide-react";
import { Button } from "../components/index";
import { t } from "../i18n/index";

function valueOrDash(value) {
  if (Array.isArray(value)) return value.length ? value.join(", ") : "-";
  return value || "-";
}

function InfoField({ label, value }) {
  return (
    <div className="project-detail-field">
      <small>{label}</small>
      <b>{valueOrDash(value)}</b>
    </div>
  );
}

export default function ProjectDetailView({
  project,
  projectID,
  projects,
  onEdit,
  onDelete,
  onRestart,
  onBuild,
  onTracking,
  onProgress,
}) {
  const hasLoaded = (projects || []).length > 0;
  if (!project) {
    return (
      <section className="panel">
        <div className="empty">
          {hasLoaded
            ? t("Project {id} was not found.", { id: projectID })
            : t("Loading project...")}
        </div>
      </section>
    );
  }

  const repo =
    project.github_repo ||
    project.image_reference ||
    project.source_archive_name ||
    project.build_source ||
    "-";
  const route = project.domain || project.exposure_mode || "-";
  const ports = (project.ports || [])
    .map((port) =>
      typeof port === "string"
        ? port
        : [port.name, port.port || port.container_port, port.protocol]
            .filter(Boolean)
            .join(" "),
    )
    .filter(Boolean);

  return (
    <div className="stack project-detail-page">
      <section className="panel project-detail-hero">
        <div className="section-head">
          <div>
            <h2>
              <Boxes size={18} /> {project.display_name || project.name}
            </h2>
            <p className="muted">
              {project.namespace || "-"} · {project.status || "-"}
            </p>
          </div>
          <span className="row-actions">
            <Button type="button" onClick={() => onTracking?.(project)}>
              <ScrollText size={15} /> {t("History")}
            </Button>
            <Button type="button" onClick={() => onProgress?.(project)}>
              <LoaderCircle size={15} /> {t("Progress")}
            </Button>
            <Button type="button" onClick={() => onEdit?.(project)}>
              <Edit3 size={15} /> {t("Edit")}
            </Button>
            <Button
              type="button"
              variant="danger"
              onClick={() => onDelete?.(project)}
            >
              <Trash2 size={15} /> {t("Delete")}
            </Button>
          </span>
        </div>
        <div className="project-detail-kpis">
          <InfoField label={t("Status")} value={project.status} />
          <InfoField label={t("Namespace")} value={project.namespace} />
          <InfoField label={t("Replicas")} value={project.replicas ?? "-"} />
          <InfoField label={t("Route")} value={route} />
        </div>
      </section>

      <section className="project-detail-grid">
        <div className="panel">
          <div className="section-head">
            <h2>
              <GitBranch size={16} /> {t("Source")}
            </h2>
          </div>
          <div className="detail-list project-detail-list">
            <span>
              {t("Repo")} <b>{repo}</b>
            </span>
            <span>
              {t("Branch")} <b>{project.github_branch || "-"}</b>
            </span>
            <span>
              {t("Build source")} <b>{project.build_source || "-"}</b>
            </span>
            <span>
              {t("Dockerfile")} <b>{project.dockerfile || "-"}</b>
            </span>
            <span>
              {t("Build context")} <b>{project.buildContext || "-"}</b>
            </span>
          </div>
        </div>

        <div className="panel">
          <div className="section-head">
            <h2>
              <Globe2 size={16} /> {t("Runtime")}
            </h2>
          </div>
          <div className="detail-list project-detail-list">
            <span>
              {t("Image")} <b>{project.image_reference || "-"}</b>
            </span>
            <span>
              {t("Domain")} <b>{project.domain || "-"}</b>
            </span>
            <span>
              {t("Exposure")} <b>{project.exposure_mode || "-"}</b>
            </span>
            <span>
              {t("Ports")} <b>{valueOrDash(ports)}</b>
            </span>
          </div>
        </div>

        <div className="panel">
          <div className="section-head">
            <h2>
              <Layers3 size={16} /> {t("Application")}
            </h2>
          </div>
          <div className="detail-list project-detail-list">
            <span>
              {t("Component")} <b>{project.component_name || project.component || "-"}</b>
            </span>
            <span>
              {t("Kind")} <b>{project.kind || "-"}</b>
            </span>
            <span>
              {t("Dependencies")} <b>{valueOrDash(project.depends_on)}</b>
            </span>
            <span>
              {t("Auto deploy")} <b>{project.auto_deploy ? t("Enabled") : t("Disabled")}</b>
            </span>
          </div>
        </div>

        <div className="panel">
          <div className="section-head">
            <h2>
              <Server size={16} /> {t("Operations")}
            </h2>
          </div>
          <div className="project-detail-actions">
            <Button type="button" onClick={() => onBuild?.(project)}>
              <Play size={15} /> {t("Rebuild")}
            </Button>
            <Button type="button" onClick={() => onRestart?.(project)}>
              <RotateCcw size={15} /> {t("Restart")}
            </Button>
          </div>
        </div>
      </section>
    </div>
  );
}

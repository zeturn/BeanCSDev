import React from "react";
import {
  Box,
  Database,
  GitBranch,
  Globe2,
  Layers3,
} from "lucide-react";
import { t } from "../i18n/index";
import { Button } from "../components/index";

export default function ApplicationsView({
  applications,
  application,
  applicationID,
  mode = "list",
  onBack,
  onDeleteApplication,
  onOpenApplication,
  onOpenProject,
}) {
  if (mode === "detail") {
    return (
      <ApplicationDetailView
        application={application}
        applicationID={applicationID}
        applications={applications}
        onBack={onBack}
        onDeleteApplication={onDeleteApplication}
        onOpenProject={onOpenProject}
      />
    );
  }

  return (
    <div className="stack">
      <div className="list-meta">
        <span className="muted">
          {t("{count} total", { count: (applications || []).length })}
        </span>
      </div>
      <div className="application-grid">
        {(applications || []).map((application) => (
          <div
            className="application-card application-card-button"
            key={application.id}
            role="button"
            tabIndex={0}
            onClick={() => onOpenApplication?.(application)}
            onKeyDown={(event) => {
              if (event.key === "Enter" || event.key === " ") {
                event.preventDefault();
                onOpenApplication?.(application);
              }
            }}
          >
            <div className="application-card-title">
              <div>
                <b>{application.display_name || application.name}</b>
                <small>{application.github_repo || application.type}</small>
              </div>
              <span className="status-chip">{application.status || "-"}</span>
            </div>
            <div className="signal-list">
              <span>
                {t("{count} projects", {
                  count: (application.projects || []).length,
                })}
              </span>
            </div>
          </div>
        ))}
        {(applications || []).length === 0 && (
          <div className="empty">{t("No applications yet.")}</div>
        )}
      </div>
    </div>
  );
}

function ApplicationDetailView({
  application,
  applicationID,
  applications,
  onBack,
  onDeleteApplication,
  onOpenProject,
}) {
  const hasLoaded = (applications || []).length > 0;
  if (!application) {
    return (
      <div className="stack">
        <section className="panel">
          <div className="section-head">
            <h2>
              <Layers3 size={16} /> {t("Application")}
            </h2>
            <Button type="button" onClick={onBack}>
              {t("Applications")}
            </Button>
          </div>
          <div className="empty">
            {hasLoaded
              ? t("Application {id} was not found.", { id: applicationID })
              : t("Loading application...")}
          </div>
        </section>
      </div>
    );
  }

  const projects = application.projects || [];
  const dependencies = application.dependencies || [];
  const components = application.components || [];

  return (
    <div className="stack">
      <section className="panel application-detail-header">
        <div className="section-head">
          <div>
            <h2>
              <Layers3 size={16} /> {application.display_name || application.name}
            </h2>
            <p className="muted">
              {application.github_repo || application.type || "-"}
              {application.github_branch ? ` · ${application.github_branch}` : ""}
            </p>
          </div>
          <span className="row-actions">
            <Button
              type="button"
              onClick={() => onDeleteApplication(application)}
              title={t("Delete application")}
              variant="danger"
            >
              <Trash2 size={15} /> {t("Delete")}
            </Button>
          </span>
        </div>
        <div className="application-detail-meta">
          <DetailStat label={t("Status")} value={application.status || "-"} />
          <DetailStat label={t("Namespace")} value={application.namespace || "-"} />
          <DetailStat label={t("Projects")} value={String(projects.length)} />
        </div>
      </section>

      <section className="panel">
        <div className="section-head">
          <h2>
            <Box size={16} /> {t("Projects")}
          </h2>
          <span className="muted">{t("{count} total", { count: projects.length })}</span>
        </div>
        <div className="table">
          <div className="tr head application-detail-project-row">
            <span>{t("Name")}</span>
            <span>{t("Component")}</span>
            <span>{t("Route")}</span>
            <span>{t("Namespace")}</span>
            <span>{t("Status")}</span>
          </div>
          {projects.map((project) => (
            <button
              className="tr application-detail-project-row application-detail-row-button"
              key={project.id}
              type="button"
              onClick={() => onOpenProject?.(project)}
            >
              <span className="strong">{project.display_name || project.name}</span>
              <span>{project.component_name || "-"}</span>
              <span>
                <Globe2 size={14} /> {project.domain || project.exposure_mode || "-"}
              </span>
              <span>{project.namespace || "-"}</span>
              <span>{project.status || "-"}</span>
            </button>
          ))}
          {projects.length === 0 && (
            <div className="empty">{t("No projects attached to this application.")}</div>
          )}
        </div>
      </section>

      <section className="panel">
        <div className="section-head">
          <h2>
            <Database size={16} /> {t("Dependencies")}
          </h2>
          <span className="muted">
            {t("{count} total", { count: dependencies.length })}
          </span>
        </div>
        <div className="table">
          <div className="tr head application-detail-dependency-row">
            <span>{t("Name")}</span>
            <span>{t("Type")}</span>
            <span>{t("Namespace")}</span>
            <span>{t("Method")}</span>
            <span>{t("Status")}</span>
          </div>
          {dependencies.map((dependency) => (
            <div className="tr application-detail-dependency-row" key={dependency.id}>
              <span className="strong">{dependency.name}</span>
              <span>{dependency.type || "-"}</span>
              <span>{dependency.namespace || "-"}</span>
              <span>{dependency.deploy_method || "-"}</span>
              <span>{dependency.status || "-"}</span>
            </div>
          ))}
          {dependencies.length === 0 && (
            <div className="empty">
              {t("No dependencies attached to this application.")}
            </div>
          )}
        </div>
      </section>

      {components.length > 0 && (
        <section className="panel">
          <div className="section-head">
            <h2>
              <GitBranch size={16} /> {t("Components")}
            </h2>
            <span className="muted">
              {t("{count} total", { count: components.length })}
            </span>
          </div>
          <div className="application-component-grid">
            {components.map((component) => (
              <div className="component-card" key={component.id || component.name}>
                <div className="component-card-head">
                  <div>
                    <b>{component.name}</b>
                    <small>{component.kind || "-"}</small>
                  </div>
                  <span className="status-chip">{component.status || "-"}</span>
                </div>
                <div className="signal-list">
                  <span>{component.depends_on?.join(", ") || t("No dependencies")}</span>
                </div>
              </div>
            ))}
          </div>
        </section>
      )}
    </div>
  );
}

function DetailStat({ label, value }) {
  return (
    <div className="application-detail-stat">
      <small>{label}</small>
      <b>{value}</b>
    </div>
  );
}

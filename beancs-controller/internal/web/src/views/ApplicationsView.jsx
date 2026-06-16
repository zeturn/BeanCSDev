import React from "react";
import { Layers3, Trash2 } from "lucide-react";
import { Button } from "../components/index";

export default function ApplicationsView({ applications, onDeleteApplication }) {
  return (
    <div className="stack">
      <section className="panel">
        <div className="section-head">
          <h2>
            <Layers3 size={16} /> Applications
          </h2>
          <span className="muted">{(applications || []).length} total</span>
        </div>
        <div className="application-grid">
          {(applications || []).map((application) => (
            <div className="application-card" key={application.id}>
              <div className="component-card-head">
                <div>
                  <b>{application.display_name || application.name}</b>
                  <small>{application.github_repo || application.type}</small>
                </div>
                <Button
                  type="button"
                  onClick={() => onDeleteApplication(application)}
                  title="Delete application"
                  variant="danger"
                >
                  <Trash2 size={15} /> Delete
                </Button>
              </div>
              <span className="status-chip">{application.status || "-"}</span>
              <div className="signal-list">
                <span>{(application.projects || []).length} projects</span>
                <span>{(application.dependencies || []).length} dependencies</span>
              </div>
              {(application.dependencies || []).length > 0 && (
                <div className="dependency-summary-list">
                  {(application.dependencies || []).map((dependency) => (
                    <span key={dependency.id}>
                      {dependency.name}
                      <small>
                        {dependency.type} · {dependency.status}
                      </small>
                    </span>
                  ))}
                </div>
              )}
            </div>
          ))}
          {(applications || []).length === 0 && (
            <div className="empty">No applications yet.</div>
          )}
        </div>
      </section>
    </div>
  );
}

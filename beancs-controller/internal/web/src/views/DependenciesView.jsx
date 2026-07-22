import React, { useEffect, useState } from "react";
import { Database, KeyRound, ShieldCheck } from "lucide-react";
import {
  Button,
  Input,
  Modal,
  PaginationBar,
} from "../components/index";
import { t } from "../i18n/index";

const databaseTypes = new Set(["mysql", "postgresql", "timescaledb"]);

function displayName(definitions, type) {
  const definition = (definitions || []).find((item) => item.name === type);
  return definition?.display_name || type;
}

export default function DependenciesView({
  definitions,
  dependencies,
  onCreateCredential,
}) {
  const [page, setPage] = useState(1);
  const pageSize = 8;
  const totalPages = Math.max(1, Math.ceil((dependencies || []).length / pageSize));
  const safePage = Math.min(page, totalPages);
  const pagedDependencies = (dependencies || []).slice(
    (safePage - 1) * pageSize,
    safePage * pageSize,
  );
  useEffect(() => {
    setPage(1);
  }, [(dependencies || []).length]);

  return (
    <div className="dependencies-page">
      <section className="dependency-entity-list">
        {pagedDependencies.map((dependency) => (
          <DependencyEntity
            key={dependency.id}
            dependency={dependency}
            typeLabel={displayName(definitions, dependency.type)}
            onCreateCredential={onCreateCredential}
          />
        ))}
        {(dependencies || []).length === 0 && (
          <div className="empty">{t("No reusable dependencies registered.")}</div>
        )}
        <PaginationBar
          page={safePage}
          pageSize={pageSize}
          total={(dependencies || []).length}
          onPageChange={setPage}
          label="dependencies"
        />
      </section>
    </div>
  );
}

function DependencyEntity({ dependency, typeLabel, onCreateCredential }) {
  const showDatabase = databaseTypes.has(dependency.type);
  const [credentialOpen, setCredentialOpen] = useState(false);
  const [credPage, setCredPage] = useState(1);
  const credSize = 6;
  const credentials = dependency.credentials || [];
  const totalPages = Math.max(1, Math.ceil(credentials.length / credSize));
  const safePage = Math.min(credPage, totalPages);
  const pagedCredentials = credentials.slice(
    (safePage - 1) * credSize,
    safePage * credSize,
  );
  useEffect(() => {
    setCredPage(1);
  }, [credentials.length]);
  return (
    <article className="dependency-entity">
      <div className="dependency-entity-head">
        <div>
          <h2>
            <Database size={18} /> {dependency.name}
          </h2>
          <p className="muted">
            {typeLabel} · {dependency.config?.host || dependency.service_name}
            {dependency.config?.port ? `:${dependency.config.port}` : ""}
          </p>
        </div>
        <div className="dependency-badges">
          <span>{t(dependency.external ? "external" : "managed")}</span>
          <span>{t(dependency.controlled ? "controlled" : "uncontrolled")}</span>
          <span>{t(dependency.status || "ready")}</span>
        </div>
      </div>
      <div className="dependency-credential-list">
        {pagedCredentials.map((credential) => (
          <div className="dependency-credential-row" key={credential.id}>
            <KeyRound size={15} />
            <span>{credential.name}</span>
            <small>{credential.config?.username || "-"}</small>
            <small>{t(credential.status || "ready")}</small>
          </div>
        ))}
        {credentials.length === 0 && (
          <div className="empty compact">{t("No credentials yet.")}</div>
        )}
        <PaginationBar
          page={safePage}
          pageSize={credSize}
          total={credentials.length}
          onPageChange={setCredPage}
          label="credentials"
        />
      </div>
      <div className="row-actions">
        <Button type="button" onClick={() => setCredentialOpen(true)}>
          <ShieldCheck size={15} /> {t("Add credential")}
        </Button>
      </div>
      {credentialOpen && (
        <Modal
          title={t("Add credential for {name}", { name: dependency.name })}
          subtitle={t("Creation and viewing are separated to reduce crowding.")}
          onClose={() => setCredentialOpen(false)}
        >
          <form
            className="component-grid dependency-credential-form"
            onSubmit={async (event) => {
              const ok = await onCreateCredential(dependency.id, event);
              if (ok) setCredentialOpen(false);
            }}
          >
            <label>{t("Credential name")}</label>
            <Input name="name" required placeholder="app" />
            {showDatabase && (
              <>
                <label>{t("Database")}</label>
                <Input name="database" required placeholder="app" />
              </>
            )}
            <label>{t("Username")}</label>
            <Input name="username" required placeholder="app" />
            <label>{t("Password")}</label>
            <Input name="password" type="password" required />
            <label>{t("Description")}</label>
            <Input name="description" placeholder={t("Used by California Beans")} />
            <div />
            <div className="modal-actions">
              <Button type="button" onClick={() => setCredentialOpen(false)}>
                {t("Cancel")}
              </Button>
              <Button type="submit">
                <ShieldCheck size={15} /> {t("Add credential")}
              </Button>
            </div>
          </form>
        </Modal>
      )}
    </article>
  );
}

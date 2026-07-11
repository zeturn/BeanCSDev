import React, { useEffect, useMemo, useState } from "react";
import { Database, KeyRound, ShieldCheck } from "lucide-react";
import {
  Button,
  Checkbox,
  DependencyConfigEditor,
  Input,
  Modal,
  PaginationBar,
  Select,
} from "../components/index";
import { dependencyDefaultConfig } from "../utils/index";
import { t } from "../i18n/index";

const defaultPorts = {
  mysql: "3306",
  postgresql: "5432",
  timescaledb: "5432",
  rabbitmq: "5672",
  redis: "6379",
};

const databaseTypes = new Set(["mysql", "postgresql", "timescaledb"]);
const controlledTypes = new Set(["mysql", "postgresql", "timescaledb", "rabbitmq"]);

function displayName(definitions, type) {
  const definition = (definitions || []).find((item) => item.name === type);
  return definition?.display_name || type;
}

export default function DependenciesView({
  definitions,
  dependencies,
  githubCredentials,
  onCreateDependency,
  onCreateCredential,
}) {
  const clusterDefinitions = useMemo(
    () =>
      (definitions || []).filter((definition) =>
        (definition.supported_deploy_methods || []).some(
          (method) => method !== "external",
        ),
      ),
    [definitions],
  );
  const externalDefinitions = useMemo(
    () =>
      (definitions || []).filter((definition) =>
        (definition.supported_deploy_methods || []).includes("external"),
      ),
    [definitions],
  );
  const [mode, setMode] = useState("cluster");
  const activeDefinitions =
    mode === "external" ? externalDefinitions : clusterDefinitions;
  const [type, setType] = useState(activeDefinitions[0]?.name || "mysql");
  const activeDefinition =
    activeDefinitions.find((definition) => definition.name === type) ||
    activeDefinitions[0];
  const deployMethods = (activeDefinition?.supported_deploy_methods || []).filter(
    (method) => (mode === "external" ? method === "external" : method !== "external"),
  );
  const defaultDeployMethod =
    deployMethods.includes(activeDefinition?.default_deploy_method)
      ? activeDefinition.default_deploy_method
      : deployMethods[0] || (mode === "external" ? "external" : "helm");
  const [deployMethod, setDeployMethod] = useState(defaultDeployMethod);
  const [config, setConfig] = useState(() =>
    dependencyDefaultConfig(activeDefinition),
  );
  const [controlled, setControlled] = useState(true);
  const [createOpen, setCreateOpen] = useState(false);
  const [page, setPage] = useState(1);
  const pageSize = 8;
  const totalPages = Math.max(1, Math.ceil((dependencies || []).length / pageSize));
  const safePage = Math.min(page, totalPages);
  const pagedDependencies = (dependencies || []).slice(
    (safePage - 1) * pageSize,
    safePage * pageSize,
  );
  const activeType = type || activeDefinition?.name || "mysql";
  const controlledSupported = controlledTypes.has(activeType);

  useEffect(() => {
    const next = activeDefinitions[0]?.name || "mysql";
    setType(next);
  }, [mode, activeDefinitions]);

  useEffect(() => {
    setDeployMethod(defaultDeployMethod);
    setConfig(dependencyDefaultConfig(activeDefinition));
  }, [activeDefinition, defaultDeployMethod]);
  useEffect(() => {
    setPage(1);
  }, [(dependencies || []).length]);

  return (
    <div className="dependencies-page">
      <div className="dependencies-toolbar">
        <Button type="button" variant="primary" onClick={() => setCreateOpen(true)}>
          <Database size={15} /> {t("Create dependency")}
        </Button>
      </div>

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
      {createOpen && (
        <Modal
          title={t("Create dependency")}
          subtitle={t(
            "Create the flow in a dialog to avoid overloading the list page.",
          )}
          className="wide-modal"
          onClose={() => setCreateOpen(false)}
        >
          <form
            className="component-grid dependency-create-form"
            onSubmit={async (event) => {
              const ok = await onCreateDependency(event);
              if (ok) setCreateOpen(false);
            }}
          >
            <label>{t("Location")}</label>
            <Select
              name="location"
              value={mode}
              onChange={(event) => setMode(event.target.value)}
            >
              <option value="cluster">{t("BeanCS cluster")}</option>
              <option value="external">{t("External service")}</option>
            </Select>
            <input
              type="hidden"
              name="external"
              value={mode === "external" ? "true" : "false"}
            />
            <input
              type="hidden"
              name="config_json"
              value={JSON.stringify(config || {})}
            />
            <label>{t("Name")}</label>
            <Input name="name" required placeholder="mysql-prod" />
            <label>{t("Display name")}</label>
            <Input name="display_name" placeholder={t("Production MySQL")} />
            <label>{t("Type")}</label>
            <Select
              name="type"
              value={activeType}
              onChange={(event) => {
                setType(event.target.value);
                if (!controlledTypes.has(event.target.value)) setControlled(false);
              }}
            >
              {activeDefinitions.map((definition) => (
                <option key={definition.name} value={definition.name}>
                  {definition.display_name || definition.name}
                </option>
              ))}
            </Select>
            {mode === "cluster" && (
              <>
                <label>{t("Deploy method")}</label>
                <Select
                  name="deploy_method"
                  value={deployMethod}
                  onChange={(event) => setDeployMethod(event.target.value)}
                >
                  {deployMethods.map((method) => (
                    <option key={method} value={method}>
                      {t(method)}
                    </option>
                  ))}
                </Select>
                <label>{t("Version")}</label>
                <Input name="version" placeholder="default" />
                <label>{t("GitOps credential")}</label>
                <Select name="github_credential_id">
                  <option value="">{t("Create record only")}</option>
                  {(githubCredentials || []).map((credential) => (
                    <option key={credential.id} value={credential.id}>
                      {credential.name}
                      {credential.gitops_repo ? ` · ${credential.gitops_repo}` : ""}
                    </option>
                  ))}
                </Select>
              </>
            )}
            {mode === "external" && (
              <>
                <input type="hidden" name="deploy_method" value="external" />
                <label>{t("Host")}</label>
                <Input name="host" required placeholder="10.0.0.20" />
                <label>{t("Port")}</label>
                <Input
                  name="port"
                  defaultValue={defaultPorts[activeType] || ""}
                  placeholder={defaultPorts[activeType] || ""}
                />
                {activeType === "rabbitmq" && (
                  <>
                    <label>{t("Management port")}</label>
                    <Input
                      name="management_port"
                      defaultValue="15672"
                      placeholder="15672"
                    />
                  </>
                )}
              </>
            )}
            <label>{t("App object")}</label>
            <Input name="application_name" placeholder={`dep-${activeType}`} />
            <label>{t("Namespace")}</label>
            <Input name="namespace" placeholder={`dep-${activeType}`} />
            {mode === "cluster" && (
              <>
                <div />
                <Checkbox name="shared" defaultChecked label={t("Reusable by other apps")} />
                <DependencyConfigEditor
                  definition={activeDefinition}
                  value={config || {}}
                  onChange={setConfig}
                />
              </>
            )}
            {mode === "external" && (
              <>
                <div />
                <Checkbox
                  name="controlled"
                  checked={controlledSupported && controlled}
                  disabled={!controlledSupported}
                  onChange={(event) => setControlled(event.target.checked)}
                  label={t("BeanCS can create credentials")}
                />
                {!controlledSupported && (
                  <>
                    <div />
                    <p className="muted">{t("Credentials for this type must be entered manually.")}</p>
                  </>
                )}
                {controlledSupported && controlled && (
                  <>
                    <label>{t("Admin username")}</label>
                    <Input name="admin_username" required placeholder="root" />
                    <label>{t("Admin password")}</label>
                    <Input name="admin_password" type="password" required />
                  </>
                )}
              </>
            )}
            <div />
            <div className="modal-actions">
              <Button type="button" onClick={() => setCreateOpen(false)}>
                {t("Cancel")}
              </Button>
              <Button type="submit" variant="primary">
                <Database size={15} /> {t("Add dependency")}
              </Button>
            </div>
          </form>
        </Modal>
      )}
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

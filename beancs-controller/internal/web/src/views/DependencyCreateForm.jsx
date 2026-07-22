import React, { useEffect, useMemo, useState } from "react";
import { Database, GitBranch, Globe2, Layers3, Server } from "lucide-react";
import {
  Button,
  Checkbox,
  DependencyConfigEditor,
  Input,
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

const controlledTypes = new Set([
  "mysql",
  "postgresql",
  "timescaledb",
  "rabbitmq",
]);

export default function DependencyCreateForm({
  definitions,
  githubCredentials,
  onSubmit,
  onCancel,
  requireGitOpsCredential = false,
  submitLabel = "Add dependency",
  embedded = false,
  showActions = true,
  onDraftChange,
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
    (method) =>
      mode === "external" ? method === "external" : method !== "external",
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
  const [name, setName] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [applicationName, setApplicationName] = useState("");
  const [namespace, setNamespace] = useState("");
  const [version, setVersion] = useState("");
  const [githubCredentialID, setGithubCredentialID] = useState("");
  const [host, setHost] = useState("");
  const [port, setPort] = useState("");
  const [managementPort, setManagementPort] = useState("15672");
  const [adminUsername, setAdminUsername] = useState("");
  const [adminPassword, setAdminPassword] = useState("");
  const activeType = type || activeDefinition?.name || "mysql";
  const controlledSupported = controlledTypes.has(activeType);

  useEffect(() => {
    setType(activeDefinitions[0]?.name || "mysql");
  }, [mode, activeDefinitions]);

  useEffect(() => {
    setDeployMethod(defaultDeployMethod);
    setConfig(dependencyDefaultConfig(activeDefinition));
  }, [activeDefinition, defaultDeployMethod]);

  useEffect(() => {
    setPort(defaultPorts[activeType] || "");
  }, [activeType, mode]);

  useEffect(() => {
    onDraftChange?.({
      mode,
      type: activeType,
      typeLabel: activeDefinition?.display_name || activeType,
      deployMethod,
      name,
      displayName,
      applicationName,
      namespace,
      version,
      githubCredentialID,
      githubCredentialName:
        (githubCredentials || []).find(
          (credential) => String(credential.id) === String(githubCredentialID),
        )?.name || "",
      host,
      port,
      managementPort,
      controlled: controlledSupported && controlled,
      shared: mode === "cluster",
      config,
    });
  }, [
    activeDefinition,
    activeType,
    applicationName,
    config,
    controlled,
    controlledSupported,
    deployMethod,
    displayName,
    githubCredentialID,
    githubCredentials,
    host,
    managementPort,
    mode,
    name,
    namespace,
    onDraftChange,
    port,
    version,
  ]);

  const Shell = embedded ? "div" : "form";
  const shellProps = embedded
    ? {}
    : {
        onSubmit: async (event) => {
          const ok = await onSubmit(event);
          if (ok) onCancel?.();
        },
      };

  return (
    <Shell className="dependency-create-form" {...shellProps}>
      <section className="dependency-form-section">
        <h3>
          <Server size={16} /> {t("Deployment target")}
        </h3>
        <div className="dependency-choice-grid">
          <Button
            type="button"
            className={mode === "cluster" ? "dependency-choice active" : "dependency-choice"}
            onClick={() => setMode("cluster")}
          >
            <Database size={18} />
            <b>{t("BeanCS cluster")}</b>
            <span>{t("Managed dependency")}</span>
          </Button>
          <Button
            type="button"
            className={mode === "external" ? "dependency-choice active" : "dependency-choice"}
            onClick={() => setMode("external")}
          >
            <Globe2 size={18} />
            <b>{t("External service")}</b>
            <span>{t("Existing endpoint")}</span>
          </Button>
        </div>
      </section>
      <input
        type="hidden"
        name="location"
        value={mode}
      />
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

      <section className="dependency-form-section">
        <h3>
          <Layers3 size={16} /> {t("Dependency identity")}
        </h3>
        <div className="dependency-form-fields">
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
          <label>{t("Name")}</label>
          <Input
            name="name"
            required
            value={name}
            onChange={(event) => setName(event.target.value)}
            placeholder="mysql-prod"
          />
          <label>{t("Display name")}</label>
          <Input
            name="display_name"
            value={displayName}
            onChange={(event) => setDisplayName(event.target.value)}
            placeholder={t("Production MySQL")}
          />
        </div>
      </section>

      {mode === "cluster" && (
        <section className="dependency-form-section">
          <h3>
            <GitBranch size={16} /> {t("GitOps rollout")}
          </h3>
          <div className="dependency-form-fields">
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
            <Input
              name="version"
              value={version}
              onChange={(event) => setVersion(event.target.value)}
              placeholder="default"
            />
            <label>{t("GitOps credential")}</label>
            <Select
              name="github_credential_id"
              value={githubCredentialID}
              required={requireGitOpsCredential}
              onChange={(event) => setGithubCredentialID(event.target.value)}
            >
              {!requireGitOpsCredential && (
                <option value="">{t("Create record only")}</option>
              )}
              {requireGitOpsCredential && (
                <option value="">{t("Choose GitOps credential")}</option>
              )}
              {(githubCredentials || []).map((credential) => (
                <option key={credential.id} value={credential.id}>
                  {credential.name}
                  {credential.gitops_repo ? ` · ${credential.gitops_repo}` : ""}
                </option>
              ))}
            </Select>
          </div>
        </section>
      )}

      {mode === "external" && (
        <section className="dependency-form-section">
          <h3>
            <Globe2 size={16} /> {t("External endpoint")}
          </h3>
          <input type="hidden" name="deploy_method" value="external" />
          <div className="dependency-form-fields">
            <label>{t("Host")}</label>
            <Input
              name="host"
              required
              value={host}
              onChange={(event) => setHost(event.target.value)}
              placeholder="10.0.0.20"
            />
            <label>{t("Port")}</label>
            <Input
              name="port"
              value={port}
              onChange={(event) => setPort(event.target.value)}
              placeholder={defaultPorts[activeType] || ""}
            />
            {activeType === "rabbitmq" && (
              <>
                <label>{t("Management port")}</label>
                <Input
                  name="management_port"
                  value={managementPort}
                  onChange={(event) => setManagementPort(event.target.value)}
                  placeholder="15672"
                />
              </>
            )}
          </div>
        </section>
      )}

      <section className="dependency-form-section">
        <h3>
          <Database size={16} /> {t("BeanCS objects")}
        </h3>
        <div className="dependency-form-fields">
          <label>{t("App object")}</label>
          <Input
            name="application_name"
            value={applicationName}
            onChange={(event) => setApplicationName(event.target.value)}
            placeholder={`dep-${activeType}`}
          />
          <label>{t("Namespace")}</label>
          <Input
            name="namespace"
            value={namespace}
            onChange={(event) => setNamespace(event.target.value)}
            placeholder={`dep-${activeType}`}
          />
        </div>
      </section>

      {mode === "cluster" && (
        <section className="dependency-form-section">
          <h3>
            <Server size={16} /> {t("Runtime settings")}
          </h3>
          <div className="dependency-form-fields">
            <div />
            <Checkbox name="shared" defaultChecked label={t("Reusable by other apps")} />
          </div>
          <DependencyConfigEditor
            definition={activeDefinition}
            value={config || {}}
            onChange={setConfig}
          />
        </section>
      )}

      {mode === "external" && (
        <section className="dependency-form-section">
          <h3>
            <Database size={16} /> {t("Credentials")}
          </h3>
          <div className="dependency-form-fields">
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
                <p className="muted">
                  {t("Credentials for this type must be entered manually.")}
                </p>
              </>
            )}
            {controlledSupported && controlled && (
              <>
                <label>{t("Admin username")}</label>
                <Input
                  name="admin_username"
                  required
                  value={adminUsername}
                  onChange={(event) => setAdminUsername(event.target.value)}
                  placeholder="root"
                />
                <label>{t("Admin password")}</label>
                <Input
                  name="admin_password"
                  type="password"
                  required
                  value={adminPassword}
                  onChange={(event) => setAdminPassword(event.target.value)}
                />
              </>
            )}
          </div>
        </section>
      )}
      {showActions && (
        <>
          <div className="modal-actions">
            <Button type="button" onClick={onCancel}>
              {t("Cancel")}
            </Button>
            <Button type="submit" variant="primary">
              <Database size={15} /> {t(submitLabel)}
            </Button>
          </div>
        </>
      )}
    </Shell>
  );
}

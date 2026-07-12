import React, { useEffect, useMemo, useState } from "react";
import { Database } from "lucide-react";
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
  const activeType = type || activeDefinition?.name || "mysql";
  const controlledSupported = controlledTypes.has(activeType);

  useEffect(() => {
    setType(activeDefinitions[0]?.name || "mysql");
  }, [mode, activeDefinitions]);

  useEffect(() => {
    setDeployMethod(defaultDeployMethod);
    setConfig(dependencyDefaultConfig(activeDefinition));
  }, [activeDefinition, defaultDeployMethod]);

  return (
    <form
      className="component-grid dependency-create-form"
      onSubmit={async (event) => {
        const ok = await onSubmit(event);
        if (ok) onCancel?.();
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
          <Select name="github_credential_id" required={requireGitOpsCredential}>
            {!requireGitOpsCredential && (
              <option value="">{t("Create record only")}</option>
            )}
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
              <p className="muted">
                {t("Credentials for this type must be entered manually.")}
              </p>
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
        <Button type="button" onClick={onCancel}>
          {t("Cancel")}
        </Button>
        <Button type="submit" variant="primary">
          <Database size={15} /> {t(submitLabel)}
        </Button>
      </div>
    </form>
  );
}

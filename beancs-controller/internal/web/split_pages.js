const fs = require('fs');

const pages = [
  { name: 'Dashboard', view: 'DashboardView', props: ['dashboard'] },
  { name: 'Deploy', view: 'DeployView', props: ['config', 'credentials', 'domains', 'namespaces', 'selectedCredential', 'setSelectedCredential', 'repos', 'selectedRepo', 'analysis', 'setAnalysis', 'form', 'setForm', 'loadRepos', 'analyzeRepo', 'checkInstallSource', 'deployProject', 'containerRegistries', 'containerImages', 'dependencyDefinitions', 'reusableDependencies', 'createTrackedImageFromDeploy', 'deployBasaltPass', 'onConnectGitHub', 'reposLoading'] },
  { name: 'Dependencies', view: 'DependenciesView', props: ['definitions', 'dependencies', 'githubCredentials', 'onCreateDependency', 'onCreateCredential'] },
  { name: 'Progress', view: 'ProgressView', props: ['projects', 'processes', 'activeProcessID', 'setActiveProcessID', 'activeProjectID', 'setActiveProjectID', 'progress', 'installProgress', 'refresh', 'refreshList', 'logFollow', 'liveLogs', 'logStatus', 'onStartLogFollow', 'onStopLogFollow'] },
  { name: 'Projects', view: 'ProjectsView', props: ['projects', 'onEdit', 'onDelete', 'onScale', 'onRestart', 'onBuild', 'onTracking', 'onProgress'] },
  { name: 'Applications', view: 'ApplicationsView', props: ['applications', 'onDeleteApplication'] },
  { name: 'Deployments', view: 'DeploymentsView', props: ['projects', 'processes', 'runtimeDeployments', 'refresh', 'onOpenProcess'] },
  { name: 'APIKeys', view: 'APIKeysView', props: ['keys', 'scopeCatalog', 'createdKey', 'onDismissCreated', 'onCreate', 'onRevoke', 'onRefresh'] },
  { name: 'ContainerRegistries', view: 'ContainerRegistriesView', props: ['presets', 'registries', 'images', 'onAddRegistry', 'onDeleteRegistry', 'onAddImage', 'onRefreshImage', 'onDeleteImage', 'onSyncAll', 'onRefresh'] },
  { name: 'WorkloadImage', view: 'WorkloadImageView', props: ['images', 'onRefresh', 'onOpenRegistry', 'onRefreshImage', 'onDeleteImage'] },
  { name: 'Alerts', view: 'AlertsView', props: ['dashboard', 'refresh'] },
  { name: 'Events', view: 'EventsView', props: ['dashboard', 'refresh'] },
  { name: 'Logs', view: 'LogsView', props: ['projects', 'activeProjectID', 'setActiveProjectID', 'progress', 'refresh', 'logFollow', 'liveLogs', 'logStatus', 'onStartLogFollow', 'onStopLogFollow', 'onOpenPods'] },
  { name: 'Metrics', view: 'MetricsView', props: ['dashboard', 'runtime', 'refresh'] },
  { name: 'Settings', view: 'SettingsView', props: ['version'] },
  { name: 'GitHub', view: 'GitHubView', props: ['credentials', 'onConnect', 'onUpdate', 'onRepos', 'onDelete', 'reposByCredential', 'repoFilters', 'setRepoFilters'] },
  { name: 'Domains', view: 'DomainsView', props: ['domains'] },
  { name: 'Networking', view: 'NetworkingView', props: ['network', 'refresh', 'onSaveService', 'onDeleteService', 'onSaveIngress', 'onDeleteIngress', 'onSaveNetworkPolicy', 'onDeleteNetworkPolicy', 'onDetail'] },
  { name: 'Cloudflare', view: 'CloudflareView', props: ['credentials', 'domains', 'selectedID', 'selectedZoneID', 'setSelectedID', 'setSelectedZoneID', 'dnsRecords', 'editingRecord', 'setEditingRecord', 'onCreate', 'onDelete', 'onLoadDNS', 'onSaveDNS', 'onDeleteDNS'] }
];

if (!fs.existsSync('src/pages')) {
  fs.mkdirSync('src/pages');
}

for (const p of pages) {
  const content = `import React from 'react';\nimport ${p.view} from '../views/${p.view}';\n\nexport default function ${p.name}Page(props) {\n  return <${p.view} {...props} />;\n}\n`;
  fs.writeFileSync(`src/pages/${p.name}Page.jsx`, content);
}
console.log("Pages generated successfully.");

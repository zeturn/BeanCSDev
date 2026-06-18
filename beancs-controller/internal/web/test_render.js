return (
    <div className="app-shell">
      <div
        className={sidebarOpen ? "sidebar-overlay active" : "sidebar-overlay"}
        onClick={() => setSidebarOpen(false)}
      />
      <aside className={sidebarOpen ? "sidebar open" : "sidebar"}>
        <div className="sidebar-product">
          <span className="brand-orb">
            <Coffee size={16} />
          </span>
          <b>BeanCS</b>
        </div>
        <label className="sidebar-search">
          <Search size={19} />
          <Input
            value={sidebarQuery}
            onChange={(event) => setSidebarQuery(event.target.value)}
            placeholder="Find..."
          />
          <kbd>F</kbd>
        </label>
        <div className="sidebar-nav">
          {filteredOverview.length > 0 && (
            <SidebarNavGroup
              items={filteredOverview}
              view={view}
              onSelect={selectNav}
            />
          )}
          {filteredNavSections.map((section) => (
            <SidebarNavGroup
              key={section.id}
              label={section.label}
              items={section.items}
              view={view}
              onSelect={selectNav}
            />
          ))}
          {sidebarQuery &&
            filteredOverview.length === 0 &&
            filteredNavSections.length === 0 && (
              <div className="nav-empty">No matches</div>
            )}
        </div>
        <div className="sidebar-user">
          <div className="user-avatar">
            {userProfile.avatar ? (
              <img
                src={userProfile.avatar}
                alt={userProfile.name || "User avatar"}
              />
            ) : (
              userProfile.initial
            )}
          </div>
          <div className="user-copy">
            <b>{userProfile.name}</b>
            <span>{userProfile.detail}</span>
          </div>
          <Button
            type="button"
            aria-label="More account actions"
            variant="icon"
          >
            <MoreHorizontal size={16} />
          </Button>
          <Button
            type="button"
            aria-label="Sign out"
            variant="icon"
            onClick={logout}
          >
            <LogOut size={16} />
          </Button>
        </div>
      </aside>
      <main className="workspace">
        <div className="mobile-topbar">
          <Button
            type="button"
            aria-label="Open navigation"
            onClick={() => setSidebarOpen(true)}
            variant="icon"
          >
            <Menu size={18} />
          </Button>
          <span className="mobile-brand">BeanCS</span>
        </div>
        <PageHeading
          title={
            view === "dashboard"
              ? dashboard?.cluster_name || "Overview"
              : titleFor(view)
          }
          topLabel={view === "dashboard" ? "Overview" : undefined}
          subtitle={
            view === "dashboard"
              ? `Kubernetes ${dashboard?.kubernetes_version || "-"}${dashboard?.k3s_version ? ` · K3s ${dashboard.k3s_version}` : ""}`
              : subtitleFor(view, runtime, projects)
          }
          actions={
            view === "dashboard" ? null : (
              <Button onClick={loadWorkspace} disabled={loading}>
                <RefreshCw size={15} /> Refresh
              </Button>
            )
          }
        />
        {notice && <div className="notice">{notice}</div>}
        {error && <div className="alert">{error}</div>}
        {shouldShowSkeleton(view, dashboard, network) ? (
          <SkeletonPage />
        ) : (
          <>
            {view === "dashboard" && (
              <DashboardView dashboard={dashboard} />
            )}
            {view === "deploy" && (
              <DeployView
                config={config}
                credentials={credentials}
                domains={domains}
                namespaces={runtime.namespaces || []}
                selectedCredential={selectedCredential}
                setSelectedCredential={setSelectedCredential}
                repos={repos}
                selectedRepo={selectedRepo}
                analysis={analysis}
                setAnalysis={setAnalysis}
                form={deployForm}
                setForm={setDeployForm}
                loadRepos={loadRepos}
                analyzeRepo={analyzeRepo}
                checkInstallSource={checkInstallSource}
                deployProject={deployProject}
                containerRegistries={containerRegistries}
                containerImages={containerImages}
                dependencyDefinitions={dependencyDefinitions}
                reusableDependencies={reusableDependencies}
                createTrackedImageFromDeploy={createTrackedImageFromDeploy}
                deployBasaltPass={deployBasaltPass}
                onConnectGitHub={connectGitHubApp}
                reposLoading={reposLoading}
              />
            )}
            {view === "dependencies" && (
              <DependenciesView
                definitions={dependencyDefinitions}
                dependencies={reusableDependencies}
                githubCredentials={credentials.github}
                onCreateDependency={createDependency}
                onCreateCredential={createDependencyCredential}
              />
            )}
            {view === "progress" && (
              <ProgressView
                projects={projects}
                processes={processRecords}
                activeProcessID={activeProcessID}
                setActiveProcessID={setActiveProcessID}
                activeProjectID={activeProgressProjectID}
                setActiveProjectID={setActiveProgressProjectID}
                progress={projectProgress}
                installProgress={installProgress}
                refresh={loadProjectProgress}
                refreshList={loadProcesses}
                logFollow={projectLogFollow}
                liveLogs={projectLiveLogs}
                logStatus={projectLogStatus}
                onStartLogFollow={startProjectLogFollow}
                onStopLogFollow={stopProjectLogFollow}
              />
            )}
            {view === "projects" && (
              <ProjectsView
                projects={projects}
                onEdit={setEditingProject}
                onDelete={deleteProject}
                onScale={scaleProject}
                onRestart={restartProject}
                onBuild={buildProject}
                onTracking={openProjectTracking}
                onProgress={(project) => {
                  setActiveProgressProjectID(String(project.id));
                  setView("progress");
                }}
              />
            )}
            {view === "applications" && (
              <ApplicationsView
                applications={applications}
                onDeleteApplication={deleteApplication}
              />
            )}
            {view === "deployments" && (
              <DeploymentsView
                projects={projects}
                processes={processRecords}
                runtimeDeployments={runtime.deployments || []}
                refresh={loadWorkspace}
                onOpenProcess={(process) => {
                  setActiveProcessID(String(process.id));
                  setActiveProgressProjectID(String(process.project_id || ""));
                  setView("progress");
                }}
              />
            )}
            {view === "apiKeys" && (
              <APIKeysView
                keys={apiKeys}
                scopeCatalog={apiKeyScopeCatalog}
                createdKey={createdAPIKey}
                onDismissCreated={() => setCreatedAPIKey(null)}
                onCreate={createAPIKey}
                onRevoke={revokeAPIKey}
                onRefresh={loadAPIKeys}
              />
            )}
            {view === "registries" && (
              <ContainerRegistriesView
                presets={registryPresets}
                registries={containerRegistries}
                images={containerImages}
                onAddRegistry={createContainerRegistry}
                onDeleteRegistry={deleteContainerRegistry}
                onAddImage={createTrackedImage}
                onRefreshImage={refreshTrackedImage}
                onDeleteImage={deleteTrackedImage}
                onSyncAll={syncAllTrackedImages}
                onRefresh={loadRegistriesPage}
              />
            )}
            {view === "workloadImage" && (
              <WorkloadImageView
                images={containerImages}
                onRefresh={loadRegistriesPage}
                onOpenRegistry={() => setView("registries")}
                onRefreshImage={refreshTrackedImage}
                onDeleteImage={deleteTrackedImage}
              />
            )}
            {view === "storage" && (
              <ComingSoonView
                title="Storage"
                description="PersistentVolumeClaims, PersistentVolumes, and StorageClasses will be manageable here in a future release."
              />
            )}
            {view === "secrets" && (
              <ComingSoonView
                title="Secrets"
                description="Kubernetes Secret inspection and rotation workflows are not wired in this console yet. Use kubectl or your GitOps pipeline for now."
              />
            )}
            {view === "alerts" && (
              <AlertsView dashboard={dashboard} refresh={loadDashboard} />
            )}
            {view === "events" && (
              <EventsView dashboard={dashboard} refresh={loadDashboard} />
            )}
            {view === "logs" && (
              <LogsView
                projects={projects}
                activeProjectID={activeProgressProjectID}
                setActiveProjectID={setActiveProgressProjectID}
                progress={projectProgress}
                refresh={loadProjectProgress}
                logFollow={projectLogFollow}
                liveLogs={projectLiveLogs}
                logStatus={projectLogStatus}
                onStartLogFollow={startProjectLogFollow}
                onStopLogFollow={stopProjectLogFollow}
                onOpenPods={() => setView("pods")}
              />
            )}
            {view === "metrics" && (
              <MetricsView
                dashboard={dashboard}
                runtime={runtime}
                refresh={loadDashboard}
              />
            )}
            {view === "settings" && <SettingsView version={appVersion} />}
            {view === "github" && (
              <GitHubView
                credentials={credentials.github}
                onConnect={connectGitHubApp}
                onUpdate={updateGitHubCredential}
                onRepos={loadRepos}
                onDelete={(id) => deleteCredential("github", id)}
                reposByCredential={reposByCredential}
                repoFilters={repoFilters}
                setRepoFilters={setRepoFilters}
              />
            )}
            {view === "domains" && <DomainsView domains={domains} />}
            {view === "networking" && (
              <NetworkingView
                network={network}
                refresh={loadNetwork}
                onSaveService={saveService}
                onDeleteService={deleteService}
                onSaveIngress={saveIngress}
                onDeleteIngress={deleteIngress}
                onSaveNetworkPolicy={saveNetworkPolicy}
                onDeleteNetworkPolicy={deleteNetworkPolicy}
                onDetail={setRuntimeDetail}
              />
            )}
            {view === "cloudflare" && (
              <CloudflareView
                credentials={credentials.cloudflare}
                domains={domains}
                selectedID={selectedCloudflareID}
                selectedZoneID={selectedCloudflareZoneID}
                setSelectedID={setSelectedCloudflareID}
                setSelectedZoneID={setSelectedCloudflareZoneID}
                dnsRecords={dnsRecords}
                editingRecord={editingDNSRecord}
                setEditingRecord={setEditingDNSRecord}
                onCreate={createCredential}
                onDelete={(id) => deleteCredential("cloudflare", id)}
                onLoadDNS={loadDNSRecords}
                onSaveDNS={saveDNSRecord}
                onDeleteDNS={deleteDNSRecord}
              />
            )}
            {view === "accessControl" && (
              <CredentialManager
                kind="basaltpass"
                rows={credentials.basaltpass}
                onCreate={createCredential}
                onDelete={deleteCredential}
              />
            )}
            {["namespaces", "pods", "nodes", "ingresses", "services"].includes(
              view,
            ) && (
              <RuntimeTable
                kind={view}
                rows={runtime[view] || []}
                nodeJoinCommand={nodeJoinCommand}
                onLoadNodeJoinCommand={loadNodeJoinCommand}
                onCreateNamespace={createNamespace}
                onPatchNamespace={patchNamespaceLabels}
                onNamespaceDetail={loadNamespaceDetail}
                onDeleteNamespace={deleteNamespace}
                onDeletePod={deletePod}
                onNodeDetail={loadNodeDetail}
                onPodLogs={loadPodLogs}
                onSaveService={saveService}
                onDeleteService={deleteService}
                onDetail={setRuntimeDetail}
              />
            )}
          </>
        )}
      </main>
      {editingProject && (
        <ProjectModal
          project={editingProject}
          onClose={() => setEditingProject(null)}
          onSubmit={updateProject}
          onLoadEnv={loadProjectEnv}
        />
      )}
      {deletingProject && (
        <DeleteProjectModal
          project={deletingProject}
          busy={loading}
          onClose={() => setDeletingProject(null)}
          onDelete={confirmDeleteProject}
        />
      )}
      {deletingApplication && (
        <DeleteApplicationModal
          application={deletingApplication}
          busy={loading}
          onClose={() => setDeletingApplication(null)}
          onDelete={confirmDeleteApplication}
        />
      )}
      {trackingProject && (
        <ProjectTrackingModal
          project={trackingProject}
          tracking={projectTracking}
          loading={trackingLoading}
          onRefresh={() => openProjectTracking(trackingProject)}
          onClose={() => {
            setTrackingProject(null);
            setProjectTracking(null);
          }}
        />
      )}
      {runtimeDetail && (
        <RuntimeDetailDrawer
          detail={runtimeDetail}
          logs={runtimeLogs}
          logFollow={runtimeLogFollow}
          logStatus={runtimeLogStatus}
          selectedLogContainer={runtimeLogContainer}
          logTail={runtimeLogTail}
          logLoaded={runtimeLogLoaded}
          nodeHealth={nodeHealth}
          onLoadNodeHealth={loadNodeHealth}
          onSaveNodeLabels={saveNodeLabels}
          onSaveNodeTaints={saveNodeTaints}
          onCordonNode={cordonNode}
          onDrainNode={drainNode}
          onDeleteNode={deleteNode}
          onSaveResourceQuota={saveResourceQuota}
          onDeleteResourceQuota={deleteResourceQuota}
          onSaveLimitRange={saveLimitRange}
          onDeleteLimitRange={deleteLimitRange}
          onSaveNamespacePermission={saveNamespacePermission}
          onDeleteNamespacePermission={deleteNamespacePermission}
          onSaveNamespaceIsolation={saveNamespaceIsolation}
          onSelectLogContainer={setRuntimeLogContainer}
          onSetLogTail={setRuntimeLogTail}
          onLoadContainerLogs={loadRuntimeContainerLogs}
          onFollowPodLogs={startRuntimeLogFollow}
          onStopPodLogs={stopRuntimeLogFollow}
          onClose={() => {
            stopRuntimeLogFollow();
            setRuntimeDetail(null);
            setRuntimeLogs("");
            setRuntimeLogContainer("");
            setRuntimeLogLoaded(false);
            setRuntimeLogStatus("");
            setNodeHealth(null);
          }}
          onSaveService={saveService}
          onPatchNamespace={patchNamespaceLabels}
        />
      )}
    </div>
  );
}
createRoot(document.getElementById("root")).render(<App />);

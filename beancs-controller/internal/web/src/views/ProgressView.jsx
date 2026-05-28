import React, { useEffect, useMemo, useRef, useState } from "react";
import * as LucideIcons from "lucide-react";
import {
  processJobsFromRecord,
  progressJobs,
  filterLogLines,
  formatTime,
} from "../utils/index";
import {
  ProgressEvidence,
  ProgressStatusIcon,
  MetricCard,
  Select,
  Button,
  Input,
} from "../components/index";
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
export default function ProgressView({
  projects,
  processes,
  activeProcessID,
  setActiveProcessID,
  activeProjectID,
  setActiveProjectID,
  progress,
  installProgress,
  refresh,
  refreshList,
  logFollow,
  liveLogs,
  logStatus,
  onStartLogFollow,
  onStopLogFollow,
}) {
  const [activeJob, setActiveJob] = useState("runtime");
  const [expandedSteps, setExpandedSteps] = useState({});
  const [logQuery, setLogQuery] = useState("");
  const selectedProcess = (processes || []).find(
    (process) => String(process.id) === String(activeProcessID),
  );
  const pods = progress?.pods || [];
  const events = progress?.events || [];
  const deployments = progress?.deployments || [];
  const currentProjectName =
    progress?.project?.name || installProgress?.project || "";
  const scopedInstallProgress =
    installProgress &&
    (!progress?.project?.name ||
      installProgress.project === progress.project.name)
      ? installProgress
      : null;
  const readyPods = pods.filter(
    (pod) =>
      Number(pod.ready_containers || 0) > 0 &&
      Number(pod.ready_containers) === Number(pod.total_containers || 0),
  ).length;
  const desiredReplicas =
    progress?.deployment?.replicas ?? progress?.project?.replicas ?? 0;
  const readyReplicas = progress?.deployment?.ready_replicas ?? 0;
  const logs = logFollow ? liveLogs : progress?.logs;
  const jobs = selectedProcess
    ? processJobsFromRecord(selectedProcess)
    : progressJobs(
        progress,
        scopedInstallProgress,
        readyPods,
        pods,
        deployments,
        events,
      );
  const detailTabID = activeJob.startsWith("detail:")
    ? activeJob.replace("detail:", "")
    : "";
  const selectedJob = detailTabID
    ? null
    : jobs.find((job) => job.id === activeJob) || jobs[0];
  const selectedJobLogs =
    selectedProcess && selectedJob
      ? selectedJob.steps.map((step) => step.log || "").join("\n")
      : "";
  const visibleLogs = filterLogLines(selectedJobLogs || "", logQuery);
  const visibleRuntimeLogs = filterLogLines(logs || "", logQuery);
  const detailTabs = [
    {
      id: "run",
      label: "Run details",
      icon: Activity,
    },
    {
      id: "install",
      label: "Install log",
      icon: ScrollText,
    },
    {
      id: "deployments",
      label: "Deployment records",
      icon: Rocket,
    },
    {
      id: "events",
      label: "Kubernetes events",
      icon: AlertTriangle,
    },
    {
      id: "runtime",
      label: "Runtime logs",
      icon: FileText,
    },
  ];
  const selectedDetailTab = detailTabs.find((tab) => tab.id === detailTabID);
  const toggleStep = (key) =>
    setExpandedSteps((current) => ({
      ...current,
      [key]: !current[key],
    }));
  if (!activeProcessID && !activeProjectID && !scopedInstallProgress) {
    return (
      <ProgressListView
        processes={processes || []}
        projects={projects}
        onSelectProcess={(process) => {
          setActiveProcessID(String(process.id));
          if (process.project_id)
            setActiveProjectID(String(process.project_id));
        }}
        refresh={refreshList}
      />
    );
  }
  const headerProject = selectedProcess?.project || progress?.project;
  return (
    <div className="process-page">
      <section className="process-topbar">
        <div>
          <h2>
            <ProgressStatusIcon
              status={
                selectedProcess?.status || selectedJob?.status || "pending"
              }
            />{" "}
            {selectedProcess?.title ||
              headerProject?.display_name ||
              headerProject?.name ||
              "Deployment process"}
          </h2>
          <p>
            {headerProject?.namespace ||
              currentProjectName ||
              "Choose a process"}
            {selectedProcess?.updated_at
              ? ` · updated ${formatTime(selectedProcess.updated_at)}`
              : progress?.checked_at
                ? ` · checked ${formatTime(progress.checked_at)}`
                : ""}
          </p>
        </div>
        <div className="progress-controls">
          <Select
            value={activeProcessID}
            onChange={(event) => {
              const next = (processes || []).find(
                (process) => String(process.id) === event.target.value,
              );
              setActiveProcessID(event.target.value);
              if (next?.project_id) setActiveProjectID(String(next.project_id));
            }}
          >
            <option value="">Choose process</option>
            {(processes || []).map((process) => (
              <option key={process.id} value={process.id}>
                #{process.id} {process.project?.name || process.title}
              </option>
            ))}
          </Select>
          <Button
            onClick={() => {
              refreshList();
              if (activeProjectID) refresh();
            }}
          >
            <RefreshCw size={15} /> Refresh
          </Button>
        </div>
      </section>

      <div className="process-shell">
        <aside className="process-sidebar">
          <Button
            type="button"
            className={
              activeJob === "summary" ? "process-nav active" : "process-nav"
            }
            onClick={() => setActiveJob("summary")}
          >
            <LayoutDashboard size={15} /> Summary
          </Button>
          <div className="process-nav-heading">All jobs</div>
          {jobs.map((job) => (
            <Button
              key={job.id}
              type="button"
              className={
                selectedJob?.id === job.id
                  ? "process-job active"
                  : "process-job"
              }
              onClick={() => setActiveJob(job.id)}
            >
              <ProgressStatusIcon status={job.status} />
              <span>{job.label}</span>
              <small>{job.detail}</small>
            </Button>
          ))}
          <div className="process-nav-heading">Run details</div>
          {detailTabs.map(({ id, label, icon: Icon }) => (
            <Button
              key={id}
              type="button"
              className={
                detailTabID === id ? "process-nav active" : "process-nav"
              }
              onClick={() => setActiveJob(`detail:${id}`)}
            >
              <Icon size={15} /> {label}
            </Button>
          ))}
        </aside>

        <section className="process-main">
          {activeJob === "summary" ? (
            <div className="process-summary">
              <div className="dashboard-kpis">
                <MetricCard
                  icon={Boxes}
                  label="Project"
                  value={progress?.project?.name || "-"}
                  detail={
                    progress?.project?.domain ||
                    progress?.project?.exposure_mode ||
                    "No route"
                  }
                />
                <MetricCard
                  icon={Server}
                  label="Runtime"
                  value={`${readyReplicas}/${desiredReplicas}`}
                  detail={`${readyPods}/${pods.length} pods ready`}
                />
                <MetricCard
                  icon={GitBranch}
                  label="Deployments"
                  value={deployments.length}
                  detail={deployments[0]?.status || "No events"}
                />
                <MetricCard
                  icon={AlertTriangle}
                  label="Warnings"
                  value={
                    events.filter((event) => event.type === "Warning").length
                  }
                  detail={`${events.length} Kubernetes events`}
                  tone={
                    events.some((event) => event.type === "Warning")
                      ? "warning"
                      : "good"
                  }
                />
              </div>
              {progress?.error && (
                <p className="error-inline">{progress.error}</p>
              )}
            </div>
          ) : detailTabID ? (
            <>
              <div className="process-job-header">
                <div>
                  <h2>{selectedDetailTab?.label || "Run details"}</h2>
                  <p>Process evidence and runtime signals for this run.</p>
                </div>
                <div className="process-log-tools">
                  {(detailTabID === "runtime" ||
                    detailTabID === "install" ||
                    detailTabID === "events" ||
                    detailTabID === "deployments") && (
                    <div className="process-search">
                      <Search size={15} />{" "}
                      <Input
                        value={logQuery}
                        onChange={(event) => setLogQuery(event.target.value)}
                        placeholder="Search details"
                      />
                    </div>
                  )}
                </div>
              </div>
              <ProgressEvidence
                activeTab={detailTabID}
                detailQuery={logQuery}
                progress={progress}
                installProgress={scopedInstallProgress}
                selectedProcess={selectedProcess}
                jobs={jobs}
                deployments={deployments}
                events={events}
                logs={visibleRuntimeLogs}
                logFollow={logFollow}
                logStatus={logStatus}
                onRefresh={() => (activeProjectID ? refresh() : refreshList())}
                onStartLogFollow={() =>
                  progress?.project?.id && onStartLogFollow(progress.project.id)
                }
                onStopLogFollow={onStopLogFollow}
              />
            </>
          ) : (
            <>
              <div className="process-job-header">
                <div>
                  <h2>{selectedJob?.label || "Job"}</h2>
                  <p>{selectedJob?.description || "Deployment job details"}</p>
                </div>
                <div className="process-log-tools">
                  <div className="process-search">
                    <Search size={15} />{" "}
                    <Input
                      value={logQuery}
                      onChange={(event) => setLogQuery(event.target.value)}
                      placeholder="Search logs"
                    />
                  </div>
                  <Button type="button" title="Settings">
                    <Settings size={15} />
                  </Button>
                </div>
              </div>

              <div className="process-step-list">
                {(selectedJob?.steps || []).map((step, index) => {
                  const stepKey = `${selectedJob?.id || "job"}:${step.label}:${index}`;
                  const isExpanded =
                    expandedSteps[stepKey] ?? Boolean(step.expanded);
                  return (
                    <div
                      className={
                        isExpanded ? "process-step expanded" : "process-step"
                      }
                      key={stepKey}
                    >
                      <div className="process-step-row">
                        <Button
                          type="button"
                          className="process-step-toggle"
                          aria-label={
                            isExpanded ? "Collapse step" : "Expand step"
                          }
                          onClick={() => toggleStep(stepKey)}
                        >
                          {isExpanded ? (
                            <ChevronDown size={16} />
                          ) : (
                            <ChevronRight size={16} />
                          )}
                        </Button>
                        <ProgressStatusIcon status={step.status} />
                        <span>{step.label}</span>
                        <small>{step.duration || ""}</small>
                      </div>
                      {isExpanded && (
                        <div className="process-log-block">
                          {step.kind === "logs" && (
                            <div className="row-actions process-log-actions">
                              <Button
                                onClick={() => refresh()}
                                disabled={logFollow}
                              >
                                <RefreshCw size={15} /> Snapshot
                              </Button>
                              {logFollow ? (
                                <Button onClick={onStopLogFollow}>
                                  Stop follow
                                </Button>
                              ) : (
                                <Button
                                  onClick={() =>
                                    progress?.project?.id &&
                                    onStartLogFollow(progress.project.id)
                                  }
                                  disabled={!progress?.project?.id}
                                  variant="primary"
                                >
                                  Follow live
                                </Button>
                              )}
                            </div>
                          )}
                          {logStatus && step.kind === "logs" && (
                            <p className="log-status">{logStatus}</p>
                          )}
                          <pre>
                            {step.kind === "logs"
                              ? visibleLogs || "No application logs yet."
                              : step.log || "No log output for this step."}
                          </pre>
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            </>
          )}
        </section>
      </div>
    </div>
  );
}

import React, { useEffect, useMemo, useRef, useState } from "react";
import * as LucideIcons from "lucide-react";
import { formatTime } from "../utils/index";
import {
  MetricCard,
  IngressForm,
  NetworkPolicyForm,
  SimpleTable,
  ServiceForm,
  Button,
  Modal,
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
export default function NetworkingView({
  network,
  refresh,
  onSaveService,
  onDeleteService,
  onSaveIngress,
  onDeleteIngress,
  onSaveNetworkPolicy,
  onDeleteNetworkPolicy,
  onDetail,
}) {
  const [activeTab, setActiveTab] = useState("services");
  const [serviceCreateOpen, setServiceCreateOpen] = useState(false);
  const [ingressCreateOpen, setIngressCreateOpen] = useState(false);
  const [policyCreateOpen, setPolicyCreateOpen] = useState(false);
  const data = network || {
    services: [],
    ingresses: [],
    endpoints: [],
    network_policies: [],
    access: [],
    controllers: {},
  };
  const controllers = data.controllers || {};
  const tabs = [
    {
      id: "services",
      label: "Services",
      icon: Database,
      count: data.services.length,
    },
    {
      id: "ingress",
      label: "Ingress & TLS",
      icon: Network,
      count: data.ingresses.length,
    },
    {
      id: "policies",
      label: "NetworkPolicy",
      icon: Lock,
      count: data.network_policies.length,
    },
    {
      id: "access",
      label: "Access addresses",
      icon: Globe2,
      count: data.access.length,
    },
    {
      id: "endpoints",
      label: "Endpoints",
      icon: Layers3,
      count: data.endpoints.length,
    },
  ];
  return (
    <div className="stack network-page">
      <section className="network-overview">
        <div className="dashboard-kpis">
          <MetricCard
            icon={Database}
            label="Services"
            value={data.services.length}
            detail="ClusterIP / NodePort / LoadBalancer"
          />
          <MetricCard
            icon={Network}
            label="Ingresses"
            value={data.ingresses.length}
            detail={`${controllers.traefik_ingresses || 0} Traefik · ${controllers.tailscale_ingresses || 0} Tailscale`}
          />
          <MetricCard
            icon={Shield}
            label="TLS"
            value={controllers.tls_ingresses || 0}
            detail="Ingress TLS bindings"
          />
          <MetricCard
            icon={Layers3}
            label="Endpoints"
            value={data.endpoints.length}
            detail="Resolved backend addresses"
          />
          <MetricCard
            icon={Lock}
            label="Policies"
            value={data.network_policies.length}
            detail="NetworkPolicy rules"
          />
          <MetricCard
            icon={Globe2}
            label="Access URLs"
            value={data.access.length}
            detail="Service access entries"
          />
        </div>
        <div className="detail-list compact-details">
          <span>
            Traefik namespaces{" "}
            <b>{(controllers.traefik_namespaces || []).join(", ") || "-"}</b>
          </span>
          <span>
            Tailscale namespaces{" "}
            <b>{(controllers.tailscale_namespaces || []).join(", ") || "-"}</b>
          </span>
          <span>
            Checked <b>{formatTime(data.checked_at)}</b>
          </span>
        </div>
      </section>

      <section className="panel network-tabs-panel">
        <div
          className="network-tabs"
          role="tablist"
          aria-label="Networking resources"
        >
          {tabs.map((tab) => {
            const Icon = tab.icon;
            return (
              <Button
                key={tab.id}
                type="button"
                role="tab"
                aria-selected={activeTab === tab.id}
                className={
                  activeTab === tab.id ? "network-tab active" : "network-tab"
                }
                onClick={() => setActiveTab(tab.id)}
              >
                <Icon size={15} />
                <span>{tab.label}</span>
                <b>{tab.count}</b>
              </Button>
            );
          })}
        </div>

        {activeTab === "services" && (
          <div className="network-tab-panel">
            <div className="panel-heading-inline">
              <h2>
                <Database size={18} /> Service, LoadBalancer and NodePort
              </h2>
              <Button type="button" onClick={() => setServiceCreateOpen(true)}>
                <Plus size={15} /> Create service
              </Button>
            </div>
            <SimpleTable
              rows={data.services}
              columns={[
                "namespace",
                "name",
                "type",
                "cluster_ip",
                "external_ip",
                "ports",
              ]}
              actions={(row) => (
                <>
                  <Button
                    onClick={() =>
                      onDetail({
                        kind: "services",
                        row,
                      })
                    }
                  >
                    Details
                  </Button>
                  <Button onClick={() => onDeleteService(row)} variant="danger">
                    <Trash2 size={15} />
                  </Button>
                </>
              )}
            />
          </div>
        )}

        {activeTab === "ingress" && (
          <div className="network-tab-panel">
            <div className="panel-heading-inline">
              <h2>
                <Network size={18} /> Ingress, domain and TLS binding
              </h2>
              <Button type="button" onClick={() => setIngressCreateOpen(true)}>
                <Plus size={15} /> Create ingress
              </Button>
            </div>
            <SimpleTable
              rows={data.ingresses}
              columns={[
                "namespace",
                "name",
                "class",
                "hosts",
                "services",
                "tls",
                "address",
              ]}
              actions={(row) => (
                <>
                  <Button
                    onClick={() =>
                      onDetail({
                        kind: "ingresses",
                        row,
                      })
                    }
                  >
                    Details
                  </Button>
                  <Button onClick={() => onDeleteIngress(row)} variant="danger">
                    <Trash2 size={15} />
                  </Button>
                </>
              )}
            />
          </div>
        )}

        {activeTab === "policies" && (
          <div className="network-tab-panel">
            <div className="panel-heading-inline">
              <h2>
                <Lock size={18} /> NetworkPolicy
              </h2>
              <Button type="button" onClick={() => setPolicyCreateOpen(true)}>
                <Plus size={15} /> Create policy
              </Button>
            </div>
            <SimpleTable
              rows={data.network_policies}
              columns={[
                "namespace",
                "name",
                "pod_selector",
                "policy_types",
                "ingress_rules",
                "egress_rules",
              ]}
              actions={(row) => (
                <>
                  <Button
                    onClick={() =>
                      onDetail({
                        kind: "network-policy",
                        row,
                      })
                    }
                  >
                    Details
                  </Button>
                  <Button
                    onClick={() => onDeleteNetworkPolicy(row)}
                    variant="danger"
                  >
                    <Trash2 size={15} />
                  </Button>
                </>
              )}
            />
          </div>
        )}

        {activeTab === "access" && (
          <div className="network-tab-panel">
            <h2>
              <Globe2 size={18} /> Service access addresses
            </h2>
            <SimpleTable
              rows={data.access || []}
              columns={[
                "namespace",
                "service",
                "type",
                "class",
                "urls",
                "node_ports",
                "load_balancer",
              ]}
              actions={(row) => (
                <Button
                  onClick={() =>
                    onDetail({
                      kind: "service-access",
                      row,
                    })
                  }
                >
                  Details
                </Button>
              )}
            />
          </div>
        )}

        {activeTab === "endpoints" && (
          <div className="network-tab-panel">
            <h2>
              <Layers3 size={18} /> Endpoints
            </h2>
            <SimpleTable
              rows={data.endpoints || []}
              columns={["namespace", "name", "addresses", "ports"]}
              actions={(row) => (
                <Button
                  onClick={() =>
                    onDetail({
                      kind: "endpoints",
                      row,
                    })
                  }
                >
                  Details
                </Button>
              )}
              compact
            />
          </div>
        )}
      </section>
      {serviceCreateOpen && (
        <Modal
          title="Create service"
          subtitle="创建入口独立到弹窗，避免与列表混在一起。"
          className="wide-modal"
          onClose={() => setServiceCreateOpen(false)}
        >
          <ServiceForm
            onSubmit={async (event) => {
              await onSaveService(event);
              setServiceCreateOpen(false);
            }}
          />
          <div className="modal-actions">
            <Button type="button" onClick={() => setServiceCreateOpen(false)}>
              Close
            </Button>
          </div>
        </Modal>
      )}
      {ingressCreateOpen && (
        <Modal
          title="Create ingress"
          subtitle="创建入口独立到弹窗，保持列表简洁。"
          className="wide-modal"
          onClose={() => setIngressCreateOpen(false)}
        >
          <IngressForm
            onSubmit={async (event) => {
              await onSaveIngress(event);
              setIngressCreateOpen(false);
            }}
          />
          <div className="modal-actions">
            <Button type="button" onClick={() => setIngressCreateOpen(false)}>
              Close
            </Button>
          </div>
        </Modal>
      )}
      {policyCreateOpen && (
        <Modal
          title="Create network policy"
          subtitle="创建入口独立到弹窗，降低单页复杂度。"
          className="wide-modal"
          onClose={() => setPolicyCreateOpen(false)}
        >
          <NetworkPolicyForm
            onSubmit={async (event) => {
              await onSaveNetworkPolicy(event);
              setPolicyCreateOpen(false);
            }}
          />
          <div className="modal-actions">
            <Button type="button" onClick={() => setPolicyCreateOpen(false)}>
              Close
            </Button>
          </div>
        </Modal>
      )}
    </div>
  );
}

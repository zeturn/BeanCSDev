import React, { useState } from "react";
import { Database, HardDrive, Layers3, RefreshCw, Server } from "lucide-react";
import { formatTime } from "../utils/index";
import { Button, MetricCard, SimpleTable } from "../components/index";

export default function StorageView({ storage, refresh }) {
  const [activeTab, setActiveTab] = useState("claims");
  const data = storage || {
    persistent_volume_claims: [],
    persistent_volumes: [],
    storage_classes: [],
  };
  const claims = data.persistent_volume_claims || [];
  const volumes = data.persistent_volumes || [];
  const classes = data.storage_classes || [];
  const boundClaims = claims.filter((claim) => claim.phase === "Bound").length;
  const mountedClaims = claims.filter((claim) => claim.mounted_by?.length).length;
  const tabs = [
    { id: "claims", label: "PersistentVolumeClaims", icon: Database, count: claims.length },
    { id: "volumes", label: "PersistentVolumes", icon: HardDrive, count: volumes.length },
    { id: "classes", label: "StorageClasses", icon: Layers3, count: classes.length },
  ];

  return (
    <div className="stack storage-page">
      <section className="panel storage-overview">
        <div className="action-panel">
          <div>
            <h2>
              <HardDrive size={18} /> Storage resources
            </h2>
            <p>Inspect claim binding, mounted workloads, persistent volumes and available storage classes.</p>
          </div>
          <Button onClick={refresh} title="Refresh storage resources">
            <RefreshCw size={15} /> Refresh
          </Button>
        </div>
        <div className="dashboard-kpis">
          <MetricCard icon={Database} label="PVCs" value={claims.length} detail={`${boundClaims} bound`} />
          <MetricCard icon={Server} label="Mounted claims" value={mountedClaims} detail="Used by active pods" />
          <MetricCard icon={HardDrive} label="PersistentVolumes" value={volumes.length} detail="Cluster-wide volumes" />
          <MetricCard icon={Layers3} label="StorageClasses" value={classes.length} detail={`${classes.filter((item) => item.default).length} default`} />
        </div>
        <div className="detail-list compact-details">
          <span>Checked <b>{formatTime(data.checked_at)}</b></span>
          <span>Mode <b>Read only</b></span>
        </div>
      </section>

      <section className="panel storage-tabs-panel">
        <div className="storage-tabs" role="tablist" aria-label="Storage resources">
          {tabs.map((tab) => {
            const Icon = tab.icon;
            return (
              <Button
                key={tab.id}
                type="button"
                role="tab"
                aria-selected={activeTab === tab.id}
                className={activeTab === tab.id ? "storage-tab active" : "storage-tab"}
                onClick={() => setActiveTab(tab.id)}
              >
                <Icon size={15} />
                <span>{tab.label}</span>
                <b>{tab.count}</b>
              </Button>
            );
          })}
        </div>

        {activeTab === "claims" && (
          <div className="storage-tab-panel">
            <h2><Database size={18} /> PersistentVolumeClaims</h2>
            <SimpleTable
              rows={claims}
              columns={["namespace", "name", "phase", "requested_capacity", "capacity", "access_modes", "storage_class", "volume_name", "mounted_by", "age_seconds"]}
              className="storage-claims-table"
            />
          </div>
        )}
        {activeTab === "volumes" && (
          <div className="storage-tab-panel">
            <h2><HardDrive size={18} /> PersistentVolumes</h2>
            <SimpleTable
              rows={volumes}
              columns={["name", "phase", "capacity", "access_modes", "storage_class", "claim", "reclaim_policy", "age_seconds"]}
              className="storage-volumes-table"
            />
          </div>
        )}
        {activeTab === "classes" && (
          <div className="storage-tab-panel">
            <h2><Layers3 size={18} /> StorageClasses</h2>
            <SimpleTable
              rows={classes}
              columns={["name", "provisioner", "default", "reclaim_policy", "volume_binding_mode", "allow_volume_expansion", "age_seconds"]}
              className="storage-classes-table"
            />
          </div>
        )}
      </section>
    </div>
  );
}

import { memo } from "react";
import {
  Background,
  Controls,
  ReactFlow,
  type Node,
  type NodeProps,
  type NodeTypes,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { Badge } from "@/components/ui/badge";

export type ClusterNodeRole = "leader" | "follower" | "unhealthy" | "standalone";

export type ClusterMember = {
  id: string;
  name: string;
  role: ClusterNodeRole;
  address?: string;
  state?: string;
};

type ClusterTopologyProps = {
  members?: ClusterMember[];
};

type ClusterNodeData = {
  title: string;
  subtitle: string;
  role: ClusterNodeRole;
  address?: string;
  state?: string;
};

function roleClass(role: ClusterNodeRole) {
  switch (role) {
    case "leader":
      return "border-sky-500/60 bg-sky-500/12";
    case "follower":
      return "border-emerald-500/55 bg-emerald-500/10";
    case "unhealthy":
      return "border-destructive/70 bg-destructive/12";
    case "standalone":
    default:
      return "border-border bg-card";
  }
}

function roleLabel(role: ClusterNodeRole) {
  switch (role) {
    case "leader":
      return "Leader";
    case "follower":
      return "Follower";
    case "unhealthy":
      return "Unhealthy";
    case "standalone":
    default:
      return "Standalone";
  }
}

function ClusterRoleNode({ data }: NodeProps) {
  const nodeData = data as ClusterNodeData;
  return (
    <div className={`min-w-[220px] rounded-xl border p-3 shadow-sm ${roleClass(nodeData.role)}`}>
      <div className="flex items-center justify-between gap-2">
        <p className="text-sm font-semibold">{nodeData.title}</p>
        <Badge variant="outline" className="h-5 rounded-full px-2 text-[10px] uppercase tracking-wide">
          {roleLabel(nodeData.role)}
        </Badge>
      </div>
      <p className="mt-1 text-xs text-muted-foreground">{nodeData.subtitle}</p>
      {nodeData.address ? <p className="mt-2 text-[11px] text-muted-foreground">{nodeData.address}</p> : null}
      {nodeData.state ? <p className="mt-1 text-[11px] text-muted-foreground">state: {nodeData.state}</p> : null}
    </div>
  );
}

const clusterNodeTypes: NodeTypes = {
  clusterLeaderNode: memo(ClusterRoleNode),
  clusterFollowerNode: memo(ClusterRoleNode),
  clusterUnhealthyNode: memo(ClusterRoleNode),
  clusterStandaloneNode: memo(ClusterRoleNode),
};

function nodeTypeForRole(role: ClusterNodeRole): keyof typeof clusterNodeTypes {
  switch (role) {
    case "leader":
      return "clusterLeaderNode";
    case "follower":
      return "clusterFollowerNode";
    case "unhealthy":
      return "clusterUnhealthyNode";
    case "standalone":
    default:
      return "clusterStandaloneNode";
  }
}

export function ClusterTopology({ members = [] }: ClusterTopologyProps) {
  const nodes: Node[] = [];

  if (!members.length) {
    nodes.push({
      id: "cluster-standalone",
      type: "clusterStandaloneNode",
      position: { x: 220, y: 160 },
      draggable: false,
      data: {
        title: "Node-1",
        subtitle: "Standalone gateway mode",
        role: "standalone",
        state: "single-node",
      } satisfies ClusterNodeData,
    });
  } else {
    const centerX = 320;
    const centerY = 220;
    const radius = Math.max(130, members.length * 24);

    members.forEach((member, index) => {
      const angle = (index / Math.max(1, members.length)) * Math.PI * 2 - Math.PI / 2;
      const x = centerX + Math.cos(angle) * radius;
      const y = centerY + Math.sin(angle) * radius;

      nodes.push({
        id: `cluster-${member.id}`,
        type: nodeTypeForRole(member.role),
        position: { x, y },
        draggable: false,
        data: {
          title: member.name,
          subtitle: member.role === "leader" ? "Control plane leader" : "Replicated cluster member",
          role: member.role,
          address: member.address,
          state: member.state,
        } satisfies ClusterNodeData,
      });
    });
  }

  return (
    <div className="relative h-[420px] w-full overflow-hidden rounded-xl border bg-card/60">
      <div className="absolute left-3 top-3 z-10 rounded-lg border bg-background/85 px-2 py-1 text-[11px] text-muted-foreground">
        <p>v0.5.0 prep: leader/follower/unhealthy node types are ready.</p>
      </div>
      <ReactFlow
        nodes={nodes}
        edges={[]}
        nodeTypes={clusterNodeTypes}
        nodesDraggable={false}
        nodesConnectable={false}
        elementsSelectable={false}
        fitView
        fitViewOptions={{ padding: 0.2 }}
      >
        <Background gap={16} size={1} className="opacity-65" />
        <Controls showInteractive={false} />
      </ReactFlow>
    </div>
  );
}

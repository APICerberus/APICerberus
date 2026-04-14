import { useQuery } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { adminApiRequest } from "@/lib/api";
import { ReconnectingWebSocketClient } from "@/lib/ws";

export type ClusterNodeRole = "leader" | "follower" | "candidate" | "unhealthy" | "standalone";

export interface ClusterNode {
  id: string;
  name: string;
  address: string;
  role: ClusterNodeRole;
  state: "healthy" | "unhealthy" | "joining" | "leaving";
  lastSeen: string;
  metadata?: Record<string, unknown>;
}

export interface ClusterEdge {
  from: string;
  to: string;
  type: "raft" | "rpc" | "heartbeat";
  status: "connected" | "disconnected" | "lagging";
  latencyMs?: number;
}

export interface ClusterStatus {
  enabled: boolean;
  mode: "standalone" | "raft";
  nodeId: string;
  leaderId?: string;
  term: number;
  commitIndex: number;
  appliedIndex: number;
  nodes: ClusterNode[];
  edges: ClusterEdge[];
}

const CLUSTER_QUERY_KEY = "cluster";

// Placeholder data for standalone mode
const STANDALONE_STATUS: ClusterStatus = {
  enabled: false,
  mode: "standalone",
  nodeId: "local",
  term: 0,
  commitIndex: 0,
  appliedIndex: 0,
  nodes: [
    {
      id: "local",
      name: "Local Node",
      address: "127.0.0.1:8080",
      role: "standalone",
      state: "healthy",
      lastSeen: new Date().toISOString(),
    },
  ],
  edges: [],
};

async function fetchClusterStatus(): Promise<ClusterStatus> {
  try {
    return await adminApiRequest<ClusterStatus>("/admin/api/v1/cluster/status");
  } catch (error) {
    // Return standalone mode as fallback
    console.warn("Cluster API not available, using standalone mode", error);
    return STANDALONE_STATUS;
  }
}

export function useClusterStatus() {
  return useQuery({
    queryKey: [CLUSTER_QUERY_KEY],
    queryFn: fetchClusterStatus,
    refetchInterval: 5000, // Refresh every 5 seconds
    staleTime: 3000,
  });
}

export function useClusterRealtime() {
  const [status, setStatus] = useState<ClusterStatus>(STANDALONE_STATUS);
  const [isConnected, setIsConnected] = useState(false);

  useEffect(() => {
    const ws = new ReconnectingWebSocketClient<{ type: string; payload?: Partial<ClusterStatus> }>();

    ws.subscribe((message) => {
      if (message.type === "cluster" && message.payload) {
        setStatus((prev) => ({
          ...prev,
          ...message.payload,
        }));
      }
    });

    ws.onStatusChange((s) => {
      setIsConnected(s === "open");
      if (s === "open") {
        ws.send({ action: "subscribe", channel: "cluster" });
      }
    });

    ws.connect();

    return () => {
      ws.disconnect();
    };
  }, []);

  return { status, isConnected };
}

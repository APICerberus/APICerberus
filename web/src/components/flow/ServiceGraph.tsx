import type { ReactNode } from "react";
import dagre from "dagre";
import { Background, MarkerType, ReactFlow, type Edge, type Node, type NodeMouseHandler } from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { Badge } from "@/components/ui/badge";
import type { Route, Service, Upstream } from "@/lib/types";
import { flowEdgeTypes } from "./edges";

type GraphNodeType = "service" | "route" | "upstream";

type ServiceGraphProps = {
  services: Service[];
  routes: Route[];
  upstreams: Upstream[];
  onNodeNavigate?: (nodeType: GraphNodeType, id: string) => void;
};

type GraphNodeData = {
  nodeType: GraphNodeType;
  entityID: string;
  label: ReactNode;
};

const NODE_WIDTH = 220;
const NODE_HEIGHT = 96;

function styleFor(nodeType: GraphNodeType) {
  switch (nodeType) {
    case "service":
      return {
        border: "1px solid hsl(var(--chart-1) / 0.7)",
        background: "hsl(var(--chart-1) / 0.12)",
      };
    case "route":
      return {
        border: "1px solid hsl(var(--chart-3) / 0.7)",
        background: "hsl(var(--chart-3) / 0.12)",
      };
    case "upstream":
      return {
        border: "1px solid hsl(var(--chart-2) / 0.7)",
        background: "hsl(var(--chart-2) / 0.12)",
      };
    default:
      return {
        border: "1px solid hsl(var(--border))",
        background: "hsl(var(--card))",
      };
  }
}

function nodeID(nodeType: GraphNodeType, id: string) {
  return `${nodeType}:${id}`;
}

function compactList(values: string[], max = 2) {
  if (!values.length) {
    return "-";
  }
  const base = values.slice(0, max).join(", ");
  const remaining = values.length - max;
  return remaining > 0 ? `${base} +${remaining}` : base;
}

export function ServiceGraph({ services, routes, upstreams, onNodeNavigate }: ServiceGraphProps) {
  const upstreamByID = new Map(upstreams.map((upstream) => [upstream.id, upstream]));
  const serviceByID = new Map(services.map((service) => [service.id, service]));

  const nodes: Node<GraphNodeData>[] = [];
  const edges: Edge[] = [];

  for (const service of services) {
    nodes.push({
      id: nodeID("service", service.id),
      position: { x: 0, y: 0 },
      draggable: false,
      data: {
        nodeType: "service",
        entityID: service.id,
        label: (
          <div className="space-y-1">
            <p className="text-sm font-semibold">{service.name}</p>
            <div className="flex items-center gap-1 text-[11px] text-muted-foreground">
              <Badge variant="outline" className="h-5 rounded-full px-1.5 text-[10px] uppercase">
                service
              </Badge>
              <span>{service.protocol}</span>
            </div>
          </div>
        ),
      },
      style: {
        ...styleFor("service"),
        width: NODE_WIDTH,
        minHeight: NODE_HEIGHT,
        borderRadius: 12,
        color: "hsl(var(--foreground))",
      },
    });
  }

  for (const route of routes) {
    nodes.push({
      id: nodeID("route", route.id),
      position: { x: 0, y: 0 },
      draggable: false,
      data: {
        nodeType: "route",
        entityID: route.id,
        label: (
          <div className="space-y-1">
            <p className="text-sm font-semibold">{route.name}</p>
            <p className="text-[11px] text-muted-foreground">{compactList(route.methods)}</p>
            <p className="text-[11px] text-muted-foreground">{compactList(route.paths)}</p>
          </div>
        ),
      },
      style: {
        ...styleFor("route"),
        width: NODE_WIDTH,
        minHeight: NODE_HEIGHT,
        borderRadius: 12,
        color: "hsl(var(--foreground))",
      },
    });

    edges.push({
      id: `service-route:${route.service}:${route.id}`,
      source: nodeID("service", route.service),
      target: nodeID("route", route.id),
      type: "trafficEdge",
      markerEnd: { type: MarkerType.ArrowClosed },
      style: { strokeWidth: 2.1 },
    });

    const upstream = upstreamByID.get(serviceByID.get(route.service)?.upstream ?? "");
    if (upstream) {
      edges.push({
        id: `route-upstream:${route.id}:${upstream.id}`,
        source: nodeID("route", route.id),
        target: nodeID("upstream", upstream.id),
        type: "trafficEdge",
        markerEnd: { type: MarkerType.ArrowClosed },
        style: { strokeWidth: 1.8 },
      });
    }
  }

  const seenUpstreams = new Set<string>();
  for (const service of services) {
    const upstream = upstreamByID.get(service.upstream);
    if (!upstream || seenUpstreams.has(upstream.id)) {
      continue;
    }
    seenUpstreams.add(upstream.id);

    nodes.push({
      id: nodeID("upstream", upstream.id),
      position: { x: 0, y: 0 },
      draggable: false,
      data: {
        nodeType: "upstream",
        entityID: upstream.id,
        label: (
          <div className="space-y-1">
            <p className="text-sm font-semibold">{upstream.name}</p>
            <p className="text-[11px] text-muted-foreground">algorithm: {upstream.algorithm}</p>
            <p className="text-[11px] text-muted-foreground">targets: {upstream.targets.length}</p>
          </div>
        ),
      },
      style: {
        ...styleFor("upstream"),
        width: NODE_WIDTH,
        minHeight: NODE_HEIGHT,
        borderRadius: 12,
        color: "hsl(var(--foreground))",
      },
    });
  }

  const dagreGraph = new dagre.graphlib.Graph().setDefaultEdgeLabel(() => ({}));
  dagreGraph.setGraph({ rankdir: "LR", ranksep: 90, nodesep: 46, marginx: 24, marginy: 24 });

  for (const node of nodes) {
    dagreGraph.setNode(node.id, { width: NODE_WIDTH, height: NODE_HEIGHT });
  }
  for (const edge of edges) {
    dagreGraph.setEdge(edge.source, edge.target);
  }

  dagre.layout(dagreGraph);

  const layoutedNodes = nodes.map((node) => {
    const point = dagreGraph.node(node.id);
    return {
      ...node,
      position: {
        x: point.x - NODE_WIDTH / 2,
        y: point.y - NODE_HEIGHT / 2,
      },
    };
  });

  const handleNodeClick: NodeMouseHandler<Node<GraphNodeData>> = (_event, node) => {
    const data = node.data;
    if (!data?.nodeType || !data?.entityID) {
      return;
    }
    onNodeNavigate?.(data.nodeType, data.entityID);
  };

  return (
    <div className="relative h-[620px] w-full overflow-hidden rounded-xl border bg-card/60">
      <div className="absolute left-3 top-3 z-10 rounded-lg border bg-background/85 px-2 py-1 text-[11px] text-muted-foreground">
        <p>Auto-layout powered by dagre (left-to-right)</p>
      </div>
      <ReactFlow
        nodes={layoutedNodes}
        edges={edges}
        edgeTypes={flowEdgeTypes}
        onNodeClick={handleNodeClick}
        nodesConnectable={false}
        nodesDraggable={false}
        elementsSelectable={false}
        fitView
        fitViewOptions={{ padding: 0.1 }}
      >
        <Background gap={16} size={1} className="opacity-65" />
      </ReactFlow>
    </div>
  );
}

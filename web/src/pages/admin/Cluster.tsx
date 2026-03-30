import { ClusterTopology, type ClusterMember } from "@/components/flow/ClusterTopology";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

const PLACEHOLDER_MEMBERS: ClusterMember[] = [
  {
    id: "node-1",
    name: "Node-1",
    role: "standalone",
    address: "127.0.0.1:8080",
    state: "single-node",
  },
];

export function ClusterPage() {
  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>Cluster Topology</CardTitle>
          <CardDescription>
            Standalone placeholder for current architecture; role-specific node types are prepped for Raft in v0.5.0.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <ClusterTopology members={PLACEHOLDER_MEMBERS} />
        </CardContent>
      </Card>
    </div>
  );
}

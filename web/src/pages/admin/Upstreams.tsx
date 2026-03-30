import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import type { ColumnDef } from "@tanstack/react-table";
import { Plus } from "lucide-react";
import type { Upstream } from "@/lib/types";
import { ROUTES } from "@/lib/constants";
import { DataTable } from "@/components/shared/DataTable";
import { StatusBadge } from "@/components/shared/StatusBadge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useAddUpstreamTarget, useCreateUpstream, useUpstreams } from "@/hooks/use-upstreams";

const UPSTREAM_COLUMNS: ColumnDef<Upstream>[] = [
  {
    accessorKey: "name",
    header: "Upstream",
    cell: ({ row }) => <span className="font-medium">{row.original.name}</span>,
  },
  {
    accessorKey: "algorithm",
    header: "Algorithm",
  },
  {
    id: "targets",
    header: "Targets",
    cell: ({ row }) => row.original.targets.length,
  },
  {
    id: "health",
    header: "Health",
    cell: ({ row }) => <StatusBadge status={row.original.targets.length ? "active" : "pending"} />,
  },
];

export function UpstreamsPage() {
  const navigate = useNavigate();
  const upstreamsQuery = useUpstreams();
  const createUpstream = useCreateUpstream();
  const addTarget = useAddUpstreamTarget();

  const [createOpen, setCreateOpen] = useState(false);
  const [targetOpen, setTargetOpen] = useState(false);
  const [selectedUpstreamID, setSelectedUpstreamID] = useState("");
  const [name, setName] = useState("");
  const [algorithm, setAlgorithm] = useState("round_robin");
  const [targetAddress, setTargetAddress] = useState("127.0.0.1:8080");
  const [targetWeight, setTargetWeight] = useState("100");

  const upstreams = useMemo(() => upstreamsQuery.data ?? [], [upstreamsQuery.data]);

  const handleCreate = async () => {
    if (!name.trim()) {
      return;
    }
    const created = await createUpstream.mutateAsync({
      name: name.trim(),
      algorithm,
      targets: [],
    });
    setCreateOpen(false);
    setName("");
    navigate(ROUTES.upstreamDetail(created.id));
  };

  const handleAddTarget = async () => {
    if (!selectedUpstreamID || !targetAddress.trim()) {
      return;
    }
    await addTarget.mutateAsync({
      id: selectedUpstreamID,
      payload: {
        address: targetAddress.trim(),
        weight: Number(targetWeight) || 100,
      },
    });
    setTargetOpen(false);
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-semibold">Upstreams</h2>
          <p className="text-sm text-muted-foreground">Manage balancing strategy, health and target topology.</p>
        </div>

        <div className="flex items-center gap-2">
          <Dialog open={targetOpen} onOpenChange={setTargetOpen}>
            <DialogTrigger asChild>
              <Button variant="outline">
                <Plus className="mr-2 size-4" />
                Add Target
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Add Upstream Target</DialogTitle>
                <DialogDescription>Assign new backend address and traffic weight.</DialogDescription>
              </DialogHeader>
              <div className="space-y-3">
                <div className="space-y-1.5">
                  <Label>Upstream</Label>
                  <Select value={selectedUpstreamID} onValueChange={setSelectedUpstreamID}>
                    <SelectTrigger>
                      <SelectValue placeholder="Select upstream" />
                    </SelectTrigger>
                    <SelectContent>
                      {upstreams.map((upstream) => (
                        <SelectItem key={upstream.id} value={upstream.id}>
                          {upstream.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="target-address">Address</Label>
                  <Input
                    id="target-address"
                    value={targetAddress}
                    onChange={(event) => setTargetAddress(event.target.value)}
                  />
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="target-weight">Weight</Label>
                  <Input
                    id="target-weight"
                    value={targetWeight}
                    onChange={(event) => setTargetWeight(event.target.value)}
                  />
                </div>
              </div>
              <DialogFooter>
                <Button variant="outline" onClick={() => setTargetOpen(false)}>
                  Cancel
                </Button>
                <Button onClick={handleAddTarget} disabled={addTarget.isPending}>
                  Add
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>

          <Dialog open={createOpen} onOpenChange={setCreateOpen}>
            <DialogTrigger asChild>
              <Button>
                <Plus className="mr-2 size-4" />
                New Upstream
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Create Upstream</DialogTitle>
                <DialogDescription>Define balancing strategy before attaching targets.</DialogDescription>
              </DialogHeader>
              <div className="space-y-3">
                <div className="space-y-1.5">
                  <Label htmlFor="upstream-name">Name</Label>
                  <Input id="upstream-name" value={name} onChange={(event) => setName(event.target.value)} />
                </div>
                <div className="space-y-1.5">
                  <Label>Algorithm</Label>
                  <Select value={algorithm} onValueChange={setAlgorithm}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {["round_robin", "weighted_round_robin", "least_conn", "least_latency"].map((value) => (
                        <SelectItem key={value} value={value}>
                          {value}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <DialogFooter>
                <Button variant="outline" onClick={() => setCreateOpen(false)}>
                  Cancel
                </Button>
                <Button onClick={handleCreate} disabled={createUpstream.isPending}>
                  Create
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      <DataTable<Upstream, unknown>
        data={upstreams}
        columns={UPSTREAM_COLUMNS}
        searchColumn="name"
        searchPlaceholder="Search upstream..."
        fileName="upstreams"
        className="rounded-lg border bg-card p-3"
      />
    </div>
  );
}


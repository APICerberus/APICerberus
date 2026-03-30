import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Plus } from "lucide-react";
import type { ColumnDef } from "@tanstack/react-table";
import { ServiceGraph } from "@/components/flow/ServiceGraph";
import type { Service } from "@/lib/types";
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
import { useRoutes } from "@/hooks/use-routes";
import { useCreateService, useServices } from "@/hooks/use-services";
import { useUpstreams } from "@/hooks/use-upstreams";

const SERVICE_COLUMNS: ColumnDef<Service>[] = [
  {
    accessorKey: "name",
    header: "Service",
    cell: ({ row }) => <span className="font-medium">{row.original.name}</span>,
  },
  {
    accessorKey: "protocol",
    header: "Protocol",
  },
  {
    accessorKey: "upstream",
    header: "Upstream",
  },
  {
    id: "status",
    header: "Status",
    cell: () => <StatusBadge status="active" />,
  },
];

export function ServicesPage() {
  const navigate = useNavigate();
  const servicesQuery = useServices();
  const routesQuery = useRoutes();
  const upstreamsQuery = useUpstreams();
  const createService = useCreateService();
  const [open, setOpen] = useState(false);
  const [view, setView] = useState<"table" | "graph">("table");
  const [name, setName] = useState("");
  const [upstream, setUpstream] = useState("");
  const [protocol, setProtocol] = useState("http");

  const services = useMemo(() => servicesQuery.data ?? [], [servicesQuery.data]);
  const routes = useMemo(() => routesQuery.data ?? [], [routesQuery.data]);
  const upstreams = useMemo(() => upstreamsQuery.data ?? [], [upstreamsQuery.data]);

  const handleCreate = async () => {
    if (!name.trim() || !upstream.trim()) {
      return;
    }
    const created = await createService.mutateAsync({
      name: name.trim(),
      upstream: upstream.trim(),
      protocol,
    });
    setOpen(false);
    setName("");
    setUpstream("");
    setProtocol("http");
    navigate(ROUTES.serviceDetail(created.id));
  };

  const handleGraphNavigate = (nodeType: "service" | "route" | "upstream", id: string) => {
    if (nodeType === "service") {
      navigate(ROUTES.serviceDetail(id));
      return;
    }
    if (nodeType === "route") {
      navigate(ROUTES.routeDetail(id));
      return;
    }
    navigate(ROUTES.upstreamDetail(id));
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-semibold">Services</h2>
          <p className="text-sm text-muted-foreground">Manage API services and map them to upstream pools.</p>
        </div>

        <div className="flex items-center gap-2">
          <div className="inline-flex items-center gap-1 rounded-lg border bg-background p-1">
            <Button size="sm" variant={view === "table" ? "default" : "ghost"} onClick={() => setView("table")}>
              Table
            </Button>
            <Button size="sm" variant={view === "graph" ? "default" : "ghost"} onClick={() => setView("graph")}>
              Graph
            </Button>
          </div>

          <Dialog open={open} onOpenChange={setOpen}>
            <DialogTrigger asChild>
              <Button>
                <Plus className="mr-2 size-4" />
                New Service
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Create Service</DialogTitle>
                <DialogDescription>Define service name, protocol and upstream binding.</DialogDescription>
              </DialogHeader>
              <div className="space-y-3">
                <div className="space-y-1.5">
                  <Label htmlFor="service-name">Name</Label>
                  <Input id="service-name" value={name} onChange={(event) => setName(event.target.value)} />
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="service-upstream">Upstream</Label>
                  <Input id="service-upstream" value={upstream} onChange={(event) => setUpstream(event.target.value)} />
                </div>
                <div className="space-y-1.5">
                  <Label>Protocol</Label>
                  <Select value={protocol} onValueChange={setProtocol}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="http">http</SelectItem>
                      <SelectItem value="https">https</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <DialogFooter>
                <Button variant="outline" onClick={() => setOpen(false)}>
                  Cancel
                </Button>
                <Button onClick={handleCreate} disabled={createService.isPending}>
                  Create
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      {view === "table" ? (
        <DataTable<Service, unknown>
          data={services}
          columns={SERVICE_COLUMNS}
          searchColumn="name"
          searchPlaceholder="Search service..."
          fileName="services"
          className="rounded-lg border bg-card p-3"
        />
      ) : (
        <ServiceGraph
          services={services}
          routes={routes}
          upstreams={upstreams}
          onNodeNavigate={handleGraphNavigate}
        />
      )}
    </div>
  );
}

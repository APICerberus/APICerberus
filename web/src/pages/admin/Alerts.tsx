import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { ColumnDef } from "@tanstack/react-table";
import { Plus, Trash2 } from "lucide-react";
import { adminApiRequest } from "@/lib/api";
import { formatDateTime } from "@/lib/utils";
import { DataTable } from "@/components/shared/DataTable";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
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
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";

type AlertRule = {
  id: string;
  name: string;
  enabled: boolean;
  type: "error_rate" | "p99_latency" | "upstream_health";
  threshold: number;
  window: string;
  cooldown: string;
  action: {
    type: "log" | "webhook";
    webhook_url?: string;
  };
};

type AlertHistoryRow = {
  id: string;
  rule_id: string;
  rule_name: string;
  rule_type: string;
  triggered_at: string;
  value: number;
  threshold: number;
  action_type: string;
  success: boolean;
  error?: string;
};

type AlertsResponse = {
  rules: AlertRule[];
  history: AlertHistoryRow[];
};

const ALERT_HISTORY_COLUMNS: ColumnDef<AlertHistoryRow>[] = [
  {
    accessorKey: "rule_name",
    header: "Rule",
    cell: ({ row }) => <span className="font-medium">{row.original.rule_name}</span>,
  },
  {
    accessorKey: "rule_type",
    header: "Type",
  },
  {
    id: "value",
    header: "Value / Threshold",
    cell: ({ row }) => (
      <span>
        {row.original.value.toFixed(2)} / {row.original.threshold.toFixed(2)}
      </span>
    ),
  },
  {
    accessorKey: "action_type",
    header: "Action",
  },
  {
    id: "status",
    header: "Status",
    cell: ({ row }) => (
      <Badge variant="outline" className={row.original.success ? "border-emerald-500/60 bg-emerald-500/10" : "border-destructive/60 bg-destructive/10 text-destructive"}>
        {row.original.success ? "success" : "failed"}
      </Badge>
    ),
  },
  {
    accessorKey: "triggered_at",
    header: "Triggered",
    cell: ({ row }) => <span className="text-xs text-muted-foreground">{formatDateTime(row.original.triggered_at)}</span>,
  },
];

export function AlertsPage() {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);

  const [name, setName] = useState("");
  const [type, setType] = useState<AlertRule["type"]>("error_rate");
  const [threshold, setThreshold] = useState("5");
  const [window, setWindow] = useState("5m");
  const [cooldown, setCooldown] = useState("1m");
  const [actionType, setActionType] = useState<AlertRule["action"]["type"]>("log");
  const [webhookURL, setWebhookURL] = useState("");

  const alertsQuery = useQuery({
    queryKey: ["alerts"],
    queryFn: () => adminApiRequest<AlertsResponse>("/admin/api/v1/alerts"),
    refetchInterval: 10_000,
  });

  const createAlert = useMutation({
    mutationFn: (payload: Partial<AlertRule>) =>
      adminApiRequest<AlertRule>("/admin/api/v1/alerts", {
        method: "POST",
        body: payload,
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["alerts"] });
    },
  });

  const updateAlert = useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: Partial<AlertRule> }) =>
      adminApiRequest<AlertRule>(`/admin/api/v1/alerts/${id}`, {
        method: "PUT",
        body: payload,
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["alerts"] });
    },
  });

  const deleteAlert = useMutation({
    mutationFn: (id: string) =>
      adminApiRequest<null>(`/admin/api/v1/alerts/${id}`, {
        method: "DELETE",
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["alerts"] });
    },
  });

  const rules = useMemo(() => alertsQuery.data?.rules ?? [], [alertsQuery.data?.rules]);
  const history = useMemo(() => alertsQuery.data?.history ?? [], [alertsQuery.data?.history]);

  const resetForm = () => {
    setName("");
    setType("error_rate");
    setThreshold("5");
    setWindow("5m");
    setCooldown("1m");
    setActionType("log");
    setWebhookURL("");
  };

  const handleCreate = async () => {
    if (!name.trim()) {
      return;
    }
    await createAlert.mutateAsync({
      name: name.trim(),
      enabled: true,
      type,
      threshold: Number(threshold),
      window,
      cooldown,
      action: {
        type: actionType,
        webhook_url: actionType === "webhook" ? webhookURL.trim() : undefined,
      },
    });
    setOpen(false);
    resetForm();
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-semibold">Alerts</h2>
          <p className="text-sm text-muted-foreground">Configure alert rules and inspect trigger history.</p>
        </div>

        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="mr-2 size-4" />
              New Alert Rule
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Create Alert Rule</DialogTitle>
              <DialogDescription>Define threshold, window, action, and cooldown policy.</DialogDescription>
            </DialogHeader>
            <div className="space-y-3">
              <div className="space-y-1.5">
                <Label htmlFor="alert-name">Rule Name</Label>
                <Input id="alert-name" value={name} onChange={(event) => setName(event.target.value)} />
              </div>

              <div className="grid gap-3 md:grid-cols-2">
                <div className="space-y-1.5">
                  <Label>Type</Label>
                  <Select value={type} onValueChange={(value) => setType(value as AlertRule["type"])}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="error_rate">error_rate</SelectItem>
                      <SelectItem value="p99_latency">p99_latency</SelectItem>
                      <SelectItem value="upstream_health">upstream_health</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="alert-threshold">Threshold</Label>
                  <Input id="alert-threshold" value={threshold} onChange={(event) => setThreshold(event.target.value)} />
                </div>
              </div>

              <div className="grid gap-3 md:grid-cols-2">
                <div className="space-y-1.5">
                  <Label htmlFor="alert-window">Window</Label>
                  <Input id="alert-window" value={window} onChange={(event) => setWindow(event.target.value)} />
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="alert-cooldown">Cooldown</Label>
                  <Input id="alert-cooldown" value={cooldown} onChange={(event) => setCooldown(event.target.value)} />
                </div>
              </div>

              <div className="grid gap-3 md:grid-cols-2">
                <div className="space-y-1.5">
                  <Label>Action</Label>
                  <Select value={actionType} onValueChange={(value) => setActionType(value as AlertRule["action"]["type"])}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="log">log</SelectItem>
                      <SelectItem value="webhook">webhook</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="alert-webhook">Webhook URL</Label>
                  <Input
                    id="alert-webhook"
                    value={webhookURL}
                    onChange={(event) => setWebhookURL(event.target.value)}
                    placeholder="https://example.com/webhook"
                    disabled={actionType !== "webhook"}
                  />
                </div>
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setOpen(false)}>
                Cancel
              </Button>
              <Button onClick={handleCreate} disabled={createAlert.isPending}>
                Create
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>

      <div className="grid gap-3">
        {rules.map((rule) => (
          <Card key={rule.id}>
            <CardHeader className="pb-2">
              <CardTitle className="text-base">{rule.name}</CardTitle>
              <CardDescription>
                {rule.type} | threshold {rule.threshold} | window {rule.window} | cooldown {rule.cooldown}
              </CardDescription>
            </CardHeader>
            <CardContent className="flex items-center justify-between">
              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                <Badge variant="outline">{rule.action.type}</Badge>
                {rule.action.webhook_url ? <span>{rule.action.webhook_url}</span> : null}
              </div>
              <div className="flex items-center gap-2">
                <Switch
                  checked={rule.enabled}
                  onCheckedChange={(checked) =>
                    updateAlert.mutate({
                      id: rule.id,
                      payload: {
                        ...rule,
                        enabled: checked,
                      },
                    })
                  }
                />
                <Button variant="ghost" size="icon" onClick={() => deleteAlert.mutate(rule.id)}>
                  <Trash2 className="size-4 text-destructive" />
                </Button>
              </div>
            </CardContent>
          </Card>
        ))}
        {!rules.length ? <p className="text-sm text-muted-foreground">No alert rules configured.</p> : null}
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Alert History</CardTitle>
          <CardDescription>Triggered alerts and action delivery results.</CardDescription>
        </CardHeader>
        <CardContent>
          <DataTable<AlertHistoryRow, unknown>
            columns={ALERT_HISTORY_COLUMNS}
            data={history}
            searchColumn="rule_name"
            searchPlaceholder="Search history..."
            fileName="alert-history"
          />
        </CardContent>
      </Card>
    </div>
  );
}

import { useEffect, useMemo, useState } from "react";
import { useParams } from "react-router-dom";
import { Save, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ErrorState } from "@/components/shared/ErrorState";
import { useDeleteUpstreamTarget, useUpdateUpstream, useUpstream, useUpstreamHealth } from "@/hooks/use-upstreams";

export function UpstreamDetailPage() {
  const { id = "" } = useParams();
  const upstreamQuery = useUpstream(id);
  const healthQuery = useUpstreamHealth(id);
  const updateUpstream = useUpdateUpstream();
  const deleteTarget = useDeleteUpstreamTarget();

  const [name, setName] = useState("");
  const [algorithm, setAlgorithm] = useState("round_robin");
  const [healthPath, setHealthPath] = useState("/health");

  const upstream = upstreamQuery.data;
  const healthTargets = useMemo(
    () => ((healthQuery.data?.targets ?? []) as Array<Record<string, unknown>>) ?? [],
    [healthQuery.data?.targets],
  );

  useEffect(() => {
    if (!upstream) {
      return;
    }
    setName(upstream.name);
    setAlgorithm(upstream.algorithm);
    setHealthPath(String((upstream.health_check?.active as { path?: string } | undefined)?.path ?? "/health"));
  }, [upstream]);

  if (!id) {
    return <ErrorState message="Missing upstream id." />;
  }
  if (upstreamQuery.isError) {
    return <ErrorState message="Failed to load upstream detail." onRetry={() => upstreamQuery.refetch()} />;
  }

  const handleSave = async () => {
    await updateUpstream.mutateAsync({
      id,
      payload: {
        id,
        name,
        algorithm,
        targets: upstream?.targets ?? [],
        health_check: {
          active: {
            ...(upstream?.health_check?.active ?? {}),
            path: healthPath,
          },
        },
      },
    });
  };

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>Upstream Configuration</CardTitle>
          <CardDescription>Set algorithm and health-check path configuration.</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-3 md:grid-cols-3">
          <div className="space-y-1.5">
            <Label htmlFor="upstream-detail-name">Name</Label>
            <Input id="upstream-detail-name" value={name} onChange={(event) => setName(event.target.value)} />
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
          <div className="space-y-1.5">
            <Label htmlFor="upstream-health-path">Health Path</Label>
            <Input
              id="upstream-health-path"
              value={healthPath}
              onChange={(event) => setHealthPath(event.target.value)}
            />
          </div>
          <div className="md:col-span-3">
            <Button onClick={handleSave} disabled={updateUpstream.isPending}>
              <Save className="mr-2 size-4" />
              Save Upstream
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Targets</CardTitle>
          <CardDescription>Inspect and manage upstream target health.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-2">
          {(upstream?.targets ?? []).map((target) => {
            const health = healthTargets.find((item) => String(item.target_id) === target.id);
            return (
              <div key={target.id} className="flex items-center justify-between rounded-md border p-3">
                <div>
                  <p className="font-medium">{target.address}</p>
                  <p className="text-xs text-muted-foreground">weight: {target.weight}</p>
                </div>
                <div className="flex items-center gap-2">
                  <Badge variant="outline">{String(health?.healthy ?? "unknown")}</Badge>
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => deleteTarget.mutate({ id, targetId: target.id })}
                    aria-label="Delete target"
                  >
                    <Trash2 className="size-4 text-destructive" />
                  </Button>
                </div>
              </div>
            );
          })}
          {!upstream?.targets?.length ? <p className="text-sm text-muted-foreground">No targets configured.</p> : null}
        </CardContent>
      </Card>
    </div>
  );
}


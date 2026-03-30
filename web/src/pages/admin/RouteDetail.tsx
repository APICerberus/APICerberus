import { useEffect, useMemo, useState } from "react";
import { useParams } from "react-router-dom";
import { Save } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import { ErrorState } from "@/components/shared/ErrorState";
import { useRoute, useUpdateRoute } from "@/hooks/use-routes";
import { useServices } from "@/hooks/use-services";

type EditablePlugin = {
  name: string;
  enabled: boolean;
};

export function RouteDetailPage() {
  const { id = "" } = useParams();
  const routeQuery = useRoute(id);
  const servicesQuery = useServices();
  const updateRoute = useUpdateRoute();

  const [name, setName] = useState("");
  const [service, setService] = useState("");
  const [pathsText, setPathsText] = useState("");
  const [methodsText, setMethodsText] = useState("GET");
  const [plugins, setPlugins] = useState<EditablePlugin[]>([]);

  const route = routeQuery.data;
  const services = useMemo(() => servicesQuery.data ?? [], [servicesQuery.data]);

  useEffect(() => {
    if (!route) {
      return;
    }
    setName(route.name);
    setService(route.service);
    setPathsText(route.paths.join(", "));
    setMethodsText(route.methods.join(", "));
    setPlugins((route.plugins ?? []).map((plugin) => ({ name: plugin.name, enabled: plugin.enabled ?? true })));
  }, [route]);

  if (!id) {
    return <ErrorState message="Missing route id." />;
  }
  if (routeQuery.isError) {
    return <ErrorState message="Failed to load route details." onRetry={() => routeQuery.refetch()} />;
  }

  const handleSave = async () => {
    await updateRoute.mutateAsync({
      id,
      payload: {
        id,
        name,
        service,
        paths: pathsText
          .split(",")
          .map((item) => item.trim())
          .filter(Boolean),
        methods: methodsText
          .split(",")
          .map((item) => item.trim().toUpperCase())
          .filter(Boolean),
        plugins: plugins.map((plugin) => ({ name: plugin.name, enabled: plugin.enabled })),
      },
    });
  };

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>Route Configuration</CardTitle>
          <CardDescription>Edit route matching and service assignment.</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-3 md:grid-cols-2">
          <div className="space-y-1.5">
            <Label htmlFor="route-detail-name">Name</Label>
            <Input id="route-detail-name" value={name} onChange={(event) => setName(event.target.value)} />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="route-detail-service">Service</Label>
            <Input
              id="route-detail-service"
              list="service-options"
              value={service}
              onChange={(event) => setService(event.target.value)}
            />
            <datalist id="service-options">
              {services.map((svc) => (
                <option key={svc.id} value={svc.id}>
                  {svc.name}
                </option>
              ))}
            </datalist>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="route-detail-paths">Paths (comma separated)</Label>
            <Input id="route-detail-paths" value={pathsText} onChange={(event) => setPathsText(event.target.value)} />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="route-detail-methods">Methods (comma separated)</Label>
            <Input
              id="route-detail-methods"
              value={methodsText}
              onChange={(event) => setMethodsText(event.target.value)}
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Plugin Configuration</CardTitle>
          <CardDescription>Toggle route-level plugins and keep execution list visible.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-2">
          {plugins.length ? (
            plugins.map((plugin) => (
              <div key={plugin.name} className="flex items-center justify-between rounded-md border p-3">
                <div className="space-y-1">
                  <p className="font-medium">{plugin.name}</p>
                  <Badge variant="outline">route plugin</Badge>
                </div>
                <Switch
                  checked={plugin.enabled}
                  onCheckedChange={(checked) =>
                    setPlugins((current) =>
                      current.map((item) => (item.name === plugin.name ? { ...item, enabled: checked } : item)),
                    )
                  }
                />
              </div>
            ))
          ) : (
            <p className="text-sm text-muted-foreground">No plugins configured for this route.</p>
          )}
        </CardContent>
      </Card>

      <Button onClick={handleSave} disabled={updateRoute.isPending}>
        <Save className="mr-2 size-4" />
        Save Route
      </Button>
    </div>
  );
}


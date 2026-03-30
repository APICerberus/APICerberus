import { Navigate, Route, Routes } from "react-router-dom";
import { AdminLayout } from "@/components/layout/AdminLayout";
import { ThemeProvider } from "@/components/layout/ThemeProvider";
import { NAV_ITEMS } from "@/components/layout/navigation";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { ROUTES } from "@/lib/constants";
import { DashboardPage } from "@/pages/admin/Dashboard";
import { RouteDetailPage } from "@/pages/admin/RouteDetail";
import { RoutesPage } from "@/pages/admin/Routes";
import { ServiceDetailPage } from "@/pages/admin/ServiceDetail";
import { ServicesPage } from "@/pages/admin/Services";
import { UpstreamDetailPage } from "@/pages/admin/UpstreamDetail";
import { UpstreamsPage } from "@/pages/admin/Upstreams";

function PlaceholderPage({ title, description }: { title: string; description: string }) {
  return (
    <div className="mx-auto max-w-5xl">
      <Card>
        <CardHeader>
          <Badge className="w-fit" variant="secondary">
            In Progress
          </Badge>
          <CardTitle>{title}</CardTitle>
          <CardDescription>{description}</CardDescription>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">
            This screen is scaffolded and connected to app navigation. CRUD widgets and data hooks land in subsequent
            tasks.
          </p>
        </CardContent>
      </Card>
    </div>
  );
}

export function App() {
  return (
    <ThemeProvider>
      <AdminLayout>
        <Routes>
          <Route path={ROUTES.dashboard} element={<DashboardPage />} />
          <Route path={ROUTES.services} element={<ServicesPage />} />
          <Route path="/services/:id" element={<ServiceDetailPage />} />
          <Route path={ROUTES.routes} element={<RoutesPage />} />
          <Route path="/routes/:id" element={<RouteDetailPage />} />
          <Route path={ROUTES.upstreams} element={<UpstreamsPage />} />
          <Route path="/upstreams/:id" element={<UpstreamDetailPage />} />
          {NAV_ITEMS
            .filter(
              (item) =>
                item.path !== ROUTES.dashboard &&
                item.path !== ROUTES.services &&
                item.path !== ROUTES.routes &&
                item.path !== ROUTES.upstreams,
            )
            .map((item) => (
            <Route
              key={item.path}
              path={item.path}
              element={<PlaceholderPage title={item.title} description={item.description} />}
            />
            ))}
          <Route path="*" element={<Navigate to={ROUTES.dashboard} replace />} />
        </Routes>
      </AdminLayout>
    </ThemeProvider>
  );
}

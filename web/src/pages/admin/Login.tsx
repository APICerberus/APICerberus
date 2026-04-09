import { useEffect, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Shield } from "lucide-react";

const ERROR_MESSAGES: Record<string, string> = {
  missing_key: "Admin key is required.",
  invalid_key: "Invalid admin key. Please try again.",
};

export function AdminLoginPage() {
  const [searchParams] = useSearchParams();
  const [error, setError] = useState("");

  useEffect(() => {
    const loginError = searchParams.get("login");
    if (loginError && loginError in ERROR_MESSAGES) {
      setError(ERROR_MESSAGES[loginError]);
    } else {
      setError("");
    }
  }, [searchParams]);

  return (
    <div className="min-h-screen flex items-center justify-center bg-background p-4">
      <Card className="w-full max-w-sm">
        <CardHeader className="space-y-1">
          <div className="flex items-center justify-center mb-2">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-primary/20">
              <Shield className="h-5 w-5 text-primary" />
            </div>
          </div>
          <CardTitle className="text-center text-2xl">Admin Login</CardTitle>
          <CardDescription className="text-center">
            Enter your admin API key to access the dashboard.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {/*
            Traditional HTML form POST — the admin key goes directly from the
            browser form to the server without ever entering JavaScript memory.
            The server validates the key and sets an HttpOnly, SameSite=Strict
            session cookie. This prevents XSS from exfiltrating the admin key.
          */}
          <form action="/admin/login" method="POST" className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="admin-key">Admin API Key</Label>
              <Input
                id="admin-key"
                name="admin_key"
                type="password"
                placeholder="Enter admin API key"
                autoComplete="current-password"
                autoFocus
              />
            </div>
            {error && <p className="text-sm text-destructive">{error}</p>}
            <Button type="submit" className="w-full">
              Continue
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}

import type { LucideIcon } from "lucide-react";
import { Inbox } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

type EmptyStateProps = {
  title: string;
  description: string;
  actionLabel?: string;
  onAction?: () => void;
  icon?: LucideIcon;
};

export function EmptyState({ title, description, actionLabel, onAction, icon: Icon = Inbox }: EmptyStateProps) {
  return (
    <Card className="border-dashed">
      <CardHeader className="items-center text-center">
        <span className="rounded-xl bg-muted p-3 text-muted-foreground">
          <Icon className="size-6" />
        </span>
        <CardTitle className="mt-2">{title}</CardTitle>
        <CardDescription className="max-w-md">{description}</CardDescription>
      </CardHeader>
      {actionLabel && onAction ? (
        <CardContent className="flex justify-center">
          <Button onClick={onAction}>{actionLabel}</Button>
        </CardContent>
      ) : null}
    </Card>
  );
}


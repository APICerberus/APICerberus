import { useState } from "react";
import { Check, Copy } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";

type CopyButtonProps = {
  value: string;
  copiedText?: string;
  className?: string;
};

export function CopyButton({ value, copiedText = "Copied to clipboard", className }: CopyButtonProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      toast.success(copiedText);
      window.setTimeout(() => setCopied(false), 1200);
    } catch {
      toast.error("Copy failed");
    }
  };

  return (
    <Button variant="outline" size="icon" className={className} onClick={handleCopy} aria-label="Copy">
      {copied ? <Check className="size-4" /> : <Copy className="size-4" />}
    </Button>
  );
}


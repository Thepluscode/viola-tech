import { cn } from "@/lib/utils";
import { AlertTriangle } from "lucide-react";

interface ErrorStateProps {
  title?: string;
  message?: string;
  className?: string;
}

export function ErrorState({
  title = "Something went wrong",
  message = "An unexpected error occurred. Please try again.",
  className,
}: ErrorStateProps) {
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center py-16 gap-4 text-center",
        className
      )}
    >
      <div className="flex items-center justify-center w-12 h-12 rounded-md bg-severity-critical-bg border border-severity-critical-border">
        <AlertTriangle className="h-6 w-6 text-severity-critical" />
      </div>
      <div>
        <p className="text-sm font-semibold text-viola-text">{title}</p>
        <p className="text-xs text-viola-muted mt-1 max-w-sm">{message}</p>
      </div>
    </div>
  );
}

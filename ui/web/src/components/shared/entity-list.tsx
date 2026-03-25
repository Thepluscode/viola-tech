import { cn } from "@/lib/utils";
import { Monitor } from "lucide-react";

interface EntityListProps {
  entities: string[];
  className?: string;
}

export function EntityList({ entities, className }: EntityListProps) {
  if (entities.length === 0) {
    return <span className="text-xs text-viola-muted italic">No entities</span>;
  }
  return (
    <div className={cn("flex flex-col gap-1", className)}>
      {entities.map((entity) => (
        <span
          key={entity}
          className="inline-flex items-center gap-2 text-xs font-mono text-viola-text"
        >
          <Monitor className="h-3 w-3 text-viola-accent shrink-0" />
          {entity}
        </span>
      ))}
    </div>
  );
}

import { cn } from "@/lib/utils";

interface MitreTagProps {
  id?: string;
  name?: string;
  tactic?: string;
  className?: string;
}

export function MitreTag({ id, name, tactic, className }: MitreTagProps) {
  const label = name ? `${id} · ${name}` : id ?? tactic ?? "Unknown";

  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 px-2 py-0.5 rounded-md",
        "bg-[#0d1f3c] border border-[#1e3a6e] text-[#60a5fa]",
        "text-xs font-mono",
        className
      )}
      title={tactic}
    >
      {label}
    </span>
  );
}

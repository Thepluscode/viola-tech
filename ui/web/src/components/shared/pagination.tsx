"use client";

import { useRouter, usePathname, useSearchParams } from "next/navigation";
import { cn } from "@/lib/utils";
import { ChevronLeft, ChevronRight } from "lucide-react";

interface PaginationProps {
  page: number;
  hasNextPage: boolean;
  className?: string;
}

export function Pagination({ page, hasNextPage, className }: PaginationProps) {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  function navigate(newPage: number) {
    const params = new URLSearchParams(searchParams.toString());
    params.set("page", String(newPage));
    router.push(`${pathname}?${params.toString()}`);
  }

  return (
    <div className={cn("flex items-center gap-3 text-sm text-viola-muted", className)}>
      <button
        onClick={() => navigate(page - 1)}
        disabled={page <= 1}
        className={cn(
          "flex items-center gap-1 px-3 py-1.5 rounded-md border",
          "border-viola-border bg-viola-surface hover:border-viola-accent hover:text-viola-accent",
          "transition-colors disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:border-viola-border disabled:hover:text-viola-muted"
        )}
      >
        <ChevronLeft className="h-3.5 w-3.5" />
        Prev
      </button>
      <span className="font-mono text-xs text-viola-muted px-2">Page {page}</span>
      <button
        onClick={() => navigate(page + 1)}
        disabled={!hasNextPage}
        className={cn(
          "flex items-center gap-1 px-3 py-1.5 rounded-md border",
          "border-viola-border bg-viola-surface hover:border-viola-accent hover:text-viola-accent",
          "transition-colors disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:border-viola-border disabled:hover:text-viola-muted"
        )}
      >
        Next
        <ChevronRight className="h-3.5 w-3.5" />
      </button>
    </div>
  );
}

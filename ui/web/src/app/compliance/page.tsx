import { FrameworkCard } from "@/components/compliance/framework-card";
import { ControlsTable } from "@/components/compliance/controls-table";
import { AuditTrail } from "@/components/compliance/audit-trail";
import {
  MOCK_COMPLIANCE_SCORES,
  MOCK_CONTROLS,
  MOCK_AUDIT_EVENTS,
} from "@/lib/mock-compliance";
import { Shield } from "lucide-react";
import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Compliance",
};

export default function CompliancePage() {
  const scores = MOCK_COMPLIANCE_SCORES;
  const controls = MOCK_CONTROLS;
  const auditEvents = MOCK_AUDIT_EVENTS;

  const totalPassing = scores.reduce((s, f) => s + f.passing, 0);
  const totalFailing = scores.reduce((s, f) => s + f.failing, 0);
  const totalControls = scores.reduce((s, f) => s + f.total, 0);

  return (
    <div className="px-6 py-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-xl font-bold text-viola-text">Compliance Dashboard</h1>
          <p className="text-xs text-viola-muted mt-0.5">
            {totalPassing}/{totalControls} controls passing across {scores.length} frameworks
          </p>
        </div>
        <div className="flex items-center gap-2 text-xs bg-viola-surface border border-viola-border px-3 py-1.5 rounded-md">
          <Shield className="h-3.5 w-3.5 text-viola-accent" />
          <span className="text-viola-muted">{totalFailing} failing</span>
        </div>
      </div>

      {/* Framework score cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {scores.map((score) => (
          <FrameworkCard key={score.framework} score={score} />
        ))}
      </div>

      {/* Controls table + Audit trail */}
      <div className="mt-6 grid grid-cols-1 xl:grid-cols-3 gap-6">
        <div className="xl:col-span-2">
          <ControlsTable controls={controls} />
        </div>
        <div>
          <AuditTrail events={auditEvents} />
        </div>
      </div>
    </div>
  );
}

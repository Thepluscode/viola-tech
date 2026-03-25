import { notFound } from "next/navigation";
import { getAlert } from "@/lib/api-client";
import { AlertDetail } from "@/components/alerts/alert-detail";
import { AlertUpdateForm } from "@/components/alerts/alert-update-form";
import type { Metadata } from "next";

export const dynamic = "force-dynamic";

interface PageProps {
  params: { id: string };
}

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  return { title: params.id };
}

export default async function AlertPage({ params }: PageProps) {
  const alert = await getAlert(params.id);
  if (!alert) notFound();

  return (
    <div className="px-6 py-6">
      <div className="grid grid-cols-1 xl:grid-cols-[1fr_300px] gap-6 max-w-6xl">
        <AlertDetail alert={alert} />
        <div className="xl:pt-[60px]">
          <AlertUpdateForm alert={alert} />
        </div>
      </div>
    </div>
  );
}

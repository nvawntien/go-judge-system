import { PageHeader } from "@/components/layout/page-header"

export default function Page() {
  return (
    <div>
      <PageHeader title="Problems" description="Placeholder for the Problems page." />
      <div className="rounded-lg border bg-white p-8 shadow-sm">
        <p className="text-slate-500">Content for Problems will go here.</p>
      </div>
    </div>
  )
}

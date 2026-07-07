import { PageHeader } from "@/components/layout/page-header"

export default function Page() {
  return (
    <div>
      <PageHeader title="System Logs" description="Placeholder for the System Logs page." />
      <div className="rounded-lg border bg-white p-8 shadow-sm">
        <p className="text-slate-500">Content for System Logs will go here.</p>
      </div>
    </div>
  )
}

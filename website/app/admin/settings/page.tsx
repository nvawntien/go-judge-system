import { PageHeader } from "@/components/layout/page-header"

export default function Page() {
  return (
    <div>
      <PageHeader title="Settings" description="Placeholder for the Settings page." />
      <div className="rounded-lg border bg-white p-8 shadow-sm">
        <p className="text-slate-500">Content for Settings will go here.</p>
      </div>
    </div>
  )
}

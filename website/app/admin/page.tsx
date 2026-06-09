import { PageHeader } from "@/components/layout/page-header"

export default function Page() {
  return (
    <div>
      <PageHeader title="Admin Dashboard" description="Placeholder for the Admin Dashboard page." />
      <div className="rounded-lg border bg-white p-8 shadow-sm">
        <p className="text-slate-500">Content for Admin Dashboard will go here.</p>
      </div>
    </div>
  )
}

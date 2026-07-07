import { PageHeader } from "@/components/layout/page-header"

export default function Page() {
  return (
    <div>
      <PageHeader title="Submissions" description="Placeholder for the Submissions page." />
      <div className="rounded-lg border bg-white p-8 shadow-sm">
        <p className="text-slate-500">Content for Submissions will go here.</p>
      </div>
    </div>
  )
}

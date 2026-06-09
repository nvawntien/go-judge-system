import { PageHeader } from "@/components/layout/page-header"

export default function Page() {
  return (
    <div>
      <PageHeader title="Ranking" description="Placeholder for the Ranking page." />
      <div className="rounded-lg border bg-white p-8 shadow-sm">
        <p className="text-slate-500">Content for Ranking will go here.</p>
      </div>
    </div>
  )
}

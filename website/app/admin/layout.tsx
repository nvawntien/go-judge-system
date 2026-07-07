import { AdminSidebar } from "@/components/layout/admin-sidebar"
import { AdminTopBar } from "@/components/layout/admin-top-bar"

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="grid min-h-screen w-full md:grid-cols-[256px_1fr]">
      <AdminSidebar />
      <div className="flex flex-col">
        <AdminTopBar />
        <main className="flex-1 p-4 lg:p-6 bg-slate-50">
          {children}
        </main>
      </div>
    </div>
  )
}

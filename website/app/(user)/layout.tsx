import { UserTopNav } from "@/components/layout/user-top-nav"

export default function UserLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="relative flex min-h-screen flex-col bg-slate-50 dark:bg-slate-950 transition-colors">
      <UserTopNav />
      <main className="flex-1">
        <div className="container mx-auto px-4 md:px-6 pt-6 pb-8">
          {children}
        </div>
      </main>
    </div>
  )
}

import { UserTopNav } from "@/components/layout/user-top-nav"

export default function UserLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="relative flex min-h-screen flex-col bg-slate-50 dark:bg-slate-950 transition-colors">
      <UserTopNav />
      <main className="flex-1 mx-auto w-full max-w-[1440px] px-4 sm:px-5 lg:px-6 pt-4 pb-6">
        {children}
      </main>
    </div>
  )
}

import Link from "next/link"
import { USER_NAV_ITEMS } from "@/lib/constants/navigation"

export function UserTopNav() {
  return (
    <header className="sticky top-0 z-50 w-full border-b bg-white">
      <div className="container mx-auto flex h-16 items-center px-4 md:px-6">
        <Link href="/" className="mr-6 flex items-center space-x-2">
          <span className="text-xl font-bold text-purple-600">JudgeHub</span>
        </Link>
        <nav className="flex items-center space-x-6 text-sm font-medium text-slate-600">
          {USER_NAV_ITEMS.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              className="transition-colors hover:text-purple-600"
            >
              {item.name}
            </Link>
          ))}
        </nav>
        <div className="ml-auto flex items-center space-x-4">
          <div className="h-8 w-8 rounded-full bg-slate-200"></div>
        </div>
      </div>
    </header>
  )
}

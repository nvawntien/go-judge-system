"use client";

import Link from "next/link"
import { usePathname } from "next/navigation"
import { USER_NAV_ITEMS } from "@/lib/constants/navigation"
import { Search, Bell, ChevronDown } from "lucide-react"
import { ThemeToggle } from "@/components/theme/theme-toggle"

export function UserTopNav() {
  const pathname = usePathname()

  return (
    <header className="sticky top-0 z-50 w-full border-b border-slate-200 dark:border-slate-800/80 bg-white dark:bg-slate-950 transition-colors">
      <div className="mx-auto flex h-16 w-full max-w-[1440px] items-center px-4 sm:px-5 lg:px-6">
        
        {/* Brand Area */}
        <Link href="/" className="mr-6 lg:mr-12 flex items-center space-x-2 shrink-0">
          <span className="text-purple-600 dark:text-violet-500 font-bold text-xl tracking-tighter">{`{<>}`}</span>
          <span className="text-xl font-bold text-slate-900 dark:text-slate-100">JudgeHub</span>
        </Link>
        
        {/* Navigation Menu */}
        <nav className="hidden md:flex items-center space-x-4 lg:space-x-8 h-full overflow-hidden">
          {USER_NAV_ITEMS.map((item) => {
            const isActive = pathname === item.href || pathname?.startsWith(item.href + '/');
            return (
              <Link
                key={item.href}
                href={item.href}
                className={`flex items-center h-full text-sm font-medium transition-colors border-b-2 ${
                  isActive 
                    ? "border-purple-600 text-purple-600 dark:border-violet-500 dark:text-violet-400" 
                    : "border-transparent text-slate-500 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-100"
                }`}
              >
                {item.name}
              </Link>
            )
          })}
        </nav>
        
        {/* Right Side Actions */}
        <div className="ml-auto flex items-center space-x-3 sm:space-x-4 lg:space-x-6 shrink-0">
          
          {/* Search Input */}
          <div className="relative hidden lg:flex items-center">
            <Search className="absolute left-3 h-4 w-4 text-slate-400 dark:text-slate-500" />
            <input 
              type="text" 
              placeholder="Search problems..." 
              className="h-9 w-48 xl:w-64 rounded-full border border-slate-200 dark:border-slate-800/80 bg-slate-50 dark:bg-slate-900/50 pl-10 pr-12 text-sm text-slate-900 dark:text-slate-100 outline-none placeholder:text-slate-400 dark:placeholder:text-slate-500 focus:border-purple-600 dark:focus:border-violet-500 focus:ring-1 focus:ring-purple-600 dark:focus:ring-violet-500 transition-all"
            />
            <div className="absolute right-2 flex items-center rounded border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 px-1.5 h-6 text-[10px] font-medium text-slate-400 dark:text-slate-500">
              ⌘K
            </div>
          </div>
          <div className="flex lg:hidden items-center text-slate-500 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-100 cursor-pointer transition-colors">
             <Search className="h-5 w-5" />
          </div>

          {/* Theme Toggle */}
          <ThemeToggle />

          {/* Notifications */}
          <button className="relative text-slate-500 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-100 transition-colors">
            <Bell className="h-5 w-5" />
            <span className="absolute -top-1 -right-1 flex h-4 w-4 items-center justify-center rounded-full bg-purple-600 dark:bg-violet-600 text-[9px] font-bold text-white border border-white dark:border-slate-950">
              3
            </span>
          </button>

          {/* User Profile */}
          <Link 
            href="/profile"
            aria-label="Open profile"
            className="flex items-center gap-2 cursor-pointer group rounded-full sm:rounded-lg sm:pl-1 sm:pr-2 sm:py-1 hover:bg-slate-100 dark:hover:bg-slate-800/50 transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-violet-500 focus-visible:ring-offset-2 dark:focus-visible:ring-offset-slate-950"
          >
            <div className="h-8 w-8 overflow-hidden rounded-full border border-slate-200 dark:border-slate-800 bg-slate-100 dark:bg-slate-800 flex items-center justify-center shrink-0">
               <svg viewBox="0 0 36 36" fill="none" role="img" xmlns="http://www.w3.org/2000/svg" width="32" height="32" className="opacity-90 dark:opacity-80">
                 <mask id="mask__beam" maskUnits="userSpaceOnUse" x="0" y="0" width="36" height="36">
                   <rect width="36" height="36" rx="72" fill="#FFFFFF"></rect>
                 </mask>
                 <g mask="url(#mask__beam)">
                   <rect width="36" height="36" fill="#f8fafc"></rect>
                   <rect x="0" y="0" width="36" height="36" transform="translate(4 4) rotate(241 18 18) scale(1)" fill="#e2e8f0" rx="36"></rect>
                   <g transform="translate(0 0) rotate(4 18 18)">
                     <path d="M15 19c2 1 4 1 6 0" stroke="#475569" fill="none" strokeLinecap="round"></path>
                     <rect x="10" y="14" width="1.5" height="2" rx="1" stroke="none" fill="#475569"></rect>
                     <rect x="24" y="14" width="1.5" height="2" rx="1" stroke="none" fill="#475569"></rect>
                   </g>
                 </g>
               </svg>
            </div>
            <span className="hidden sm:block text-sm font-medium text-slate-700 group-hover:text-slate-900 dark:text-slate-300 dark:group-hover:text-slate-100 transition-colors">Alex</span>
            <ChevronDown className="hidden sm:block h-4 w-4 text-slate-400 group-hover:text-slate-700 dark:text-slate-500 dark:group-hover:text-slate-300 transition-colors" />
          </Link>

        </div>
      </div>
    </header>
  )
}

"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Search } from "lucide-react";

import { MobileNav } from "@/components/layout/mobile-nav";
import { mainNavItems, manageNavItem } from "@/components/layout/nav-items";
import { ThemeToggle } from "@/components/layout/theme-toggle";
import { UserMenu } from "@/components/layout/user-menu";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

function isActivePath(pathname: string, href: string) {
  return pathname === href || pathname.startsWith(`${href}/`);
}

export function AppNavbar() {
  const pathname = usePathname();

  return (
    <header className="sticky top-0 z-40 border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/80">
      <div className="container flex h-16 items-center gap-3 sm:gap-4">
        <MobileNav />

        <Link href="/" className="flex min-w-max items-center gap-3">
          <div className="flex h-9 w-9 items-center justify-center rounded-md bg-primary text-sm font-semibold text-primary-foreground">
            GJ
          </div>
          <div className="hidden leading-tight sm:block">
            <span className="block text-sm font-semibold">Go Judge System</span>
            <span className="block text-xs text-muted-foreground">Online judge</span>
          </div>
        </Link>

        <nav className="hidden items-center gap-1 md:flex">
          {mainNavItems.map((item) => {
            const active = isActivePath(pathname, item.href);
            return (
              <Button
                key={item.href}
                asChild
                variant="ghost"
                className={cn(
                  "h-9 rounded-none border-b-2 border-transparent px-3 text-sm text-muted-foreground hover:text-foreground",
                  active && "border-primary bg-primary/10 text-primary hover:bg-primary/10 hover:text-primary",
                )}
              >
                <Link href={item.href} aria-current={active ? "page" : undefined}>
                  {item.label}
                </Link>
              </Button>
            );
          })}
          <Button
            asChild
            variant="ghost"
            className={cn(
              "h-9 rounded-none border-b-2 border-transparent px-3 text-sm text-muted-foreground hover:text-foreground",
              isActivePath(pathname, manageNavItem.href) && "border-primary bg-primary/10 text-primary hover:bg-primary/10 hover:text-primary",
            )}
          >
            <Link
              href={manageNavItem.href}
              aria-current={isActivePath(pathname, manageNavItem.href) ? "page" : undefined}
            >
              {manageNavItem.label}
            </Link>
          </Button>
        </nav>

        <div className="ml-auto flex items-center gap-1 sm:gap-2">
          <div className="relative hidden w-[min(28vw,320px)] lg:block">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input className="h-9 pl-9" placeholder="Search problems, contests, submissions" />
          </div>
          <ThemeToggle />
          <UserMenu />
        </div>
      </div>
    </header>
  );
}

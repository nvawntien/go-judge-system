"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Menu, Search } from "lucide-react";

import { mainNavItems, manageNavItem } from "@/components/layout/nav-items";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";

export function MobileNav() {
  const pathname = usePathname();

  return (
    <Sheet>
      <SheetTrigger asChild>
        <Button variant="ghost" size="icon" className="md:hidden" aria-label="Open navigation">
          <Menu className="h-5 w-5" />
        </Button>
      </SheetTrigger>
      <SheetContent>
        <SheetHeader>
          <SheetTitle>Go Judge System</SheetTitle>
          <SheetDescription>Navigate the online judge workspace.</SheetDescription>
        </SheetHeader>

        <div className="mt-6 space-y-6">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input className="pl-9" placeholder="Search problems or submissions" />
          </div>

          <nav className="space-y-1">
            {[...mainNavItems, manageNavItem].map((item) => {
              const active = pathname === item.href || pathname.startsWith(`${item.href}/`);

              return (
                <SheetClose asChild key={item.href}>
                  <Link
                    href={item.href}
                    aria-current={active ? "page" : undefined}
                    className={
                      active
                        ? "flex h-10 items-center rounded-md border border-primary/20 bg-primary/10 px-3 text-sm font-medium text-primary transition-colors"
                        : "flex h-10 items-center rounded-md px-3 text-sm font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
                    }
                  >
                    {item.label}
                  </Link>
                </SheetClose>
              );
            })}
          </nav>

          <Separator />

          <div className="rounded-lg border bg-muted/40 p-3">
            <p className="text-sm font-medium text-foreground">Guest workspace</p>
            <p className="mt-1 text-sm text-muted-foreground">User actions stay as placeholders until the auth milestone begins.</p>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  );
}

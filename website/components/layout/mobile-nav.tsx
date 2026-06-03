"use client";

import Link from "next/link";
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
            {[...mainNavItems, manageNavItem].map((item) => (
              <SheetClose asChild key={item.href}>
                <Link
                  href={item.href}
                  className="flex h-10 items-center rounded-md px-3 text-sm font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
                >
                  {item.label}
                </Link>
              </SheetClose>
            ))}
          </nav>

          <Separator />

          <p className="text-sm text-muted-foreground">User menu and authenticated navigation will connect after auth flow.</p>
        </div>
      </SheetContent>
    </Sheet>
  );
}

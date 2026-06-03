import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";

export function AppShellPlaceholder() {
  return (
    <main className="app-surface min-h-screen text-foreground">
      <header className="border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/80">
        <div className="container flex h-16 items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-md bg-primary text-sm font-semibold text-primary-foreground">
              GJ
            </div>
            <div className="space-y-0.5">
              <p className="text-sm font-semibold tracking-tight">Go Judge System</p>
              <p className="text-xs text-muted-foreground">Unified top navigation foundation</p>
            </div>
          </div>
          <nav className="hidden items-center gap-6 text-sm text-muted-foreground md:flex">
            <span className="font-medium text-foreground">Problems</span>
            <span>Contests</span>
            <span>Submissions</span>
            <span>Ranking</span>
          </nav>
          <div className="hidden w-full max-w-xs md:block">
            <Input disabled value="Global search will arrive in the next milestone" />
          </div>
        </div>
      </header>

      <section className="container py-10">
        <div className="grid gap-6 lg:grid-cols-[1.2fr_0.8fr]">
          <Card>
            <CardHeader>
              <CardTitle>Frontend foundation is ready for the app shell milestone</CardTitle>
              <CardDescription>
                Next.js App Router, TypeScript, Tailwind CSS, shadcn/ui primitives, and light or dark theme tokens are in place.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="grid gap-3 sm:grid-cols-2">
                <div className="app-panel p-4">
                  <p className="text-xs uppercase text-muted-foreground">Stack</p>
                  <p className="mt-2 text-sm font-medium">Next.js + TypeScript + Tailwind + shadcn/ui</p>
                </div>
                <div className="app-panel p-4">
                  <p className="text-xs uppercase text-muted-foreground">Layout direction</p>
                  <p className="mt-2 text-sm font-medium">Top navigation, no default sidebar</p>
                </div>
              </div>
              <div className="flex flex-wrap gap-3">
                <Button>Primary Action</Button>
                <Button variant="outline">Secondary Action</Button>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Next implementation steps</CardTitle>
              <CardDescription>
                This placeholder intentionally stops before problem pages, auth flows, or live API integration.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-3 text-sm text-muted-foreground">
              <p>1. Build the unified app shell.</p>
              <p>2. Add typed API client utilities against the gateway contract.</p>
              <p>3. Start the problem list page inside the approved layout system.</p>
            </CardContent>
          </Card>
        </div>
      </section>
    </main>
  );
}

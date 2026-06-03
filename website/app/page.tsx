import { AppShell } from "@/components/layout/app-shell";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

export default function HomePage() {
  return (
    <AppShell>
      <div className="grid gap-6 lg:grid-cols-[1.2fr_0.8fr]">
        <Card>
          <CardHeader>
            <CardTitle>Unified app shell is ready for feature pages</CardTitle>
            <CardDescription>
              Top navigation, global search UI, theme toggle, user menu placeholder, and mobile navigation are in place.
            </CardDescription>
          </CardHeader>
          <CardContent className="grid gap-3 sm:grid-cols-2">
            <div className="app-panel p-4">
              <p className="text-xs uppercase text-muted-foreground">Layout</p>
              <p className="mt-2 text-sm font-medium">Unified top navigation, no default sidebar</p>
            </div>
            <div className="app-panel p-4">
              <p className="text-xs uppercase text-muted-foreground">Theme</p>
              <p className="mt-2 text-sm font-medium">Light default with dark mode support</p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Next milestone</CardTitle>
            <CardDescription>
              This shell intentionally stops before problem lists, auth flows, submissions, or admin screens.
            </CardDescription>
          </CardHeader>
        </Card>
      </div>
    </AppShell>
  );
}

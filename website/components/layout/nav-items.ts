export type NavItem = {
  label: string;
  href: string;
};

export const mainNavItems: NavItem[] = [
  { label: "Problems", href: "/problems" },
  { label: "Contests", href: "/contests" },
  { label: "Submissions", href: "/submissions" },
  { label: "Ranking", href: "/ranking" },
];

export const manageNavItem: NavItem = {
  label: "Manage",
  href: "/admin",
};

import type { Role } from '@/lib/types';

const ROLE_LEVEL: Record<Role, number> = {
  user: 1,
  contributor: 2,
  moderator: 3,
  admin: 4,
};

export function roleAtLeast(role: Role | undefined | null, minimum: Role): boolean {
  if (!role) return false;
  return ROLE_LEVEL[role] >= ROLE_LEVEL[minimum];
}

/**
 * Contributor Workspace is a dedicated authoring surface, not a projection of
 * every role that inherits contributor-level backend capabilities.
 */
export function isContributorWorkspaceUser(role: Role | undefined | null): role is 'contributor' {
  return role === 'contributor';
}

export function roleLabel(role: Role): string {
  return role.charAt(0).toUpperCase() + role.slice(1);
}

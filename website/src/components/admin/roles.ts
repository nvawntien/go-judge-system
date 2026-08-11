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

export function roleLabel(role: Role): string {
  return role.charAt(0).toUpperCase() + role.slice(1);
}

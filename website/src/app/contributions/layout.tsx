import type { ReactNode } from 'react';
import { ContributorShell } from '@/components/contributions/ContributorShell';

export default function ContributionsLayout({ children }: { children: ReactNode }) {
  return <ContributorShell>{children}</ContributorShell>;
}

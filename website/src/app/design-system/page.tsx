'use client';

import { AppShell, PageHeading } from '@/components/AppShell';
import { Card, CodePath, Icon } from '@/components/ui';

const TOKENS = [
  '--bg',
  '--surface',
  '--surface2',
  '--surface3',
  '--border',
  '--border2',
  '--accent',
  '--accent-fg',
  '--accent-soft',
  '--success',
  '--warn',
  '--error',
];

export default function DesignSystemPage() {
  return (
    <AppShell maxWidth={900}>
      <PageHeading title="Design system notes" subtitle="The reusable AstraCode component language, in both themes." />

      <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        <Card padding={20}>
          <h2 style={{ margin: '0 0 4px', fontSize: 14, fontWeight: 650 }}>Foundations</h2>
          <p style={{ margin: '0 0 14px', fontSize: 12.5, color: 'var(--text2)', lineHeight: 1.6 }}>
            UI type is <strong>Instrument Sans</strong>; everything technical — code, IDs, runtimes,
            ratings — is <strong>JetBrains Mono</strong>. All colours are CSS variables, so light and
            dark are the same product. Purple is reserved for active navigation, primary actions,
            focus, progress, and the current user. Status never relies on colour alone — every verdict
            pairs an icon and a text label.
          </p>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
            {TOKENS.map((token) => (
              <span
                key={token}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 7,
                  border: '1px solid var(--border)',
                  borderRadius: 8,
                  padding: '6px 10px',
                }}
              >
                <span
                  style={{
                    width: 16,
                    height: 16,
                    borderRadius: 5,
                    background: `var(${token})`,
                    border: '1px solid var(--border2)',
                  }}
                />
                <span style={{ fontFamily: 'var(--font-mono)', fontSize: 10.5, color: 'var(--text2)' }}>
                  {token}
                </span>
              </span>
            ))}
          </div>
        </Card>

        <Card padding={20}>
          <h2 style={{ margin: '0 0 12px', fontSize: 14, fontWeight: 650 }}>Buttons &amp; inputs</h2>
          <div
            style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 10, marginBottom: 14 }}
          >
            <button type="button" className="ac-button ac-button-primary">
              Primary
            </button>
            <button type="button" className="ac-button ac-button-secondary">
              Secondary
            </button>
            <button type="button" className="ac-button ac-button-ghost">
              Ghost
            </button>
            <button type="button" className="ac-button ac-button-danger">
              Destructive
            </button>
            <button
              type="button"
              aria-label="Icon button example"
              className="ac-icon-button"
            >
              <Icon.Search />
            </button>
          </div>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 10 }}>
            <input placeholder="Text input" aria-label="Example input" className="ac-field" style={input} />
            <select aria-label="Example select" style={{ ...input, width: 130 }}>
              <option>Select</option>
            </select>
            <input
              placeholder="Invalid input"
              aria-label="Example invalid input"
              aria-invalid="true"
              style={{ ...input, width: 150, borderColor: 'var(--error)' }}
            />
          </div>
        </Card>

        <Card padding={20}>
          <h2 style={{ margin: '0 0 12px', fontSize: 14, fontWeight: 650 }}>Badges</h2>
          <div style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 8 }}>
            <Badge color="var(--easy)" bg="var(--success-bg)">
              Easy
            </Badge>
            <Badge color="var(--med)" bg="var(--warn-bg)">
              Medium
            </Badge>
            <Badge color="var(--hard)" bg="var(--error-bg)">
              Hard
            </Badge>
            <span style={{ width: 1, height: 18, background: 'var(--border2)' }} />
            {['Go', 'C++', 'Python', 'Java'].map((language) => (
              <span
                key={language}
                style={{
                  fontSize: 11.5,
                  color: 'var(--text2)',
                  background: 'var(--surface2)',
                  border: '1px solid var(--border)',
                  borderRadius: 5,
                  padding: '1px 8px',
                  fontFamily: 'var(--font-mono)',
                }}
              >
                {language}
              </span>
            ))}
            <span style={{ width: 1, height: 18, background: 'var(--border2)' }} />
            <Badge color="var(--success)" bg="var(--success-bg)">
              ✓ Accepted
            </Badge>
            <Badge color="var(--error)" bg="var(--error-bg)">
              ✕ Wrong Answer
            </Badge>
            <Badge color="var(--warn)" bg="var(--warn-bg)">
              ◷ Time Limit
            </Badge>
          </div>
        </Card>

        <Card padding={20}>
          <h2 style={{ margin: '0 0 4px', fontSize: 14, fontWeight: 650 }}>The code-path motif</h2>
          <p style={{ margin: '0 0 14px', fontSize: 12.5, color: 'var(--text2)', lineHeight: 1.6 }}>
            AstraCode&apos;s signature: thin lines connecting small nodes, read left to right like
            execution through test cases. It appears in the logo, the streak chip, judge loading
            states, and empty states — never as pure decoration. Filled nodes are progress; hollow
            nodes are what&apos;s ahead.
          </p>
          <CodePath total={5} done={3} width={220} height={28} />
          <p style={{ margin: '14px 0 0', fontSize: 12.5, color: 'var(--text2)', lineHeight: 1.6 }}>
            <strong>Interaction notes:</strong> theme follows the OS by default and persists once
            chosen; every animation respects{' '}
            <span style={{ fontFamily: 'var(--font-mono)', fontSize: 11.5 }}>
              prefers-reduced-motion
            </span>
            ; pages enter with one staggered fade (never per-card); tables use hover rows and 44px+
            touch targets on mobile; the workspace split is draggable and stacks vertically under
            1020px.
          </p>
        </Card>
      </div>
    </AppShell>
  );
}

function Badge({
  children,
  color,
  bg,
}: {
  children: React.ReactNode;
  color: string;
  bg: string;
}) {
  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 5,
        fontSize: 11.5,
        fontWeight: 600,
        color,
        background: bg,
        borderRadius: 6,
        padding: '2px 9px',
      }}
    >
      {children}
    </span>
  );
}

const input: React.CSSProperties = {
  height: 38,
  borderRadius: 8,
  border: '1px solid var(--border2)',
  background: 'var(--surface)',
  padding: '0 12px',
  fontSize: 13,
  color: 'var(--text)',
  width: 180,
};

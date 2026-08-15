'use client';

import { Fragment, useMemo } from 'react';

/**
 * Problem statements come back as a plain string. Rather than pull in a
 * markdown runtime, this renders the small subset authors actually use:
 * paragraphs, `inline code`, **bold**, `#`-headings and `-`/`*` bullet lists.
 */

const CODE_STYLE: React.CSSProperties = {
  fontFamily: 'var(--font-mono)',
  fontSize: 12,
  background: 'var(--surface2)',
  border: '1px solid var(--border)',
  borderRadius: 4,
  padding: '1px 5px',
};

function renderInline(text: string, keyPrefix: string) {
  const parts = text.split(/(`[^`]*`|\*\*[^*]+\*\*)/g);
  return parts.map((part, index) => {
    const key = `${keyPrefix}-${index}`;
    if (part.startsWith('`') && part.endsWith('`') && part.length > 1) {
      return (
        <code key={key} style={CODE_STYLE}>
          {part.slice(1, -1)}
        </code>
      );
    }
    if (part.startsWith('**') && part.endsWith('**') && part.length > 3) {
      return <strong key={key}>{part.slice(2, -2)}</strong>;
    }
    return <Fragment key={key}>{part}</Fragment>;
  });
}

export function RichText({ text, muted = false }: { text: string; muted?: boolean }) {
  const blocks = useMemo(() => (text ?? '').replace(/\r\n/g, '\n').split(/\n{2,}/), [text]);

  return (
    <>
      {blocks.map((block, blockIndex) => {
        const trimmed = block.trim();
        if (!trimmed) return null;

        const heading = /^(#{1,4})\s+(.*)$/.exec(trimmed);
        if (heading) {
          return (
            <h3 key={blockIndex} className="ac-rich-text-heading">{heading[2]}</h3>
          );
        }

        const lines = trimmed.split('\n');
        const isList = lines.every((line) => /^\s*[-*]\s+/.test(line));
        if (isList) {
          return (
            <ul
              key={blockIndex}
              style={{
                margin: '0 0 14px',
                paddingLeft: 20,
                fontSize: 13,
                lineHeight: 1.65,
                color: muted ? 'var(--text2)' : 'var(--text)',
                maxWidth: '70ch',
              }}
            >
              {lines.map((line, lineIndex) => (
                <li key={lineIndex} style={{ marginBottom: 4 }}>
                  {renderInline(line.replace(/^\s*[-*]\s+/, ''), `${blockIndex}-${lineIndex}`)}
                </li>
              ))}
            </ul>
          );
        }

        return (
          <p
            key={blockIndex}
            className="ac-statement-paragraph"
            style={{
              color: muted ? 'var(--text2)' : 'var(--text)',
              whiteSpace: 'pre-wrap',
            }}
          >
            {renderInline(trimmed, String(blockIndex))}
          </p>
        );
      })}
    </>
  );
}

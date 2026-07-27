'use client';

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { tokenizeLine } from '@/lib/highlight';
import type { LanguageCode } from '@/lib/types';

interface CodeEditorProps {
  value: string;
  onChange: (value: string) => void;
  language: LanguageCode;
  fontSize: number;
  tabSize: number;
  readOnly?: boolean;
  /** 1-based line to scroll to and highlight (compiler/runtime error jumps). */
  highlightLine?: number | null;
}

/**
 * Textarea + highlighted overlay. Both layers share font metrics and padding so
 * the caret lands exactly on the painted glyphs; the textarea's own text is
 * transparent and only its caret and selection show through.
 */
export function CodeEditor({
  value,
  onChange,
  language,
  fontSize,
  tabSize,
  readOnly = false,
  highlightLine = null,
}: CodeEditorProps) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const [caret, setCaret] = useState({ line: 1, column: 1 });

  const lines = useMemo(() => value.split('\n'), [value]);
  const highlighted = useMemo(
    () => lines.map((line) => tokenizeLine(line, language)),
    [lines, language],
  );

  const syncCaret = useCallback(() => {
    const el = textareaRef.current;
    if (!el) return;
    const upTo = el.value.slice(0, el.selectionStart);
    const parts = upTo.split('\n');
    setCaret({ line: parts.length, column: parts[parts.length - 1].length + 1 });
  }, []);

  useEffect(() => {
    if (!highlightLine) return;
    const el = textareaRef.current;
    const scroller = scrollRef.current;
    if (!el || !scroller) return;

    const offset = lines.slice(0, highlightLine - 1).reduce((sum, line) => sum + line.length + 1, 0);
    el.focus();
    el.setSelectionRange(offset, offset + (lines[highlightLine - 1]?.length ?? 0));
    scroller.scrollTop = Math.max(0, (highlightLine - 4) * fontSize * 1.65);
    syncCaret();
  }, [highlightLine, lines, fontSize, syncCaret]);

  const onKeyDown = (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key !== 'Tab' || readOnly) return;
    event.preventDefault();

    const el = event.currentTarget;
    const { selectionStart, selectionEnd } = el;
    const indent = ' '.repeat(tabSize);
    const next = `${value.slice(0, selectionStart)}${indent}${value.slice(selectionEnd)}`;
    onChange(next);

    requestAnimationFrame(() => {
      el.selectionStart = el.selectionEnd = selectionStart + indent.length;
      syncCaret();
    });
  };

  const lineHeight = 1.65;
  const gutterWidth = `${Math.max(2, String(lines.length).length) * 0.62 + 1.6}em`;

  const textLayer: React.CSSProperties = {
    margin: 0,
    padding: '10px 16px 10px 12px',
    fontFamily: 'var(--font-mono)',
    fontSize,
    lineHeight,
    whiteSpace: 'pre',
    wordBreak: 'normal',
    overflowWrap: 'normal',
    tabSize,
  };

  return (
    <>
      <div
        ref={scrollRef}
        style={{
          flex: 1,
          minHeight: 120,
          overflow: 'auto',
          background: 'var(--code-bg)',
          position: 'relative',
        }}
      >
        <div style={{ display: 'flex', width: 'max-content', minWidth: '100%', minHeight: '100%' }}>
          <div
            aria-hidden="true"
            style={{
              position: 'sticky',
              left: 0,
              zIndex: 2,
              background: 'var(--code-bg)',
              borderRight: '1px solid var(--code-line)',
              padding: '10px 8px 10px 10px',
              textAlign: 'right',
              minWidth: gutterWidth,
              boxSizing: 'content-box',
              fontFamily: 'var(--font-mono)',
              fontSize,
              lineHeight,
              color: 'var(--gutter)',
              userSelect: 'none',
              flexShrink: 0,
            }}
          >
            {lines.map((_, index) => (
              <div
                key={index}
                style={{
                  color: index + 1 === caret.line ? 'var(--accent-fg)' : 'var(--gutter)',
                  fontWeight: index + 1 === caret.line ? 600 : 400,
                }}
              >
                {index + 1}
              </div>
            ))}
          </div>

          <div style={{ position: 'relative', flex: 1, minWidth: 0 }}>
            <pre aria-hidden="true" style={{ ...textLayer, color: 'var(--code-fg)' }}>
              {highlighted.map((tokens, index) => (
                <div
                  key={index}
                  style={{
                    background: index + 1 === caret.line ? 'var(--code-line)' : 'transparent',
                    minHeight: `${fontSize * lineHeight}px`,
                  }}
                >
                  {tokens.length === 0 ? (
                    <span> </span>
                  ) : (
                    tokens.map((token, tokenIndex) => (
                      <span key={tokenIndex} style={{ color: token.color }}>
                        {token.text}
                      </span>
                    ))
                  )}
                </div>
              ))}
            </pre>

            <textarea
              ref={textareaRef}
              value={value}
              readOnly={readOnly}
              spellCheck={false}
              autoCapitalize="off"
              autoCorrect="off"
              aria-label={`Code editor, ${language} solution`}
              onChange={(event) => {
                onChange(event.target.value);
                syncCaret();
              }}
              onKeyUp={syncCaret}
              onClick={syncCaret}
              onSelect={syncCaret}
              onKeyDown={onKeyDown}
              style={{
                ...textLayer,
                position: 'absolute',
                inset: 0,
                width: '100%',
                height: '100%',
                boxSizing: 'border-box',
                border: 'none',
                outline: 'none',
                resize: 'none',
                background: 'transparent',
                color: 'transparent',
                caretColor: 'var(--accent)',
                overflow: 'hidden',
              }}
            />
          </div>
        </div>
      </div>

      <div
        aria-label="Editor status"
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 14,
          padding: '5px 14px',
          borderTop: '1px solid var(--border)',
          background: 'var(--surface)',
          flexShrink: 0,
          fontFamily: 'var(--font-mono)',
          fontSize: 11,
          color: 'var(--text2)',
          overflowX: 'auto',
          whiteSpace: 'nowrap',
        }}
      >
        <span>
          Ln {caret.line}, Col {caret.column}
        </span>
        <span>Tab: {tabSize}</span>
        <span>{lines.length} lines</span>
        <span>{new Blob([value]).size} B</span>
      </div>
    </>
  );
}

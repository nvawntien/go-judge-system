'use client';

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { tokenizeLine } from '@/lib/highlight';
import { handleCodeEditorKeyDown } from '@/lib/codeEditorKeys';
import type { CodeDiagnostic, LanguageCode } from '@/lib/types';

interface CodeEditorProps {
  value: string;
  onChange: (value: string) => void;
  language: LanguageCode;
  fontSize: number;
  tabSize: number;
  readOnly?: boolean;
  /** 1-based line to scroll to and highlight (compiler/runtime error jumps). */
  highlightLine?: number | null;
  diagnostics?: CodeDiagnostic[];
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
  diagnostics = [],
}: CodeEditorProps) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const [caret, setCaret] = useState({ line: 1, column: 1 });

  const lines = useMemo(() => value.split('\n'), [value]);
  const highlighted = useMemo(
    () => lines.map((line) => tokenizeLine(line, language)),
    [lines, language],
  );
  const diagnosticsByLine = useMemo(() => {
    const byLine = new Map<number, CodeDiagnostic[]>();
    for (const diagnostic of diagnostics) {
      if (!diagnostic.line || diagnostic.line < 1) continue;
      const current = byLine.get(diagnostic.line) ?? [];
      current.push(diagnostic);
      byLine.set(diagnostic.line, current);
    }
    return byLine;
  }, [diagnostics]);

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
    if (readOnly) return;
    handleCodeEditorKeyDown({
      event,
      value,
      onChange,
      tabSize,
      syncCaret,
    });
  };

  const lineHeight = 1.65;
  const gutterWidth = `${Math.max(2, String(lines.length).length) * 0.58 + 0.95}em`;

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
              padding: '10px 5px 10px 7px',
              textAlign: 'right',
              minWidth: gutterWidth,
              boxSizing: 'content-box',
              fontFamily: 'var(--font-mono)',
              fontSize: Math.max(11, fontSize - 1),
              lineHeight,
              color: 'var(--gutter)',
              userSelect: 'none',
              flexShrink: 0,
            }}
          >
            {lines.map((_, index) => (
              (() => {
                const lineNo = index + 1;
                const lineDiagnostics = diagnosticsByLine.get(lineNo) ?? [];
                const hasDiagnostic = lineDiagnostics.length > 0;
                return (
                  <div
                    key={index}
                    title={lineDiagnostics.map((item) => item.message).join('\n')}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'flex-end',
                      gap: 3,
                      color: hasDiagnostic
                        ? 'var(--error)'
                        : lineNo === caret.line
                          ? 'var(--accent-fg)'
                          : 'var(--gutter)',
                      fontWeight: hasDiagnostic || lineNo === caret.line ? 600 : 400,
                    }}
                  >
                    {hasDiagnostic && (
                      <span
                        role="img"
                        aria-label="Error"
                        style={{
                          width: 10,
                          height: 10,
                          borderRadius: '50%',
                          background: 'var(--error-bg)',
                          color: 'var(--error)',
                          display: 'inline-flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          fontSize: 8,
                          lineHeight: 1,
                        }}
                      >
                        !
                      </span>
                    )}
                    {lineNo}
                  </div>
                );
              })()
            ))}
          </div>

          <div style={{ position: 'relative', flex: 1, minWidth: 0 }}>
            <pre aria-hidden="true" style={{ ...textLayer, color: 'var(--code-fg)' }}>
              {highlighted.map((tokens, index) => (
                <div
                  key={index}
                  title={(diagnosticsByLine.get(index + 1) ?? []).map((item) => item.message).join('\n')}
                  style={{
                    background:
                      (diagnosticsByLine.get(index + 1)?.length ?? 0) > 0
                        ? 'color-mix(in srgb, var(--error-bg) 45%, transparent)'
                        : index + 1 === caret.line
                          ? 'var(--code-line)'
                          : 'transparent',
                    boxShadow:
                      (diagnosticsByLine.get(index + 1)?.length ?? 0) > 0
                        ? 'inset 3px 0 0 var(--error)'
                        : 'none',
                    textDecoration:
                      (diagnosticsByLine.get(index + 1)?.length ?? 0) > 0
                        ? 'underline wavy var(--error)'
                        : 'none',
                    textDecorationSkipInk: 'none',
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

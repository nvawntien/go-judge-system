import type { KeyboardEvent } from 'react';

const pairs: Record<string, string> = {
  '(': ')',
  '[': ']',
  '{': '}',
  '"': '"',
  "'": "'",
  '`': '`',
};

const openingChars = new Set(Object.keys(pairs));
const closingChars = new Set(Object.values(pairs));
const indentOpeners = new Set(['{', '(', '[']);

interface CodeEditorKeyOptions {
  event: KeyboardEvent<HTMLTextAreaElement>;
  value: string;
  onChange: (value: string) => void;
  tabSize: number;
  syncCaret: () => void;
}

interface EditResult {
  value: string;
  selectionStart: number;
  selectionEnd: number;
}

export function handleCodeEditorKeyDown({
  event,
  value,
  onChange,
  tabSize,
  syncCaret,
}: CodeEditorKeyOptions): boolean {
  if (event.defaultPrevented || event.altKey || event.ctrlKey || event.metaKey) {
    return false;
  }

  const el = event.currentTarget;
  const { selectionStart, selectionEnd } = el;
  const key = event.key;
  let edit: EditResult | null = null;

  if (key === 'Tab') {
    edit = event.shiftKey
      ? outdentSelection(value, selectionStart, selectionEnd, tabSize)
      : indentSelection(value, selectionStart, selectionEnd, tabSize);
  } else if (key === 'Backspace') {
    edit = deletePairedCharacter(value, selectionStart, selectionEnd);
  } else if (key === 'Enter') {
    edit = insertIndentedNewline(value, selectionStart, selectionEnd, tabSize);
  } else if (closingChars.has(key) && shouldSkipClosingCharacter(value, selectionStart, selectionEnd, key)) {
    edit = {
      value,
      selectionStart: selectionStart + 1,
      selectionEnd: selectionStart + 1,
    };
  } else if (openingChars.has(key)) {
    edit = insertPair(value, selectionStart, selectionEnd, key, pairs[key]);
  }

  if (!edit) {
    return false;
  }

  event.preventDefault();
  onChange(edit.value);
  requestAnimationFrame(() => {
    el.selectionStart = edit.selectionStart;
    el.selectionEnd = edit.selectionEnd;
    syncCaret();
  });
  return true;
}

function insertPair(value: string, selectionStart: number, selectionEnd: number, open: string, close: string): EditResult {
  const before = value.slice(0, selectionStart);
  const selected = value.slice(selectionStart, selectionEnd);
  const after = value.slice(selectionEnd);

  if (selectionStart !== selectionEnd) {
    return {
      value: `${before}${open}${selected}${close}${after}`,
      selectionStart: selectionStart + open.length,
      selectionEnd: selectionEnd + open.length,
    };
  }

  return {
    value: `${before}${open}${close}${after}`,
    selectionStart: selectionStart + open.length,
    selectionEnd: selectionStart + open.length,
  };
}

function shouldSkipClosingCharacter(
  value: string,
  selectionStart: number,
  selectionEnd: number,
  key: string,
): boolean {
  if (selectionStart !== selectionEnd || value[selectionStart] !== key) {
    return false;
  }
  if ((key === '"' || key === "'" || key === '`') && isEscapedAt(value, selectionStart)) {
    return false;
  }
  return true;
}

function deletePairedCharacter(value: string, selectionStart: number, selectionEnd: number): EditResult | null {
  if (selectionStart !== selectionEnd || selectionStart === 0) {
    return null;
  }

  const previous = value[selectionStart - 1];
  const next = value[selectionStart];
  if (pairs[previous] !== next) {
    return null;
  }

  return {
    value: `${value.slice(0, selectionStart - 1)}${value.slice(selectionStart + 1)}`,
    selectionStart: selectionStart - 1,
    selectionEnd: selectionStart - 1,
  };
}

function insertIndentedNewline(
  value: string,
  selectionStart: number,
  selectionEnd: number,
  tabSize: number,
): EditResult {
  const before = value.slice(0, selectionStart);
  const after = value.slice(selectionEnd);
  const lineStart = before.lastIndexOf('\n') + 1;
  const currentIndent = getLineIndent(value.slice(lineStart, selectionStart));
  const indentUnit = ' '.repeat(tabSize);
  const previous = value[selectionStart - 1];
  const next = value[selectionEnd];

  if (selectionStart === selectionEnd && pairs[previous] === next && indentOpeners.has(previous)) {
    const innerIndent = currentIndent + indentUnit;
    return {
      value: `${before}\n${innerIndent}\n${currentIndent}${after}`,
      selectionStart: before.length + 1 + innerIndent.length,
      selectionEnd: before.length + 1 + innerIndent.length,
    };
  }

  const nextIndent = currentIndent + (indentOpeners.has(previous) ? indentUnit : '');
  return {
    value: `${before}\n${nextIndent}${after}`,
    selectionStart: before.length + 1 + nextIndent.length,
    selectionEnd: before.length + 1 + nextIndent.length,
  };
}

function indentSelection(value: string, selectionStart: number, selectionEnd: number, tabSize: number): EditResult {
  const indent = ' '.repeat(tabSize);
  if (selectionStart === selectionEnd) {
    return {
      value: `${value.slice(0, selectionStart)}${indent}${value.slice(selectionEnd)}`,
      selectionStart: selectionStart + indent.length,
      selectionEnd: selectionStart + indent.length,
    };
  }

  const range = getSelectedLineRange(value, selectionStart, selectionEnd);
  const block = value.slice(range.start, range.end);
  const lines = block.split('\n');
  const nextBlock = lines.map((line) => `${indent}${line}`).join('\n');

  return {
    value: `${value.slice(0, range.start)}${nextBlock}${value.slice(range.end)}`,
    selectionStart: selectionStart + indent.length,
    selectionEnd: selectionEnd + indent.length * lines.length,
  };
}

function outdentSelection(value: string, selectionStart: number, selectionEnd: number, tabSize: number): EditResult {
  const range = getSelectedLineRange(value, selectionStart, selectionEnd);
  const block = value.slice(range.start, range.end);
  const lines = block.split('\n');
  const removals: number[] = [];
  const nextBlock = lines
    .map((line) => {
      const removal = countOutdentChars(line, tabSize);
      removals.push(removal);
      return line.slice(removal);
    })
    .join('\n');

  const firstRemoval = removals[0] ?? 0;
  const removedBeforeStart =
    selectionStart > range.start ? Math.min(firstRemoval, selectionStart - range.start) : 0;
  const totalRemoved = removals.reduce((sum, count) => sum + count, 0);

  return {
    value: `${value.slice(0, range.start)}${nextBlock}${value.slice(range.end)}`,
    selectionStart: selectionStart - removedBeforeStart,
    selectionEnd: Math.max(selectionStart - removedBeforeStart, selectionEnd - totalRemoved),
  };
}

function getSelectedLineRange(value: string, selectionStart: number, selectionEnd: number) {
  const start = value.lastIndexOf('\n', Math.max(0, selectionStart - 1)) + 1;
  const effectiveEnd =
    selectionEnd > selectionStart && value[selectionEnd - 1] === '\n' ? selectionEnd - 1 : selectionEnd;
  const nextLineBreak = value.indexOf('\n', effectiveEnd);
  const end = nextLineBreak === -1 ? value.length : nextLineBreak;
  return { start, end };
}

function countOutdentChars(line: string, tabSize: number): number {
  if (line.startsWith('\t')) {
    return 1;
  }
  let spaces = 0;
  while (spaces < Math.min(tabSize, line.length) && line[spaces] === ' ') {
    spaces += 1;
  }
  return spaces;
}

function getLineIndent(linePrefix: string): string {
  return linePrefix.match(/^[\t ]*/)?.[0] ?? '';
}

function isEscapedAt(value: string, index: number): boolean {
  let slashCount = 0;
  for (let i = index - 1; i >= 0 && value[i] === '\\'; i -= 1) {
    slashCount += 1;
  }
  return slashCount % 2 === 1;
}

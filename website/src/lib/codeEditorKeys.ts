import type { KeyboardEvent } from 'react';
import type { LanguageCode } from './types';

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
const angleBracketLanguages = new Set<LanguageCode>(['CPP', 'JAVA']);

interface CodeEditorKeyOptions {
  event: KeyboardEvent<HTMLTextAreaElement>;
  value: string;
  onChange: (value: string) => void;
  language: LanguageCode;
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
  language,
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
    edit =
      deleteIndentationToPreviousTabStop(value, selectionStart, selectionEnd, tabSize) ??
      deletePairedCharacter(value, selectionStart, selectionEnd, language);
  } else if (key === 'Enter') {
    edit = insertIndentedNewline(value, selectionStart, selectionEnd, tabSize);
  } else if (
    (closingChars.has(key) || key === '>') &&
    shouldSkipClosingCharacter(value, selectionStart, selectionEnd, key, language)
  ) {
    edit = {
      value,
      selectionStart: selectionStart + 1,
      selectionEnd: selectionStart + 1,
    };
  } else {
    const close = getOpeningPair(value, selectionStart, selectionEnd, key, language);
    if (close) {
      edit = insertPair(value, selectionStart, selectionEnd, key, close);
    }
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

function getOpeningPair(
  value: string,
  selectionStart: number,
  selectionEnd: number,
  key: string,
  language: LanguageCode,
): string | null {
  if (openingChars.has(key)) return pairs[key];
  if (key !== '<' || !angleBracketLanguages.has(language)) return null;
  if (selectionStart !== selectionEnd) return '>';

  return isLikelyTemplateStart(value, selectionStart) ? '>' : null;
}

function isLikelyTemplateStart(value: string, selectionStart: number): boolean {
  const previous = value[selectionStart - 1];
  if (!previous || /\s/.test(previous)) return false;

  // `vector<` and `map<string,` are useful template/generic starts. Requiring an
  // adjacent identifier-like token keeps ordinary spaced comparisons (`a < b`)
  // native and unsurprising without parsing the document on every keypress.
  return /[A-Za-z0-9_>]/.test(previous);
}

function shouldSkipClosingCharacter(
  value: string,
  selectionStart: number,
  selectionEnd: number,
  key: string,
  language: LanguageCode,
): boolean {
  if (selectionStart !== selectionEnd || value[selectionStart] !== key) {
    return false;
  }
  if (key === '>' && !angleBracketLanguages.has(language)) return false;
  if ((key === '"' || key === "'" || key === '`') && isEscapedAt(value, selectionStart)) {
    return false;
  }
  return true;
}

function deletePairedCharacter(
  value: string,
  selectionStart: number,
  selectionEnd: number,
  language: LanguageCode,
): EditResult | null {
  if (selectionStart !== selectionEnd || selectionStart === 0) {
    return null;
  }

  const previous = value[selectionStart - 1];
  const next = value[selectionStart];
  const expectedClose =
    previous === '<' &&
    angleBracketLanguages.has(language) &&
    isLikelyTemplateStart(value, selectionStart - 1)
      ? '>'
      : pairs[previous];
  if (expectedClose !== next) {
    return null;
  }

  return {
    value: `${value.slice(0, selectionStart - 1)}${value.slice(selectionStart + 1)}`,
    selectionStart: selectionStart - 1,
    selectionEnd: selectionStart - 1,
  };
}

function deleteIndentationToPreviousTabStop(
  value: string,
  selectionStart: number,
  selectionEnd: number,
  tabSize: number,
): EditResult | null {
  if (selectionStart !== selectionEnd || selectionStart === 0) return null;

  const lineStart = value.lastIndexOf('\n', selectionStart - 1) + 1;
  const lineEnd = value.indexOf('\n', selectionStart);
  const line = value.slice(lineStart, lineEnd === -1 ? value.length : lineEnd);
  if (!/^[\t ]*$/.test(line) || selectionStart === lineStart) return null;

  const prefix = value.slice(lineStart, selectionStart);
  const columns = visualColumns(prefix, tabSize);
  const column = columns[columns.length - 1];
  const targetColumn = Math.floor((column - 1) / tabSize) * tabSize;
  let deleteOffset = prefix.length;

  while (deleteOffset > 0 && columns[deleteOffset] > targetColumn) {
    deleteOffset -= 1;
  }
  const deleteStart = lineStart + deleteOffset;

  return {
    value: `${value.slice(0, deleteStart)}${value.slice(selectionStart)}`,
    selectionStart: deleteStart,
    selectionEnd: deleteStart,
  };
}

function visualColumns(value: string, tabSize: number): number[] {
  const columns = [0];
  for (const character of value) {
    const column = columns[columns.length - 1];
    columns.push(character === '\t' ? column + tabSize - (column % tabSize) : column + 1);
  }
  return columns;
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

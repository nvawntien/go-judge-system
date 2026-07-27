import type { LanguageCode } from './types';

/**
 * Tiny per-line tokenizer, ported from the AstraCode prototype. It is
 * deliberately shallow — enough for the editor's colour language without
 * pulling a full parser into the bundle.
 */

export interface Token {
  text: string;
  color: string;
}

const KEYWORDS: Record<LanguageCode, string[]> = {
  GO: [
    'package', 'import', 'func', 'for', 'range', 'if', 'else', 'return', 'nil', 'var', 'const',
    'type', 'struct', 'interface', 'map', 'chan', 'go', 'defer', 'switch', 'case', 'default',
    'break', 'continue', 'true', 'false',
  ],
  CPP: [
    'include', 'using', 'namespace', 'for', 'while', 'if', 'else', 'return', 'auto', 'const',
    'struct', 'class', 'public', 'private', 'template', 'typename', 'break', 'continue', 'switch',
    'case', 'true', 'false', 'nullptr', 'new', 'delete',
  ],
  PYTHON: [
    'def', 'class', 'for', 'in', 'if', 'elif', 'else', 'return', 'is', 'not', 'and', 'or', 'None',
    'True', 'False', 'import', 'from', 'as', 'while', 'try', 'except', 'finally', 'with', 'lambda',
    'yield', 'pass', 'break', 'continue',
  ],
  JAVA: [
    'import', 'package', 'public', 'private', 'protected', 'class', 'static', 'void', 'new',
    'return', 'if', 'else', 'for', 'while', 'try', 'catch', 'finally', 'throws', 'throw', 'this',
    'extends', 'implements', 'interface', 'true', 'false', 'null', 'final', 'switch', 'case',
    'break', 'continue',
  ],
};

const TYPES: Record<LanguageCode, string[]> = {
  GO: ['int', 'int64', 'string', 'bool', 'byte', 'rune', 'float64', 'make', 'len', 'cap', 'append', 'error'],
  CPP: ['vector', 'unordered_map', 'map', 'set', 'string', 'int', 'long', 'void', 'double', 'size_t', 'pair'],
  PYTHON: ['enumerate', 'range', 'len', 'int', 'str', 'list', 'dict', 'set', 'tuple', 'sum', 'min', 'max', 'sorted'],
  JAVA: ['int', 'long', 'double', 'String', 'boolean', 'char', 'List', 'ArrayList', 'Map', 'HashMap', 'Scanner'],
};

const LINE_COMMENT: Record<LanguageCode, string> = {
  GO: '//',
  CPP: '//',
  JAVA: '//',
  PYTHON: '#',
};

const TOKEN_RE = /("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*')|(\b\d[\d._]*\b)|(\b[A-Za-z_]\w*\b)|([^\w\s]+)|(\s+)/g;

export function tokenizeLine(line: string, language: LanguageCode): Token[] {
  const comment = LINE_COMMENT[language];
  const commentIndex = line.indexOf(comment);

  // Comment that is not inside a string literal — colour the rest of the line.
  if (commentIndex >= 0 && !isInsideString(line, commentIndex)) {
    const head = line.slice(0, commentIndex);
    return [
      ...(head ? tokenizeCode(head, language) : []),
      { text: line.slice(commentIndex), color: 'var(--syn-com)' },
    ];
  }

  return tokenizeCode(line, language);
}

function isInsideString(line: string, index: number): boolean {
  let quote: string | null = null;
  for (let i = 0; i < index; i += 1) {
    const ch = line[i];
    if (ch === '\\') {
      i += 1;
      continue;
    }
    if (quote) {
      if (ch === quote) quote = null;
    } else if (ch === '"' || ch === "'") {
      quote = ch;
    }
  }
  return quote !== null;
}

function tokenizeCode(line: string, language: LanguageCode): Token[] {
  const keywords = KEYWORDS[language] ?? [];
  const types = TYPES[language] ?? [];
  const tokens: Token[] = [];

  TOKEN_RE.lastIndex = 0;
  let match: RegExpExecArray | null;

  while ((match = TOKEN_RE.exec(line))) {
    const text = match[0];
    let color = 'var(--code-fg)';

    if (match[1]) {
      color = 'var(--syn-str)';
    } else if (match[2]) {
      color = 'var(--syn-num)';
    } else if (match[3]) {
      if (keywords.includes(text)) color = 'var(--syn-kw)';
      else if (types.includes(text)) color = 'var(--syn-type)';
      else if (line[TOKEN_RE.lastIndex] === '(') color = 'var(--syn-fn)';
    } else if (match[4] && text.startsWith('#') && (language === 'CPP' || language === 'JAVA')) {
      color = 'var(--syn-kw)';
    }

    tokens.push({ text, color });
  }

  return tokens;
}

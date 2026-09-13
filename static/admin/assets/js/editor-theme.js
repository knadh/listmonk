// Forked from https://github.com/fsegurai/codemirror-themes
// MIT License - Copyright (c) 2025 fsegurai

import { HighlightStyle, syntaxHighlighting } from '@codemirror/language';
import { EditorView } from '@codemirror/view';
import { tags } from '@lezer/highlight';

const background = 'var(--cm-bg)';
const foreground = 'var(--cm-fg)';
const caret = 'var(--cm-caret)';
const selection = 'var(--cm-selection)';
const selectionMatch = 'var(--cm-selection-match)';
const searchMatch = 'var(--cm-search-match)';
const lineHighlight = 'var(--cm-active-line)';
const border = 'var(--cm-border)';
const gutterForeground = 'var(--cm-gutter-fg)';
const gutterActiveForeground = 'var(--cm-gutter-active-fg)';
const keywordColor = 'var(--cm-keyword)';
const controlKeywordColor = 'var(--cm-control)';
const variableColor = 'var(--cm-variable)';
const classTypeColor = 'var(--cm-type)';
const functionColor = 'var(--cm-function)';
const numberColor = 'var(--cm-number)';
const operatorColor = 'var(--cm-operator)';
const regexpColor = 'var(--cm-regexp)';
const stringColor = 'var(--cm-string)';
const commentColor = 'var(--cm-comment)';
const linkColor = 'var(--cm-link)';
const invalidColor = 'var(--cm-invalid)';

const editorTheme = /* @__PURE__ */EditorView.theme({
  '&': {
    color: foreground,
    backgroundColor: background,
    fontFamily: 'Menlo, Monaco, Consolas, "Andale Mono", "Ubuntu Mono", "Courier New", monospace',
  },
  '.cm-content': {
    caretColor: caret,
  },
  '.cm-cursor, .cm-dropCursor': {
    borderLeftColor: caret,
  },
  '&.cm-focused > .cm-scroller > .cm-selectionLayer .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection': {
    backgroundColor: selection,
  },
  '.cm-selectionMatch': {
    backgroundColor: selectionMatch,
  },
  '.cm-searchMatch': {
    backgroundColor: searchMatch,
    outline: `1px solid ${border}`,
  },
  '.cm-activeLine': {
    backgroundColor: lineHighlight,
  },
  '.cm-gutters': {
    backgroundColor: background,
    color: gutterForeground,
    border: 'none',
  },
  '.cm-activeLineGutter': {
    backgroundColor: lineHighlight,
    color: gutterActiveForeground,
  },
  '&.cm-editor .cm-panels': {
    backgroundColor: background,
    color: foreground,
  },
  '&.cm-editor .cm-panels-top': {
    borderBottom: `1px solid ${border}`,
  },
  '&.cm-editor .cm-panels-bottom': {
    borderTop: `1px solid ${border}`,
  },
  '&.cm-editor .cm-button': {
    backgroundImage: 'none',
    backgroundColor: 'var(--cm-button-bg)',
    color: foreground,
    border: `1px solid ${border}`,
    borderRadius: 'var(--radius-small)',
    '&:active': {
      backgroundColor: 'var(--cm-button-active-bg)',
    },
  },
  '&.cm-editor .cm-textfield': {
    backgroundColor: background,
    color: foreground,
    border: `1px solid ${border}`,
    borderRadius: 'var(--radius-small)',
  },
  '&.cm-editor .cm-tooltip': {
    backgroundColor: background,
    color: foreground,
    border: `1px solid ${border}`,
  },
}, { dark: false });

const editorHighlightStyle = /* @__PURE__ */HighlightStyle.define([
  {
    tag: [
      tags.keyword,
      tags.operatorKeyword,
      tags.modifier,
      tags.color,
      /* @__PURE__ */tags.constant(tags.name),
      /* @__PURE__ */tags.standard(tags.name),
      /* @__PURE__ */tags.standard(tags.tagName),
      /* @__PURE__ */tags.special(tags.brace),
      tags.atom,
      tags.bool,
      /* @__PURE__ */tags.special(tags.variableName),
    ],
    color: keywordColor,
  },
  { tag: [tags.moduleKeyword, tags.controlKeyword], color: controlKeywordColor },
  {
    tag: [
      tags.name,
      tags.deleted,
      tags.character,
      tags.macroName,
      tags.propertyName,
      tags.variableName,
      tags.labelName,
      /* @__PURE__ */tags.definition(tags.name),
    ],
    color: variableColor,
  },
  { tag: tags.heading, fontWeight: 'bold', color: variableColor },
  {
    tag: [
      tags.typeName,
      tags.className,
      tags.tagName,
      tags.number,
      tags.changed,
      tags.annotation,
      tags.self,
      tags.namespace,
    ],
    color: classTypeColor,
  },
  {
    tag: [/* @__PURE__ */tags.function(tags.variableName), /* @__PURE__ */tags.function(tags.propertyName)],
    color: functionColor,
  },
  { tag: [tags.number], color: numberColor },
  {
    tag: [tags.operator, tags.punctuation, tags.separator, tags.url, tags.escape, tags.regexp],
    color: operatorColor,
  },
  { tag: [tags.regexp], color: regexpColor },
  {
    tag: [/* @__PURE__ */tags.special(tags.string), tags.processingInstruction, tags.string, tags.inserted],
    color: stringColor,
  },
  { tag: [tags.meta, tags.comment], color: commentColor },
  { tag: tags.link, color: linkColor, textDecoration: 'underline' },
  { tag: tags.invalid, color: invalidColor },
  { tag: tags.strong, fontWeight: 'bold' },
  { tag: tags.emphasis, fontStyle: 'italic' },
  { tag: tags.strikethrough, textDecoration: 'line-through' },
]);

const editorExtensions = [
  editorTheme,
  /* @__PURE__ */syntaxHighlighting(editorHighlightStyle),
];

export { editorExtensions, editorHighlightStyle, editorTheme };

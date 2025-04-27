// Forked from https://github.com/fsegurai/codemirror-themes
// MIT License - Copyright (c) 2025 fsegurai

import { EditorView } from '@codemirror/view';
import { HighlightStyle, syntaxHighlighting } from '@codemirror/language';
import { tags } from '@lezer/highlight';

// VSCode Light theme color definitions
const background = '#ffffff';
const foreground = '#383a42';
const caret = '#000000';
const selection = '#add6ff';
const selectionMatch = '#a8ac94';
const lineHighlight = '#99999926';
const gutterBackground = '#ffffff';
const gutterForeground = '#0055d4';
const gutterActiveForeground = '#0b216f';
const keywordColor = '#0055d4';
const controlKeywordColor = '#af00db';
const variableColor = '#e45649';
const classTypeColor = '#0055d4';
const functionColor = '#795e26';
const numberColor = '#098658';
const operatorColor = '#383a42';
const regexpColor = '#af00db';
const stringColor = '#50a14f';
const commentColor = '#999';
const linkColor = '#0055d4';
const invalidColor = '#e45649';

// Define the editor theme styles for VSCode Light
const vsCodeLightTheme = /* @__PURE__ */EditorView.theme({
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
  '.cm-searchMatch': {
    backgroundColor: selectionMatch,
    outline: `1px solid ${lineHighlight}`,
  },
  '.cm-activeLine': {
    backgroundColor: lineHighlight,
  },
  '.cm-gutters': {
    backgroundColor: gutterBackground,
    color: gutterForeground,
  },
  '.cm-activeLineGutter': {
    color: gutterActiveForeground,
  },
}, { dark: false });
// Define the highlighting style for code in the VSCode Light theme
const vsCodeLightHighlightStyle = /* @__PURE__ */HighlightStyle.define([
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
// Extension to enable the VSCode Light theme (both the editor theme and the highlight style)
const vsCodeLight = [
  vsCodeLightTheme,
  /* @__PURE__ */syntaxHighlighting(vsCodeLightHighlightStyle),
];

export { vsCodeLight, vsCodeLightHighlightStyle, vsCodeLightTheme };

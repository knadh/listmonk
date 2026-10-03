// Fork of @usewaypoint/block-text (MIT) that applies per-element inline styles to rendered Markdown.
// vite.config.ts aliases the package name to this file so the renderer bundled inside
// @usewaypoint/email-builder uses it too.
import insane, { AllowedTags } from 'insane';
import { marked, Renderer } from 'marked';
import React, { useMemo } from 'react';
import { z } from 'zod';

import { cleanMarkdownStyles, getActiveMarkdownStyles, MarkdownStyles } from './markdownStyles';

const ALLOWED_TAGS: AllowedTags[] = [
  'a', 'article', 'b', 'blockquote', 'br', 'caption', 'code', 'del', 'details', 'div', 'em',
  'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'hr', 'i', 'img', 'ins', 'kbd', 'li', 'main', 'ol', 'p', 'pre',
  'section', 'span', 'strong', 'sub', 'summary', 'sup', 'table', 'tbody', 'td', 'th', 'thead', 'tr',
  'u', 'ul',
];
const GENERIC_ALLOWED_ATTRIBUTES = ['style', 'title'];

function sanitizer(html: string) {
  return insane(html, {
    allowedTags: ALLOWED_TAGS,
    allowedAttributes: {
      ...ALLOWED_TAGS.reduce((res, tag) => {
        res[tag] = [...GENERIC_ALLOWED_ATTRIBUTES];
        return res;
      }, {} as Record<AllowedTags, string[]>),
      img: ['src', 'srcset', 'alt', 'width', 'height', ...GENERIC_ALLOWED_ATTRIBUTES],
      table: ['width', ...GENERIC_ALLOWED_ATTRIBUTES],
      td: ['align', 'width', ...GENERIC_ALLOWED_ATTRIBUTES],
      th: ['align', 'width', ...GENERIC_ALLOWED_ATTRIBUTES],
      a: ['href', 'target', ...GENERIC_ALLOWED_ATTRIBUTES],
      ol: ['start', ...GENERIC_ALLOWED_ATTRIBUTES],
      ul: ['start', ...GENERIC_ALLOWED_ATTRIBUTES],
    },
  });
}

class StyledRenderer extends Renderer {
  private styles: MarkdownStyles;

  constructor(styles: MarkdownStyles) {
    super();
    this.styles = styles;
  }

  // Adds the configured style to the opening <tag> at the start of the rendered html.
  private st(html: string, tag: string) {
    const style = this.styles[tag];
    if (!style) {
      return html;
    }
    return html.replace(new RegExp(`^<${tag}\\b`), (m) => `${m} style="${style}"`);
  }

  heading(text: string, level: number, raw: string) {
    return this.st(super.heading(text, level, raw), `h${level}`);
  }

  paragraph(text: string) {
    return this.st(super.paragraph(text), 'p');
  }

  list(body: string, ordered: boolean, start: number | '') {
    return this.st(super.list(body, ordered, start), ordered ? 'ol' : 'ul');
  }

  listitem(text: string, task: boolean, checked: boolean) {
    return this.st(super.listitem(text, task, checked), 'li');
  }

  blockquote(quote: string) {
    return this.st(super.blockquote(quote), 'blockquote');
  }

  code(code: string, infostring: string | undefined, escaped: boolean) {
    return this.st(super.code(code, infostring, escaped), 'pre');
  }

  codespan(text: string) {
    return this.st(super.codespan(text), 'code');
  }

  strong(text: string) {
    return this.st(super.strong(text), 'strong');
  }

  em(text: string) {
    return this.st(super.em(text), 'em');
  }

  hr() {
    return this.st(super.hr(), 'hr');
  }

  tablecell(content: string, flags: { header: boolean; align: 'center' | 'left' | 'right' | null }) {
    return this.st(super.tablecell(content, flags), flags.header ? 'th' : 'td');
  }

  table(header: string, body: string) {
    return this.st(`<table width="100%">\n<thead>\n${header}</thead>\n<tbody>\n${body}</tbody>\n</table>`, 'table');
  }

  link(href: string, title: string | null | undefined, text: string) {
    const t = title ? ` title="${title}"` : '';
    return this.st(`<a href="${href}"${t} target="_blank">${text}</a>`, 'a');
  }
}

export function renderMarkdownString(str: string, styles: MarkdownStyles) {
  const html = marked.parse(str, {
    async: false,
    breaks: true,
    gfm: true,
    pedantic: false,
    silent: false,
    renderer: new StyledRenderer(styles),
  });
  if (typeof html !== 'string') {
    throw new Error('marked.parse did not return a string');
  }
  return sanitizer(html);
}

function EmailMarkdown({
  markdown, styles, ...props
}: { markdown: string; styles: MarkdownStyles } & React.HTMLAttributes<HTMLDivElement>) {
  const key = JSON.stringify(styles);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  const data = useMemo(() => renderMarkdownString(markdown, styles), [markdown, key]);
  return <div {...props} dangerouslySetInnerHTML={{ __html: data }} />;
}

const FONT_FAMILY_SCHEMA = z
  .enum([
    'MODERN_SANS',
    'BOOK_SANS',
    'ORGANIC_SANS',
    'GEOMETRIC_SANS',
    'HEAVY_SANS',
    'ROUNDED_SANS',
    'MODERN_SERIF',
    'BOOK_SERIF',
    'MONOSPACE',
  ])
  .nullable()
  .optional();

function getFontFamily(fontFamily: z.infer<typeof FONT_FAMILY_SCHEMA>) {
  switch (fontFamily) {
    case 'MODERN_SANS':
      return '"Helvetica Neue", "Arial Nova", "Nimbus Sans", Arial, sans-serif';
    case 'BOOK_SANS':
      return 'Optima, Candara, "Noto Sans", source-sans-pro, sans-serif';
    case 'ORGANIC_SANS':
      return 'Seravek, "Gill Sans Nova", Ubuntu, Calibri, "DejaVu Sans", source-sans-pro, sans-serif';
    case 'GEOMETRIC_SANS':
      return 'Avenir, "Avenir Next LT Pro", Montserrat, Corbel, "URW Gothic", source-sans-pro, sans-serif';
    case 'HEAVY_SANS':
      return 'Bahnschrift, "DIN Alternate", "Franklin Gothic Medium", "Nimbus Sans Narrow", sans-serif-condensed, sans-serif';
    case 'ROUNDED_SANS':
      return 'ui-rounded, "Hiragino Maru Gothic ProN", Quicksand, Comfortaa, Manjari, "Arial Rounded MT Bold", Calibri, source-sans-pro, sans-serif';
    case 'MODERN_SERIF':
      return 'Charter, "Bitstream Charter", "Sitka Text", Cambria, serif';
    case 'BOOK_SERIF':
      return '"Iowan Old Style", "Palatino Linotype", "URW Palladio L", P052, serif';
    case 'MONOSPACE':
      return '"Nimbus Mono PS", "Courier New", "Cutive Mono", monospace';
  }
  return undefined;
}

const COLOR_SCHEMA = z
  .string()
  .regex(/^#[0-9a-fA-F]{6}$/)
  .nullable()
  .optional();

const PADDING_SCHEMA = z
  .object({
    top: z.number(),
    bottom: z.number(),
    right: z.number(),
    left: z.number(),
  })
  .optional()
  .nullable();

const getPadding = (padding: z.infer<typeof PADDING_SCHEMA>) => (
  padding ? `${padding.top}px ${padding.right}px ${padding.bottom}px ${padding.left}px` : undefined
);

export const TextPropsSchema = z.object({
  style: z
    .object({
      color: COLOR_SCHEMA,
      backgroundColor: COLOR_SCHEMA,
      fontSize: z.number().gte(0).optional().nullable(),
      fontFamily: FONT_FAMILY_SCHEMA,
      fontWeight: z.enum(['bold', 'normal']).optional().nullable(),
      textAlign: z.enum(['left', 'center', 'right']).optional().nullable(),
      padding: PADDING_SCHEMA,
    })
    .optional()
    .nullable(),
  props: z
    .object({
      markdown: z.boolean().optional().nullable(),
      text: z.string().optional().nullable(),
    })
    .optional()
    .nullable(),
});

export type TextProps = z.infer<typeof TextPropsSchema>;

export const TextPropsDefaults = {
  text: '',
};

// `markdownStyles` is passed explicitly by the editor canvas. The renderer inside
// @usewaypoint/email-builder doesn't pass it, so fall back to the active document's styles.
export function Text({ style, props, markdownStyles }: TextProps & { markdownStyles?: MarkdownStyles | null }) {
  const wStyle: React.CSSProperties = {
    color: style?.color ?? undefined,
    backgroundColor: style?.backgroundColor ?? undefined,
    fontSize: style?.fontSize ?? undefined,
    fontFamily: getFontFamily(style?.fontFamily),
    fontWeight: style?.fontWeight ?? undefined,
    textAlign: style?.textAlign ?? undefined,
    padding: getPadding(style?.padding),
  };
  const text = props?.text ?? TextPropsDefaults.text;
  if (props?.markdown) {
    const styles = cleanMarkdownStyles(markdownStyles ?? getActiveMarkdownStyles());
    return <EmailMarkdown style={wStyle} markdown={text} styles={styles} />;
  }
  return <div style={wStyle}>{text}</div>;
}

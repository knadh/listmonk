import { renderToStaticMarkup } from '@usewaypoint/email-builder';
import { TEditorConfiguration } from './documents/editor/core';

export function renderHtmlWithMeta(document: TEditorConfiguration, options: { rootBlockId: string }): string {
  // The upstream renderer strips the custom `embed` prop before rendering, so
  // collect URLs from blocks marked for embedding and re-tag the matching <img>
  // with a `data-embed` flag after rendering. The backend resolves the src
  // filename to a media item at compile time.
  const embedURLs: string[] = [];
  for (const block of Object.values(document)) {
    if (!block || (block as { type?: string }).type !== 'Image') {
      continue;
    }
    const props = ((block as { data?: { props?: { url?: string; embed?: boolean } } }).data || {}).props || {};
    if (props.embed && props.url) {
      embedURLs.push(props.url);
    }
  }

  let html = renderToStaticMarkup(document, options);

  for (const url of embedURLs) {
    const re = new RegExp(`<img\\b[^>]*?\\bsrc="${escapeRegExp(url)}"[^>]*>`, 'g');
    html = html.replace(re, (tag) => (
      /\bdata-embed\b/.test(tag) ? tag : tag.replace(/(\bsrc="[^"]*")/, '$1 data-embed="true"')
    ));
  }

  return html.replace(
    '<html>',
    '<html><head><meta name="viewport" content="width=device-width, initial-scale=1.0"></head>',
  );
}

function escapeRegExp(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

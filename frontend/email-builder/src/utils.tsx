import { renderToStaticMarkup } from '@usewaypoint/email-builder';
import { TEditorConfiguration } from './documents/editor/core';

export function renderHtmlWithMeta(document: TEditorConfiguration, options: { rootBlockId: string }): string {
  // The upstream renderer strips embed/uuid props before
  // rendering, so collect them from the document first and reapply them as
  // `data-embed` attribs after rendering.
  const embeds: Array<{ url: string; uuid: string }> = [];
  for (const block of Object.values(document)) {
    if (!block || (block as { type?: string }).type !== 'Image') {
      continue;
    }
    const props = ((block as { data?: { props?: { url?: string; embed?: boolean; uuid?: string } } }).data || {}).props || {};
    if (props.embed && props.uuid && props.url) {
      embeds.push({ url: props.url, uuid: props.uuid });
    }
  }

  let html = renderToStaticMarkup(document, options);

  for (const { url, uuid } of embeds) {
    const re = new RegExp(`(<img\\b(?:(?!data-embed=)[^>])*?\\bsrc="${escapeRegExp(url)}")([^>]*>)`);
    html = html.replace(re, `$1 data-embed="${uuid}"$2`);
  }

  return html.replace(
    '<html>',
    '<html><head><meta name="viewport" content="width=device-width, initial-scale=1.0"></head>',
  );
}

function escapeRegExp(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

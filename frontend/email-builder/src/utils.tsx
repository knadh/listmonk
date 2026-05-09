import { renderToStaticMarkup } from '@usewaypoint/email-builder';
import { TEditorConfiguration } from './documents/editor/core';
import { postProcessForOutlook } from './outlook';

const VIEWPORT_META = '<meta name="viewport" content="width=device-width, initial-scale=1.0">';
const MSO_DOCUMENT_SETTINGS = '<!--[if mso]><noscript><xml xmlns:o="urn:schemas-microsoft-com:office:office"><o:OfficeDocumentSettings><o:AllowPNG/><o:PixelsPerInch>96</o:PixelsPerInch></o:OfficeDocumentSettings></xml></noscript><![endif]-->';

function injectHeadContents(html: string, contents: string) {
  const headMatch = html.match(/<head\b([^>]*)>/i);
  if (headMatch) {
    return html.replace(/<head\b([^>]*)>/i, `<head$1>${contents}`);
  }

  const htmlMatch = html.match(/<html\b([^>]*)>/i);
  if (htmlMatch) {
    return html.replace(/<html\b([^>]*)>/i, `<html$1><head>${contents}</head>`);
  }

  return `<head>${contents}</head>${html}`;
}

function collectImageEmbedURLs(document: TEditorConfiguration): string[] {
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

  return embedURLs;
}

function applyImageEmbeds(html: string, embedURLs: string[]): string {
  let output = html;

  for (const url of embedURLs) {
    const re = new RegExp(`<img\\b[^>]*?\\bsrc="${escapeRegExp(url)}"[^>]*>`, 'g');
    output = output.replace(re, (tag) => (
      /\bdata-embed\b/.test(tag) ? tag : tag.replace(/(\bsrc="[^"]*")/, '$1 data-embed="true"')
    ));
  }

  return output;
}

export function renderHtmlWithMeta(
  document: TEditorConfiguration,
  options: { rootBlockId: string; outlook?: boolean }
): string {
  const embedURLs = collectImageEmbedURLs(document);
  const html = renderToStaticMarkup(document, options);
  const rendered = options.outlook ? postProcessForOutlook(html) : html;
  const output = applyImageEmbeds(rendered, embedURLs);
  const head = options.outlook ? `${VIEWPORT_META}${MSO_DOCUMENT_SETTINGS}` : VIEWPORT_META;

  return injectHeadContents(output, head);
}

function escapeRegExp(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

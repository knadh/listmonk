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

function collectImageEmbeds(document: TEditorConfiguration): Array<{ url: string; uuid: string }> {
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

  return embeds;
}

function applyImageEmbeds(html: string, embeds: Array<{ url: string; uuid: string }>): string {
  let output = html;

  for (const { url, uuid } of embeds) {
    const re = new RegExp(`(<img\\b(?:(?!data-embed=)[^>])*?\\bsrc="${escapeRegExp(url)}")([^>]*>)`);
    output = output.replace(re, `$1 data-embed="${uuid}"$2`);
  }

  return output;
}

export function renderHtmlWithMeta(
  document: TEditorConfiguration,
  options: { rootBlockId: string; outlook?: boolean }
): string {
  const embeds = collectImageEmbeds(document);
  const html = renderToStaticMarkup(document, options);
  const rendered = options.outlook ? postProcessForOutlook(html) : html;
  const output = applyImageEmbeds(rendered, embeds);
  const head = options.outlook ? `${VIEWPORT_META}${MSO_DOCUMENT_SETTINGS}` : VIEWPORT_META;

  return injectHeadContents(output, head);
}

function escapeRegExp(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

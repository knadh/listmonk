import { renderToStaticMarkup } from '@usewaypoint/email-builder';
import { TEditorConfiguration } from './documents/editor/core';
import { postProcessForOutlook } from './outlook';

const VIEWPORT_META = '<meta name="viewport" content="width=device-width, initial-scale=1.0">';
const MSO_DOCUMENT_SETTINGS = '<!--[if mso]><noscript><xml><o:OfficeDocumentSettings><o:AllowPNG/><o:PixelsPerInch>96</o:PixelsPerInch></o:OfficeDocumentSettings></xml></noscript><![endif]-->';

export function renderHtmlWithMeta(document: TEditorConfiguration, options: { rootBlockId: string }): string {
  const html = postProcessForOutlook(renderToStaticMarkup(document, options));

  return html.replace(
    /<head([^>]*)>/i,
    `<head$1>${VIEWPORT_META}${MSO_DOCUMENT_SETTINGS}`
  );
}

import { renderToStaticMarkup } from '@usewaypoint/email-builder';
import { TEditorConfiguration } from './documents/editor/core';

export function renderHtmlWithMeta(document: TEditorConfiguration, options: { rootBlockId: string }): string {
  const html = renderToStaticMarkup(document, options);
  // Insert <head> with viewport meta after <html>
  return html.replace(
    '<html>',
    '<html><head><meta name="viewport" content="width=device-width, initial-scale=1.0"></head>'
  );
}

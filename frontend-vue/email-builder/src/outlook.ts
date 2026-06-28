type TStyleMap = Record<string, string>;
type TPaddingValues = {
  top: number;
  right: number;
  bottom: number;
  left: number;
};

const PRESENTATION_TABLE_STYLE = 'border-collapse:collapse;mso-table-lspace:0pt;mso-table-rspace:0pt;';

function appendMissingStyles(style: string | null, declarations: Array<[string, string]>) {
  const current = (style || '').trim();
  const lower = current.toLowerCase();
  const missing = declarations
    .filter(([property]) => !lower.includes(`${property.toLowerCase()}:`))
    .map(([property, value]) => `${property}:${value}`);

  if (missing.length === 0) {
    return current;
  }

  return [current.replace(/;+\s*$/, ''), ...missing]
    .filter(Boolean)
    .join(';');
}

function setStyleValues(style: string | null, declarations: Array<[string, string | null]>) {
  const styleMap = parseStyleMap(style);

  declarations.forEach(([property, value]) => {
    const key = property.toLowerCase();
    if (value === null || value === '') {
      delete styleMap[key];
      return;
    }

    styleMap[key] = value;
  });

  return Object.entries(styleMap)
    .map(([property, value]) => `${property}:${value}`)
    .join(';');
}

function parseStyleMap(style: string | null) {
  return (style || '')
    .split(';')
    .map((entry) => entry.trim())
    .filter(Boolean)
    .reduce<TStyleMap>((acc, entry) => {
      const separator = entry.indexOf(':');
      if (separator === -1) {
        return acc;
      }

      const property = entry.slice(0, separator).trim().toLowerCase();
      const value = entry.slice(separator + 1).trim();
      if (property) {
        acc[property] = value;
      }
      return acc;
    }, {});
}

function getPixelValue(value?: string) {
  if (!value) {
    return null;
  }

  const match = value.trim().match(/^(-?\d+(?:\.\d+)?)px$/i);
  if (!match) {
    return null;
  }

  return Math.round(Number(match[1]));
}

function getPixelWidthFromImage(img: HTMLImageElement) {
  const attrWidth = img.getAttribute('width');
  if (attrWidth && /^\d+$/.test(attrWidth)) {
    return attrWidth;
  }

  const style = img.getAttribute('style') || '';
  const widthMatch = style.match(/(?:^|;)\s*width\s*:\s*(\d+)px(?:;|$)/i);
  if (widthMatch) {
    return widthMatch[1];
  }

  const maxWidthMatch = style.match(/(?:^|;)\s*max-width\s*:\s*(\d+)px(?:;|$)/i);
  if (maxWidthMatch) {
    return maxWidthMatch[1];
  }

  return null;
}

function getPaddingValues(styleMap: TStyleMap): TPaddingValues {
  const shorthand = styleMap.padding?.trim().split(/\s+/) || [];

  const [
    topFromShorthand,
    rightFromShorthand = topFromShorthand,
    bottomFromShorthand = topFromShorthand,
    leftFromShorthand = rightFromShorthand,
  ] = shorthand;

  return {
    top: getPixelValue(styleMap['padding-top'] || topFromShorthand) || 0,
    right: getPixelValue(styleMap['padding-right'] || rightFromShorthand) || 0,
    bottom: getPixelValue(styleMap['padding-bottom'] || bottomFromShorthand) || 0,
    left: getPixelValue(styleMap['padding-left'] || leftFromShorthand) || 0,
  };
}

function escapeHtml(value: string) {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;');
}

function escapeAttribute(value: string) {
  return escapeHtml(value).replace(/"/g, '&quot;');
}

function createFragmentFromHtml(node: Element, html: string) {
  const range = node.ownerDocument.createRange();
  range.selectNode(node);
  return range.createContextualFragment(html);
}

function replaceNodeWithHtml(node: Element, html: string) {
  node.replaceWith(createFragmentFromHtml(node, html));
}

function escapeTemplateString(value: string) {
  return value
    .replace(/\\/g, '\\\\')
    .replace(/"/g, '\\"');
}

function makeSafeTemplate(raw: string) {
  // Encode angle brackets so DOMParser does not consume Outlook conditional comments
  // before the Go template expression is evaluated.
  const escaped = escapeTemplateString(raw)
    .replace(/</g, '\\x3c')
    .replace(/>/g, '\\x3e');

  return `{{ Safe "${escaped}" }}`;
}

function getWrapperOptions(style: string | null) {
  const styleValue = style || '';
  const styleMap = parseStyleMap(styleValue);
  const align = styleMap['text-align'] || 'left';
  const backgroundColor = styleMap['background-color'];
  const bgcolorAttr = backgroundColor ? ` bgcolor="${escapeAttribute(backgroundColor)}"` : '';

  return { styleValue, styleMap, align, bgcolorAttr };
}

function buildPresentationTable(contents: string, width: string = '100%') {
  const widthAttr = width && width !== 'auto' ? ` width="${escapeAttribute(width)}"` : '';

  return `<table role="presentation"${widthAttr} cellpadding="0" cellspacing="0" border="0" style="${PRESENTATION_TABLE_STYLE}">${contents}</table>`;
}

function hasSingleChildMatching(div: HTMLDivElement, predicate: (child: Element) => boolean) {
  const children = Array.from(div.children);
  return children.length === 1 && predicate(children[0]);
}

function addTableDefaults(doc: Document) {
  doc.querySelectorAll('table[role="presentation"]').forEach((table) => {
    if (!table.getAttribute('cellpadding')) {
      table.setAttribute('cellpadding', '0');
    }
    if (!table.getAttribute('cellspacing')) {
      table.setAttribute('cellspacing', '0');
    }
    if (!table.getAttribute('border')) {
      table.setAttribute('border', '0');
    }

    table.setAttribute(
      'style',
      appendMissingStyles(table.getAttribute('style'), [
        ['border-collapse', 'collapse'],
        ['mso-table-lspace', '0pt'],
        ['mso-table-rspace', '0pt'],
      ])
    );
  });
}

function isStandaloneImage(img: HTMLImageElement) {
  const parent = img.parentElement;
  if (!parent) {
    return false;
  }

  if (parent.tagName === 'DIV') {
    return hasSingleChildMatching(parent as HTMLDivElement, (child) => child.tagName === 'IMG');
  }

  if (parent.tagName === 'A' && parent.children.length === 1) {
    const grandparent = parent.parentElement;
    return grandparent?.tagName === 'DIV'
      && hasSingleChildMatching(grandparent as HTMLDivElement, (child) => child.tagName === 'A');
  }

  return false;
}

function hardenImages(doc: Document) {
  doc.querySelectorAll('img').forEach((img) => {
    img.setAttribute('border', '0');

    const width = getPixelWidthFromImage(img);
    if (width && !img.getAttribute('width')) {
      img.setAttribute('width', width);
    }

    const standaloneImage = isStandaloneImage(img);
    const declarations: Array<[string, string | null]> = [
      ['border', '0'],
      ['outline', 'none'],
      ['text-decoration', 'none'],
      ['height', 'auto'],
      ['-ms-interpolation-mode', 'bicubic'],
    ];

    if (standaloneImage) {
      declarations.unshift(['display', 'block']);
      declarations.push(['vertical-align', null]);
    }

    img.setAttribute('style', setStyleValues(img.getAttribute('style'), declarations));

    const parent = img.parentElement;
    if (standaloneImage && parent?.tagName === 'A') {
      parent.setAttribute('style', setStyleValues(parent.getAttribute('style'), [
        ['display', 'inline-block'],
        ['border', '0'],
        ['text-decoration', 'none'],
      ]));
    }
  });
}

function transformImageBlocks(doc: Document) {
  const wrappers = Array.from(doc.querySelectorAll('div')).filter((div) => hasSingleChildMatching(div as HTMLDivElement, (child) => {
    if (child.tagName === 'IMG') {
      return true;
    }

    return child.tagName === 'A' && child.children.length === 1 && child.querySelector('img') !== null;
  })) as HTMLDivElement[];

  wrappers.forEach((div) => {
    const { styleValue, align, bgcolorAttr } = getWrapperOptions(div.getAttribute('style'));
    const content = div.innerHTML;

    const innerTable = buildPresentationTable(`<tbody><tr><td align="${escapeAttribute(align)}">${content}</td></tr></tbody>`, 'auto');
    const html = buildPresentationTable(
      `<tbody><tr><td align="${escapeAttribute(align)}"${bgcolorAttr} style="${escapeAttribute(styleValue)}">${innerTable}</td></tr></tbody>`
    );

    replaceNodeWithHtml(div, html);
  });
}

function transformSimpleDivBlocks(doc: Document) {
  const wrappers = Array.from(doc.querySelectorAll('div')).filter((div) => {
    const { styleMap } = getWrapperOptions(div.getAttribute('style'));

    if (!styleMap.padding && !styleMap.height) {
      return false;
    }

    if (div.children.length > 0) {
      const firstChild = div.children[0];
      if (firstChild.tagName === 'A' || firstChild.tagName === 'IMG' || firstChild.tagName === 'TABLE') {
        return false;
      }
    }

    if (styleMap['min-height'] && styleMap.width === '100%') {
      return false;
    }

    return true;
  }) as HTMLDivElement[];

  wrappers.forEach((div) => {
    const { styleValue, styleMap, align, bgcolorAttr } = getWrapperOptions(div.getAttribute('style'));
    const height = getPixelValue(styleMap.height);
    const isSpacer = div.children.length === 0 && (div.textContent || '').trim() === '' && height !== null;

    if (isSpacer) {
      const spacerHtml = buildPresentationTable(
        `<tbody><tr><td${bgcolorAttr} height="${height}" style="${escapeAttribute(styleValue)};line-height:${height}px;font-size:${height}px;">&nbsp;</td></tr></tbody>`
      );

      replaceNodeWithHtml(div, spacerHtml);
      return;
    }

    const blockHtml = buildPresentationTable(
      `<tbody><tr><td align="${escapeAttribute(align)}"${bgcolorAttr} style="${escapeAttribute(styleValue)}">${div.innerHTML}</td></tr></tbody>`
    );

    replaceNodeWithHtml(div, blockHtml);
  });
}

function buildBulletproofButton(anchor: HTMLAnchorElement, wrapperStyle: string) {
  const anchorStyleMap = parseStyleMap(anchor.getAttribute('style'));
  const wrapperStyleMap = parseStyleMap(wrapperStyle);
  const text = anchor.textContent?.replace(/\s+/g, ' ').trim() || '';
  const href = anchor.getAttribute('href') || '#';
  const target = anchor.getAttribute('target');
  const align = wrapperStyleMap['text-align'] || 'left';
  const buttonColor = anchorStyleMap['background-color'] || '#0055d4';
  const textColor = anchorStyleMap.color || '#ffffff';
  const fontSize = getPixelValue(anchorStyleMap['font-size']) || 16;
  const fontWeight = anchorStyleMap['font-weight'] || 'bold';
  const fontFamily = anchorStyleMap['font-family'] || 'Arial, sans-serif';
  const borderRadius = getPixelValue(anchorStyleMap['border-radius']) || 0;
  const paddingValues = getPaddingValues(anchorStyleMap);
  const lineHeight = getPixelValue(anchorStyleMap['line-height']) || Math.round(fontSize * 1.2);
  const display = (anchorStyleMap.display || '').toLowerCase();
  const fullWidth = display === 'block' || anchorStyleMap.width === '100%';

  const targetAttr = target ? ` target="${escapeAttribute(target)}"` : '';

  if (fullWidth) {
    const anchorStyle = setStyleValues(anchor.getAttribute('style'), [
      ['display', 'block'],
      ['text-align', 'center'],
      ['border', '1px solid ' + buttonColor],
    ]);

    return [
      buildPresentationTable(
        `<tbody><tr><td align="${escapeAttribute(align)}" style="${escapeAttribute(wrapperStyle)}">${buildPresentationTable(
          `<tbody><tr><td bgcolor="${escapeAttribute(buttonColor)}" style="background-color:${escapeAttribute(buttonColor)};border-radius:${borderRadius}px;"><a href="${escapeAttribute(href)}"${targetAttr} style="${escapeAttribute(anchorStyle)}">${escapeHtml(text)}</a></td></tr></tbody>`
        )}</td></tr></tbody>`
      ),
    ].join('');
  }

  const estimatedTextWidth = Math.max(1, Math.round(text.length * fontSize * (fontWeight.toLowerCase() === 'bold' ? 0.68 : 0.62)));
  const estimatedWidth = Math.max(40, estimatedTextWidth + paddingValues.left + paddingValues.right);
  const estimatedHeight = Math.max(lineHeight + paddingValues.top + paddingValues.bottom, 32);
  const arcsize = Math.max(0, Math.min(50, Math.round((borderRadius / estimatedHeight) * 100)));
  const cleanAnchorStyle = anchor.getAttribute('style') || '';
  const msoStart = makeSafeTemplate('<!--[if mso]>');
  const msoEnd = makeSafeTemplate('<![endif]-->');
  const vml = `<v:roundrect xmlns:v="urn:schemas-microsoft-com:vml" xmlns:w="urn:schemas-microsoft-com:office:word" href="${escapeAttribute(href)}" style="height:${estimatedHeight}px;v-text-anchor:middle;width:${estimatedWidth}px;" arcsize="${arcsize}%" strokecolor="${escapeAttribute(buttonColor)}" fillcolor="${escapeAttribute(buttonColor)}"><w:anchorlock/><center style="color:${escapeAttribute(textColor)};font-family:${escapeAttribute(fontFamily)};font-size:${fontSize}px;font-weight:${escapeAttribute(fontWeight)};">${escapeHtml(text)}</center></v:roundrect>`;
  const nonMsoStart = makeSafeTemplate('<!--[if !mso]><!-->');
  const nonMsoEnd = makeSafeTemplate('<!--<![endif]-->');

  return buildPresentationTable(
    `<tbody><tr><td align="${escapeAttribute(align)}" style="${escapeAttribute(wrapperStyle)}">${msoStart}${vml}${msoEnd}${nonMsoStart}<a href="${escapeAttribute(href)}"${targetAttr} style="${escapeAttribute(cleanAnchorStyle)}">${escapeHtml(text)}</a>${nonMsoEnd}</td></tr></tbody>`
  );
}

function transformButtonBlocks(doc: Document) {
  const wrappers = Array.from(doc.querySelectorAll('div')).filter((div) => hasSingleChildMatching(div as HTMLDivElement, (child) => {
    if (child.tagName !== 'A' || child.querySelector('img')) {
      return false;
    }

    const styleMap = parseStyleMap((child as HTMLAnchorElement).getAttribute('style'));
    return Boolean(styleMap['background-color'] && styleMap.padding);
  })) as HTMLDivElement[];

  wrappers.forEach((div) => {
    const anchor = div.children[0] as HTMLAnchorElement;
    replaceNodeWithHtml(div, buildBulletproofButton(anchor, div.getAttribute('style') || ''));
  });
}

export function postProcessForOutlook(html: string) {
  if (typeof DOMParser === 'undefined') {
    return html;
  }

  const doc = new DOMParser().parseFromString(html, 'text/html');

  addTableDefaults(doc);
  hardenImages(doc);
  transformButtonBlocks(doc);
  transformImageBlocks(doc);
  transformSimpleDivBlocks(doc);
  addTableDefaults(doc);
  hardenImages(doc);

  return `<!doctype html>\n${doc.documentElement.outerHTML}`;
}

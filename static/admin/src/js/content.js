// Different content conversions for campaign bodies, eg: html -> plaintext

// Trim multilines and optionally remove empty lines.
function trimLines(str, removeEmptyLines) {
  const out = str.split('\n');
  for (let i = 0; i < out.length; i += 1) {
    const line = out[i].trim();
    if (removeEmptyLines) {
      out[i] = line;
    } else if (line === '') {
      out[i] = '';
    }
  }

  // 3+ lines, compress to 2.
  return out.join('\n').replace(/\n\s*\n\s*\n/g, '\n\n');
}

// Pretty-print an HTML string with 4-space indentation.
class HTMLBeautifier {
  // Tags whose content flows inline and so must not force line breaks.
  static INLINE_TAGS = new Set(['a', 'span', 'b', 'strong', 'em', 'i', 'code',
    'h1', 'h2', 'h3', 'h4', 'h5', 'h6']);

  // Void elements never have a closing tag / children.
  static VOID_TAGS = new Set(['area', 'base', 'br', 'col', 'embed', 'hr', 'img',
    'input', 'link', 'meta', 'param', 'source', 'track', 'wbr']);

  // Raw-text elements whose contents must be preserved verbatim (they can hold
  // stray '<' that isn't markup).
  static RAW_TAGS = new Set(['script', 'style', 'pre', 'textarea']);

  static INDENT = '  ';

  // Pretty-print HTML.
  beautify(html) {
    const lines = [];
    let depth = 0;
    let buf = '';

    const pad = () => HTMLBeautifier.INDENT.repeat(Math.max(0, depth));
    const flush = () => {
      const t = buf.replace(/\s+/g, ' ').trim();
      if (t) {
        lines.push(pad() + t);
      }

      buf = '';
    };

    this._tokenize(html).forEach((tk) => {
      if (tk.type === 'text') {
        buf += tk.raw;
      } else if (tk.type === 'raw') {
        flush();
        lines.push(tk.raw.replace(/^\n+|\s+$/g, ''));
      } else if (tk.type === 'other') {
        flush();
        lines.push(pad() + tk.raw.trim());
      } else if (HTMLBeautifier.INLINE_TAGS.has(tk.name)) {
        buf += tk.raw;
      } else if (tk.closing) {
        flush();
        depth -= 1;
        lines.push(pad() + tk.raw.trim());
      } else if (tk.selfClose) {
        flush();
        lines.push(pad() + tk.raw.trim());
      } else {
        flush();
        lines.push(pad() + tk.raw.trim());
        depth += 1;
      }
    });
    flush();

    return lines.join('\n').replace(/\n{3,}/g, '\n\n').trim();
  }

  // Find the index just past the '>' that closes a tag opened at `start`,
  // skipping any '>' inside quoted attribute values.
  _findTagEnd(html, start) {
    let quote = null;
    for (let j = start + 1; j < html.length; j += 1) {
      const ch = html[j];
      if (quote) {
        if (ch === quote) quote = null;
      } else if (ch === '"' || ch === '\'') {
        quote = ch;
      } else if (ch === '>') {
        return j + 1;
      }
    }
    return html.length;
  }

  // Split HTML into a list of tokens. Won't error-out on malformed tags.
  _tokenize(html) {
    const tokens = [];
    const n = html.length;
    let i = 0;

    while (i < n) {
      if (html[i] !== '<') {
        const next = html.indexOf('<', i);
        const stop = next === -1 ? n : next;
        tokens.push({ type: 'text', raw: html.slice(i, stop) });
        i = stop;
      } else if (html.startsWith('<!--', i)) {
        const end = html.indexOf('-->', i + 4);
        const stop = end === -1 ? n : end + 3;
        tokens.push({ type: 'other', raw: html.slice(i, stop) });
        i = stop;
      } else if (html[i + 1] === '!' || html[i + 1] === '?') {
        const stop = this._findTagEnd(html, i);
        tokens.push({ type: 'other', raw: html.slice(i, stop) });
        i = stop;
      } else {
        const m = /^<(\/?)([a-zA-Z][a-zA-Z0-9:-]*)/.exec(html.slice(i));
        if (!m) {
          // A stray '<' that isn't a tag is considered text.
          tokens.push({ type: 'text', raw: '<' });
          i += 1;
        } else {
          const stop = this._findTagEnd(html, i);
          const raw = html.slice(i, stop);
          const name = m[2].toLowerCase();
          const closing = m[1] === '/';
          const selfClose = /\/>\s*$/.test(raw) || HTMLBeautifier.VOID_TAGS.has(name);
          tokens.push({
            type: 'tag', raw, name, closing, selfClose,
          });
          i = stop;

          // Capture raw text element contents verbatim up to its close tag.
          if (!closing && !selfClose && HTMLBeautifier.RAW_TAGS.has(name)) {
            const close = new RegExp(`</${name}\\s*>`, 'i').exec(html.slice(i));
            const contentEnd = close ? i + close.index : n;
            if (contentEnd > i) {
              tokens.push({ type: 'raw', raw: html.slice(i, contentEnd) });
            }
            i = contentEnd;
          }
        }
      }
    }

    return tokens;
  }
}

// Simple HTML to Markdown (lossy!) converter.
class HTMLToMarkdown {
  static BLOCK_TAGS = new Set(['div', 'section', 'article', 'main', 'header', 'footer',
    'aside', 'figure', 'fieldset', 'table', 'thead', 'tbody', 'tfoot', 'tr', 'td', 'th',
    'ul', 'ol', 'li', 'p', 'blockquote', 'pre', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6']);

  convert(html) {
    const div = document.createElement('div');
    div.innerHTML = html || '';

    // Drop trailing spaces before paragraph breaks (keeps Markdown's "  \n" hard
    // breaks intact) and compress blank lines.
    return this._block(div)
      .replace(/[^\S\n]+\n\n/g, '\n\n')
      .replace(/\n{3,}/g, '\n\n')
      .trim();
  }

  _isBlock(n) {
    return HTMLToMarkdown.BLOCK_TAGS.has(n.tagName.toLowerCase())
      || [...n.children].some((c) => HTMLToMarkdown.BLOCK_TAGS.has(c.tagName.toLowerCase()));
  }

  // Render inline elements (bold, italics, links, code, images).
  _inline(node) {
    let out = '';
    node.childNodes.forEach((n) => {
      if (n.nodeType === Node.TEXT_NODE) {
        out += n.textContent.replace(/\s+/g, ' ');
        return;
      }
      if (n.nodeType !== Node.ELEMENT_NODE) {
        return;
      }
      const t = n.tagName.toLowerCase();
      if (t === 'strong' || t === 'b') {
        out += `**${this._inline(n).trim()}**`;
      } else if (t === 'em' || t === 'i') {
        out += `*${this._inline(n).trim()}*`;
      } else if (t === 'code') {
        out += `\`${n.textContent}\``;
      } else if (t === 'br') {
        out += '  \n';
      } else if (t === 'a') {
        out += `[${this._inline(n).trim()}](${n.getAttribute('href') || ''})`;
      } else if (t === 'img') {
        out += `![${n.getAttribute('alt') || ''}](${n.getAttribute('src') || ''})`;
      } else {
        out += this._inline(n);
      }
    });
    return out;
  }

  // Render block-level content.
  _block(node) {
    let out = '';
    node.childNodes.forEach((n) => {
      if (n.nodeType === Node.TEXT_NODE) {
        out += n.textContent.replace(/\s+/g, ' ').replace(/^\s+$/, '');
        return;
      }
      if (n.nodeType !== Node.ELEMENT_NODE) {
        return;
      }
      const t = n.tagName.toLowerCase();
      if (/^h[1-6]$/.test(t)) {
        out += `\n${'#'.repeat(Number(t[1]))} ${this._inline(n).trim()}\n\n`;
      } else if (t === 'p') {
        out += `${this._inline(n).trim()}\n\n`;
      } else if (t === 'hr') {
        out += '\n---\n\n';
      } else if (t === 'blockquote') {
        out += `${this._block(n).trim().split('\n').map((l) => `> ${l}`).join('\n')}\n\n`;
      } else if (t === 'pre') {
        out += `\`\`\`\n${n.textContent.trim()}\n\`\`\`\n\n`;
      } else if (t === 'ul' || t === 'ol') {
        [...n.children].forEach((li, i) => {
          out += `${t === 'ol' ? `${i + 1}.` : '-'} ${this._inline(li).trim()}\n`;
        });
        out += '\n';
      } else if (t === 'br') {
        out += '\n\n';
      } else if (t === 'img') {
        out += `![${n.getAttribute('alt') || ''}](${n.getAttribute('src') || ''})\n\n`;
      } else if (this._isBlock(n)) {
        // Structural container (div, table, etc.)
        out += `${this._block(n).trim()}\n\n`;
      } else {
        out += this._inline(n);
      }
    });
    return out;
  }
}

// Simple Markdown to HTML (lossy!) converter.
class MarkdownToHTML {
  convert(md) {
    // Normalise newlines and drop NULs (used internally as placeholders).
    const src = String(md ?? '').replace(/\r\n?/g, '\n').replace(/ /g, '');
    return this._blocks(src.split('\n')).trim();
  }

  // Render an array of lines into block-level HTML.
  _blocks(lines) {
    const out = [];
    let i = 0;

    while (i < lines.length) {
      const line = lines[i];

      // Skip empty lines.
      if (line.trim() === '') { i += 1; continue; }

      // Fenced code blocks (``` or ~~~), keep them as-is.
      const fence = line.match(/^ {0,3}(`{3,}|~{3,})(.*)$/);
      if (fence) {
        const marker = fence[1][0];
        const lang = fence[2].trim().split(/\s+/)[0] || '';
        const close = new RegExp(`^ {0,3}\\${marker}{${fence[1].length},}\\s*$`);
        const body = [];
        i += 1;
        while (i < lines.length && !close.test(lines[i])) { body.push(lines[i]); i += 1; }
        i += 1; // Skip the closing fence.
        const cls = lang ? ` class="language-${this._escapeAttr(lang)}"` : '';
        out.push(`<pre><code${cls}>${this._escape(body.join('\n'))}\n</code></pre>`);
        continue;
      }

      // Headings (# .. ######)
      const h = line.match(/^ {0,3}(#{1,6})(?:\s+(.*?))?\s*$/);
      if (h) {
        const text = (h[2] || '').replace(/\s+#+\s*$/, '');
        const id = this._slug(text);
        out.push(`<h${h[1].length}${id ? ` id="${id}"` : ''}>${this._inline(text)}</h${h[1].length}>`);
        i += 1;
        continue;
      }

      // Linebreaks (---, ***, ___).
      if (/^ {0,3}([-*_])[ \t]*(?:\1[ \t]*){2,}$/.test(line)) {
        out.push('<hr />');
        i += 1;
        continue;
      }

      // Blockquotes (>).
      if (/^ {0,3}>/.test(line)) {
        const inner = [];
        while (i < lines.length && /^ {0,3}>/.test(lines[i])) {
          inner.push(lines[i].replace(/^ {0,3}> ?/, ''));
          i += 1;
        }
        out.push(`<blockquote>\n${this._blocks(inner)}\n</blockquote>`);
        continue;
      }

      // Table.
      if (line.includes('|') && this._isTableDelim(lines[i + 1])) {
        const [html, used] = this._table(lines, i);
        out.push(html);
        i += used;
        continue;
      }

      // List (ordered or unordered).
      if (/^ {0,3}([-*+]|\d{1,9}[.)])(\s+|$)/.test(line)) {
        const [html, used] = this._list(lines, i);
        out.push(html);
        i += used;
        continue;
      }

      // Raw HTML block.
      if (/^ {0,3}<(\/?[a-zA-Z][\w-]*|!--)/.test(line)) {
        const buf = [];
        while (i < lines.length && lines[i].trim() !== '') { buf.push(lines[i]); i += 1; }
        out.push(buf.join('\n'));
        continue;
      }

      // A paragraph, which gathers lines until a blank line or a new block start.
      const para = [];
      while (i < lines.length && lines[i].trim() !== '' && !this._isBlockStart(lines, i)) {
        if (para.length && /^ {0,3}(=+|-+)\s*$/.test(lines[i])) break;
        para.push(lines[i]);
        i += 1;
      }

      const setext = i < lines.length && para.length && lines[i].match(/^ {0,3}(=+|-+)\s*$/);
      const text = para.join('\n').trim();
      if (setext) {
        const level = setext[1][0] === '=' ? 1 : 2;
        out.push(`<h${level} id="${this._slug(text)}">${this._inline(text)}</h${level}>`);
        i += 1;
      } else {
        out.push(`<p>${this._inline(text)}</p>`);
      }
    }

    return out.join('\n');
  }

  // Does line `i` begin a block-level construct (used to break paragraphs)?
  _isBlockStart(lines, i) {
    const l = lines[i];
    return /^ {0,3}(`{3,}|~{3,})/.test(l)
      || /^ {0,3}#{1,6}(\s|$)/.test(l)
      || /^ {0,3}([-*_])[ \t]*(?:\1[ \t]*){2,}$/.test(l)
      || /^ {0,3}>/.test(l)
      || /^ {0,3}([-*+]|\d{1,9}[.)])(\s+|$)/.test(l)
      || /^ {0,3}<(\/?[a-zA-Z][\w-]*|!--)/.test(l)
      || (l.includes('|') && this._isTableDelim(lines[i + 1]));
  }

  _isTableDelim(l) {
    return !!l && /^ {0,3}\|?[ \t]*:?-+:?[ \t]*(\|[ \t]*:?-+:?[ \t]*)*\|?[ \t]*$/.test(l);
  }

  // Parse a list starting at `start`. Returns [html, linesConsumed].
  _list(lines, start) {
    const marker = /^(\s*)([-*+]|\d{1,9}[.)])(\s+|$)/;
    const first = lines[start].match(marker);
    const ordered = /\d/.test(first[2]);
    const indent = first[1].length;
    const items = [];
    let loose = false;
    let blank = false;
    let i = start;

    while (i < lines.length) {
      const m = lines[i].match(marker);
      if (!m || m[1].length !== indent) break;
      if (blank && items.length) loose = true;
      blank = false;

      const contentAt = m[1].length + m[2].length + (m[3].length || 1);
      const item = [lines[i].slice(contentAt)];
      i += 1;
      while (i < lines.length) {
        if (lines[i].trim() === '') { item.push(''); blank = true; i += 1; continue; }

        if (/^\s*/.exec(lines[i])[0].length >= contentAt) {
          if (blank) loose = true;
          item.push(lines[i].slice(contentAt));
          i += 1;
          continue;
        }

        break;
      }

      items.push(item);
    }

    const tag = ordered ? 'ol' : 'ul';
    const n = ordered ? parseInt(first[2], 10) : 1;
    const startAttr = ordered && n !== 1 ? ` start="${n}"` : '';
    const lis = items.map((item) => {
      let content = item;
      let prefix = '';
      const task = item[0].match(/^\[([ xX])\]\s+(.*)$/);
      if (task) {
        prefix = `<input type="checkbox" disabled${task[1] === ' ' ? '' : ' checked'} /> `;
        content = [task[2], ...item.slice(1)];
      }

      let inner = this._blocks(content).trim();
      if (!loose) inner = inner.replace(/^<p>([\s\S]*?)<\/p>/, '$1');
      return `<li>${prefix}${inner}</li>`;
    });

    return [`<${tag}${startAttr}>\n${lis.join('\n')}\n</${tag}>`, i - start];
  }

  // Parse a table starting at `start`. Returns [html, linesConsumed].
  _table(lines, start) {
    const split = (row) => {
      const s = row.trim().replace(/^\|/, '').replace(/\|$/, '');
      const cells = [];
      let cur = '';
      for (let k = 0; k < s.length; k += 1) {
        if (s[k] === '\\' && s[k + 1] === '|') { cur += '|'; k += 1; } else if (s[k] === '|') { cells.push(cur); cur = ''; } else cur += s[k];
      }
      cells.push(cur);
      return cells.map((c) => c.trim());
    };

    const header = split(lines[start]);
    const aligns = split(lines[start + 1]).map((d) => {
      const l = d.startsWith(':');
      const r = d.endsWith(':');
      if (l && r) return 'center';
      if (r) return 'right';

      return l ? 'left' : '';
    });

    const rows = [];
    let i = start + 2;
    while (i < lines.length && lines[i].trim() !== '' && lines[i].includes('|')) {
      rows.push(split(lines[i]));
      i += 1;
    }

    const cell = (t, text, a) => `<${t}${a ? ` align="${a}"` : ''}>${this._inline(text)}</${t}>`;
    let html = `<table>\n<thead>\n<tr>\n${header.map((c, k) => cell('th', c, aligns[k])).join('\n')}\n</tr>\n</thead>\n`;
    if (rows.length) {
      const body = rows.map((r) => `<tr>\n${header.map((_, k) => cell('td', r[k] || '', aligns[k])).join('\n')}\n</tr>`);
      html += `<tbody>\n${body.join('\n')}\n</tbody>\n`;
    }

    return [`${html}</table>`, i - start];
  }

  // Render inline Markdown (emphasis, code, links, images).
  _inline(text) {
    const stash = [];
    const hold = (html) => ` ${stash.push(html) - 1} `;
    let s = text;

    // Backslash escapes and hard line breaks.
    s = s.replace(/\\([\\`*_{}[\]()#+\-.!>~|"'])/g, (_, ch) => hold(this._escape(ch)));
    s = s.replace(/(?: {2,}|\\)\n/g, () => `${hold('<br />')}\n`);

    // Code spans, autolinks and raw HTML, keep as-is.
    s = s.replace(/(`+)([\s\S]*?[^`])\1(?!`)/g, (_, __, code) => hold(`<code>${this._escape(code.replace(/^ (.*) $/, '$1'))}</code>`));
    s = s.replace(/<((?:https?|ftp|mailto):[^\s<>]+)>/gi, (_, url) => hold(`<a href="${this._escapeAttr(url)}">${this._escape(url.replace(/^mailto:/i, ''))}</a>`));
    s = s.replace(/<([^\s<>@]+@[^\s<>@]+\.[^\s<>]+)>/g, (_, mail) => hold(`<a href="mailto:${this._escapeAttr(mail)}">${this._escape(mail)}</a>`));
    s = s.replace(/<\/?[a-zA-Z][^<>]*>|<!--[\s\S]*?-->/g, (m) => hold(m));

    // Escape stray text before emphasis/links inject their own markup.
    s = s.replace(/&(?![a-zA-Z][\w]*;|#\d+;|#x[\da-fA-F]+;)/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');

    // Images.
    s = s.replace(/!\[([^\]]*)\]\(\s*([^\s)]*)(?:\s+"([^"]*)")?\s*\)/g,
      (_, alt, src, title) => hold(`<img src="${this._escapeAttr(src)}" alt="${this._escapeAttr(alt)}"${title ? ` title="${this._escapeAttr(title)}"` : ''} />`));

    // Links.
    s = s.replace(/\[([^\]]*)\]\(\s*([^\s)]*)(?:\s+"([^"]*)")?\s*\)/g,
      (_, txt, href, title) => hold(`<a href="${this._escapeAttr(href)}"${title ? ` title="${this._escapeAttr(title)}"` : ''}>${this._emphasis(txt)}</a>`));

    s = this._emphasis(s);

    // Resolve placeholders.
    const resolve = (_, k) => stash[k].replace(/ (\d+) /g, resolve);
    return s.replace(/ (\d+) /g, resolve);
  }

  // Convert strikethrough, bold and italic.
  _emphasis(s) {
    return s
      .replace(/~~(?=\S)([\s\S]*?\S)~~/g, '<del>$1</del>')
      .replace(/\*\*(?=\S)([\s\S]*?\S)\*\*/g, '<strong>$1</strong>')
      .replace(/(^|[^\w])__(?=\S)([\s\S]*?\S)__(?!\w)/g, '$1<strong>$2</strong>')
      .replace(/\*(?=\S)([\s\S]*?\S)\*/g, '<em>$1</em>')
      .replace(/(^|[^\w])_(?=\S)([\s\S]*?\S)_(?!\w)/g, '$1<em>$2</em>');
  }

  _escape(s) {
    return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
  }

  _escapeAttr(s) {
    return this._escape(s).replace(/"/g, '&quot;');
  }

  // Make a slug for headingg ids.
  _slug(text) {
    return text
      .replace(/[*_`~]/g, '')
      .replace(/!?\[([^\]]*)\]\([^)]*\)/g, '$1')
      .toLowerCase()
      .replace(/[^\w\- ]+/g, '')
      .trim()
      .replace(/\s+/g, '-');
  }
}

// A simple HTML to plaintext converter and converts links to "text (url)".
class HTMLToText {
  convert(html) {
    const div = document.createElement('div');
    div.innerHTML = html || '';
    return trimLines(this._walk(div), true).trim();
  }

  _walk(node) {
    let out = '';
    node.childNodes.forEach((n) => {
      if (n.nodeType === Node.TEXT_NODE) {
        out += n.textContent.replace(/\s+/g, ' ');
        return;
      }
      if (n.nodeType !== Node.ELEMENT_NODE) {
        return;
      }
      const t = n.tagName.toLowerCase();
      if (t === 'br') {
        out += '\n';
      } else if (t === 'a') {
        const text = this._walk(n).trim();
        const href = n.getAttribute('href') || '';
        out += href && href !== text ? `${text} (${href})` : text;
      } else if (t === 'li') {
        out += `- ${this._walk(n).trim()}\n`;
      } else if (/^h[1-6]$/.test(t) || ['p', 'div', 'ul', 'ol', 'blockquote', 'tr', 'table'].includes(t)) {
        out += `${this._walk(n).trim()}\n\n`;
      } else {
        out += this._walk(n);
      }
    });
    return out;
  }
}

// Convert Markdown to email-builder JSON struct.
class MarkdownToVisual {
  convert(markdown) {
    const lines = markdown.split('\n');
    const blocks = [];
    const idBase = Date.now();
    let textBuf = [];

    const createBlock = (type, props, style = {}) => ({
      id: `block-${idBase + blocks.length}`,
      type,
      data: {
        props,
        style: {
          padding: {
            top: 16, bottom: 16, right: 24, left: 24,
          },
          ...style,
        },
      },
    });

    const flushText = () => {
      if (textBuf.length > 0) {
        blocks.push(createBlock('Text', { markdown: true, text: textBuf.join('\n') }));
        textBuf = [];
      }
    };

    lines.forEach((line) => {
      // Handle ATX headings (# Heading).
      const heading = line.match(/^(#+)\s+(.*)/);
      if (heading) {
        flushText();

        blocks.push(createBlock('Heading', {
          text: heading[2],
          level: `h${Math.min(heading[1].length, 6)}`,
        }));
        return;
      }

      // Setext headings (===== or -----).
      const trimmed = line.trim();
      if (/^(=+|-+)$/.test(trimmed) && textBuf.length > 0) {
        const lastLine = textBuf.pop();
        if (lastLine.trim()) {
          flushText();

          blocks.push(createBlock('Heading', {
            text: lastLine,
            level: trimmed[0] === '=' ? 'h1' : 'h2',
          }));

          return;
        }

        textBuf.push(lastLine, line);
      } else {
        textBuf.push(line);
      }
    });

    flushText();

    return {
      root: {
        type: 'EmailLayout',
        data: { childrenIds: blocks.map((b) => b.id) },
      },
      ...Object.fromEntries(blocks.map((b) => [b.id, { type: b.type, data: b.data }])),
    };
  }
}

// Init the singletons.
const beautifier = new HTMLBeautifier();
const mdConverter = new HTMLToMarkdown();
const htmlConverter = new MarkdownToHTML();
const textConverter = new HTMLToText();
const visualConverter = new MarkdownToVisual();


class ContentConverter {
  // Convert a campaign body from one content type to another. Returns
  // { body, bodySource }. All conversions run client-side.
  convert(body, from, to) {
    let out = body ?? '';
    let bodySource = null;

    let isHTML = false;
    if (from === 'richtext' || from === 'html' || from === 'visual') {
      // Normalize the source HTML.
      const d = document.createElement('div');
      d.innerHTML = out;
      out = d.innerHTML.trim();
      isHTML = true;
    }

    if (isHTML) {
      switch (to) {
        case 'plain': {
          const d = document.createElement('div');
          d.innerHTML = out;
          out = trimLines(d.innerText.trim(), true);
          break;
        }

        case 'markdown':
          out = mdConverter.convert(out);
          break;

        case 'visual':
          bodySource = JSON.stringify(visualConverter.convert(mdConverter.convert(out)));
          break;

        default:
          // Prettify when switching between HTML formats (richtext <> html)
          out = beautifier.beautify(out);
          break;
      }
    } else if (from === 'markdown' && (to === 'richtext' || to === 'html')) {
      out = beautifier.beautify(htmlConverter.convert(out).trim());
    } else if (from === 'plain' && (to === 'richtext' || to === 'html')) {
      out = out.replace(/\n/ig, '<br>\n');
    } else if (to === 'visual') {
      bodySource = JSON.stringify(visualConverter.convert(out));
    }

    return { body: out, bodySource };
  }
}

const converter = new ContentConverter();

// Export APIs.
export const convert = (body, from, to) => converter.convert(body, from, to);
export const beautifyHTML = (html) => beautifier.beautify(html);
export const htmlToText = (html) => textConverter.convert(html);

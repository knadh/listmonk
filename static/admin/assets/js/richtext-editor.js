// <richtext-editor> is a WebComponent wrapping Squire (squire-rte).
import Squire from 'squire-rte';
import { beautifyHTML } from './content.js';
import { setupTable } from './richtext/table.js';
import { setupResize } from './richtext/resize.js';
import {
  i18n, t, closestInSelection, PALETTE, EMOJIS, CHARS,
} from './richtext/data.js';

const TRACK_SUFFIX = '@TrackLink';
const TRACK_KEY = 'trackLink';

// Simple toolbar toggles.
const TOGGLES = {
  bold: ['B', 'bold', 'removeBold'],
  italic: ['I', 'italic', 'removeItalic'],
  underline: ['U', 'underline', 'removeUnderline'],
  strikethrough: ['S', 'strikethrough', 'removeStrikethrough'],
  subscript: ['SUB', 'subscript', 'removeSubscript'],
  superscript: ['SUP', 'superscript', 'removeSuperscript'],
  ul: ['UL', 'makeUnorderedList', 'removeList'],
  ol: ['OL', 'makeOrderedList', 'removeList'],
};

const sanitizeToDOMFragment = (html) => {
  const tpl = document.createElement('template');
  tpl.innerHTML = html || '';
  return document.importNode(tpl.content, true);
};

class RichtextEditor extends HTMLElement {
  editor = null;
  #source = null;
  #toolbar = null;
  #pending = null;
  #mute = false;
  #resizeObs = null;

  connectedCallback() {
    if (Object.hasOwn(this, 'value')) {
      const v = this.value;
      delete this.value;
      this.value = v;
    }

    this.#source = this.querySelector('[data-rte-source]');
    this.#toolbar = this.querySelector('.rte-toolbar');
    this.content = this.querySelector('[data-rte-content]');
    if (this.#source) {
      this.#source.hidden = true;
    }

    this.content.setAttribute('spellcheck', 'true');
    this.editor = new Squire(this.content, { blockTag: 'P', sanitizeToDOMFragment });
    this.editor.setHTML(this.#source?.value ?? this.#pending ?? this.getAttribute('value') ?? '');

    const disabled = this.hasAttribute('disabled');
    if (disabled) {
      this.content.setAttribute('contenteditable', 'false');
      this.classList.add('is-disabled');
    } else {
      this.initToolbar();
      this.initShortcuts();
      this.#setupFind();
      this.#setupOverflow();
      this.#fillGrids();
      this.#setupColorInputs();
      setupTable(this);
      setupResize(this);

      // Generic cancel buttons close their dialog.
      this.querySelectorAll('[data-rte-act="cancel"]').forEach((b) => {
        b.addEventListener('click', () => b.closest('dialog').close());
      });
    }

    this.editor.addEventListener('input', () => this.sync());
    this.editor.addEventListener('pathChange', () => this.#updateState());
    this.editor.addEventListener('select', () => this.#updateState());
    this.editor.addEventListener('undoStateChange', (e) => this.#updateUndo(e.detail));
    this.#updateState();
    this.#updateWordcount();

    // Signal the host that the editor is initialized.
    queueMicrotask(() => {
      this.dispatchEvent(new CustomEvent('editor-ready', { bubbles: true }));
      if (!disabled) {
        this.editor.focus();
      }
    });
  }

  disconnectedCallback() {
    this.#resizeObs?.disconnect();
    this.#resizeObs = null;
    this.editor?.destroy();
    this.editor = null;
  }

  // Main value getter/setter.
  get value() {
    return this.editor
      ? this.editor.getHTML()
      : (this.#pending ?? this.getAttribute('value') ?? '');
  }

  set value(val) {
    const v = val == null ? '' : String(val);
    if (!this.editor) {
      this.#pending = v;
    } else if (v !== this.editor.getHTML()) {
      this.#mute = true;
      this.editor.setHTML(v);
      this.#mute = false;
    }

    if (this.#source) {
      this.#source.value = v;
    }
  }

  // Push the current content to the host.
  sync() {
    if (!this.editor || this.#mute) {
      return;
    }
    if (this.#source) {
      this.#source.value = this.editor.getHTML();
    }

    this.#updateWordcount();
    this.dispatchEvent(new Event('input', { bubbles: true }));
  }

  // Toolbar.
  initToolbar() {
    // Keep the caret/selection in the editor when a toolbar control is pressed.
    this.#toolbar.addEventListener('mousedown', (e) => {
      if (e.target.closest('button')) {
        e.preventDefault();
      }
    });

    this.#toolbar.addEventListener('click', (e) => {
      const btn = e.target.closest('[data-cmd]');

      if (btn && this.#toolbar.contains(btn)) {
        e.preventDefault();
        this.#run(btn.dataset.cmd, btn.dataset.value);
      }
    });
  }

  #run(cmd, value) {
    const ed = this.editor;
    if (TOGGLES[cmd]) {
      const [tag, on, off] = TOGGLES[cmd];
      ed[ed.hasFormat(tag) ? off : on]();
    } else {
      switch (cmd) {
        case 'undo': ed.undo(); break;
        case 'redo': ed.redo(); break;
        case 'fontsize': ed.setFontSize(value); break;
        case 'forecolor': ed.setTextColor(value || ''); break;
        case 'backcolor': ed.setHighlightColor(value || ''); break;
        case 'block': this.#setBlock(value); break;
        case 'align': ed.setTextAlignment(value); break;
        case 'indent': this.#indent('increase'); break;
        case 'outdent': this.#indent('decrease'); break;
        case 'hr': ed.insertHTML('<hr>'); break;
        case 'removeformat': ed.removeAllFormatting(); break;
        case 'link': this.#openLink(); break;
        case 'anchor': this.#insertAnchor(); break;
        case 'image': this.#openImage(); break;
        case 'fullscreen': this.classList.toggle('is-fullscreen'); break;
        case 'help': this.#dialog('help').showModal(); break;
        case 'source':
        case 'insert-html':
          this.#openCode(cmd);
          return;
        default: break;
      }
    }

    // Table/charmap/emoji/find are native popovers.
    this.#closePopovers();
  }

  // Block-level formatting.
  #setBlock(tag) {
    const ed = this.editor;
    if (tag === 'BLOCKQUOTE') {
      if (!ed.hasFormat('BLOCKQUOTE')) {
        ed.increaseQuoteLevel();
      }

      return;
    }
    if (tag === 'PRE') {
      ed[ed.hasFormat('PRE') ? 'removeCode' : 'code']();
      return;
    }

    const rename = (node) => {
      for (const child of [...node.childNodes]) {
        if (!/^(?:P|DIV|H[1-6]|BLOCKQUOTE|PRE)$/.test(child.nodeName)) {
          rename(child);
          continue;
        }

        const el = document.createElement(tag);
        for (const attr of ['style', 'dir']) {
          if (child.hasAttribute(attr)) {
            el.setAttribute(attr, child.getAttribute(attr));
          }
        }

        el.append(...child.childNodes);
        child.replaceWith(el);
      }
    };

    ed.modifyBlocks((frag) => {
      rename(frag);
      return frag;
    });
  }

  // Indent/outdent.
  #indent(dir) {
    const ed = this.editor;
    const inList = /[OU]L>/.test(ed.getPath()) || ed.hasFormat('LI');
    ed[dir + (inList ? 'ListLevel' : 'QuoteLevel')]();
  }

  // Active-state.
  #updateState() {
    const ed = this.editor;
    const path = ed.getPath() || '';

    for (const btn of this.#toolbar.querySelectorAll('[data-format]')) {
      btn.classList.toggle('is-active', ed.hasFormat(btn.dataset.format));
    }

    // Colour buttons.
    const font = ed.getFontInfo();
    this.#toolbar.querySelector('[popovertarget="rte-forecolor"]')?.classList.toggle('is-active', !!font.color);
    this.#toolbar.querySelector('[popovertarget="rte-backcolor"]')?.classList.toggle('is-active', !!font.backgroundColor);

    // Link / image.
    this.#toolbar.querySelector('[data-cmd="link"]')
      ?.classList.toggle('is-active', !!closestInSelection(ed, this.content, 'a[href]'));
    this.#toolbar.querySelector('[data-cmd="image"]')
      ?.classList.toggle('is-active', !!closestInSelection(ed, this.content, 'img'));

    // Alignment.
    const align = this.getAlign();
    for (const btn of this.#toolbar.querySelectorAll('[data-cmd="align"]')) {
      btn.classList.toggle('is-active', btn.dataset.value === align);
    }

    // Block-format label.
    const m = path.match(/(H[1-6]|BLOCKQUOTE|PRE)/);
    this.#toolbar.querySelector('[data-rte-blocklabel]').textContent = !m
      ? t('campaigns.editor.paragraph')
      : ({ BLOCKQUOTE: t('campaigns.editor.blockquote'), PRE: t('campaigns.editor.code') })[m[1]]
      ?? `${t('campaigns.editor.heading')} ${m[1][1]}`;

    // Statusbar element path.
    this.querySelector('[data-rte-path]').textContent = path.replace(/^(?:P|DIV)>?/, '').replace(/>/g, ' › ').toLowerCase();
  }

  // Get the explicit alignment.
  getAlign() {
    const range = this.editor.getSelection();
    let node = range?.commonAncestorContainer;
    if (node?.nodeType === 3) {
      node = node.parentNode;
    }

    const block = node?.nodeType === 1
      ? node.closest('p, div, li, h1, h2, h3, h4, h5, h6, blockquote, pre, td, th') : null;
    if (!block || !this.content.contains(block)) {
      return '';
    }

    const val = block.style.textAlign;
    return { start: 'left', end: 'right' }[val] ?? val;
  }

  #updateUndo({ canUndo, canRedo }) {
    this.#toolbar.querySelector('[data-cmd="undo"]').disabled = !canUndo;
    this.#toolbar.querySelector('[data-cmd="redo"]').disabled = !canRedo;
  }

  #updateWordcount() {
    const text = this.content.textContent.trim();
    const n = text ? text.split(/\s+/).length : 0;
    this.querySelector('[data-rte-wordcount]').textContent = i18n.tc('campaigns.editor.words', n, { num: n });
  }

  // Responsive toolbar
  #setupOverflow() {
    const bar = this.#toolbar;
    const overflow = bar.querySelector('.rte-overflow');
    const menu = overflow.querySelector('.rte-overflow-menu');
    const groups = [...bar.querySelectorAll(':scope > .rte-group')];

    const reflow = () => {
      // Restore every group to the bar and measure against a single line.
      groups.forEach((g) => bar.insertBefore(g, overflow));
      overflow.hidden = true;
      if (bar.scrollWidth <= bar.clientWidth) {
        return;
      }

      // Show the control on overflolw.
      overflow.hidden = false;
      overflow.style.marginInlineStart = '0';
      for (let i = groups.length - 1; i >= 0 && bar.scrollWidth > bar.clientWidth; i -= 1) {
        menu.insertBefore(groups[i], menu.firstChild);
      }
      overflow.style.marginInlineStart = '';
    };

    this.#resizeObs = new ResizeObserver(() => window.requestAnimationFrame(reflow));
    this.#resizeObs.observe(bar);
    reflow();
  }

  // Dialogs.
  #dialog(name) {
    return this.querySelector(`[data-rte-dialog="${name}"]`);
  }

  #closePopovers() {
    this.querySelectorAll('.rte-menu:popover-open, .rte-colors:popover-open').forEach((m) => m.hidePopover());
  }

  // Link dialog.
  #openLink() {
    const ed = this.editor;
    const d = this.#dialog('link');
    const f = d.querySelector('form');
    const sel = ed.getSelectedText();
    const a = closestInSelection(ed, this.content, 'a');

    // Pre-fill value from an existing link under the cursor?
    const href = a?.getAttribute('href') ?? '';
    const tracked = href.endsWith(TRACK_SUFFIX);
    f.url.value = tracked ? href.slice(0, -TRACK_SUFFIX.length) : href;
    f.text.value = a ? a.textContent : sel;
    f.title.value = a?.getAttribute('title') ?? '';
    f.newwindow.checked = a?.getAttribute('target') === '_blank';
    f.track.checked = a ? tracked : localStorage.getItem(TRACK_KEY) === 'true';

    d.querySelector('[data-rte-act="submit"]').onclick = () => {
      let url = f.url.value.trim();
      if (!url) {
        d.close();
        return;
      }

      localStorage.setItem(TRACK_KEY, f.track.checked);
      if (f.track.checked && /^https?:\/\//i.test(url)) {
        url += TRACK_SUFFIX;
      }

      const attrs = {};
      if (f.title.value.trim()) {
        attrs.title = f.title.value.trim();
      }
      if (f.newwindow.checked) {
        attrs.target = '_blank';
        attrs.rel = 'noopener noreferrer';
      }

      ed.focus();
      const label = f.text.value.trim();
      if (a) {
        const range = document.createRange();
        range.selectNode(a);
        ed.setSelection(range);
        ed.makeLink(url, attrs);
      } else if (label && !sel) {
        // Nothing selected, so insert a new labelled link.
        const el = Object.assign(document.createElement('a'), { textContent: label });
        el.setAttribute('href', url);
        Object.entries(attrs).forEach(([k, v]) => el.setAttribute(k, v));
        ed.insertHTML(el.outerHTML);
      } else {
        ed.makeLink(url, attrs);
      }
      d.close();
    };

    d.querySelector('[data-rte-act="unlink"]').onclick = () => {
      ed.focus();
      ed.removeLink();
      d.close();
    };
    d.showModal();
    f.url.focus();
  }

  #insertAnchor() {
    const name = window.prompt(t('campaigns.editor.anchorName'));
    if (name) {
      this.editor.focus();
      this.editor.insertHTML(`<a id="${name.replace(/"/g, '')}"></a>`);
    }
  }

  // Image dialog.
  #openImage() {
    const ed = this.editor;
    const d = this.#dialog('image');
    const f = d.querySelector('form');
    const img = closestInSelection(ed, this.content, 'img');

    f.src.value = img?.getAttribute('src') ?? '';
    f.alt.value = img?.getAttribute('alt') ?? '';
    f.width.value = img?.getAttribute('width') ?? '';
    f.height.value = img?.getAttribute('height') ?? '';
    f.float.value = (img && ['img-float-left', 'img-float-right'].find((c) => img.classList.contains(c))) || '';
    f.embed.checked = img?.hasAttribute('data-embed') ?? false;

    // Constrain-proportions lock.
    const lock = d.querySelector('[data-rte-act="lock"]');
    const locked = () => lock.getAttribute('aria-pressed') === 'true';
    const rat = (w, h) => (w > 0 && h > 0 ? w / h : null);
    let ratio = img?.naturalWidth
      ? rat(img.naturalWidth, img.naturalHeight)
      : rat(parseFloat(f.width.value), parseFloat(f.height.value));

    // Get the ratio from an image's actual size.
    const loadRatio = (src) => {
      if (src) {
        const probe = new Image();
        probe.onload = () => { ratio = rat(probe.naturalWidth, probe.naturalHeight); };
        probe.src = src;
      }
    };
    if (!ratio) {
      loadRatio(f.src.value.trim());
    }
    f.src.oninput = () => {
      ratio = null;
      loadRatio(f.src.value.trim());
    };

    const scale = (from, to, fn) => {
      const v = parseFloat(from.value);
      if (locked() && ratio && Number.isFinite(v)) {
        to.value = Math.round(fn(v));
      }
    };
    f.width.oninput = () => scale(f.width, f.height, (w) => w / ratio);
    f.height.oninput = () => scale(f.height, f.width, (h) => h * ratio);

    lock.onclick = () => {
      const on = !locked();
      lock.setAttribute('aria-pressed', on);
      lock.classList.toggle('is-active', on);
      const use = lock.querySelector('use');
      use.setAttribute('href', `${(use.getAttribute('href') || '').split('#')[0]}#icon-${on ? 'lock' : 'unlock'}`);
    };

    d.querySelector('[data-rte-act="browse"]').onclick = () => {
      this.dispatchEvent(new CustomEvent('richtext-media', {
        bubbles: true,
        detail: {
          cb: (url) => {
            f.src.value = url;
            ratio = null;
            loadRatio(url);
          },
        },
      }));
    };

    d.querySelector('[data-rte-act="submit"]').onclick = () => {
      const src = f.src.value.trim();
      if (!src) {
        d.close();
        return;
      }
      const attrs = Object.fromEntries([
        ['alt', f.alt.value.trim()],
        ['width', f.width.value.trim()],
        ['height', f.height.value.trim()],
        ['class', f.float.value],
        ['data-embed', f.embed.checked ? 'true' : ''],
      ].filter(([, v]) => v));

      ed.focus();
      if (img) {
        // Update the existing image in place.
        ed.saveUndoState();
        img.setAttribute('src', src);
        for (const k of ['alt', 'width', 'height', 'class', 'data-embed']) {
          if (attrs[k]) {
            img.setAttribute(k, attrs[k]);
          } else {
            img.removeAttribute(k);
          }
        }

        this.sync();
      } else {
        ed.insertImage(src, attrs);
      }
      d.close();
    };
    d.showModal();
    f.src.focus();
  }

  // Source and 'Insert HTML' dialogs share a CodeMirror <code-editor>.
  #openCode(name) {
    const d = this.#dialog(name);
    const body = d.querySelector('[data-rte-code]');

    const code = document.createElement('code-editor');
    code.setAttribute('lang', 'html');
    const ta = document.createElement('textarea');
    ta.value = name === 'source' ? this.editor.getHTML() : '';
    code.append(ta);
    body.replaceChildren(code);

    d.querySelector('[data-rte-act="format"]').onclick = () => {
      code.value = beautifyHTML(code.value);
    };

    d.querySelector('[data-rte-act="submit"]').onclick = () => {
      this.editor.focus();
      if (name === 'source') {
        this.value = code.value;
        this.sync();
      } else {
        this.editor.insertHTML(code.value);
      }
      d.close();
    };
    d.addEventListener('close', () => body.replaceChildren(), { once: true });
    d.showModal();
  }

  // Find and replace.
  #setupFind() {
    const panel = this.querySelector('[data-rte-find]');
    const f = (n) => panel.querySelector(`[name="${n}"]`);
    const count = panel.querySelector('[data-rte-findcount]');
    let matches = [];
    let idx = -1;

    const clearMarks = () => {
      for (const m of this.content.querySelectorAll('mark.rte-find')) {
        const parent = m.parentNode;
        m.replaceWith(...m.childNodes);
        parent.normalize();
      }
    };

    const next = () => {
      if (!matches.length) {
        return;
      }

      matches[idx]?.classList.remove('is-current');
      idx = (idx + 1) % matches.length;
      matches[idx].classList.add('is-current');
      matches[idx].scrollIntoView({ block: 'center' });
    };

    const doFind = () => {
      clearMarks();
      matches = [];
      idx = -1;

      const term = f('find').value;
      if (!term) {
        count.textContent = '';
        return;
      }

      const re = new RegExp(`(${term.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, f('matchcase').checked ? '' : 'i');
      const walker = document.createTreeWalker(this.content, NodeFilter.SHOW_TEXT);
      const nodes = [];
      while (walker.nextNode()) {
        nodes.push(walker.currentNode);
      }
      for (const node of nodes) {
        const parts = node.nodeValue.split(re);
        if (parts.length < 2) {
          continue;
        }

        const frag = document.createDocumentFragment();
        parts.forEach((part, i) => {
          if (i % 2) {
            const mark = Object.assign(document.createElement('mark'), { className: 'rte-find', textContent: part });
            frag.append(mark);
            matches.push(mark);
          } else if (part) {
            frag.append(part);
          }
        });
        node.replaceWith(frag);
      }

      count.textContent = i18n.ts('campaigns.editor.findCount', { num: matches.length });
      if (matches.length) {
        next();
      }
    };

    const replaceOne = () => {
      if (!matches[idx]) {
        doFind();
        return;
      }

      matches[idx].replaceWith(f('replace').value);
      matches.splice(idx, 1);
      idx -= 1;
      this.sync();
      next();
    };

    const replaceAll = () => {
      matches.forEach((mark) => mark.replaceWith(f('replace').value));
      matches = [];
      idx = -1;
      count.textContent = '';
      this.sync();
    };

    f('find').addEventListener('input', doFind);
    f('matchcase').addEventListener('change', doFind);
    f('find').addEventListener('keydown', (e) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        next();
      }
    });

    panel.querySelector('[data-rte-act="next"]').addEventListener('click', next);
    panel.querySelector('[data-rte-act="replace"]').addEventListener('click', replaceOne);
    panel.querySelector('[data-rte-act="replaceAll"]').addEventListener('click', replaceAll);

    // Focus the input on open.
    panel.addEventListener('toggle', (e) => {
      if (e.newState === 'open') {
        setTimeout(() => f('find').focus(), 0);
        if (f('find').value) {
          doFind();
        }
      } else {
        clearMarks();
        this.sync();
      }
    });
  }

  // Colour / emoji / char grids.
  #fillGrids() {
    // Colour swatch palettes.
    for (const menu of this.querySelectorAll('[data-rte-palette]')) {
      for (const c of [...PALETTE, '']) {
        const b = Object.assign(document.createElement('button'), {
          type: 'button',
          className: c ? 'rte-swatch' : 'rte-swatch rte-swatch-none',
          title: c || t('globals.terms.none'),
          textContent: c ? '' : '×',
        });

        Object.assign(b.dataset, { cmd: menu.dataset.rtePalette, value: c });
        b.style.background = c;
        menu.append(b);
      }
    }

    // Special character and emoji grids.
    for (const [holder, glyphs] of [['[data-rte-charmap]', CHARS], ['[data-rte-emoji]', EMOJIS]]) {
      const grid = this.querySelector(holder);
      for (const ch of glyphs) {
        const b = Object.assign(document.createElement('button'), {
          type: 'button', className: 'ghost rte-glyph', textContent: ch,
        });
        // Keep the popover open so multiple glyphs can be inserted in sequence.
        b.addEventListener('click', () => {
          this.editor.focus();
          this.editor.insertHTML(ch);
        });

        grid.append(b);
      }
    }
  }

  // Manual hex entry + native colour picker in the colour popovers.
  #setupColorInputs() {
    const norm = (v) => {
      const s = v.trim().replace(/^#?/, '');
      return /^([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/.test(s) ? `#${s}` : null;
    };

    // Convert rgb to hex.
    const toHex = (rgb) => {
      const m = rgb?.match(/\d+/g);
      if (!m || m.length < 3) {
        return null;
      }
      return `#${m.slice(0, 3).map((n) => Number(n).toString(16).padStart(2, '0')).join('')}`;
    };

    for (const row of this.querySelectorAll('[data-rte-colorinput]')) {
      const cmd = row.dataset.rteColorinput;
      const picker = row.querySelector('input[name="picker"]');
      const hex = row.querySelector('input[name="hex"]');

      // On open, set colour of the selected item.
      row.closest('[popover]').addEventListener('toggle', (e) => {
        if (e.newState !== 'open') {
          return;
        }
        const font = this.editor.getFontInfo();
        const cur = toHex(cmd === 'forecolor' ? font.color : font.backgroundColor);
        hex.value = cur ?? '';
        if (cur) {
          picker.value = cur;
        }
      });

      // Picker to hex.
      picker.addEventListener('input', () => { hex.value = picker.value; });

      // Hex to picker (preview).
      hex.addEventListener('input', () => {
        const c = norm(hex.value);
        if (c) {
          picker.value = c;
        }
      });

      const apply = () => {
        const c = norm(hex.value) ?? picker.value;
        this.#run(cmd, c);
      };

      row.querySelector('[data-rte-act="applycolor"]').addEventListener('click', apply);
      hex.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
          e.preventDefault();
          apply();
        }
      });
    }
  }

  // Shortcuts.
  initShortcuts() {
    const emit = (name) => (self, e) => {
      e.preventDefault();
      this.dispatchEvent(new CustomEvent(name, { bubbles: true }));
    };

    const link = (self, e) => {
      e.preventDefault();
      this.#openLink();
    };

    const keys = {
      'Ctrl-s': emit('campaign-save'),
      'Meta-s': emit('campaign-save'),
      F9: emit('campaign-preview'),
      'Ctrl-k': link,
      'Meta-k': link,

      // Capture Enter+Shift/Enter to porperly handle para line and line
      // break insertion across contexts. Squire's own implementation doesn't
      // work properly inside <td>
      Enter: (self, e) => this.#splitBlock(self, e),
      'Shift-Enter': (self, e) => this.#splitBlock(self, e),
    };

    Object.entries(keys).forEach(([key, fn]) => this.editor.setKeyHandler(key, fn));
  }

  // Handle Enter (<p>) and Shift+Enter (<br>) separateely. Squire doesn't have this out of the box.
  #splitBlock(ed, e) {
    e.preventDefault();
    ed.splitBlock(e.shiftKey);

    const range = ed.getSelection();
    if (!range?.collapsed) {
      return;
    }
    const { startContainer: node, startOffset } = range;
    const br = node.nodeType === 1 ? node.childNodes[startOffset - 1]
      : (startOffset === 0 ? node.previousSibling : null);
    if (br?.nodeName === 'BR' && !br.nextSibling) {
      br.after(document.createElement('br'));
    }
  }
}

if (!customElements.get('richtext-editor')) {
  customElements.define('richtext-editor', RichtextEditor);
}

import { closestInSelection } from './data.js';

export function setupResize(cmp) {
  const ed = cmp.editor;
  let img = null;

  const overlay = document.createElement('div');
  overlay.className = 'rte-resize';
  overlay.hidden = true;
  overlay.innerHTML = ['nw', 'ne', 'sw', 'se'].map((pos) => `<span class="rte-resize-handle rte-${pos}" data-pos="${pos}"></span>`).join('');
  cmp.append(overlay);

  const hide = () => {
    img = null;
    overlay.hidden = true;
  };

  const place = () => {
    if (!img || !cmp.contains(img)) {
      hide();
      return;
    }

    const wrap = cmp.getBoundingClientRect();
    const r = img.getBoundingClientRect();
    overlay.hidden = false;

    Object.assign(overlay.style, {
      top: `${r.top - wrap.top + cmp.scrollTop}px`,
      left: `${r.left - wrap.left}px`,
      width: `${r.width}px`,
      height: `${r.height}px`,
    });
  };

  const track = () => {
    img = closestInSelection(ed, cmp.content, 'img');
    if (img) {
      place();
    } else {
      hide();
    }
  };

  // Track selection changes and direct clicks on images.
  ed.addEventListener('pathChange', track);
  ed.addEventListener('select', track);
  ed.addEventListener('blur', () => setTimeout(() => {
    if (!overlay.matches(':hover')) {
      hide();
    }
  }, 150));

  cmp.content.addEventListener('mouseup', (e) => {
    if (e.target.nodeName === 'IMG') {
      img = e.target;
      place();
    }
  });

  cmp.addEventListener('scroll', () => img && place(), true);

  // Drag a corner handle to resize, keeping the aspect ratio.
  overlay.addEventListener('mousedown', (e) => {
    const handle = e.target.closest('.rte-resize-handle');
    if (!handle || !img) {
      return;
    }

    e.preventDefault();
    const r = img.getBoundingClientRect();
    const sign = handle.dataset.pos.endsWith('w') ? -1 : 1;
    const ratio = r.width / (r.height || 1);
    let w = 0;
    let h = 0;

    const onMove = (ev) => {
      w = Math.max(16, Math.round(r.width + sign * (ev.clientX - e.clientX)));
      h = Math.round(w / ratio);
      Object.assign(overlay.style, { width: `${w}px`, height: `${h}px` });
    };

    document.addEventListener('mousemove', onMove);
    document.addEventListener('mouseup', () => {
      document.removeEventListener('mousemove', onMove);
      if (img && w) {
        ed.saveUndoState();
        img.setAttribute('width', w);
        img.setAttribute('height', h);
        img.style.removeProperty('width');
        img.style.removeProperty('height');
        cmp.sync();
      }
      setTimeout(place, 0);
    }, { once: true });
  });
}

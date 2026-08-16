// Custom table plugin for the Squire-based <richtext-editor>
import { t, closestInSelection } from './data.js';

const GRID = 10;
const TABLE_STYLE = 'border-collapse:collapse;width:100%';
const CELL_STYLE = 'border:1px solid #ccc;padding:8px';

const tableHTML = (rows, cols) => {
  const row = `<tr>${`<td style="${CELL_STYLE}"><br></td>`.repeat(cols)}</tr>`;
  return `<table style="${TABLE_STYLE}" border="1"><tbody>${row.repeat(rows)}</tbody></table>`;
};

const newCell = (like) => {
  const el = document.createElement(like?.nodeName || 'TD');
  el.setAttribute('style', CELL_STYLE);
  el.append(document.createElement('br'));

  return el;
};

// Row/column/table operations relative to the given cell.
function run(op, cell) {
  const row = cell.parentNode;
  const table = cell.closest('table');
  const idx = [...row.children].indexOf(cell);
  const rows = [...table.querySelectorAll('tr')];

  switch (op) {
    case 'rowAbove':
    case 'rowBelow': {
      const tr = document.createElement('tr');
      tr.append(...[...row.children].map((c) => newCell(c)));
      row.parentNode.insertBefore(tr, op === 'rowAbove' ? row : row.nextSibling);

      break;
    }
    case 'rowDelete':
      if (rows.length > 1) {
        row.remove();
      } else {
        table.remove();
      }

      break;
    case 'colLeft':
    case 'colRight':
      for (const r of rows) {
        const ref = r.children[idx];
        r.insertBefore(newCell(ref), (op === 'colLeft' ? ref : ref?.nextSibling) ?? null);
      }

      break;
    case 'colDelete':
      if (row.children.length > 1) {
        rows.forEach((r) => r.children[idx]?.remove());
      } else {
        table.remove();
      }

      break;
    case 'tableDelete':
      table.remove();

      break;
    default:
      break;
  }
}

export function setupTable(cmp) {
  const ed = cmp.editor;
  const currentCell = () => closestInSelection(ed, cmp.content, 'td,th');

  // Grid-size picker.
  const grid = cmp.querySelector('[data-rte-tablegrid]');
  const label = cmp.querySelector('[data-rte-tablegrid-label]');
  const cells = Array.from({ length: GRID * GRID }, (_, i) => {
    const cell = document.createElement('span');
    cell.className = 'rte-tablegrid-cell';
    Object.assign(cell.dataset, { r: Math.floor(i / GRID) + 1, c: (i % GRID) + 1 });
    grid.append(cell);
    return cell;
  });
  const paint = (rows, cols) => {
    cells.forEach((cell) => cell.classList.toggle('is-active', cell.dataset.r <= rows && cell.dataset.c <= cols));
    label.textContent = rows ? `${rows} × ${cols}` : '';
  };

  paint(0, 0);

  grid.addEventListener('mouseover', (e) => {
    const cell = e.target.closest('.rte-tablegrid-cell');
    if (cell) {
      paint(+cell.dataset.r, +cell.dataset.c);
    }
  });

  grid.addEventListener('mouseleave', () => paint(0, 0));

  grid.addEventListener('click', (e) => {
    const cell = e.target.closest('.rte-tablegrid-cell');
    if (!cell) {
      return;
    }
    ed.focus();
    ed.insertHTML(tableHTML(+cell.dataset.r, +cell.dataset.c));
    paint(0, 0);
    grid.closest('[popover]')?.hidePopover();
  });

  // Floating cell toolbar.
  const base = (cmp.querySelector('.rte-toolbar svg.icon use')?.getAttribute('href') ?? '').split('#')[0];
  const btn = (op, icon, glyph = '') => `<button type="button" class="ghost" data-op="${op}" title="${t(`campaigns.editor.${op}`)}">`
    + `<svg class="icon"><use href="${base}#icon-${icon}"></use></svg>${glyph && `<span>${glyph}</span>`}</button>`;

  const bar = document.createElement('div');
  bar.className = 'rte-table-tools';
  bar.hidden = true;
  bar.innerHTML = btn('rowAbove', 'plus', '↑') + btn('rowBelow', 'plus', '↓') + btn('rowDelete', 'trash-2', '—')
    + btn('colLeft', 'plus', '←') + btn('colRight', 'plus', '→') + btn('colDelete', 'trash-2', '|')
    + btn('tableDelete', 'trash-2');
  cmp.append(bar);

  const reposition = () => {
    const cell = currentCell();
    bar.hidden = !cell;
    if (!cell) {
      return;
    }

    const wrap = cmp.getBoundingClientRect();
    const r = cell.closest('table').getBoundingClientRect();
    bar.style.top = `${r.top - wrap.top - bar.offsetHeight - 4 + cmp.scrollTop}px`;
    bar.style.left = `${r.left - wrap.left}px`;
  };

  ed.addEventListener('pathChange', reposition);
  ed.addEventListener('select', reposition);
  ed.addEventListener('blur', () => { bar.hidden = true; });
  cmp.content.addEventListener('mouseup', reposition);

  bar.addEventListener('click', (e) => {
    const b = e.target.closest('[data-op]');
    const cell = b && currentCell();
    if (!cell) {
      return;
    }

    e.preventDefault();
    ed.saveUndoState();
    run(b.dataset.op, cell);
    cmp.sync();
    setTimeout(reposition, 0);
  });
}

const markdownToVisualBlock = (markdown) => {
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
    // Handle ATX headings (# Heading)
    const heading = line.match(/^(#+)\s+(.*)/);
    if (heading) {
      flushText();

      blocks.push(createBlock('Heading', {
        text: heading[2],
        level: `h${Math.min(heading[1].length, 6)}`,
      }));
      return;
    }

    // Handle Setext headings (===== or -----)
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
};

export default markdownToVisualBlock;

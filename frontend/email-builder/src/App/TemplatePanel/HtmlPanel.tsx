import React, { useMemo } from 'react';

import { useDocument } from '../../documents/editor/EditorContext';
import { renderHtmlWithMeta } from '../../utils';

import HighlightedCodePanel from './helper/HighlightedCodePanel';

export default function HtmlPanel() {
  const document = useDocument();
  const code = useMemo(() => renderHtmlWithMeta(document, { rootBlockId: 'root' }), [document]);
  return <HighlightedCodePanel type="html" value={code} />;
}

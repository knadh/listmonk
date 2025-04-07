import React, { createContext, useContext } from 'react';

import { EditorBlock as CoreEditorBlock } from './core';
import { useDocument } from './EditorContext';

const EditorBlockContext = createContext<string | null>(null);
export const useCurrentBlockId = () => useContext(EditorBlockContext)!;

type EditorBlockProps = {
  id: string;
};

/**
 *
 * @param id - Block id
 * @returns EditorBlock component that loads data from the EditorDocumentContext
 */
export default function EditorBlock({ id }: EditorBlockProps) {
  const document = useDocument();
  const block = document[id];
  if (!block) {
    throw new Error('Could not find block');
  }
  return (
    <EditorBlockContext.Provider value={id}>
      <CoreEditorBlock {...block} />
    </EditorBlockContext.Provider>
  );
}

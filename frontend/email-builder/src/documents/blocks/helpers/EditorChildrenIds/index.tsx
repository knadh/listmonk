import React, { Fragment } from 'react';

import { TEditorBlock } from '../../../editor/core';
import EditorBlock from '../../../editor/EditorBlock';

import AddBlockButton from './AddBlockMenu';

export type EditorChildrenChange = {
  blockId: string;
  block: TEditorBlock;
  childrenIds: string[];
};

function generateId() {
  return `block-${Date.now()}`;
}

export type EditorChildrenIdsProps = {
  childrenIds: string[] | null | undefined;
  onChange: (val: EditorChildrenChange) => void;
};
export default function EditorChildrenIds({ childrenIds, onChange }: EditorChildrenIdsProps) {
  const appendBlock = (block: TEditorBlock) => {
    const blockId = generateId();
    return onChange({
      blockId,
      block,
      childrenIds: [...(childrenIds || []), blockId],
    });
  };

  const insertBlock = (block: TEditorBlock, index: number) => {
    const blockId = generateId();
    const newChildrenIds = [...(childrenIds || [])];
    newChildrenIds.splice(index, 0, blockId);
    return onChange({
      blockId,
      block,
      childrenIds: newChildrenIds,
    });
  };

  if (!childrenIds || childrenIds.length === 0) {
    return <AddBlockButton placeholder onSelect={appendBlock} />;
  }

  return (
    <>
      {childrenIds.map((childId, i) => (
        <Fragment key={childId}>
          <AddBlockButton onSelect={(block) => insertBlock(block, i)} />
          <EditorBlock id={childId} />
        </Fragment>
      ))}
      <AddBlockButton onSelect={appendBlock} />
    </>
  );
}

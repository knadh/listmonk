import React from 'react';

import { ArrowDownwardOutlined, ArrowUpwardOutlined, ContentCopyOutlined, DeleteOutlined } from '@mui/icons-material';
import { IconButton, Paper, Stack, SxProps, Tooltip } from '@mui/material';

import { TEditorBlock } from '../../../editor/core';
import { resetDocument, setSelectedBlockId, useDocument } from '../../../editor/EditorContext';
import { ColumnsContainerProps } from '../../ColumnsContainer/ColumnsContainerPropsSchema';

const sx: SxProps = {
  position: 'absolute',
  top: 0,
  left: -56,
  borderRadius: 64,
  paddingX: 0.5,
  paddingY: 1
};

type Props = {
  blockId: string;
};
export default function TuneMenu({ blockId }: Props) {
  const document = useDocument();

  const handleDeleteClick = () => {
    const filterChildrenIds = (childrenIds: string[] | null | undefined) => {
      if (!childrenIds) {
        return childrenIds;
      }
      return childrenIds.filter((f) => f !== blockId);
    };
    const nDocument: typeof document = { ...document };
    for (const [id, b] of Object.entries(nDocument)) {
      const block = b as TEditorBlock;
      if (id === blockId) {
        continue;
      }
      switch (block.type) {
        case 'EmailLayout':
          nDocument[id] = {
            ...block,
            data: {
              ...block.data,
              childrenIds: filterChildrenIds(block.data.childrenIds),
            },
          };
          break;
        case 'Container':
          nDocument[id] = {
            ...block,
            data: {
              ...block.data,
              props: {
                ...block.data.props,
                childrenIds: filterChildrenIds(block.data.props?.childrenIds),
              },
            },
          };
          break;
        case 'ColumnsContainer':
          nDocument[id] = {
            type: 'ColumnsContainer',
            data: {
              style: block.data.style,
              props: {
                ...block.data.props,
                columns: block.data.props?.columns?.map((c) => ({
                  childrenIds: filterChildrenIds(c.childrenIds),
                })),
              },
            } as ColumnsContainerProps,
          };
          break;
        default:
          nDocument[id] = block;
      }
    }
    delete nDocument[blockId];
    resetDocument(nDocument);
  };

  const handleDuplicateClick = () => {
    const block = document[blockId] as TEditorBlock;
    if (!block) {
      return;
    }

    // Recursively clone a block and all its descendants, assigning new IDs.
    const clones: Record<string, TEditorBlock> = {};
    const cloneBlock = (srcId: string): string => {
      const newId = `block-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;
      const src = document[srcId] as TEditorBlock;
      if (!src) {
        return newId;
      }
      const cloned: TEditorBlock = JSON.parse(JSON.stringify(src));

      // Remap childrenIds for container types so each child is also cloned.
      if (cloned.type === 'Container' && cloned.data.props?.childrenIds) {
        cloned.data.props.childrenIds = cloned.data.props.childrenIds.map(cloneBlock);
      } else if (cloned.type === 'ColumnsContainer' && cloned.data.props?.columns) {
        cloned.data.props.columns = cloned.data.props.columns.map((c) => ({
          childrenIds: c.childrenIds?.map(cloneBlock) ?? [],
        }));
      }

      clones[newId] = cloned;
      return newId;
    };
    const newBlockId = cloneBlock(blockId);

    const insertAfter = (ids: string[] | null | undefined) => {
      if (!ids) {
        return ids;
      }
      const index = ids.indexOf(blockId);
      if (index < 0) {
        return ids;
      }
      const newIds = [...ids];
      newIds.splice(index + 1, 0, newBlockId);
      return newIds;
    };

    const nDocument: typeof document = {
      ...document,
      ...clones,
    };
    for (const [id, b] of Object.entries(nDocument)) {
      const entry = b as TEditorBlock;
      if (id === blockId || id in clones) {
        continue;
      }
      switch (entry.type) {
        case 'EmailLayout':
          nDocument[id] = {
            ...entry,
            data: {
              ...entry.data,
              childrenIds: insertAfter(entry.data.childrenIds),
            },
          };
          break;
        case 'Container':
          nDocument[id] = {
            ...entry,
            data: {
              ...entry.data,
              props: {
                ...entry.data.props,
                childrenIds: insertAfter(entry.data.props?.childrenIds),
              },
            },
          };
          break;
        case 'ColumnsContainer':
          nDocument[id] = {
            type: 'ColumnsContainer',
            data: {
              style: entry.data.style,
              props: {
                ...entry.data.props,
                columns: entry.data.props?.columns?.map((c) => ({
                  childrenIds: insertAfter(c.childrenIds),
                })),
              },
            } as ColumnsContainerProps,
          };
          break;
        default:
          nDocument[id] = entry;
      }
    }
    resetDocument(nDocument);
    setSelectedBlockId(newBlockId);
  };

  const handleMoveClick = (direction: 'up' | 'down') => {
    const moveChildrenIds = (ids: string[] | null | undefined) => {
      if (!ids) {
        return ids;
      }
      const index = ids.indexOf(blockId);
      if (index < 0) {
        return ids;
      }
      const childrenIds = [...ids];
      if (direction === 'up' && index > 0) {
        [childrenIds[index], childrenIds[index - 1]] = [childrenIds[index - 1], childrenIds[index]];
      } else if (direction === 'down' && index < childrenIds.length - 1) {
        [childrenIds[index], childrenIds[index + 1]] = [childrenIds[index + 1], childrenIds[index]];
      }
      return childrenIds;
    };
    const nDocument: typeof document = { ...document };
    for (const [id, b] of Object.entries(nDocument)) {
      const block = b as TEditorBlock;
      if (id === blockId) {
        continue;
      }
      switch (block.type) {
        case 'EmailLayout':
          nDocument[id] = {
            ...block,
            data: {
              ...block.data,
              childrenIds: moveChildrenIds(block.data.childrenIds),
            },
          };
          break;
        case 'Container':
          nDocument[id] = {
            ...block,
            data: {
              ...block.data,
              props: {
                ...block.data.props,
                childrenIds: moveChildrenIds(block.data.props?.childrenIds),
              },
            },
          };
          break;
        case 'ColumnsContainer':
          nDocument[id] = {
            type: 'ColumnsContainer',
            data: {
              style: block.data.style,
              props: {
                ...block.data.props,
                columns: block.data.props?.columns?.map((c) => ({
                  childrenIds: moveChildrenIds(c.childrenIds),
                })),
              },
            } as ColumnsContainerProps,
          };
          break;
        default:
          nDocument[id] = block;
      }
    }

    resetDocument(nDocument);
    setSelectedBlockId(blockId);
  };

  return (
    <Paper sx={sx} onClick={(ev) => ev.stopPropagation()}>
      <Stack>
        <Tooltip title="Move up" placement="left-start">
          <IconButton onClick={() => handleMoveClick('up')} sx={{ color: 'text.primary' }}>
            <ArrowUpwardOutlined fontSize="small" />
          </IconButton>
        </Tooltip>
        <Tooltip title="Move down" placement="left-start">
          <IconButton onClick={() => handleMoveClick('down')} sx={{ color: 'text.primary' }}>
            <ArrowDownwardOutlined fontSize="small" />
          </IconButton>
        </Tooltip>
        <Tooltip title="Duplicate" placement="left-start">
          <IconButton onClick={handleDuplicateClick} sx={{ color: 'text.primary' }}>
            <ContentCopyOutlined fontSize="small" />
          </IconButton>
        </Tooltip>
        <Tooltip title="Delete" placement="left-start">
          <IconButton onClick={handleDeleteClick} sx={{ color: 'text.primary' }}>
            <DeleteOutlined fontSize="small" />
          </IconButton>
        </Tooltip>
      </Stack>
    </Paper>
  );
}

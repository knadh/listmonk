import React, { useMemo } from 'react';

import { FileDownloadOutlined } from '@mui/icons-material';
import { IconButton, Tooltip } from '@mui/material';

import { useDocument } from '../../../documents/editor/EditorContext';

export default function DownloadJson() {
  const doc = useDocument();
  const href = useMemo(() => {
    return `data:text/plain,${encodeURIComponent(JSON.stringify(doc, null, '  '))}`;
  }, [doc]);
  return (
    <Tooltip title="Download JSON file">
      <IconButton href={href} download="emailTemplate.json">
        <FileDownloadOutlined fontSize="small" />
      </IconButton>
    </Tooltip>
  );
}

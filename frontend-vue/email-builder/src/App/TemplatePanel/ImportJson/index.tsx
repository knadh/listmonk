import React, { useState } from 'react';

import { FileUploadOutlined } from '@mui/icons-material';
import { IconButton, Tooltip } from '@mui/material';

import ImportJsonDialog from './ImportJsonDialog';

export default function ImportJson() {
  const [open, setOpen] = useState(false);

  let dialog = null;
  if (open) {
    dialog = <ImportJsonDialog onClose={() => setOpen(false)} />;
  }

  return (
    <>
      <Tooltip title="Import JSON">
        <IconButton onClick={() => setOpen(true)}>
          <FileUploadOutlined fontSize="small" />
        </IconButton>
      </Tooltip>
      {dialog}
    </>
  );
}

import React, { useEffect, useState } from 'react';

import { AddOutlined } from '@mui/icons-material';
import { Fade, IconButton } from '@mui/material';

type Props = {
  buttonElement: HTMLElement | null;
  onClick: () => void;
};
export default function DividerButton({ buttonElement, onClick }: Props) {
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    function listener({ clientX, clientY }: MouseEvent) {
      if (!buttonElement) {
        return;
      }
      const rect = buttonElement.getBoundingClientRect();
      const rectY = rect.y;
      const bottomX = rect.x;
      const topX = bottomX + rect.width;

      if (Math.abs(clientY - rectY) < 20) {
        if (bottomX < clientX && clientX < topX) {
          setVisible(true);
          return;
        }
      }
      setVisible(false);
    }
    window.addEventListener('mousemove', listener);
    return () => {
      window.removeEventListener('mousemove', listener);
    };
  }, [buttonElement, setVisible]);

  return (
    <Fade in={visible}>
      <IconButton
        size="small"
        sx={{
          p: 0.12,
          position: 'absolute',
          top: '-12px',
          left: '50%',
          transform: 'translateX(-10px)',
          bgcolor: 'brand.blue',
          color: 'primary.contrastText',
          '&:hover, &:active, &:focus': {
            bgcolor: 'brand.blue',
            color: 'primary.contrastText',
          },
        }}
        onClick={(ev) => {
          ev.stopPropagation();
          onClick();
        }}
      >
        <AddOutlined fontSize="small" />
      </IconButton>
    </Fade>
  );
}

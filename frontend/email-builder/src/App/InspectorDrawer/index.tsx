import React from 'react';

import {
  Box, Drawer, Tab, Tabs,
} from '@mui/material';

import {
  reclampInspectorDrawerWidth, setInspectorDrawerResizing, setInspectorDrawerWidth, setSidebarTab, useInspectorDrawerOpen,
  useInspectorDrawerResizing, useInspectorDrawerWidth, useSelectedSidebarTab,
} from '../../documents/editor/EditorContext';

import ConfigurationPanel from './ConfigurationPanel';
import StylesPanel from './StylesPanel';

export default function InspectorDrawer() {
  const selectedSidebarTab = useSelectedSidebarTab();
  const inspectorDrawerOpen = useInspectorDrawerOpen();
  const width = useInspectorDrawerWidth();
  const dragging = useInspectorDrawerResizing();

  const onPointerDown = (e: React.PointerEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.currentTarget.setPointerCapture(e.pointerId);
    setInspectorDrawerResizing(true);
  };
  const onPointerMove = (e: React.PointerEvent<HTMLDivElement>) => {
    if (dragging) {
      setInspectorDrawerWidth(window.innerWidth - e.clientX);
    }
  };
  const onPointerCancel = (e: React.PointerEvent<HTMLDivElement>) => {
    // The pointer position is unreliable on cancel, so end the drag without saving.
    if (dragging) {
      e.currentTarget.releasePointerCapture(e.pointerId);
      setInspectorDrawerResizing(false);
    }
  };
  const onPointerUp = (e: React.PointerEvent<HTMLDivElement>) => {
    if (!dragging) {
      return;
    }
    e.currentTarget.releasePointerCapture(e.pointerId);
    setInspectorDrawerResizing(false);
    setInspectorDrawerWidth(window.innerWidth - e.clientX);
  };

  // Keep the panel within bounds when the window is resized.
  React.useEffect(() => {
    const onResize = () => reclampInspectorDrawerWidth();
    window.addEventListener('resize', onResize);
    return () => window.removeEventListener('resize', onResize);
  }, []);

  const renderCurrentSidebarPanel = () => {
    switch (selectedSidebarTab) {
      case 'block-configuration':
        return <ConfigurationPanel />;
      case 'styles':
        return <StylesPanel />;
    }
  };

  return (
    <Drawer
      variant="persistent"
      anchor="right"
      className="sidebar"
      open={inspectorDrawerOpen}
      sx={{
        width: inspectorDrawerOpen ? width : 0,
      }}
      // Make the drawer relative to the wrapper instead of body.
      PaperProps={{ style: { position: 'absolute', zIndex: 0, userSelect: dragging ? 'none' : undefined } }}
      ModalProps={{
        container: document.querySelector('.email-builder-container'),
        style: { position: 'absolute', zIndex: 0 },
      }}
    >
      <Box
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={onPointerUp}
        onPointerCancel={onPointerCancel}
        sx={{
          position: 'absolute',
          top: 0,
          left: 0,
          bottom: 0,
          width: 6,
          zIndex: 1,
          cursor: 'ew-resize',
          touchAction: 'none',
          backgroundColor: dragging ? 'primary.main' : 'transparent',
          opacity: 0.5,
          '&:hover': { backgroundColor: 'primary.main' },
        }}
      />
      <Box sx={{
        width, height: 49, borderBottom: 1, borderColor: 'divider',
      }}
      >
        <Box px={2}>
          <Tabs value={selectedSidebarTab} onChange={(_, v) => setSidebarTab(v)}>
            <Tab value="styles" label="Styles" />
            <Tab value="block-configuration" label="Inspect" />
          </Tabs>
        </Box>
      </Box>
      <Box sx={{ width, height: 'calc(100% - 49px)', overflow: 'auto' }}>
        {renderCurrentSidebarPanel()}
      </Box>
    </Drawer>
  );
}

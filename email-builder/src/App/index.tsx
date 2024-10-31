import React from 'react';
import { Stack, useTheme } from '@mui/material';
import { renderToStaticMarkup } from '@usewaypoint/email-builder';

import { TEditorConfiguration } from '../documents/editor/core';
import { useInspectorDrawerOpen, useSamplesDrawerOpen, subscribeDocument, setDocument } from '../documents/editor/EditorContext';
import InspectorDrawer, { INSPECTOR_DRAWER_WIDTH } from './InspectorDrawer';
import TemplatePanel from './TemplatePanel';

export const DEFAULT_SOURCE: TEditorConfiguration = {
  "root": {
    "type": "EmailLayout",
    "data": {}
  }
}

function useDrawerTransition(cssProperty: 'margin-left' | 'margin-right', open: boolean) {
  const { transitions } = useTheme();
  return transitions.create(cssProperty, {
    easing: !open ? transitions.easing.sharp : transitions.easing.easeOut,
    duration: !open ? transitions.duration.leavingScreen : transitions.duration.enteringScreen,
  });
}

export interface AppProps {
  // Initial configuration to load. Optional.
  data?: TEditorConfiguration,
  // Callback for any change in document. Optional.
  onChange?: (json: TEditorConfiguration, html: String) => void,
  // Optional height for the Stack component.
  height?: string,
}

export default function App(props: AppProps) {
  const inspectorDrawerOpen = useInspectorDrawerOpen();
  const samplesDrawerOpen = useSamplesDrawerOpen();

  const marginLeftTransition = useDrawerTransition('margin-left', samplesDrawerOpen);
  const marginRightTransition = useDrawerTransition('margin-right', inspectorDrawerOpen);

  if (props.data) {
    setDocument(props.data)
  } else {
    setDocument(DEFAULT_SOURCE)
  }

  if (props.onChange) {
    subscribeDocument ((document) => {
      props.onChange?.(document, renderToStaticMarkup(document, { rootBlockId: 'root' }))
    })
  }

  return (
    <>
      <InspectorDrawer />

      <Stack
        sx={{
          marginRight: inspectorDrawerOpen ? `${INSPECTOR_DRAWER_WIDTH}px` : 0,
          transition: [marginLeftTransition, marginRightTransition].join(', '),
          height: props.height ? props.height : 'auto',
        }}
      >
        <TemplatePanel />
      </Stack>
    </>
  );
}

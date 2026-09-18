import { create } from 'zustand';
import { subscribeWithSelector } from 'zustand/middleware';

import getConfiguration from '../../getConfiguration';

import { TEditorConfiguration } from './core';

export const INSPECTOR_MIN_WIDTH = 280;
export const INSPECTOR_DEFAULT_WIDTH = 320;

function clampInspectorWidth(width: number) {
  // Leave at least 320px for the canvas.
  const max = Math.max(INSPECTOR_MIN_WIDTH, window.innerWidth - 320);
  return Math.min(Math.max(width, INSPECTOR_MIN_WIDTH), max);
}

type TValue = {
  document: TEditorConfiguration;

  selectedBlockId: string | null;
  selectedSidebarTab: 'block-configuration' | 'styles';
  selectedMainTab: 'editor' | 'preview' | 'json' | 'html';
  selectedScreenSize: 'desktop' | 'mobile';

  inspectorDrawerOpen: boolean;
  inspectorDrawerWidth: number;
  inspectorDrawerResizing: boolean;
  samplesDrawerOpen: boolean;
};

const editorStateStore = create(subscribeWithSelector<TValue>(() => ({
  document: getConfiguration(window.location.hash),
  selectedBlockId: null,
  selectedSidebarTab: 'styles',
  selectedMainTab: 'editor',
  selectedScreenSize: 'desktop',

  inspectorDrawerOpen: true,
  inspectorDrawerWidth: INSPECTOR_DEFAULT_WIDTH,
  inspectorDrawerResizing: false,
  samplesDrawerOpen: true,
})));

export function useDocument() {
  return editorStateStore((s) => s.document);
}

export function subscribeDocument (listener: (selectedState: TEditorConfiguration, previousSelectedState: TEditorConfiguration) => void) {
  editorStateStore.subscribe((state) => state.document, listener)
}

export function useSelectedBlockId() {
  return editorStateStore((s) => s.selectedBlockId);
}

export function useSelectedScreenSize() {
  return editorStateStore((s) => s.selectedScreenSize);
}

export function useSelectedMainTab() {
  return editorStateStore((s) => s.selectedMainTab);
}

export function setSelectedMainTab(selectedMainTab: TValue['selectedMainTab']) {
  return editorStateStore.setState({ selectedMainTab });
}

export function useSelectedSidebarTab() {
  return editorStateStore((s) => s.selectedSidebarTab);
}

export function useInspectorDrawerOpen() {
  return editorStateStore((s) => s.inspectorDrawerOpen);
}

export function useInspectorDrawerWidth() {
  return editorStateStore((s) => s.inspectorDrawerWidth);
}

export function useInspectorDrawerResizing() {
  return editorStateStore((s) => s.inspectorDrawerResizing);
}

export function setInspectorDrawerResizing(inspectorDrawerResizing: boolean) {
  return editorStateStore.setState({ inspectorDrawerResizing });
}

export function setInspectorDrawerWidth(width: number) {
  return editorStateStore.setState({ inspectorDrawerWidth: clampInspectorWidth(width) });
}

export function reclampInspectorDrawerWidth() {
  const { inspectorDrawerWidth } = editorStateStore.getState();
  const clamped = clampInspectorWidth(inspectorDrawerWidth);
  if (clamped !== inspectorDrawerWidth) {
    editorStateStore.setState({ inspectorDrawerWidth: clamped });
  }
}

export function useSamplesDrawerOpen() {
  return editorStateStore((s) => s.samplesDrawerOpen);
}

export function setSelectedBlockId(selectedBlockId: TValue['selectedBlockId']) {
  const selectedSidebarTab = selectedBlockId === null ? 'styles' : 'block-configuration';
  const options: Partial<TValue> = {};
  if (selectedBlockId !== null) {
    options.inspectorDrawerOpen = true;
  }
  return editorStateStore.setState({
    selectedBlockId,
    selectedSidebarTab,
    ...options,
  });
}

export function setSidebarTab(selectedSidebarTab: TValue['selectedSidebarTab']) {
  return editorStateStore.setState({ selectedSidebarTab });
}

export function resetDocument(document: TValue['document']) {
  return editorStateStore.setState({
    document,
    selectedSidebarTab: 'styles',
    selectedBlockId: null,
  });
}

export function setDocument(document: TValue['document']) {
  const originalDocument = editorStateStore.getState().document;
  return editorStateStore.setState({
    document: {
      ...originalDocument,
      ...document,
    },
  });
}

export function toggleInspectorDrawerOpen() {
  const inspectorDrawerOpen = !editorStateStore.getState().inspectorDrawerOpen;
  return editorStateStore.setState({ inspectorDrawerOpen });
}

export function toggleSamplesDrawerOpen() {
  const samplesDrawerOpen = !editorStateStore.getState().samplesDrawerOpen;
  return editorStateStore.setState({ samplesDrawerOpen });
}

export function setSelectedScreenSize(selectedScreenSize: TValue['selectedScreenSize']) {
  return editorStateStore.setState({ selectedScreenSize });
}

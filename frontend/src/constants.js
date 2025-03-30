export const models = Object.freeze({
  serverConfig: 'serverConfig',
  lang: 'lang',
  dashboard: 'dashboard',
  // This loading state is used across all contexts where lists are loaded
  // via the instant "minimal" API.
  lists: 'lists',
  // This is used only on the lists page where lists are loaded with full
  // context (subscriber counts), which can be slow and expensive.
  listsFull: 'listsFull',
  subscribers: 'subscribers',
  campaigns: 'campaigns',
  templates: 'templates',
  media: 'media',
  bounces: 'bounces',
  users: 'users',
  profile: 'profile',
  userRoles: 'userRoles',
  listRoles: 'listRoles',
  settings: 'settings',
  logs: 'logs',
  maintenance: 'maintenance',
});

// Ad-hoc URIs that are used outside of vuex requests.
const rootURL = import.meta.env.VUE_APP_ROOT_URL || '/';
const baseURL = import.meta.env.BASE_URL.replace(/\/$/, '');

export const uris = Object.freeze({
  previewCampaign: '/api/campaigns/:id/preview',
  previewTemplate: '/api/templates/:id/preview',
  previewRawTemplate: '/api/templates/preview',
  exportSubscribers: '/api/subscribers/export',
  errorEvents: '/api/events?type=error',
  base: `${baseURL}/static`,
  root: rootURL,
  static: `${baseURL}/static`,
});

// Keys used in Vuex store.
export const storeKeys = Object.freeze({
  models: 'models',
  isLoading: 'isLoading',
});

export const timestamp = 'ddd D MMM YYYY, hh:mm A';

export const colors = Object.freeze({
  primary: '#0055d4',
});

export const regDuration = '[0-9]+(ms|s|m|h|d)';

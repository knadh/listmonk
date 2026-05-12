import { ToastProgrammatic as Toast } from 'buefy';
import axios from 'axios';
import qs from 'qs';
import store from '../store';
import { models } from '../constants';
import Utils from '../utils';

const http = axios.create({
  baseURL: import.meta.env.VUE_APP_ROOT_URL || '/',
  withCredentials: false,
  responseType: 'json',

  // Override the default serializer to switch params from becoming []id=a&[]id=b ...
  // in GET and DELETE requests to id=a&id=b.
  paramsSerializer: (params) => qs.stringify(params, { arrayFormat: 'repeat' }),
});

const utils = new Utils();

// Intercept requests to set the 'loading' state of a model.
http.interceptors.request.use((config) => {
  if ('loading' in config) {
    store.commit('setLoading', { model: config.loading, status: true });
  }
  return config;
}, (error) => Promise.reject(error));

// Intercept responses to set them to store.
http.interceptors.response.use((resp) => {
  // Clear the loading state for a model.
  if ('loading' in resp.config) {
    store.commit('setLoading', { model: resp.config.loading, status: false });
  }

  let data = {};
  if (typeof resp.data.data === 'object') {
    if (resp.data.data.constructor === Object) {
      data = { ...resp.data.data };
    } else {
      data = [...resp.data.data];
    }

    // Transform keys to camelCase.
    switch (typeof resp.config.camelCase) {
      case 'function':
        data = utils.camelKeys(data, resp.config.camelCase);
        break;
      case 'boolean':
        if (resp.config.camelCase) {
          data = utils.camelKeys(data);
        }
        break;
      default:
        data = utils.camelKeys(data);
        break;
    }
  } else {
    data = resp.data.data;
  }

  // Store the API response for a model.
  if ('store' in resp.config) {
    store.commit('setModelResponse', { model: resp.config.store, data });
  }

  return data;
}, (err) => {
  // Clear the loading state for a model.
  if ('loading' in err.config) {
    store.commit('setLoading', { model: err.config.loading, status: false });
  }

  let msg = '';
  if (err.response && err.response.data && err.response.data.message) {
    msg = err.response.data.message;
  } else {
    msg = err.toString();
  }

  if (!err.config.disableToast) {
    Toast.open({
      message: msg,
      type: 'is-danger',
      queue: false,
      position: 'is-top',
      pauseOnHover: true,
    });
  }

  return Promise.reject(err);
});

// API calls accept the following config keys.
// loading: modelName (set's the loading status in the global store: eg: store.loading.lists = true)
// store: modelName (set's the API response in the global store. eg: store.lists: { ... } )

// Health check endpoint that does not throw a toast.
export const getHealth = () => http.get(
  '/api/health',
  { disableToast: true },
);

export const reloadApp = () => http.post('/api/admin/reload');

// Dashboard
export const getDashboardCounts = () => http.get(
  '/api/dashboard/counts',
  { loading: models.dashboard },
);

export const getDashboardCharts = () => http.get(
  '/api/dashboard/charts',
  { loading: models.dashboard },
);

export const getDashboardFeatureCounts = () => http.get(
  '/api/dashboard/features',
  { loading: models.dashboard, camelCase: false },
);

// Lists.
export const getLists = (params) => http.get(
  '/api/lists',
  {
    params: (!params ? { per_page: 'all' } : params),
    loading: models.lists,
    store: models.lists,
  },
);

export const queryLists = (params) => http.get(
  '/api/lists',
  {
    params: (!params ? { per_page: 'all' } : params),
    loading: models.listsFull,
  },
);

export const getList = async (id) => http.get(
  `/api/lists/${id}`,
  { loading: models.list },
);

export const createList = (data) => http.post(
  '/api/lists',
  data,
  { loading: models.lists },
);

export const updateList = (data) => http.put(
  `/api/lists/${data.id}`,
  data,
  { loading: models.lists },
);

export const deleteList = (id) => http.delete(
  `/api/lists/${id}`,
  { loading: models.lists },
);

export const deleteLists = (params) => http.delete(
  '/api/lists',
  { params, loading: models.lists },
);

// Subscribers.
export const getSubscribers = async (params) => http.get(
  '/api/subscribers',
  {
    params,
    loading: models.subscribers,
    store: models.subscribers,
    camelCase: (keyPath) => !keyPath.startsWith('.results.*.attribs'),
  },
);

export const getSubscriber = async (id) => http.get(
  `/api/subscribers/${id}`,
  { loading: models.subscribers },
);

export const getSubscriberActivity = async (id) => http.get(
  `/api/subscribers/${id}/activity`,
  { loading: models.subscribers },
);

export const getSubscriberBounces = async (id) => http.get(
  `/api/subscribers/${id}/bounces`,
  { loading: models.bounces },
);

export const deleteSubscriberBounces = async (id) => http.delete(
  `/api/subscribers/${id}/bounces`,
  { loading: models.bounces },
);

export const deleteBounce = async (id) => http.delete(
  `/api/bounces/${id}`,
  { loading: models.bounces },
);

export const deleteBounces = async (params) => http.delete(
  '/api/bounces',
  { params, loading: models.bounces },
);

export const blocklistBouncedSubscribers = async () => http.put(
  '/api/bounces/blocklist',
  { loading: models.bounces },
);

export const createSubscriber = (data) => http.post(
  '/api/subscribers',
  data,
  { loading: models.subscribers },
);

export const updateSubscriber = (data) => http.put(
  `/api/subscribers/${data.id}`,
  data,
  { loading: models.subscribers },
);

export const sendSubscriberOptin = (id) => http.post(
  `/api/subscribers/${id}/optin`,
  {},
  { loading: models.subscribers },
);

export const deleteSubscriber = (id) => http.delete(
  `/api/subscribers/${id}`,
  { loading: models.subscribers },
);

export const addSubscribersToLists = (data) => http.put(
  '/api/subscribers/lists',
  data,
  { loading: models.subscribers },
);

export const addSubscribersToListsByQuery = (data) => http.put(
  '/api/subscribers/query/lists',
  data,

  { loading: models.subscribers },
);

export const blocklistSubscribers = (data) => http.put(
  '/api/subscribers/blocklist',
  data,
  { loading: models.subscribers },
);

export const blocklistSubscribersByQuery = (data) => http.put(
  '/api/subscribers/query/blocklist',
  data,
  { loading: models.subscribers },
);

export const deleteSubscribers = (params) => http.delete(
  '/api/subscribers',
  { params, loading: models.subscribers },
);

export const deleteSubscribersByQuery = (data) => http.post(
  '/api/subscribers/query/delete',
  data,
  { loading: models.subscribers },
);

// Subscriber import.
export const importSubscribers = (data) => http.post('/api/import/subscribers', data);

export const getImportStatus = () => http.get('/api/import/subscribers');

export const getImportLogs = async () => http.get(
  '/api/import/subscribers/logs',
  { camelCase: false },
);

export const stopImport = () => http.delete('/api/import/subscribers');

// Bounces.
export const getBounces = async (params) => http.get(
  '/api/bounces',
  { params, loading: models.bounces },
);

// Campaigns.
export const getCampaigns = async (params) => http.get('/api/campaigns', {
  params,
  loading: models.campaigns,
  store: models.campaigns,
  camelCase: (keyPath) => !keyPath.startsWith('.results.*.headers'),
});

export const getCampaign = async (id) => http.get(`/api/campaigns/${id}`, {
  loading: models.campaigns,
  camelCase: (keyPath) => !keyPath.startsWith('.headers'),
});

export const getCampaignStats = async () => http.get('/api/campaigns/running/stats', {});

export const createCampaign = async (data) => http.post(
  '/api/campaigns',
  data,
  { loading: models.campaigns },
);

export const getCampaignViewCounts = async (params) => http.get(
  '/api/campaigns/analytics/views',
  { params, loading: models.campaigns },
);

export const getCampaignClickCounts = async (params) => http.get(
  '/api/campaigns/analytics/clicks',
  { params, loading: models.campaigns },
);

export const getCampaignBounceCounts = async (params) => http.get(
  '/api/campaigns/analytics/bounces',
  { params, loading: models.campaigns },
);

export const getCampaignLinkCounts = async (params) => http.get(
  '/api/campaigns/analytics/links',
  { params, loading: models.campaigns },
);

export const convertCampaignContent = async (data) => http.post(
  `/api/campaigns/${data.id}/content`,
  data,
  { loading: models.campaigns },
);

export const testCampaign = async (data) => http.post(
  `/api/campaigns/${data.id}/test`,
  data,
  { loading: models.campaigns },
);

export const updateCampaign = async (id, data) => http.put(
  `/api/campaigns/${id}`,
  data,
  { loading: models.campaigns },
);

export const changeCampaignStatus = async (id, status) => http.put(
  `/api/campaigns/${id}/status`,
  { status },

  { loading: models.campaigns },
);

export const updateCampaignArchive = async (id, data) => http.put(
  `/api/campaigns/${id}/archive`,
  data,
  { loading: models.campaigns },
);

export const updateCampaignEvergreen = async (id, isEvergreen) => http.put(
  `/api/campaigns/${id}/evergreen`,
  { is_evergreen: isEvergreen },
  { loading: models.campaigns },
);

export const rewindCampaign = async (id) => http.post(
  `/api/campaigns/${id}/rewind`,
  null,
  { loading: models.campaigns },
);

// Solomon fork: clear in-memory sliding-window rate-limit state on a running
// campaign. Wired to the "Reset rate limit" button on the campaign detail.
export const resetCampaignWindow = async (id) => http.post(
  `/api/campaigns/${id}/reset-window`,
  null,
  { loading: models.campaigns },
);

// Solomon fork: delete failed-status rows from campaign_send_log so the worker
// retries those subscribers. Wired to the "Retry N failed" button on the
// Send Log tab.
export const retryFailedCampaignSends = async (id) => http.post(
  `/api/campaigns/${id}/send-log/retry-failed`,
  null,
  { loading: models.campaigns },
);

export const deleteCampaign = async (id) => http.delete(
  `/api/campaigns/${id}`,
  { loading: models.campaigns },
);

export const deleteCampaigns = (params) => http.delete(
  '/api/campaigns',
  { params, loading: models.campaigns },
);

// Media.
export const getMedia = async (params) => http.get(
  '/api/media',
  { params, loading: models.media, store: models.media },
);

export const uploadMedia = (data) => http.post(
  '/api/media',
  data,
  { loading: models.media },
);

export const deleteMedia = (id) => http.delete(
  `/api/media/${id}`,
  { loading: models.media },
);

// Templates.
export const createTemplate = async (data) => http.post(
  '/api/templates',
  data,
  { loading: models.templates },
);

export const getTemplates = async () => http.get(
  '/api/templates',
  { loading: models.templates, store: models.templates },
);

export const getTemplate = async (id) => http.get(
  `/api/templates/${id}`,
  { loading: models.templates },
);

export const updateTemplate = async (data) => http.put(
  `/api/templates/${data.id}`,
  data,
  { loading: models.templates },
);

export const makeTemplateDefault = async (id) => http.put(
  `/api/templates/${id}/default`,
  {},
  { loading: models.templates },
);

export const deleteTemplate = async (id) => http.delete(
  `/api/templates/${id}`,
  { loading: models.templates },
);

// Settings.
export const getServerConfig = async () => http.get(
  '/api/config',
  { loading: models.serverConfig, store: models.serverConfig, camelCase: false },
);

export const getSettings = async () => http.get(
  '/api/settings',
  { loading: models.settings, store: models.settings, camelCase: false },
);

export const updateSettings = async (data) => http.put(
  '/api/settings',
  data,
  { loading: models.settings },
);

export const updateSettingsByKey = async (key, data) => http.put(
  `/api/settings/${key}`,
  data,
  { loading: models.settings },
);

export const testSMTP = async (data) => http.post(
  '/api/settings/smtp/test',
  data,
  { loading: models.settings, disableToast: true },
);

export const getLogs = async () => http.get(
  '/api/logs',
  { loading: models.logs, camelCase: false },
);

export const getLang = async (lang) => http.get(
  `/api/lang/${lang}`,
  { loading: models.lang, camelCase: false },
);

export const logout = async () => http.post('/api/logout');

export const deleteGCCampaignAnalytics = async (typ, beforeDate) => http.delete(
  `/api/maintenance/analytics/${typ}`,
  { loading: models.maintenance, params: { before_date: beforeDate } },
);

export const deleteGCSubscribers = async (typ) => http.delete(
  `/api/maintenance/subscribers/${typ}`,
  { loading: models.maintenance },
);

export const deleteGCSubscriptions = async (beforeDate) => http.delete(
  '/api/maintenance/subscriptions/unconfirmed',
  { loading: models.maintenance, params: { before_date: beforeDate } },
);

// Users.
export const getUsers = () => http.get(
  '/api/users',
  {
    loading: models.users,
    store: models.users,
  },
);

export const queryUsers = () => http.get(
  '/api/users',
  {
    loading: models.users,
    store: models.users,
  },
);

export const getUser = async (id) => http.get(
  `/api/users/${id}`,
  { loading: models.users },
);

export const createUser = (data) => http.post(
  '/api/users',
  data,
  { loading: models.users },
);

export const updateUser = (data) => http.put(
  `/api/users/${data.id}`,
  data,
  { loading: models.users },
);

export const deleteUser = (id) => http.delete(
  `/api/users/${id}`,
  { loading: models.users },
);

export const getUserProfile = () => http.get(
  '/api/profile',
  { loading: models.users, store: models.profile },
);

export const updateUserProfile = (data) => http.put(
  '/api/profile',
  data,
  { loading: models.users, store: models.profile },
);

// Companies (multi-tenant, v7.17.0+).
export const getCompanies = () => http.get(
  '/api/companies',
  { loading: models.companies, store: models.companies },
);

export const getCompanyStats = () => http.get(
  '/api/companies/stats',
  { loading: models.companies },
);

export const getCompany = (id) => http.get(
  `/api/companies/${id}`,
  { loading: models.companies },
);

export const createCompany = (data) => http.post(
  '/api/companies',
  data,
  { loading: models.companies },
);

export const updateCompany = (data) => http.put(
  `/api/companies/${data.id}`,
  data,
  { loading: models.companies },
);

export const deleteCompany = (id) => http.delete(
  `/api/companies/${id}`,
  { loading: models.companies },
);

export const getUserRoles = async () => http.get(
  '/api/roles/users',
  { loading: models.userRoles, store: models.userRoles },
);

export const getListRoles = async () => http.get(
  '/api/roles/lists',
  { loading: models.listRoles, store: models.listRoles },
);

export const createUserRole = (data) => http.post(
  '/api/roles/users',
  data,
  { loading: models.userRoles },
);

export const createListRole = (data) => http.post(
  '/api/roles/lists',
  data,
  { loading: models.listRoles },
);

export const updateUserRole = (data) => http.put(
  `/api/roles/users/${data.id}`,
  data,
  { loading: models.userRoles },
);

export const updateListRole = (data) => http.put(
  `/api/roles/lists/${data.id}`,
  data,
  { loading: models.userRoles },
);

export const deleteRole = (id) => http.delete(
  `/api/roles/${id}`,
  { loading: models.userRoles },
);

// TOTP 2FA APIs
export const getTOTPQR = (id) => http.get(
  `/api/users/${id}/twofa/totp`,
  { camelCase: true },
);

export const enableTOTP = (id, data) => http.put(
  `/api/users/${id}/twofa`,
  data,
);

export const disableTOTP = (id, data) => http.delete(
  `/api/users/${id}/twofa`,
  { data },
);

// =====================================================
// Solomon Platform Extensions
// =====================================================

// Segments.
export const getSegments = async (params) => http.get('/api/segments', { params });
export const getSegment = async (id) => http.get(`/api/segments/${id}`);
export const createSegment = (data) => http.post('/api/segments', data);
export const updateSegment = (id, data) => http.put(`/api/segments/${id}`, data);
export const deleteSegment = (id) => http.delete(`/api/segments/${id}`);
export const getSegmentCount = async (id) => http.get(`/api/segments/${id}/count`);
export const getSegmentSubscribers = async (id, params) => http.get(`/api/segments/${id}/subscribers`, { params });

// Webhooks.
export const getWebhooks = async (params) => http.get('/api/webhooks', { params });
export const getWebhook = async (id) => http.get(`/api/webhooks/${id}`);
export const createWebhook = (data) => http.post('/api/webhooks', data);
export const updateWebhook = (id, data) => http.put(`/api/webhooks/${id}`, data);
export const deleteWebhook = (id) => http.delete(`/api/webhooks/${id}`);
export const getWebhookLog = async (id, params) => http.get(`/api/webhooks/${id}/log`, { params });
export const testWebhook = (id) => http.post(`/api/webhooks/${id}/test`);

// Drip campaigns.
export const getDripCampaigns = async (params) => http.get('/api/drips', { params });
export const getDripCampaign = async (id) => http.get(`/api/drips/${id}`);
export const createDripCampaign = (data) => http.post('/api/drips', data);
export const updateDripCampaign = (id, data) => http.put(`/api/drips/${id}`, data);
export const updateDripCampaignStatus = (id, status) => http.put(`/api/drips/${id}/status`, { status });
export const deleteDripCampaign = (id) => http.delete(`/api/drips/${id}`);
export const getDripSteps = async (id) => http.get(`/api/drips/${id}/steps`);
export const createDripStep = (id, data) => http.post(`/api/drips/${id}/steps`, data);
export const updateDripStep = (id, stepId, data) => http.put(`/api/drips/${id}/steps/${stepId}`, data);
export const deleteDripStep = (id, stepId) => http.delete(`/api/drips/${id}/steps/${stepId}`);
export const getDripEnrollments = async (id, params) => http.get(`/api/drips/${id}/enrollments`, { params });
export const enrollSubscriberInDrip = (id, subscriberId) => http.post(`/api/drips/${id}/enroll`, { subscriber_id: subscriberId });
export const bulkEnrollInDrip = (id, subscriberIds) => http.post(`/api/drips/${id}/enroll-bulk`, { subscriber_ids: subscriberIds });

// Warming (camelCase: false to keep snake_case keys matching Go json tags).
export const getWarmingAddresses = async () => http.get('/api/warming/addresses', { camelCase: false });
export const createWarmingAddress = (data) => http.post('/api/warming/addresses', data);
export const updateWarmingAddress = (id, data) => http.put(`/api/warming/addresses/${id}`, data);
export const deleteWarmingAddress = (id) => http.delete(`/api/warming/addresses/${id}`);
export const getWarmingSenders = async () => http.get('/api/warming/senders', { camelCase: false });
export const createWarmingSender = (data) => http.post('/api/warming/senders', data);
export const updateWarmingSender = (id, data) => http.put(`/api/warming/senders/${id}`, data);
export const deleteWarmingSender = (id) => http.delete(`/api/warming/senders/${id}`);
export const getWarmingTemplates = async () => http.get('/api/warming/templates', { camelCase: false });
export const createWarmingTemplate = (data) => http.post('/api/warming/templates', data);
export const updateWarmingTemplate = (id, data) => http.put(`/api/warming/templates/${id}`, data);
export const deleteWarmingTemplate = (id) => http.delete(`/api/warming/templates/${id}`);
export const getWarmingConfig = async () => http.get('/api/warming/config', { camelCase: false });
export const updateWarmingConfig = (data) => http.put('/api/warming/config', data);
export const getWarmingSendLog = async (params) => http.get('/api/warming/log', { params, camelCase: false });
export const getWarmingStats = async () => http.get('/api/warming/stats', { camelCase: false });
export const getWarmingCampaigns = async () => http.get('/api/warming/campaigns', { camelCase: false });
export const createWarmingCampaign = (data) => http.post('/api/warming/campaigns', data);
export const updateWarmingCampaign = (id, data) => http.put(`/api/warming/campaigns/${id}`, data);
export const deleteWarmingCampaign = (id) => http.delete(`/api/warming/campaigns/${id}`);
export const getWarmingCampaignStats = async (id) => http.get(`/api/warming/campaigns/${id}/stats`, { camelCase: false });

// A/B tests.
export const getABTest = async (id) => http.get(`/api/ab-tests/${id}`);
export const getABTestByCampaign = async (campaignId) => http.get(`/api/campaigns/${campaignId}/ab-test`);
export const createABTest = (data) => http.post('/api/ab-tests', data);
export const updateABTest = (id, data) => http.put(`/api/ab-tests/${id}`, data);
export const deleteABTest = (id) => http.delete(`/api/ab-tests/${id}`);
export const createABVariant = (testId, data) => http.post(`/api/ab-tests/${testId}/variants`, data);
export const updateABVariant = (testId, variantId, data) => http.put(`/api/ab-tests/${testId}/variants/${variantId}`, data);
export const deleteABVariant = (testId, variantId) => http.delete(`/api/ab-tests/${testId}/variants/${variantId}`);

// Automations.
export const getAutomations = async (params) => http.get('/api/automations', { params });
export const getAutomation = async (id) => http.get(`/api/automations/${id}`);
export const createAutomation = (data) => http.post('/api/automations', data);
export const updateAutomation = (id, data) => http.put(`/api/automations/${id}`, data);
export const deleteAutomation = (id) => http.delete(`/api/automations/${id}`);
export const createAutomationNode = (id, data) => http.post(`/api/automations/${id}/nodes`, data);
export const updateAutomationNode = (id, nodeId, data) => http.put(`/api/automations/${id}/nodes/${nodeId}`, data);
export const deleteAutomationNode = (id, nodeId) => http.delete(`/api/automations/${id}/nodes/${nodeId}`);
export const createAutomationEdge = (id, data) => http.post(`/api/automations/${id}/edges`, data);
export const deleteAutomationEdge = (id, edgeId) => http.delete(`/api/automations/${id}/edges/${edgeId}`);

// Contact scoring.
export const getScoringRules = async () => http.get('/api/scoring/rules');
export const getScoringRule = async (id) => http.get(`/api/scoring/rules/${id}`);
export const createScoringRule = (data) => http.post('/api/scoring/rules', data);
export const updateScoringRule = (id, data) => http.put(`/api/scoring/rules/${id}`, data);
export const deleteScoringRule = (id) => http.delete(`/api/scoring/rules/${id}`);
export const getSubscriberScoreLog = async (id, params) => http.get(`/api/subscribers/${id}/score-log`, { params });

// CRM: deals and activities.
export const getDeals = async (params) => http.get('/api/deals', { params });
export const getDeal = async (id) => http.get(`/api/deals/${id}`);
export const createDeal = (data) => http.post('/api/deals', data);
export const updateDeal = (id, data) => http.put(`/api/deals/${id}`, data);
export const deleteDeal = (id) => http.delete(`/api/deals/${id}`);
export const getDealPipeline = async () => http.get('/api/deals/pipeline');
export const getSubscriberCRMActivities = async (id, params) => http.get(`/api/subscribers/${id}/activities`, { params });
export const createContactActivity = (subscriberId, data) => http.post(`/api/subscribers/${subscriberId}/activities`, data);
export const deleteContactActivity = (activityId) => http.delete(`/api/activities/${activityId}`);

import Vue from 'vue';
import Vuex from 'vuex';
import { models } from '../constants';

Vue.use(Vuex);

export default new Vuex.Store({
  state: {
    // Data from API responses for different models, eg: lists, campaigns.
    // The API responses are stored in this map as-is. This is invoked by
    // API requests in `http`. This initialises lists: {}, campaigns: {}
    // etc. on state.
    ...Object.keys(models).reduce((obj, cur) => ({ ...obj, [cur]: [] }), {}),

    // Map of loading status (true, false) indicators for different model keys
    // like lists, campaigns etc. loading: {lists: true, campaigns: true ...}.
    // The Axios API global request interceptor marks a model as loading=true
    // and the response interceptor marks it as false. The model keys are being
    // pre-initialised here to fix "reactivity" issues on first loads.
    loading: Object.keys(models).reduce((obj, cur) => ({ ...obj, [cur]: false }), {}),
  },

  mutations: {
    // Set data from API responses. `model` is 'lists', 'campaigns' etc.
    setModelResponse(state, { model, data }) {
      state[model] = data;
    },

    // Set the loading status for a model globally. When a request starts,
    // status is set to true which is used by the UI to show loaders and block
    // forms. When a response is received, the status is set to false. This is
    // invoked by API requests in `http`.
    setLoading(state, { model, status }) {
      state.loading[model] = status;
    },
  },

  getters: {
    [models.lists]: (state) => state[models.lists],
    [models.subscribers]: (state) => state[models.subscribers],
    [models.campaigns]: (state) => state[models.campaigns],
    [models.media]: (state) => state[models.media],
    [models.templates]: (state) => state[models.templates],
    [models.settings]: (state) => state[models.settings],
    [models.serverConfig]: (state) => state[models.serverConfig],
    [models.logs]: (state) => state[models.logs],
  },

  modules: {
  },
});

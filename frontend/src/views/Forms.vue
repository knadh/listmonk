<template>
  <section class="forms relative">
    <header class="row page-header">
      <div class="col-8">
        <h1>
          {{ $t('forms.title') }}
        </h1>
      </div>
    </header>

    <div class="card page-content" :aria-busy="loading.lists ? 'true' : null" data-spinner="large overlay">
      <p v-if="!loading.lists && publicLists.length === 0">
        {{ $t('forms.noPublicLists') }}
      </p>
      <div class="row" v-else-if="publicLists.length > 0">
        <div class="col-4">
          <h4>{{ $t('forms.publicLists') }}</h4>
          <p>{{ $t('forms.selectHelp') }}</p>

          <ul class="no" data-cy="lists">
            <li v-for="(l, i) in publicLists" :key="l.id">
              <oat-checkbox v-model="checked" :native-value="i">
                {{ l.name }}
              </oat-checkbox>
            </li>
          </ul>

          <template v-if="serverConfig.public_subscription.enabled">
            <hr />
            <h4>{{ $t('forms.publicSubPage') }}</h4>
            <p>
              <a :href="`${serverConfig.root_url}/subscription/form`" target="_blank" rel="noopener noreferer"
                data-cy="url">
                {{ serverConfig.root_url }}/subscription/form
              </a>
            </p>
          </template>

          <hr />
          <h4>{{ $t('forms.redirectURL') }}</h4>
          <p class="text-light text-7">
            {{ $t('forms.redirectURLHelp') }}
          </p>
          <ul v-if="redirectURLs.length > 0" class="no" data-cy="redirect-urls">
            <li>
              <oat-radio v-model="selectedRedirectURL" native-value="">
                {{ $t('globals.terms.none') }}
              </oat-radio>
            </li>
            <li v-for="url in redirectURLs" :key="url">
              <oat-radio v-model="selectedRedirectURL" :native-value="url">
                {{ url }}
              </oat-radio>
            </li>
          </ul>
        </div>
        <div class="col-12" data-cy="form">
          <h4>{{ $t('forms.formHTML') }}</h4>
          <p>
            {{ $t('forms.formHTMLHelp') }}
          </p>

          <code-editor lang="html" v-if="checked.length > 0" v-model="html" disabled />
        </div>
      </div><!-- row -->
    </div>
  </section>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import CodeEditor from '../components/CodeEditor.vue';

export default Vue.extend({
  name: 'ListForm',

  components: {
    'code-editor': CodeEditor,
  },

  data() {
    return {
      checked: [],
      html: '',
      selectedRedirectURL: '',
    };
  },

  methods: {
    escapeAttr(value) {
      return String(value)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
    },

    renderHTML() {
      let h = `<form method="post" action="${this.serverConfig.root_url}/subscription/form" class="listmonk-form">\n`
        + '  <div>\n'
        + `    <h3>${this.$t('public.sub')}</h3>\n`
        + '    <input type="hidden" name="nonce" />\n';

      if (this.selectedRedirectURL) {
        h += `    <input type="hidden" name="next" value="${this.escapeAttr(this.selectedRedirectURL)}" />\n`;
      }

      h += '\n'
        + `    <p><input type="email" name="email" required placeholder="${this.$t('subscribers.email')}" /></p>\n`
        + `    <p><input type="text" name="name" placeholder="${this.$t('public.subName')}" /></p>\n\n`;

      this.checked.forEach((i) => {
        const l = this.publicLists[parseInt(i, 10)];

        h += '    <p>\n'
          + `      <input id="${l.uuid.substr(0, 5)}" type="checkbox" name="l" checked value="${l.uuid}" />\n`
          + `      <label for="${l.uuid.substr(0, 5)}">${l.name}</label>\n`;

        if (l.description) {
          h += '      <br />\n'
            + `      <span>${l.description}</span>\n`;
        }

        h += '    </p>\n';
      });

      // Captcha?
      if (this.serverConfig.public_subscription.captcha_enabled) {
        if (this.serverConfig.public_subscription.captcha_provider === 'altcha') {
          h += '\n'
            + `    <altcha-widget challengeurl="${this.serverConfig.root_url}/api/public/captcha/altcha"></altcha-widget>\n`
            + `    <${'script'} type="module" src="${this.serverConfig.root_url}/public/static/altcha.umd.js" async defer></${'script'}>\n`;
        } else if (this.serverConfig.public_subscription.captcha_provider === 'hcaptcha') {
          h += '\n'
            + `    <div class="h-captcha" data-sitekey="${this.serverConfig.public_subscription.captcha_key}"></div>\n`
            + `    <${'script'} src="https://js.hcaptcha.com/1/api.js" async defer></${'script'}>\n`;
        }
      }

      h += '\n'
        + `    <input type="submit" value="${this.$t('public.sub')} " />\n`
        + '  </div>\n'
        + '</form>';

      this.html = h;
    },
  },

  computed: {
    ...mapState(['loading', 'lists', 'serverConfig']),

    publicLists() {
      if (!this.lists.results) {
        return [];
      }
      return this.lists.results.filter((l) => l.type === 'public');
    },

    redirectURLs() {
      const urls = this.serverConfig.public_subscription
        ? this.serverConfig.public_subscription.redirect_urls
        : [];
      return Array.isArray(urls) ? urls : [];
    },
  },

  watch: {
    checked() {
      this.renderHTML();
    },

    selectedRedirectURL() {
      this.renderHTML();
    },
  },
});
</script>

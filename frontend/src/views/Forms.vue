<template>
  <section class="forms content relative">
    <h1 class="title is-4">
      {{ $t('forms.title') }}
    </h1>
    <hr />

    <b-loading v-if="loading.lists" :active="loading.lists" :is-full-page="false" />
    <p v-else-if="publicLists.length === 0">
      {{ $t('forms.noPublicLists') }}
    </p>

    <div class="columns" v-else-if="publicLists.length > 0">
      <div class="column is-4">
        <h4>{{ $t('forms.publicLists') }}</h4>
        <p>{{ $t('forms.selectHelp') }}</p>

        <b-loading :active="loading.lists" :is-full-page="false" />
        <ul class="no" data-cy="lists">
          <li v-for="(l, i) in publicLists" :key="l.id">
            <b-checkbox v-model="checked" :native-value="i">
              {{ l.name }}
            </b-checkbox>
          </li>
        </ul>

        <template v-if="settings['app.enable_public_subscription_page']">
          <hr />
          <h4>{{ $t('forms.publicSubPage') }}</h4>
          <p>
            <a :href="`${settings['app.root_url']}/subscription/form`" target="_blank" rel="noopener noreferer"
              data-cy="url">
              {{ settings['app.root_url'] }}/subscription/form
            </a>
          </p>
        </template>
      </div>
      <div class="column" data-cy="form">
        <h4>{{ $t('forms.formHTML') }}</h4>
        <p>
          {{ $t('forms.formHTMLHelp') }}
        </p>

        <html-editor v-if="checked.length > 0" v-model="html" disabled />
      </div>
    </div><!-- columns -->
  </section>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import HTMLEditor from '../components/HTMLEditor.vue';

export default Vue.extend({
  name: 'ListForm',

  components: {
    'html-editor': HTMLEditor,
  },

  data() {
    return {
      checked: [],
      html: '',
    };
  },

  methods: {
    renderHTML() {
      let h = `<form method="post" action="${this.settings['app.root_url']}/subscription/form" class="listmonk-form">\n`
        + '  <div>\n'
        + `    <h3>${this.$t('public.sub')}</h3>\n`
        + '    <input type="hidden" name="nonce" />\n\n'
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
      if (this.settings['security.enable_captcha']) {
        h += '\n'
          + `    <div class="h-captcha" data-sitekey="${this.settings['security.captcha_key']}"></div>\n`
          + `    <${'script'} src="https://js.hcaptcha.com/1/api.js" async defer></${'script'}>\n`;
      }

      h += '\n'
        + `    <input type="submit" value="${this.$t('public.sub')} " />\n`
        + '  </div>\n'
        + '</form>';

      this.html = h;
    },
  },

  computed: {
    ...mapState(['loading', 'lists', 'settings']),

    publicLists() {
      if (!this.lists.results) {
        return [];
      }
      return this.lists.results.filter((l) => l.type === 'public');
    },
  },

  watch: {
    checked() {
      this.renderHTML();
    },
  },
});
</script>

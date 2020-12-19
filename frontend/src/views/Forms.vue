<template>
  <section class="forms content relative">
    <h1 class="title is-4">{{ $t('forms.title') }}</h1>
    <hr />
    <b-loading v-if="loading.lists" :active="loading.lists" :is-full-page="false" />
    <div class="columns" v-else-if="publicLists.length > 0">
      <div class="column is-4">
        <h4>{{ $t('forms.publicLists') }}</h4>
        <p>{{ $t('forms.selectHelp') }}</p>

        <b-loading :active="loading.lists" :is-full-page="false" />
        <ul class="no">
          <li v-for="l in publicLists" :key="l.id">
            <b-checkbox v-model="checked"
              :native-value="l.uuid">{{ l.name }}</b-checkbox>
          </li>
        </ul>
      </div>
      <div class="column">
        <h4>{{ $t('forms.formHTML') }}</h4>
        <p>
          {{ $t('forms.formHTMLHelp') }}
        </p>

        <!-- eslint-disable max-len -->
        <pre>&lt;form method=&quot;post&quot; action=&quot;{{ serverConfig.rootURL }}/subscription/form&quot; class=&quot;listmonk-form&quot;&gt;
    &lt;div&gt;
        &lt;h3&gt;Subscribe&lt;/h3&gt;
        &lt;p&gt;&lt;input type=&quot;text&quot; name=&quot;email&quot; placeholder=&quot;E-mail&quot; /&gt;&lt;/p&gt;
        &lt;p&gt;&lt;input type=&quot;text&quot; name=&quot;name&quot; placeholder=&quot;Name (optional)&quot; /&gt;&lt;/p&gt;
      <template v-for="l in publicLists"><span v-if="l.uuid in selected" :key="l.id" :set="id = l.uuid.substr(0, 5)">
        &lt;p&gt;
          &lt;input id=&quot;{{ id }}&quot; type=&quot;checkbox&quot; name=&quot;l&quot; value=&quot;{{ l.uuid }}&quot; /&gt;
          &lt;label for=&quot;{{ id }}&quot;&gt;{{ l.name }}&lt;/label&gt;
        &lt;/p&gt;</span></template>
        &lt;p&gt;&lt;input type=&quot;submit&quot; value=&quot;Subscribe&quot; /&gt;&lt;/p&gt;
    &lt;/div&gt;
&lt;/form&gt;</pre>
      </div>
    </div><!-- columns -->

    <p v-else></p>
  </section>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';

export default Vue.extend({
  name: 'ListForm',

  data() {
    return {
      checked: [],
    };
  },

  methods: {
    getPublicLists(lists) {
      return lists.filter((l) => l.type === 'public');
    },
  },

  computed: {
    ...mapState(['lists', 'loading', 'serverConfig']),

    publicLists() {
      if (!this.lists.results) {
        return [];
      }
      return this.lists.results.filter((l) => l.type === 'public');
    },

    selected() {
      const sel = [];
      this.checked.forEach((uuid) => {
        sel[uuid] = true;
      });
      return sel;
    },
  },
});
</script>

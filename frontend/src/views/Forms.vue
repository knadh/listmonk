<template>
  <section class="forms content relative">
    <h1 class="title is-4">Forms</h1>
    <hr />
    <b-loading v-if="loading.lists" :active="loading.lists" :is-full-page="false" />
    <div class="columns" v-else-if="publicLists.length > 0">
      <div class="column is-4">
        <h4>Public lists</h4>
        <p>Select lists to add to the form.</p>

        <b-loading :active="loading.lists" :is-full-page="false" />
        <ul class="no">
          <li v-for="l in publicLists" :key="l.id">
            <b-checkbox v-model="checked"
              :native-value="l.uuid">{{ l.name }}</b-checkbox>
          </li>
        </ul>
      </div>
      <div class="column">
        <h4>Form HTML</h4>
        <p>
          Use the following HTML to show a subscription form on an external webpage.
        </p>
        <p>
          The form should have the <code>email</code> field and one or more <code>l</code>
          (list UUID) fields. The <code>name</code> field is optional.
        </p>

        <pre><!-- eslint-disable max-len -->&lt;form method=&quot;post&quot; action=&quot;http://localhost:9000/subscription/form&quot; class=&quot;listmonk-form&quot;&gt;
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

    <p v-else>There are no public lists to create forms.</p>
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
    ...mapState(['lists', 'loading']),

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

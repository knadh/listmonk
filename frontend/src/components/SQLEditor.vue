<template>
  <div class="sql-editor">
    <div class="field has-addons">
      <div class="control is-expanded">
        <b-input
          :value="value"
          @input="$emit('input', $event)"
          @keydown.native.enter="$emit('enter')"
          type="textarea"
          :placeholder="placeholder"
          :rows="rows"
          ref="textarea"
        />
      </div>
      <div v-if="showSnippetsButton" class="control" style="position: relative;">
        <b-button
          @click="toggleSnippetsDropdown($event)"
          type="is-light"
          icon-left="code"
          :class="{ 'is-info': showSnippetsDropdown }"
          :disabled="sqlSnippets.length === 0"
          :title="`Snippets count: ${sqlSnippets.length}`"
        >
          {{ $t('sqlSnippets.snippet') }} ({{ sqlSnippets.length }})
        </b-button>

        <!-- SQL Snippets Dropdown -->
        <div v-if="showSnippetsDropdown" class="dropdown is-active" style="position: absolute; top: 100%; right: 0; width: 400px; z-index: 9999;">
          <div class="dropdown-menu" style="width: 100%;">
            <div class="dropdown-content">
              <div v-if="sqlSnippets.length === 0" class="dropdown-item has-text-grey">
                <em>{{ $t('globals.messages.emptyState') }}</em>
              </div>
              <a v-for="snippet in sqlSnippets" :key="snippet.id"
                @click="selectSQLSnippet(snippet)"
                @keydown.enter="selectSQLSnippet(snippet)"
                class="dropdown-item"
                tabindex="0"
                style="cursor: pointer;">
                <div class="media">
                  <div class="media-left">
                    <b-icon icon="code" size="is-small" class="has-text-info" />
                  </div>
                  <div class="media-content">
                    <div class="content">
                      <p class="is-size-6 has-text-weight-semibold mb-1">{{ snippet.name }}</p>
                      <p v-if="snippet.description" class="is-size-7 has-text-grey mb-2">{{ snippet.description }}</p>
                      <p class="is-size-7 has-text-dark">
                        <code class="has-background-light">{{ snippet.querySql || snippet.query_sql }}</code>
                      </p>
                    </div>
                  </div>
                </div>
              </a>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'SQLEditor',

  props: {
    value: {
      type: String,
      default: '',
    },
    placeholder: {
      type: String,
      default: 'Enter SQL query...',
    },
    rows: {
      type: Number,
      default: 4,
    },
    showSnippetsButton: {
      type: Boolean,
      default: true,
    },
  },

  data() {
    return {
      sqlSnippets: [],
      showSnippetsDropdown: false,
    };
  },

  mounted() {
    if (this.showSnippetsButton) {
      this.loadSQLSnippets();
      // Add click outside listener for dropdown
      document.addEventListener('click', this.handleClickOutside);
    }
  },

  beforeDestroy() {
    if (this.showSnippetsButton) {
      // Remove click outside listener
      document.removeEventListener('click', this.handleClickOutside);
    }
  },

  methods: {
    // Load active SQL snippets for autocomplete
    loadSQLSnippets() {
      this.$api.getSQLSnippets({ is_active: true }).then((data) => {
        // API returns the array directly, not wrapped in results
        this.sqlSnippets = data || [];
      }).catch(() => {
        // Silently fail for SQL snippets loading - this is optional functionality
        this.sqlSnippets = [];
      });
    },

    // Handle SQL snippet selection
    selectSQLSnippet(snippet) {
      if (snippet && (snippet.querySql || snippet.query_sql)) {
        const sqlQuery = snippet.querySql || snippet.query_sql;
        this.$emit('input', sqlQuery);
        this.$emit('snippet-selected', sqlQuery);
        this.showSnippetsDropdown = false;
      }
    },

    // Toggle snippets dropdown
    toggleSnippetsDropdown(event) {
      event.stopPropagation();
      this.showSnippetsDropdown = !this.showSnippetsDropdown;
    },

    // Handle clicks outside dropdown to close it
    handleClickOutside(event) {
      const dropdown = event.target.closest('.dropdown');
      const button = event.target.closest('.control');

      if (!dropdown && !button) {
        this.showSnippetsDropdown = false;
      }
    },

    // Focus the textarea
    focus() {
      this.$refs.textarea.focus();
    },

    // Refresh SQL snippets (public method)
    refreshSnippets() {
      this.loadSQLSnippets();
    },
  },
};
</script>

<style scoped>
.sql-editor {
  position: relative;
}
</style>

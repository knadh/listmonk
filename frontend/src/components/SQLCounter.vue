<template>
  <div v-if="query && query.trim() !== ''" class="sql-counter mb-3">
    <div class="field">
      <div class="control">
        <div class="notification is-light p-3">
          <div class="level is-mobile">
            <div class="level-left">
              <div class="level-item">
                <div class="media">
                  <div class="media-left">
                    <b-icon icon="account-search" size="is-small" class="has-text-info" />
                  </div>
                  <div class="media-content">
                    <div v-if="subscriberCount.loading" class="is-flex is-align-items-center">
                      <b-loading :is-full-page="false" v-model="subscriberCount.loading" :can-cancel="false" />
                      <span class="is-size-6 has-text-grey ml-2">{{ $t('globals.messages.loading') }}...</span>
                    </div>
                    <div v-else-if="subscriberCount.error" class="has-text-danger is-size-6">
                      <b-icon icon="alert-circle" size="is-small" class="mr-1" />
                      {{ $t('sqlSnippets.invalidQuery') }}
                    </div>
                    <div v-else class="is-size-6">
                      <span class="has-text-weight-semibold has-text-info">{{ subscriberCount.found.toLocaleString() }}</span>
                      {{ $t('subscribers.matchingSubscribers') }}
                      <span class="has-text-grey">
                        ({{ $t('subscribers.outOfTotal', { total: subscriberCount.total.toLocaleString() }) }})
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
            <div v-if="showLiveToggle" class="level-right">
              <div class="level-item">
                <b-field>
                  <b-checkbox v-model="isLiveValidationEnabled" size="is-small">
                    {{ $t('sqlSnippets.liveValidation') }}
                  </b-checkbox>
                </b-field>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'SQLCounter',
  props: {
    query: {
      type: String,
      default: '',
    },
    showLiveToggle: {
      type: Boolean,
      default: true,
    },
    liveValidationEnabled: {
      type: Boolean,
      default: true,
    },
  },

  data() {
    return {
      subscriberCount: {
        loading: false,
        error: false,
        found: 0,
        total: 0,
      },
      countDebounceTimer: null,
      isLiveValidationEnabled: this.liveValidationEnabled,
    };
  },

  watch: {
    query(newQuery) {
      if (this.isLiveValidationEnabled) {
        this.updateSubscriberCount(newQuery);
      } else {
        // Reset state when live validation is disabled
        this.subscriberCount.loading = false;
        this.subscriberCount.error = false;
        this.subscriberCount.found = 0;
      }
    },

    isLiveValidationEnabled(enabled) {
      this.$emit('update:liveValidationEnabled', enabled);
      if (enabled && this.query) {
        this.updateSubscriberCount(this.query, true); // immediate when toggling on
      } else {
        // Reset state when disabled
        this.subscriberCount.loading = false;
        this.subscriberCount.error = false;
        this.subscriberCount.found = 0;
      }
    },

    liveValidationEnabled(enabled) {
      this.isLiveValidationEnabled = enabled;
    },
  },

  mounted() {
    this.loadTotalSubscriberCount();
    // Always run initial count if there's a query, regardless of live validation setting
    if (this.query) {
      this.updateSubscriberCount(this.query, true); // immediate = true for initial load
    }
  },

  beforeDestroy() {
    if (this.countDebounceTimer) {
      clearTimeout(this.countDebounceTimer);
    }
  },

  methods: {
    // Load total subscriber count
    loadTotalSubscriberCount() {
      this.$api.countSQLSnippet({ query_sql: '' }).then((data) => {
        this.subscriberCount.total = data.total || 0;
      }).catch(() => {
        // Silently fail for total count
        this.subscriberCount.total = 0;
      });
    },

    // Update subscriber count with debouncing
    updateSubscriberCount(query, immediate = false) {
      if (this.countDebounceTimer) {
        clearTimeout(this.countDebounceTimer);
      }

      // Don't count if query is empty
      if (!query || query.trim() === '') {
        this.subscriberCount.found = 0;
        this.subscriberCount.error = false;
        return;
      }

      const executeCount = () => {
        this.subscriberCount.loading = true;
        this.subscriberCount.error = false;

        this.$api.countSQLSnippet({ query_sql: query }).then((data) => {
          this.subscriberCount.found = data.matched || 0;
          this.subscriberCount.total = data.total || this.subscriberCount.total;
          this.subscriberCount.loading = false;
        }).catch(() => {
          // Only show error state in UI, don't show toast for live validation
          this.subscriberCount.error = true;
          this.subscriberCount.loading = false;
        });
      };

      if (immediate) {
        // Execute immediately for initial load
        executeCount();
      } else {
        // Use longer debounce time to reduce annoying error messages
        this.countDebounceTimer = setTimeout(executeCount, 1500);
      }
    },

    // Manual validation (for when live validation is disabled)
    validateQuery() {
      if (!this.query || this.query.trim() === '') {
        this.$utils.toast(this.$t('sqlSnippets.emptyQuery'), 'is-warning');
        return;
      }

      this.updateSubscriberCount(this.query);
    },

    // Update count immediately (public method for programmatic calls)
    updateImmediately() {
      if (this.query) {
        this.updateSubscriberCount(this.query, true);
      }
    },
  },
};
</script>

<style scoped>
.sql-counter {
  position: relative;
}
</style>

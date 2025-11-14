<template>
  <div class="subscriber-activity">
    <div v-if="isLoading" class="has-text-centered">
      <b-loading :active="true" :is-full-page="false"></b-loading>
    </div>

    <div v-else>
      <!-- Summary Stats -->
      <div class="columns">
        <div class="column is-4">
          <div class="box has-text-centered">
            <p class="heading">{{ $t('subscribers.activity.totalCampaignsViewed') }}</p>
            <p class="title">{{ activity.campaignViews ? activity.campaignViews.length : 0 }}</p>
          </div>
        </div>
        <div class="column is-4">
          <div class="box has-text-centered">
            <p class="heading">{{ $t('subscribers.activity.totalViews') }}</p>
            <p class="title">{{ totalViews }}</p>
          </div>
        </div>
        <div class="column is-4">
          <div class="box has-text-centered">
            <p class="heading">{{ $t('subscribers.activity.totalClicks') }}</p>
            <p class="title">{{ totalClicks }}</p>
          </div>
        </div>
      </div>

      <!-- Campaign Views Section -->
      <div class="section-header mb-4">
        <h5 class="title is-5">
          <b-icon icon="email-open-outline" size="is-small" />
          {{ $t('subscribers.activity.campaignViews') }}
        </h5>
      </div>

      <div v-if="activity.campaignViews && activity.campaignViews.length > 0">
        <b-table :data="activity.campaignViews" hoverable default-sort="lastViewedAt" default-sort-direction="desc"
          paginated :per-page="10" :pagination-simple="false" class="campaign-views-table">
          <b-table-column v-slot="props" field="subject" :label="$tc('globals.terms.campaign', 1)" sortable>
            <div v-if="props.row.uuid">
              <router-link :to="{ name: 'campaign', params: { id: props.row.id } }">
                {{ props.row.subject || props.row.name }}
              </router-link>
            </div>
            <div v-else>
              <em class="has-text-grey">{{ $t('subscribers.activity.campaignDeleted') }}</em>
            </div>
          </b-table-column>

          <b-table-column v-slot="props" field="viewCount" :label="$t('subscribers.activity.views')" sortable
            numeric>
            <span class="tag is-light">{{ props.row.viewCount }}</span>
          </b-table-column>

          <b-table-column v-slot="props" field="lastViewedAt" :label="$t('subscribers.activity.lastViewed')" sortable>
            <span v-if="props.row.lastViewedAt">
              {{ $utils.niceDate(props.row.lastViewedAt, true) }}
            </span>
          </b-table-column>
        </b-table>
      </div>
      <div v-else class="has-text-centered has-text-grey p-6">
        <b-icon icon="email-outline" size="is-large" />
        <p class="mt-2">{{ $t('subscribers.activity.noCampaignViews') }}</p>
      </div>

      <!-- Link Clicks Section -->
      <div class="section-header mb-4 mt-6">
        <h5 class="title is-5">
          <b-icon icon="cursor-default-click-outline" size="is-small" />
          {{ $t('subscribers.activity.linkClicks') }}
        </h5>
      </div>

      <div v-if="activity.linkClicks && activity.linkClicks.length > 0">
        <b-table :data="activity.linkClicks" hoverable default-sort="lastClickedAt" default-sort-direction="desc"
          paginated :per-page="10" :pagination-simple="false" class="link-clicks-table">
          <b-table-column v-slot="props" field="url" :label="$t('subscribers.activity.url')" sortable>
            <a :href="props.row.url" target="_blank" rel="noopener noreferrer" class="is-size-7">
              {{ props.row.url }}
              <b-icon icon="open-in-new" size="is-small" />
            </a>
          </b-table-column>

          <b-table-column v-slot="props" field="campaignName" :label="$tc('globals.terms.campaign', 1)" sortable>
            <div v-if="props.row.campaignUuid">
              <router-link :to="{ name: 'campaign', params: { id: props.row.campaignId } }">
                {{ props.row.campaignSubject || props.row.campaignName }}
              </router-link>
            </div>
            <div v-else>
              <em class="has-text-grey">{{ $t('subscribers.activity.campaignDeleted') }}</em>
            </div>
          </b-table-column>

          <b-table-column v-slot="props" field="clickCount" :label="$t('subscribers.activity.clicks')" sortable
            numeric>
            <span class="tag is-light">{{ props.row.clickCount }}</span>
          </b-table-column>

          <b-table-column v-slot="props" field="lastClickedAt" :label="$t('subscribers.activity.lastClicked')" sortable>
            <span v-if="props.row.lastClickedAt">
              {{ $utils.niceDate(props.row.lastClickedAt, true) }}
            </span>
          </b-table-column>
        </b-table>
      </div>
      <div v-else class="has-text-centered has-text-grey p-6">
        <b-icon icon="link-variant-off" size="is-large" />
        <p class="mt-2">{{ $t('subscribers.activity.noLinkClicks') }}</p>
      </div>
    </div>
  </div>
</template>

<script>
import Vue from 'vue';

export default Vue.extend({
  props: {
    subscriberId: {
      type: Number,
      required: true,
    },
  },

  data() {
    return {
      isLoading: false,
      activity: {
        campaignViews: [],
        linkClicks: [],
      },
    };
  },

  computed: {
    totalViews() {
      if (!this.activity.campaignViews) return 0;
      return this.activity.campaignViews.reduce((sum, v) => sum + (v.viewCount || 0), 0);
    },

    totalClicks() {
      if (!this.activity.linkClicks) return 0;
      return this.activity.linkClicks.reduce((sum, c) => sum + (c.clickCount || 0), 0);
    },
  },

  mounted() {
    this.getActivity();
  },

  methods: {
    getActivity() {
      this.isLoading = true;
      this.$api.getSubscriberActivity(this.subscriberId).then((data) => {
        this.activity = data;
        this.isLoading = false;
      }).catch(() => {
        this.isLoading = false;
      });
    },
  },
});
</script>

<style scoped>
.subscriber-activity {
  min-height: 300px;
}

.section-header {
  border-bottom: 1px solid #dbdbdb;
  padding-bottom: 0.5rem;
}

.box {
  height: 100%;
}

.campaign-views-table,
.link-clicks-table {
  margin-top: 1rem;
}
</style>

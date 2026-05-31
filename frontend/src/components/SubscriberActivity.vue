<template>
  <div class="subscriber-activity">
    <div v-if="isLoading" class="align-center">
      <oat-loading :active="true" :is-full-page="false" />
    </div>

    <div v-else>
      <!-- Summary Stats -->
      <div class="row">
        <div class="col-4">
          <div class="card align-center">
            <p class="heading">{{ $t('globals.terms.campaigns') }}</p>
            <p>{{ activity.campaignViews ? activity.campaignViews.length : 0 }}</p>
          </div>
        </div>
        <div class="col-4">
          <div class="card align-center">
            <p class="heading">{{ $t('campaigns.views') }}</p>
            <p>{{ totalViews }}</p>
          </div>
        </div>
        <div class="col-4">
          <div class="card align-center">
            <p class="heading">{{ $t('campaigns.clicks') }}</p>
            <p>{{ totalClicks }}</p>
          </div>
        </div>
      </div>

      <!-- Campaign Views Section -->
      <div class="stack-header mb-4">
        <h5>
          {{ $t('campaigns.views') }}
        </h5>
      </div>

      <div v-if="activity.campaignViews && activity.campaignViews.length > 0">
        <oat-data-table :data="activity.campaignViews" default-sort="lastViewedAt" default-sort-direction="desc"
          paginated :per-page="10" :pagination-simple="false" class="campaign-views-table">
          <oat-table-column v-slot="props" field="subject" :label="$tc('globals.terms.campaign', 1)" sortable>
            <div v-if="props.row.uuid">
              <router-link :to="{ name: 'campaign', params: { id: props.row.id } }">
                {{ props.row.name }}
              </router-link>
              <p class="text-light text-7">{{ props.row.subject }}</p>
            </div>
            <div v-else>
              <em class="text-light">{{ $t('subscribers.activity.campaignDeleted') }}</em>
            </div>
          </oat-table-column>

          <oat-table-column v-slot="props" field="viewCount" :label="$t('campaigns.views')" sortable numeric>
            <span class="badge ">{{ props.row.viewCount }}</span>
          </oat-table-column>

          <oat-table-column v-slot="props" field="lastViewedAt" :label="$t('globals.fields.createdAt')" sortable>
            <span v-if="props.row.lastViewedAt">
              {{ $utils.niceDate(props.row.lastViewedAt, true) }}
            </span>
          </oat-table-column>
</oat-data-table>
      </div>
      <div v-else class="align-center text-light p-6">
        <p class="mt-2">{{ $t('globals.messages.emptyState') }}</p>
      </div>

      <!-- Link Clicks Section -->
      <div class="stack-header mb-4 mt-6">
        <h5>
          {{ $t('campaigns.clicks') }}
        </h5>
      </div>

      <div v-if="activity.linkClicks && activity.linkClicks.length > 0">
        <oat-data-table :data="activity.linkClicks" default-sort="lastClickedAt" default-sort-direction="desc"
          paginated :per-page="10" :pagination-simple="false" class="link-clicks-table">
          <oat-table-column v-slot="props" field="url" :label="$t('globals.terms.url')" cell-class="link-click-url"
            sortable>
            <a :href="props.row.url" target="_blank" rel="noopener noreferrer">
              {{ props.row.url }}
            </a>
          </oat-table-column>

          <oat-table-column v-slot="props" field="campaignName" :label="$tc('globals.terms.campaign', 1)" sortable>
            <div v-if="props.row.campaignUuid">
              <router-link :to="{ name: 'campaign', params: { id: props.row.campaignId } }">
                {{ props.row.campaignSubject || props.row.campaignName }}
              </router-link>
            </div>
            <div v-else>
              &mdash;
            </div>
          </oat-table-column>

          <oat-table-column v-slot="props" field="clickCount" :label="$t('campaigns.clicks')" sortable numeric>
            <span class="badge ">{{ props.row.clickCount }}</span>
          </oat-table-column>

          <oat-table-column v-slot="props" field="lastClickedAt" :label="$t('globals.fields.createdAt')" sortable>
            <span v-if="props.row.lastClickedAt">
              {{ $utils.niceDate(props.row.lastClickedAt, true) }}
            </span>
          </oat-table-column>
</oat-data-table>
      </div>
      <div v-else class="align-center text-light p-6">
        <p class="mt-2">{{ $t('globals.messages.emptyState') }}</p>
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

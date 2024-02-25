<template>
  <div class="field list-selector">
    <div :class="['list-tags', ...classes]">
      <b-taglist>
        <b-tag v-for="l in selectedItems" :key="l.id" :class="l.subscriptionStatus" :closable="!$props.disabled"
          :data-id="l.id" @close="removeList(l.id)" class="list">
          {{ l.name }} <sup v-if="l.optin === 'double'">{{ $t(`subscribers.status.${l.subscriptionStatus}`) }}</sup>
        </b-tag>
      </b-taglist>
    </div>

    <b-field :message="message" :label="label + (selectedItems ? ` (${selectedItems.length})` : '')"
      label-position="on-border">
      <b-autocomplete v-model="query" :placeholder="placeholder" clearable dropdown-position="top"
        :disabled="all.length === 0 || $props.disabled" :keep-first="true" :clear-on-select="true" :open-on-focus="true"
        :data="filteredLists" @select="selectList" field="name" />
    </b-field>
  </div>
</template>

<script>
import Vue from 'vue';

export default {
  name: 'ListSelector',

  props: {
    label: { type: String, default: '' },
    placeholder: { type: String, default: '' },
    message: { type: String, default: '' },
    required: Boolean,
    disabled: Boolean,
    classes: {
      type: Array,
      default: () => [],
    },
    selected: {
      type: Array,
      default: () => [],
    },
    all: {
      type: Array,
      default: () => [],
    },
  },

  data() {
    return {
      query: '',
      selectedItems: [],
    };
  },

  methods: {
    selectList(l) {
      if (!l) {
        return;
      }
      this.selectedItems.push(l);
      this.query = '';

      // Propagate the items to the parent's v-model binding.
      Vue.nextTick(() => {
        this.$emit('input', this.selectedItems);
      });
    },

    removeList(id) {
      this.selectedItems = this.selectedItems.filter((l) => l.id !== id);

      // Propagate the items to the parent's v-model binding.
      Vue.nextTick(() => {
        this.$emit('input', this.selectedItems);
      });
    },
  },

  computed: {
    // Return the list of unselected lists.
    filteredLists() {
      // Get a map of IDs of the user subscriptions. eg: {1: true, 2: true};
      const subIDs = this.selectedItems.reduce((obj, item) => ({ ...obj, [item.id]: true }), {});

      // Filter lists from the global lists whose IDs are not in the user's
      // subscribed ist.
      const q = this.query.toLowerCase();
      return this.$props.all.filter(
        (l) => (!(l.id in subIDs) && l.name.toLowerCase().indexOf(q) >= 0),
      );
    },
  },

  watch: {
    // This is required to update the array of lists to propagate from parent
    // components and "react" on the selector.
    selected() {
      // Deep-copy.
      this.selectedItems = JSON.parse(JSON.stringify(this.selected));
    },
  },

  mounted() {
    if (this.selected) {
      this.selectedItems = JSON.parse(JSON.stringify(this.selected));
    }
  },
};
</script>

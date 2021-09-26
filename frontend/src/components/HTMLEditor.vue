<template>
  <div ref="htmlEditor" id="html-editor" class="html-editor"></div>
</template>

<script>
import CodeFlask from 'codeflask';
import { colors } from '../constants';

export default {
  props: {
    value: String,
    disabled: Boolean,
  },

  data() {
    return {
      flask: null,
    };
  },

  methods: {
    initHTMLEditor(body) {
      // CodeFlask editor is rendered in a shadow DOM tree to keep its styles
      // sandboxed away from the global styles.
      const el = document.createElement('code-flask');
      el.attachShadow({ mode: 'open' });
      el.shadowRoot.innerHTML = `
        <style>
          .codeflask .codeflask__flatten { font-size: 15px; }
          .codeflask .codeflask__lines { background: #fafafa; z-index: 10; }
          .codeflask .token.tag { font-weight: bold; }
          .codeflask .token.attr-name { color: #111; }
          .codeflask .token.attr-value { color: ${colors.primary} !important; }
        </style>
        <div id="area"></area>
      `;
      this.$refs.htmlEditor.appendChild(el);

      this.flask = new CodeFlask(el.shadowRoot.getElementById('area'), {
        language: 'html',
        lineNumbers: false,
        styleParent: el.shadowRoot,
        readonly: this.disabled,
      });

      this.flask.onUpdate((b) => {
        this.$emit('input', b);
      });

      // Set the initial value.
      this.flask.updateCode(body);
    },
  },

  mounted() {
    this.initHTMLEditor(this.$props.value || '');
  },
};
</script>

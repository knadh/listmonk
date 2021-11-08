<template>
  <div ref="appearanceEditor" id="appearance-editor" class="html-editor"></div>
</template>

<script>
import CodeFlask from 'codeflask';

export default {
  props: {
    value: String,
    language: String,
  },

  data() {
    return {
      data: '',
      flask: null,
    };
  },

  methods: {
    initAppearanceEditor(body) {
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
      this.$refs.appearanceEditor.appendChild(el);

      this.flask = new CodeFlask(el.shadowRoot.getElementById('area'), {
        language: this.language,
        lineNumbers: false,
        styleParent: el.shadowRoot,
        defaultTheme: false,
      });

      this.flask.onUpdate((v) => {
        this.data = v;
        this.$emit('input', v);
      });

      // Set the initial value.
      this.flask.updateCode(body);
    },
  },

  mounted() {
    if (this.$props.value) {
      this.initAppearanceEditor(this.$props.value || '');
    } else {
      this.initAppearanceEditor('');
    }
  },

  watch: {
    value(newVal) {
      if (newVal !== this.data) {
        if (newVal) {
          this.flask.updateCode(newVal);
        } else {
          this.flask.updateCode('');
        }
      }
    },
  },
};

</script>

module.exports = {
  root: true,
  env: {
    node: true,
    // es2022: true,
  },
  plugins: ['vue'],
  extends: [
    'eslint:recommended',
    'plugin:vue/essential',
    'plugin:vue/strongly-recommended',
    '@vue/eslint-config-airbnb',
  ],
  parser: 'vue-eslint-parser',
  rules: {
    'class-methods-use-this': 'off',
    'vue/multi-word-component-names': 'off',
    'vue/quote-props': 'off',
    'vue/first-attribute-linebreak': 'off',
    'vue/no-child-content': 'off',
    'vue/max-attributes-per-line': 'off',
    'vue/html-indent': 'off',
    'vue/html-closing-bracket-newline': 'off',
    'vue/max-len': ['error', {
      code: 200,
      template: 200,
      comments: 200,
    }],
  },
};

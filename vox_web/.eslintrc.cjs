module.exports = {
  root: true,
  env: { browser: true, es2022: true },
  extends: ['plugin:vue/vue3-essential'],
  parser: 'vue-eslint-parser',
  parserOptions: {
    parser: '@typescript-eslint/parser',
    ecmaVersion: 'latest',
    sourceType: 'module',
  },
  rules: {},
}

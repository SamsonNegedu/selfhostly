module.exports = {
  root: true,
  env: { browser: true, es2020: true },
  extends: ['eslint:recommended', 'plugin:@typescript-eslint/recommended', 'plugin:react-hooks/recommended'],
  ignorePatterns: ['dist', 'node_modules', '.eslintrc.cjs'],
  parser: '@typescript-eslint/parser',
  plugins: ['react-refresh'],
  rules: {
    // Files that export a hook or a context next to a component (ThemeProvider, Toast, Button variants) are
    // deliberate here, so Fast Refresh falling back to a full reload for them is accepted.
    'react-refresh/only-export-components': 'off',
    // Older API hooks use `any` in error callbacks. New code should type them, and this stays off until the
    // old ones are cleaned up.
    '@typescript-eslint/no-explicit-any': 'off',
    // A leading underscore marks a value that is deliberately unused.
    '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_', varsIgnorePattern: '^_', ignoreRestSiblings: true }],
  },
}

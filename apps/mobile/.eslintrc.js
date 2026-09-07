module.exports = {
  extends: ['expo'],
  rules: {
    '@typescript-eslint/no-unused-vars': ['warn', { argsIgnorePattern: '^_' }],
    '@typescript-eslint/no-explicit-any': 'warn',
  },
  overrides: [
    {
      files: ['src/services/**/*.ts'],
      rules: {
        'no-restricted-syntax': [
          'error',
          {
            selector: "CallExpression[callee.name='fetch']",
            message:
              'Use fetchOrNetworkError from ../utils/http instead of bare fetch, so a dropped connection surfaces as NetworkError rather than a bare TypeError.',
          },
        ],
      },
    },
  ],
};

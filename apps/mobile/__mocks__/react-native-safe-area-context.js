/* eslint-env jest */
const insets = { top: 0, right: 0, bottom: 0, left: 0 };
const frame = { width: 390, height: 844, x: 0, y: 0 };

module.exports = {
  useSafeAreaInsets: jest.fn(() => insets),
  useSafeAreaFrame: jest.fn(() => frame),
  SafeAreaProvider: ({ children }) => children,
  SafeAreaView: ({ children }) => children,
  SafeAreaConsumer: ({ children }) => children(insets),
  initialWindowMetrics: { insets, frame },
};

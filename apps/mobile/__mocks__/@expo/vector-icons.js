const React = require('react');
const { Text } = require('react-native');

const createIconSet = () => {
  const Icon = ({ name, size, color, testID }) =>
    React.createElement(Text, { testID }, name);
  Icon.glyphMap = new Proxy({}, { get: () => 0 });
  return Icon;
};

const Ionicons = createIconSet();

module.exports = {
  Ionicons,
  MaterialIcons: createIconSet(),
  AntDesign: createIconSet(),
  Feather: createIconSet(),
  FontAwesome: createIconSet(),
  FontAwesome5: createIconSet(),
  createIconSet,
};

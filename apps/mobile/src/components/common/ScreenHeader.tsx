import React, { ReactNode, useState } from 'react';
import { View, Text, TouchableOpacity, StyleSheet, useColorScheme } from 'react-native';
import { Svg, Path } from 'react-native-svg';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { GlobalStyles } from '../../constants/globalStyles';
import { Colors, Spacing } from '../../constants/theme';
import { HeaderSearch, HeaderSearchProps } from './HeaderSearch';

interface ScreenHeaderProps {
  title: string;
  onBack?: () => void;
  rightActions?: ReactNode;
  /** Adds the header search; its open pill covers the title. */
  search?: HeaderSearchProps;
}

const BackButton: React.FC<{ onPress: () => void; color: string }> = ({ onPress, color }) => (
  <TouchableOpacity style={styles.backButton} onPress={onPress} activeOpacity={0.7} accessibilityRole="button" accessibilityLabel="Back">
    <Svg width={16} height={13} viewBox="0 0 17.5 14.5" fill="none">
      <Path
        d="M7.41667 0.75L0.75 7.25L7.41667 13.75M0.75 7.25H16.75"
        stroke={color}
        strokeWidth={1.5}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </Svg>
  </TouchableOpacity>
);

export const ScreenHeader: React.FC<ScreenHeaderProps> = ({
  title,
  onBack,
  rightActions,
  search,
}) => {
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;
  const insets = useSafeAreaInsets();
  // Where the title starts within the header; the open search pill grows to it.
  const [titleX, setTitleX] = useState(0);

  return (
    <View
      style={[
        GlobalStyles.header,
        {
          backgroundColor: colors.background,
          paddingTop: insets.top + Spacing.md,
          height: undefined,
        },
      ]}
    >
      <View style={GlobalStyles.headerLeft}>
        {onBack && <BackButton onPress={onBack} color={colors.text} />}
        <Text
          style={[GlobalStyles.headerTitle, { color: colors.text }]}
          onLayout={(e) => setTitleX(e.nativeEvent.layout.x)}
        >
          {title}
        </Text>
      </View>
      {search && (
        <HeaderSearch
          {...search}
          leftEdge={Spacing.md + titleX}
          style={rightActions ? styles.searchBeforeActions : undefined}
        />
      )}
      {rightActions && (
        <View style={GlobalStyles.headerRight}>{rightActions}</View>
      )}
    </View>
  );
};

const styles = StyleSheet.create({
  searchBeforeActions: {
    marginRight: Spacing.sm,
  },
  backButton: {
    width: 42,
    height: 42,
    borderRadius: 21,
    alignItems: 'center',
    justifyContent: 'center',
  },
});

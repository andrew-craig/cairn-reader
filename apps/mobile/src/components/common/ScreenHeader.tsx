import React, { ReactNode } from 'react';
import { View, Text, useColorScheme } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useNavigation } from '@react-navigation/native';
import { IconButton } from './IconButton';
import { GlobalStyles } from '../../constants/globalStyles';
import { Colors, Spacing } from '../../constants/theme';

interface ScreenHeaderProps {
  title: string;
  onBack?: () => void;
  rightActions?: ReactNode;
}

export const ScreenHeader: React.FC<ScreenHeaderProps> = ({
  title,
  onBack,
  rightActions,
}) => {
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;
  const insets = useSafeAreaInsets();
  const navigation = useNavigation();

  const handleBack = onBack ?? (() => navigation.goBack());

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
        <IconButton icon="chevron-back" onPress={handleBack} />
        <Text style={[GlobalStyles.headerTitle, { color: colors.text }]}>{title}</Text>
      </View>
      {rightActions && (
        <View style={GlobalStyles.headerRight}>{rightActions}</View>
      )}
    </View>
  );
};

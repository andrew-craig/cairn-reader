import React from 'react';
import { View, Text, TouchableOpacity, StyleSheet, useColorScheme } from 'react-native';
import { ContentList } from '@cairn/shared';
import { Colors, Spacing, FontSizes, BorderRadius, FontFamily } from '../../constants';

const LIST_OPTIONS: Array<{ list: ContentList; label: string }> = [
  { list: 'feed', label: 'Feed' },
  { list: 'reads', label: 'Reads' },
];

interface ListToggleProps {
  /** What is being routed, for accessibility labels: "Send {subject} to Feed". */
  subject: string;
  list: ContentList;
  onChange: (list: ContentList) => void;
  disabled?: boolean;
}

/** Two-way Feed | Reads picker. */
export const ListToggle: React.FC<ListToggleProps> = ({ subject, list, onChange, disabled = false }) => {
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;

  return (
    <View style={[styles.toggle, { borderColor: colors.border }]}>
      {LIST_OPTIONS.map((option) => {
        const selected = option.list === list;
        return (
          <TouchableOpacity
            key={option.list}
            style={[styles.option, selected && { backgroundColor: colors.hover }]}
            onPress={() => !selected && onChange(option.list)}
            disabled={disabled}
            activeOpacity={0.7}
            accessibilityRole="button"
            accessibilityLabel={`Send ${subject} to ${option.label}`}
            accessibilityState={{ selected, disabled }}
          >
            <Text
              style={[
                styles.label,
                { color: selected ? colors.text : colors.textSecondary },
                selected && { fontFamily: FontFamily.defaultBold },
              ]}
            >
              {option.label}
            </Text>
          </TouchableOpacity>
        );
      })}
    </View>
  );
};

const styles = StyleSheet.create({
  toggle: {
    flexDirection: 'row',
    alignSelf: 'flex-start',
    borderWidth: 1,
    borderRadius: BorderRadius.full,
    overflow: 'hidden',
  },
  option: {
    paddingHorizontal: Spacing.md,
    paddingVertical: Spacing.xs,
  },
  label: {
    fontSize: FontSizes.sm,
    fontFamily: FontFamily.default,
  },
});

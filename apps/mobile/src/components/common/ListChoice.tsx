import React from 'react';
import { View, Text, TouchableOpacity, StyleSheet, useColorScheme } from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import { ContentList } from '@cairn/shared';
import { Colors, Spacing, FontSizes, BorderRadius, FontFamily } from '../../constants';

const LIST_OPTIONS: { list: ContentList; label: string; description: string }[] = [
  {
    list: 'feed',
    label: 'Feed',
    description: 'Skim in an endless scroll. Items are deleted after 30 days unless you favorite them.',
  },
  {
    list: 'reads',
    label: 'Reads',
    description: 'Read or triage each item. Items are kept and available offline.',
  },
];

interface ListChoiceProps {
  list: ContentList;
  onChange: (list: ContentList) => void;
  disabled?: boolean;
}

/** Explained Feed / Reads choice for a new subscription. Rows use ListToggle instead. */
export const ListChoice: React.FC<ListChoiceProps> = ({ list, onChange, disabled = false }) => {
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;

  return (
    <View style={styles.container} accessibilityRole="radiogroup">
      <Text style={[styles.heading, { color: colors.text }]}>Where should new items go?</Text>
      {LIST_OPTIONS.map((option) => {
        const selected = option.list === list;
        return (
          <TouchableOpacity
            key={option.list}
            style={[
              styles.option,
              { backgroundColor: colors.background, borderColor: selected ? colors.text : 'transparent' },
            ]}
            onPress={() => !selected && onChange(option.list)}
            disabled={disabled}
            activeOpacity={0.7}
            accessibilityRole="radio"
            accessibilityLabel={`${option.label}: ${option.description}`}
            accessibilityState={{ checked: selected, disabled }}
          >
            <Ionicons
              name={selected ? 'radio-button-on' : 'radio-button-off'}
              size={22}
              color={selected ? colors.text : colors.textSecondary}
            />
            <View style={styles.optionText}>
              <Text style={[styles.label, { color: colors.text }]}>{option.label}</Text>
              <Text style={[styles.description, { color: colors.textSecondary }]}>{option.description}</Text>
            </View>
          </TouchableOpacity>
        );
      })}
      <Text style={[styles.note, { color: colors.textSecondary }]}>
        You can change this later on the RSS page.
      </Text>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    gap: Spacing.sm,
  },
  heading: {
    fontSize: FontSizes.md,
    fontFamily: FontFamily.defaultSemiBold,
  },
  option: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    gap: Spacing.sm,
    padding: Spacing.md,
    borderWidth: 2,
    borderRadius: BorderRadius.md,
  },
  optionText: {
    flex: 1,
    gap: Spacing.xs,
  },
  label: {
    fontSize: FontSizes.md,
    fontFamily: FontFamily.defaultBold,
  },
  description: {
    fontSize: FontSizes.sm,
    fontFamily: FontFamily.default,
  },
  note: {
    fontSize: FontSizes.sm,
    fontFamily: FontFamily.default,
  },
});

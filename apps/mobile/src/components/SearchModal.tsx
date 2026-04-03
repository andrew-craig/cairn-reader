import React, { useState } from 'react';
import {
  Modal,
  View,
  TextInput,
  TouchableOpacity,
  StyleSheet,
  useColorScheme,
  KeyboardAvoidingView,
  Platform,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { Colors, Spacing, FontSizes, BorderRadius, FontFamily, Layout } from '../constants';

interface SearchModalProps {
  visible: boolean;
  onClose: () => void;
  onSearch: (query: string) => void;
  placeholder?: string;
}

export const SearchModal: React.FC<SearchModalProps> = ({
  visible,
  onClose,
  onSearch,
  placeholder = 'Search articles',
}) => {
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;
  const insets = useSafeAreaInsets();

  const modalTopPosition = insets.top + Layout.headerHeight;

  const [query, setQuery] = useState('');

  const handleSubmit = () => {
    const trimmed = query.trim();
    if (trimmed) {
      onSearch(trimmed);
      setQuery('');
      onClose();
    }
  };

  const handleClose = () => {
    setQuery('');
    onClose();
  };

  return (
    <Modal
      visible={visible}
      transparent
      animationType="fade"
      onRequestClose={handleClose}
    >
      <TouchableOpacity
        style={[styles.overlay, { paddingTop: modalTopPosition }]}
        activeOpacity={1}
        onPress={handleClose}
      >
        <KeyboardAvoidingView
          behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
          style={styles.keyboardView}
        >
          <TouchableOpacity
            activeOpacity={1}
            style={[
              styles.modalContent,
              { backgroundColor: colors.hover }
            ]}
            onPress={(e) => e.stopPropagation()}
          >
            <View style={styles.inputContainer}>
              <TextInput
                style={[
                  styles.input,
                  {
                    backgroundColor: colors.background,
                    color: colors.text,
                  }
                ]}
                placeholder={placeholder}
                placeholderTextColor={colors.textSecondary}
                value={query}
                onChangeText={setQuery}
                autoCapitalize="none"
                autoCorrect={false}
                autoFocus
                returnKeyType="search"
                onSubmitEditing={handleSubmit}
              />
            </View>
          </TouchableOpacity>
        </KeyboardAvoidingView>
      </TouchableOpacity>
    </Modal>
  );
};

const styles = StyleSheet.create({
  overlay: {
    flex: 1,
    backgroundColor: 'rgba(255, 255, 255, 0.2)',
    justifyContent: 'flex-start',
    paddingHorizontal: Spacing.md,
  },
  keyboardView: {
    width: '100%',
  },
  modalContent: {
    borderRadius: 21,
    padding: Spacing.md,
    shadowColor: 'rgba(15, 23, 42, 0.2)',
    shadowOffset: {
      width: 0,
      height: 2,
    },
    shadowOpacity: 1,
    shadowRadius: 4,
    elevation: 4,
  },
  inputContainer: {
    width: '100%',
  },
  input: {
    height: 48,
    borderRadius: BorderRadius.md,
    borderWidth: 0,
    paddingHorizontal: Spacing.md,
    fontSize: FontSizes.md,
    fontFamily: FontFamily.default,
  },
});

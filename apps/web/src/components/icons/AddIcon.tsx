interface AddIconProps {
  size?: number;
}

// Ionicons `add-outline`, the icon mobile's Reads header uses.
// Colour follows `currentColor`.
export default function AddIcon({ size = 24 }: AddIconProps) {
  return (
    <svg
      aria-hidden="true"
      width={size}
      height={size}
      viewBox="0 0 512 512"
      fill="none"
      stroke="currentColor"
      strokeWidth="32"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="M256 112v288M400 256H112" />
    </svg>
  );
}

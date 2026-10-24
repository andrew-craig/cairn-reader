interface SearchIconProps {
  size?: number;
}

// Ionicons `search-outline`, the icon mobile's Reads header uses.
// Colour follows `currentColor`.
export default function SearchIcon({ size = 24 }: SearchIconProps) {
  return (
    <svg
      aria-hidden="true"
      width={size}
      height={size}
      viewBox="0 0 512 512"
      fill="none"
      stroke="currentColor"
      strokeWidth="32"
      strokeMiterlimit="10"
    >
      <path d="M221.09 64a157.09 157.09 0 10157.09 157.09A157.1 157.1 0 00221.09 64z" />
      <path strokeLinecap="round" d="M338.29 338.29L448 448" />
    </svg>
  );
}

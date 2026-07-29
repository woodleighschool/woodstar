export const PACKAGE_FORM_TAB_SEARCH_VALUES = [
  "contents",
  "requirements",
  "installation",
  "uninstall",
  "scripts",
  "alerts",
  "advanced",
] as const;

export const PACKAGE_FORM_TAB_VALUES = ["basic", ...PACKAGE_FORM_TAB_SEARCH_VALUES] as const;

export function packageFormTabSearchValue(value: string) {
  return PACKAGE_FORM_TAB_SEARCH_VALUES.find((tab) => tab === value);
}

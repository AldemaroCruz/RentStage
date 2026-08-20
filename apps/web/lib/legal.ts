export const DEFAULT_SUPPORT_EMAIL = "support@rentstage.local";

export function legalContactEmail(value = process.env.NEXT_PUBLIC_SUPPORT_EMAIL): string {
  const normalized = value?.trim().toLowerCase();
  return normalized && normalized.includes("@") ? normalized : DEFAULT_SUPPORT_EMAIL;
}

export function isPlaceholderLegalContact(value: string): boolean {
  return value.endsWith("@rentstage.local");
}

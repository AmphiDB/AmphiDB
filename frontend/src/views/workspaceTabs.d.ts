export function filterTabsForDatabase<T extends { database: string }>(
  items: T[],
  database: string | null | undefined
): T[];

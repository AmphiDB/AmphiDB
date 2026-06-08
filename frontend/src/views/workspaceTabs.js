export const filterTabsForDatabase = (items, database) => {
  if (!database) return [];
  return items.filter(item => item.database === database);
};

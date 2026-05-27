/**
 * Normalized data store (plan.md §18).
 * type Normalized<T> = { entities: Record<string, T>; ids: string[] }
 * O(1) updates, prevents unnecessary re-renders.
 */

export interface Normalized<T> {
  entities: Record<string, T>;
  ids: string[];
}

export function createNormalized<T>(): Normalized<T> {
  return { entities: {}, ids: [] };
}

export function normalize<T extends { id: number | string }>(
  items: T[]
): Normalized<T> {
  const entities: Record<string, T> = {};
  const ids: string[] = [];

  for (const item of items) {
    const key = String(item.id);
    entities[key] = item;
    ids.push(key);
  }

  return { entities, ids };
}

export function denormalize<T>(normalized: Normalized<T>): T[] {
  return normalized.ids
    .map((id) => normalized.entities[id])
    .filter(Boolean);
}

export function upsertEntity<T extends { id: number | string }>(
  state: Normalized<T>,
  entity: T
): Normalized<T> {
  const key = String(entity.id);
  const isNew = !(key in state.entities);

  return {
    entities: { ...state.entities, [key]: entity },
    ids: isNew ? [...state.ids, key] : state.ids,
  };
}

export function removeEntity<T>(
  state: Normalized<T>,
  id: string
): Normalized<T> {
  const { [id]: _, ...rest } = state.entities;
  return {
    entities: rest,
    ids: state.ids.filter((i) => i !== id),
  };
}

/**
 * Merge with timestamp check (plan.md §16).
 * Only update if incoming entity is newer.
 */
export function mergeIfNewer<T extends { id: number | string; updated_at?: string }>(
  state: Normalized<T>,
  incoming: T
): Normalized<T> {
  const key = String(incoming.id);
  const existing = state.entities[key];

  if (existing && existing.updated_at && incoming.updated_at) {
    if (new Date(incoming.updated_at) < new Date(existing.updated_at)) {
      // Stale event → ignore
      return state;
    }
  }

  return upsertEntity(state, incoming);
}

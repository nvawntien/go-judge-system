/**
 * Cross-source data resolver (plan.md §26).
 *
 * Priority order for conflict resolution:
 *   1. Local optimistic state (if newer)
 *   2. WebSocket event
 *   3. API response
 *
 * All resolved using updatedAt timestamp — never overwrite newer data with older source.
 */

export type DataSource = "LOCAL_OPTIMISTIC" | "WEBSOCKET" | "API";

const SOURCE_PRIORITY: Record<DataSource, number> = {
  LOCAL_OPTIMISTIC: 3,  // Highest
  WEBSOCKET: 2,
  API: 1,               // Lowest
};

interface Timestamped {
  updated_at?: string;
}

interface SourcedData<T> {
  data: T;
  source: DataSource;
  receivedAt: number;
}

/**
 * Resolve which data should win given two competing updates.
 * Returns the winner.
 */
export function resolveConflict<T extends Timestamped>(
  current: SourcedData<T> | null,
  incoming: SourcedData<T>
): SourcedData<T> {
  if (!current) return incoming;

  // Rule 1: If both have timestamps, compare them
  if (current.data.updated_at && incoming.data.updated_at) {
    const currentTime = new Date(current.data.updated_at).getTime();
    const incomingTime = new Date(incoming.data.updated_at).getTime();

    if (incomingTime > currentTime) {
      return incoming; // Incoming is newer
    }
    if (incomingTime < currentTime) {
      return current; // Current is newer — ignore incoming
    }
    // Equal timestamps: fall through to source priority
  }

  // Rule 2: If timestamps are equal or missing, use source priority
  if (SOURCE_PRIORITY[incoming.source] >= SOURCE_PRIORITY[current.source]) {
    return incoming;
  }

  return current;
}

/**
 * Merge fields from incoming into current without overwriting entire object.
 * Only updates fields present in the incoming payload (plan.md §16).
 * Preserves local optimistic fields not in the incoming event.
 */
export function partialMerge<T extends Record<string, unknown>>(
  current: T,
  incoming: Partial<T>
): T {
  const merged = { ...current };
  for (const key of Object.keys(incoming) as Array<keyof T>) {
    if (incoming[key] !== undefined) {
      merged[key] = incoming[key] as T[keyof T];
    }
  }
  return merged;
}

/**
 * Offline action queue (plan.md §24).
 * Queues mutations when offline, replays on reconnect.
 */

export interface PendingAction {
  id: string;
  type: "SUBMIT";
  payload: unknown;
  createdAt: string;
}

const STORAGE_KEY = "oj-offline-queue";

function loadQueue(): PendingAction[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    return raw ? JSON.parse(raw) : [];
  } catch {
    return [];
  }
}

function saveQueue(queue: PendingAction[]): void {
  if (typeof window === "undefined") return;
  localStorage.setItem(STORAGE_KEY, JSON.stringify(queue));
}

export function enqueue(action: Omit<PendingAction, "id" | "createdAt">): PendingAction {
  const pending: PendingAction = {
    ...action,
    id: crypto.randomUUID(),
    createdAt: new Date().toISOString(),
  };

  const queue = loadQueue();
  queue.push(pending);
  saveQueue(queue);
  return pending;
}

export function dequeue(id: string): void {
  const queue = loadQueue();
  saveQueue(queue.filter((a) => a.id !== id));
}

export function peekAll(): PendingAction[] {
  return loadQueue();
}

export function clearQueue(): void {
  if (typeof window === "undefined") return;
  localStorage.removeItem(STORAGE_KEY);
}

/**
 * Replay all pending actions using a provided executor.
 * Removes successfully executed actions from the queue.
 */
export async function replayQueue(
  executor: (action: PendingAction) => Promise<void>
): Promise<{ succeeded: number; failed: number }> {
  const queue = loadQueue();
  let succeeded = 0;
  let failed = 0;

  for (const action of queue) {
    try {
      await executor(action);
      dequeue(action.id);
      succeeded++;
    } catch {
      failed++;
    }
  }

  return { succeeded, failed };
}

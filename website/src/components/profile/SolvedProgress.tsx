import { difficultyMeta } from '@/lib/format';

export type DifficultyProgressValue = {
  solved: number;
  total?: number;
};

export type DifficultyProgress = {
  easy: DifficultyProgressValue;
  medium: DifficultyProgressValue;
  hard: DifficultyProgressValue;
};

const DIFFICULTIES = ['easy', 'medium', 'hard'] as const;
const RING_RADIUS = 50;
const RING_CIRCUMFERENCE = 2 * Math.PI * RING_RADIUS;

export function SolvedProgress({
  solved,
  attempted,
}: {
  solved: number;
  attempted: number;
}) {
  const safeSolved = Math.max(0, solved);
  const safeAttempted = Math.max(0, attempted);
  const overallRatio = safeAttempted > 0 ? Math.min(safeSolved / safeAttempted, 1) : 0;
  const overallLabel = safeAttempted > 0
    ? `Overall solved progress: ${safeSolved} of ${safeAttempted} attempted problems.`
    : `Overall solved progress: ${safeSolved} solved; no attempted problems.`;

  return (
    <div className="ac-solved-progress-overall">
      <div className="ac-solved-progress-ring">
        <svg viewBox="0 0 120 120" role="img" aria-label={overallLabel}>
          <circle className="ac-solved-progress-track" cx="60" cy="60" r={RING_RADIUS} />
          {overallRatio > 0 ? (
            <circle
              className="ac-solved-progress-value"
              cx="60"
              cy="60"
              r={RING_RADIUS}
              stroke="var(--accent)"
              strokeDasharray={`${overallRatio * RING_CIRCUMFERENCE} ${RING_CIRCUMFERENCE}`}
            />
          ) : null}
        </svg>
        <div className="ac-solved-progress-center" aria-hidden="true">
          <strong>{safeSolved.toLocaleString()}</strong>
          <span>Solved</span>
        </div>
      </div>
      <span className="ac-solved-progress-caption">Overall progress</span>
      <p className="ac-visually-hidden">{overallLabel}</p>
    </div>
  );
}

export function DifficultyBreakdown({ progress }: { progress?: DifficultyProgress }) {
  return (
    <div className="ac-solved-progress-breakdown">
      <div className="ac-solved-progress-label">Difficulty breakdown</div>
      <div className="ac-solved-progress-difficulties">
        {DIFFICULTIES.map((difficulty) => {
          const meta = difficultyMeta(difficulty);
          const value = progress?.[difficulty];
          return (
            <div key={difficulty} className="ac-solved-progress-difficulty">
              <span><i aria-hidden="true" style={{ background: meta.color }} />{meta.label}</span>
              <strong>{formatDifficultyProgress(value)}</strong>
            </div>
          );
        })}
      </div>
      {!progress && <p>Difficulty data unavailable</p>}
      <p className="ac-visually-hidden">
        {progress
          ? DIFFICULTIES.map((difficulty) => `${difficultyMeta(difficulty).label}: ${formatDifficultyProgress(progress[difficulty])}`).join('. ')
          : 'Difficulty breakdown unavailable.'}
      </p>
    </div>
  );
}

function formatDifficultyProgress(value: DifficultyProgressValue | undefined) {
  if (!value) return '—';
  const solved = Math.max(0, value.solved);
  if (typeof value.total === 'number') return `${solved.toLocaleString()} / ${Math.max(0, value.total).toLocaleString()}`;
  return `${solved.toLocaleString()} solved`;
}

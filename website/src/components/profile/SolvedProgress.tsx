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
const DIFFICULTY_ARC_GAP = 4;

export function SolvedProgress({
  solved,
  attempted,
  difficultyProgress,
}: {
  solved: number;
  attempted: number;
  difficultyProgress?: DifficultyProgress;
}) {
  const safeSolved = Math.max(0, solved);
  const safeAttempted = Math.max(0, attempted);
  const overallRatio = safeAttempted > 0 ? Math.min(safeSolved / safeAttempted, 1) : 0;
  const overallLabel = safeAttempted > 0
    ? `Overall solved progress: ${safeSolved} of ${safeAttempted} attempted problems.`
    : `Overall solved progress: ${safeSolved} solved; no attempted problems.`;
  const difficultyArcs = buildDifficultyArcs(difficultyProgress);
  const ringLabel = difficultyArcs.length > 0
    ? `Solved composition by difficulty. ${DIFFICULTIES.map((difficulty) => {
        const meta = difficultyMeta(difficulty);
        return `${meta.label}: ${difficultyProgress?.[difficulty].solved ?? 0} solved`;
      }).join(', ')}.`
    : overallLabel;

  return (
    <div className="ac-solved-progress">
      <div className="ac-solved-progress-overall">
        <div className="ac-solved-progress-ring">
          <svg viewBox="0 0 120 120" role="img" aria-label={ringLabel}>
            <circle className="ac-solved-progress-track" cx="60" cy="60" r={RING_RADIUS} />
            {difficultyArcs.length > 0 ? difficultyArcs.map((arc) => (
              <circle
                key={arc.difficulty}
                className="ac-solved-progress-value"
                cx="60"
                cy="60"
                r={RING_RADIUS}
                stroke={difficultyMeta(arc.difficulty).color}
                strokeDasharray={`${arc.length} ${RING_CIRCUMFERENCE - arc.length}`}
                strokeDashoffset={-arc.offset}
              />
            )) : overallRatio > 0 ? (
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
      </div>

      <div className="ac-solved-progress-breakdown">
        <div className="ac-solved-progress-label">Difficulty breakdown</div>
        <div className="ac-solved-progress-difficulties">
          {DIFFICULTIES.map((difficulty) => {
            const meta = difficultyMeta(difficulty);
            const value = difficultyProgress?.[difficulty];
            return (
              <div key={difficulty} className="ac-solved-progress-difficulty">
                <span><i aria-hidden="true" style={{ background: meta.color }} />{meta.label}</span>
                <strong>{formatDifficultyProgress(value)}</strong>
              </div>
            );
          })}
        </div>
        {!difficultyProgress && <p>Difficulty data unavailable</p>}
      </div>

      <p className="ac-visually-hidden">
        {overallLabel} {difficultyProgress ? ringLabel : 'Difficulty breakdown unavailable.'}
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

function buildDifficultyArcs(progress: DifficultyProgress | undefined) {
  if (!progress) return [];
  const totalSolved = DIFFICULTIES.reduce((total, difficulty) => total + Math.max(0, progress[difficulty].solved), 0);
  if (totalSolved === 0) return [];

  let offset = 0;
  return DIFFICULTIES.flatMap((difficulty) => {
    const solved = Math.max(0, progress[difficulty].solved);
    const segmentLength = (solved / totalSolved) * RING_CIRCUMFERENCE;
    const arc = solved > 0 ? [{
      difficulty,
      length: Math.max(1, segmentLength - DIFFICULTY_ARC_GAP),
      offset,
    }] : [];
    offset += segmentLength;
    return arc;
  });
}

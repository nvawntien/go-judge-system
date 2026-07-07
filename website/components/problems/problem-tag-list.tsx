interface ProblemTagListProps {
  tags: string[];
}

export function ProblemTagList({ tags }: ProblemTagListProps) {
  if (!tags || tags.length === 0) return null;

  const visibleTags = tags.slice(0, 2);
  const remainingCount = tags.length - 2;

  return (
    <div className="flex flex-nowrap items-center gap-1.5 overflow-hidden" title={tags.join(", ")}>
      {visibleTags.map((tag) => (
        <span key={tag} className="inline-flex items-center px-2 py-0.5 rounded text-[11px] bg-slate-50 dark:bg-slate-800/50 border border-slate-100 dark:border-slate-800 text-slate-500 dark:text-slate-400 whitespace-nowrap transition-colors">
          {tag}
        </span>
      ))}
      {remainingCount > 0 && (
        <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[11px] bg-slate-50 dark:bg-slate-800/50 border border-slate-100 dark:border-slate-800 text-slate-500 dark:text-slate-400 whitespace-nowrap transition-colors">
          +{remainingCount}
        </span>
      )}
    </div>
  );
}

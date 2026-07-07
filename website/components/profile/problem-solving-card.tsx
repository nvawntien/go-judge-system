"use client";

import { useState } from "react";
import { ProfileStats, DifficultyProgress } from "@/lib/mocks/profile";

interface ProblemSolvingCardProps {
  stats: ProfileStats;
  difficultyProgress: DifficultyProgress[];
  totalProblemsInPlatform?: number;
}

type HoverState = "default" | "overall" | "Easy" | "Medium" | "Hard";

export function ProblemSolvingCard({
  stats,
  difficultyProgress,
  totalProblemsInPlatform = 660,
}: ProblemSolvingCardProps) {
  const [hoverState, setHoverState] = useState<HoverState>("default");

  const easyStats = difficultyProgress.find(d => d.difficulty === "Easy") || { solved: 0, total: 0 };
  const mediumStats = difficultyProgress.find(d => d.difficulty === "Medium") || { solved: 0, total: 0 };
  const hardStats = difficultyProgress.find(d => d.difficulty === "Hard") || { solved: 0, total: 0 };

  let centerTextTop = "";
  let centerTextBottom = "";
  let ringPercentage = 0;
  let ringColor = "";

  const formatPercent = (solved: number, total: number) => {
    if (total === 0) return "0%";
    return parseFloat(((solved / total) * 100).toFixed(1)) + "%";
  };

  switch (hoverState) {
    case "overall":
      centerTextTop = `${stats.acceptanceRate}%`;
      centerTextBottom = "Acceptance";
      ringPercentage = (stats.solvedProblems / totalProblemsInPlatform) * 100;
      ringColor = "#8b5cf6"; // Violet
      break;
    case "Easy":
      centerTextTop = formatPercent(easyStats.solved, easyStats.total);
      centerTextBottom = "Easy";
      ringPercentage = easyStats.total > 0 ? (easyStats.solved / easyStats.total) * 100 : 0;
      ringColor = "#10b981"; // Emerald
      break;
    case "Medium":
      centerTextTop = formatPercent(mediumStats.solved, mediumStats.total);
      centerTextBottom = "Medium";
      ringPercentage = mediumStats.total > 0 ? (mediumStats.solved / mediumStats.total) * 100 : 0;
      ringColor = "#f59e0b"; // Amber
      break;
    case "Hard":
      centerTextTop = formatPercent(hardStats.solved, hardStats.total);
      centerTextBottom = "Hard";
      ringPercentage = hardStats.total > 0 ? (hardStats.solved / hardStats.total) * 100 : 0;
      ringColor = "#ef4444"; // Red
      break;
    case "default":
    default:
      centerTextTop = `${stats.solvedProblems}`;
      centerTextBottom = "Solved";
      ringPercentage = (stats.solvedProblems / totalProblemsInPlatform) * 100;
      ringColor = "#8b5cf6"; // Violet
      break;
  }

  // Calculate SVG stroke-dasharray properties
  const radius = 38;
  const circumference = 2 * Math.PI * radius;
  const strokeDashoffset = circumference - (ringPercentage / 100) * circumference;

  // Add attempted mock
  const attempted = stats.solvedProblems + 45; // Mock attempted count

  return (
    <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800/80 shadow-sm transition-colors overflow-hidden h-full flex flex-col">
      <div className="p-4 border-b border-slate-100 dark:border-slate-800 transition-colors">
        <h3 className="font-semibold text-slate-900 dark:text-slate-100">Problem Solving</h3>
      </div>

      <div className="p-4 sm:p-5 flex flex-col sm:flex-row items-center justify-center gap-6 sm:gap-4 xl:gap-8 flex-1">
        
        {/* Left: Circular Progress Ring & Attempted */}
        <div className="flex flex-col items-center justify-center">
          <div 
            className="relative flex items-center justify-center w-32 h-32 sm:w-36 sm:h-36 shrink-0 cursor-pointer"
            onMouseEnter={() => setHoverState("overall")}
            onMouseLeave={() => setHoverState("default")}
            onFocus={() => setHoverState("overall")}
            onBlur={() => setHoverState("default")}
            tabIndex={0}
            aria-label="Overall problem solving progress"
          >
            {/* Base Track */}
            <svg className="w-full h-full transform -rotate-90" viewBox="0 0 100 100">
              <circle
                cx="50"
                cy="50"
                r={radius}
                className="stroke-slate-100 dark:stroke-slate-800/60 transition-colors"
                strokeWidth="5"
                fill="transparent"
              />
              {/* Dynamic Animated Progress Segment */}
              <circle
                cx="50"
                cy="50"
                r={radius}
                stroke={ringColor}
                strokeWidth="6"
                fill="transparent"
                strokeLinecap="round"
                strokeDasharray={circumference}
                strokeDashoffset={strokeDashoffset}
                className="transition-all duration-500 ease-out"
              />
            </svg>

            {/* Center Content */}
            <div className="absolute inset-0 flex flex-col items-center justify-center text-center">
              <span className="text-2xl sm:text-[28px] leading-none font-bold text-slate-900 dark:text-slate-100 transition-colors">
                {centerTextTop}
              </span>
              {(hoverState === "default") && (
                <span className="text-[11px] text-slate-400 font-medium">/ {totalProblemsInPlatform}</span>
              )}
              <span className="text-[11px] font-semibold text-slate-500 dark:text-slate-400 mt-1 transition-colors uppercase tracking-wider">
                {centerTextBottom}
              </span>
            </div>
          </div>
          
          <div className="mt-3 text-xs font-medium text-slate-500 dark:text-slate-400">
            <span className="text-slate-700 dark:text-slate-300 font-semibold">{attempted}</span> Attempted
          </div>
        </div>

        {/* Right: Difficulty Breakdown Boxes */}
        <div className="w-full sm:w-[130px] space-y-2.5 shrink-0">
          {/* Easy Box */}
          <div 
            className={`flex flex-col items-center justify-center p-2.5 rounded-lg border transition-all cursor-pointer outline-none focus-visible:ring-2 focus-visible:ring-emerald-500 ${
              hoverState === "Easy" 
                ? "bg-emerald-50 dark:bg-emerald-500/10 border-emerald-200 dark:border-emerald-500/30 shadow-sm" 
                : "bg-slate-50 dark:bg-slate-800/50 border-slate-100 dark:border-slate-800/80 hover:bg-emerald-50/50 dark:hover:bg-emerald-500/5"
            }`}
            onMouseEnter={() => setHoverState("Easy")}
            onMouseLeave={() => setHoverState("default")}
            onFocus={() => setHoverState("Easy")}
            onBlur={() => setHoverState("default")}
          >
            <span className="text-[13px] font-semibold text-emerald-600 dark:text-emerald-400">Easy</span>
            <span className="text-[13px] font-bold text-slate-900 dark:text-slate-100 mt-0.5">
              {easyStats.solved} <span className="text-slate-500 dark:text-slate-400 font-medium">/ {easyStats.total}</span>
            </span>
          </div>

          {/* Medium Box */}
          <div 
            className={`flex flex-col items-center justify-center p-2.5 rounded-lg border transition-all cursor-pointer outline-none focus-visible:ring-2 focus-visible:ring-amber-500 ${
              hoverState === "Medium" 
                ? "bg-amber-50 dark:bg-amber-500/10 border-amber-200 dark:border-amber-500/30 shadow-sm" 
                : "bg-slate-50 dark:bg-slate-800/50 border-slate-100 dark:border-slate-800/80 hover:bg-amber-50/50 dark:hover:bg-amber-500/5"
            }`}
            onMouseEnter={() => setHoverState("Medium")}
            onMouseLeave={() => setHoverState("default")}
            onFocus={() => setHoverState("Medium")}
            onBlur={() => setHoverState("default")}
          >
            <span className="text-[13px] font-semibold text-amber-600 dark:text-amber-400">Medium</span>
            <span className="text-[13px] font-bold text-slate-900 dark:text-slate-100 mt-0.5">
              {mediumStats.solved} <span className="text-slate-500 dark:text-slate-400 font-medium">/ {mediumStats.total}</span>
            </span>
          </div>

          {/* Hard Box */}
          <div 
            className={`flex flex-col items-center justify-center p-2.5 rounded-lg border transition-all cursor-pointer outline-none focus-visible:ring-2 focus-visible:ring-red-500 ${
              hoverState === "Hard" 
                ? "bg-red-50 dark:bg-red-500/10 border-red-200 dark:border-red-500/30 shadow-sm" 
                : "bg-slate-50 dark:bg-slate-800/50 border-slate-100 dark:border-slate-800/80 hover:bg-red-50/50 dark:hover:bg-red-500/5"
            }`}
            onMouseEnter={() => setHoverState("Hard")}
            onMouseLeave={() => setHoverState("default")}
            onFocus={() => setHoverState("Hard")}
            onBlur={() => setHoverState("default")}
          >
            <span className="text-[13px] font-semibold text-red-600 dark:text-red-400">Hard</span>
            <span className="text-[13px] font-bold text-slate-900 dark:text-slate-100 mt-0.5">
              {hardStats.solved} <span className="text-slate-500 dark:text-slate-400 font-medium">/ {hardStats.total}</span>
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}

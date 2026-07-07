export function ProblemTableSkeleton() {
  return (
    <div className="w-full">
      <div className="border-b border-slate-100 pb-3 flex mb-3">
        <div className="w-16"></div>
        <div className="flex-1 px-4">
          <div className="h-4 w-24 bg-slate-100 rounded"></div>
        </div>
        <div className="w-24 px-4 hidden sm:block">
          <div className="h-4 w-16 bg-slate-100 rounded"></div>
        </div>
      </div>
      
      {Array.from({ length: 5 }).map((_, i) => (
        <div key={i} className="flex items-center py-4 border-b border-slate-50 last:border-0">
          <div className="w-16 flex justify-center">
            <div className="h-4 w-4 rounded-full bg-slate-100"></div>
          </div>
          <div className="flex-1 px-4">
            <div className="h-5 w-48 bg-slate-100 rounded mb-2"></div>
            <div className="flex gap-2">
              <div className="h-4 w-16 bg-slate-50 rounded"></div>
              <div className="h-4 w-16 bg-slate-50 rounded"></div>
            </div>
          </div>
          <div className="w-24 px-4 hidden sm:block">
            <div className="h-6 w-16 bg-slate-100 rounded-full"></div>
          </div>
          <div className="w-24 text-right px-4">
            <div className="h-8 w-16 bg-slate-100 rounded-lg ml-auto"></div>
          </div>
        </div>
      ))}
    </div>
  )
}

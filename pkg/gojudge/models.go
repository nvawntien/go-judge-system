package gojudge

// Request represents a request to the go-judge /run endpoint.
type Request struct {
	Cmd []*Cmd `json:"cmd"`
}

// Cmd represents a single command execution step in the sandbox.
type Cmd struct {
	Args  []string `json:"args"`
	Env   []string `json:"env,omitempty"`
	Files []*File  `json:"files,omitempty"`
	
	// File paths mapped from Sandbox to Host
	CopyIn map[string]*File `json:"copyIn,omitempty"`
	
	// Files to copy out
	CopyOut       []string `json:"copyOut,omitempty"`
	CopyOutCached []string `json:"copyOutCached,omitempty"`
	CopyOutDir    string   `json:"copyOutDir,omitempty"`

	CPULimit    uint64 `json:"cpuLimit,omitempty"`    // ns
	ClockLimit  uint64 `json:"clockLimit,omitempty"`  // ns
	MemoryLimit uint64 `json:"memoryLimit,omitempty"` // bytes
	ProcLimit   uint64 `json:"procLimit,omitempty"`
}

type File struct {
	Content *string `json:"content,omitempty"`
	FileID  *string `json:"fileId,omitempty"`
	Symlink *string `json:"symlink,omitempty"`
	// For stdout/stderr capturing
	Name    *string `json:"name,omitempty"`
	Max     *int64  `json:"max,omitempty"`
	Src     *string `json:"src,omitempty"`
}

// Response represents the result from the go-judge /run endpoint.
type Response []Result

// Result represents the execution outcome of a single Cmd block.
type Result struct {
	Status     string            `json:"status"` // Accepted, Memory Limit Exceeded, Time Limit Exceeded, Output Limit Exceeded, File Error, Non Zero Exit Status, Signalled, Internal Error, Run Error
	ExitStatus int               `json:"exitStatus"`
	Error      string            `json:"error,omitempty"`
	Time       uint64            `json:"time"`   // ns
	RunTime    uint64            `json:"runTime"`// ns
	Memory     uint64            `json:"memory"` // bytes
	Files      map[string]string `json:"files,omitempty"` // output files if copyOut is requested
	FileIDs    map[string]string `json:"fileIds,omitempty"`
}

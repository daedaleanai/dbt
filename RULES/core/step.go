package core

// BuildStep represents one build step (i.e., one build command).
// Each BuildStep produces `Out` and `Outs` from `Ins` and `In` by running `Cmd`.
type BuildStep struct {
	Out     OutPath
	Outs    OutPaths
	In      Path
	Ins     Paths
	Depfile OutPath
	Cmd     string
	Script  string
	Descr   string
}

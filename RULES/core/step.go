package core

// BuildStep represents one build step (i.e., one build command).
// Each BuildStep produces `Out` from `Ins` and `In` by running `Cmd`.
type BuildStep struct {
	Out     OutPath
	In      Path
	Ins     Paths
	Depfile *OutPath
	Cmd     string
	Descr   string
}

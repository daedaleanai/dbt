package cc

import "dbt-rules/RULES/core"

// CondToolchainLibrary is a library that can vary based on the toolchain.
// Use:
// var MyLib = CondToolchainLibrary(func(tc cc.Toolchain) Library {
//   .. test features of toolchain and return Library
// })
type CondToolchainLibrary func(tc Toolchain) Library

// CcLibrary returns the toolchain-specific library.
func (ctl CondToolchainLibrary) CcLibrary(tc Toolchain) Library {
	return ctl(tc).CcLibrary(tc)
}

// Build builds the library with the default toolchain.
func (ctl CondToolchainLibrary) Build(ctx core.Context) {
	ctl(DefaultToolchain()).Build(ctx)
}

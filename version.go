package main

var (
	// The commit that the current build.
	Commit string

	// If the current build for a tag, this includes the tag’s name.
	Tag string

	// For builds not triggered by a pull request this is the name of the branch
	// currently being built; whereas for builds triggered by a pull request
	// this is the name of the branch targeted by the pull request
	// (in many cases this will be master).
	Branch string

	// The number of the current build (for example, “4”).
	BuildNumber string
)

func CreateVersionString() string {

	version := "unknown"

	switch {
	case Tag != "":
		version = Tag

	case Commit != "" && Branch != "":
		version = Branch + "_" + Commit
	}

	out := "Version: " + version

	if BuildNumber != "" {
		out = out + " BuildNumber: " + BuildNumber
	}

	return out
}

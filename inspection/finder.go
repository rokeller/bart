package inspection

import (
	"io/fs"
	"os"
)

// Visitor defines the contract for a visitor of a finder.
type Visitor interface {
	VisitDir(string, fs.DirEntry)
	VisitFile(string, fs.DirEntry)
}

// FileFinder defines the contract for an inspector that finds and enumerates/navigates files.
type FileFinder interface {
	Discover(v Visitor)
}

type discoverContext struct {
	Visitor
}

func Discover(basePath string, v Visitor) error {
	rootFS := os.DirFS(basePath)
	ctx := discoverContext{
		Visitor: v,
	}

	return fs.WalkDir(rootFS, ".", ctx.walkDir)
}

func (c *discoverContext) walkDir(path string, d fs.DirEntry, err error) error {
	if d.IsDir() {
		// TODO: check for .bartignore and handle it. return SkipDir error when
		//       this directory is ignored through .bartignore rules
		c.VisitDir(path, d)
	} else {
		// if d.Name() != ".bartignore" {
		c.VisitFile(path, d)
		// }
	}

	return nil
}

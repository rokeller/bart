package inspection

import (
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/rokeller/bart/domain"
)

// Visitor defines the contract for a visitor of a finder.
type Visitor interface {
	Start()
	Visit(ctx Context)
	Done()
}

// FileFinder defines the contract for an inspector that finds and enumerates/navigates files.
type FileFinder interface {
	Find(visitor Visitor)
}

// Context holds contextual information for a found file.
type Context struct {
	Depth int
	Path  string
	Item  os.FileInfo

	visitor Visitor
	owner   *fileFinder
}

type fileFinder struct {
	basePath string
}

// NewFileFinder creates a new FileFinder for the given base path.
func NewFileFinder(basePath string) FileFinder {
	return &fileFinder{
		basePath: basePath,
	}
}

func (f *fileFinder) Find(visitor Visitor) {
	item, err := os.Stat(f.basePath)

	if nil != err {
		log.Panicf("Failed to get file info: %v", err)
	}

	visitor.Start()
	defer visitor.Done()

	f.navigateRoot(item, visitor)
}

func (f *fileFinder) navigateRoot(item os.FileInfo, visitor Visitor) {
	if !item.IsDir() {
		log.Panic("The base path must be a directory.")
	}

	f.navigate(Context{
		Depth:   -1,
		Path:    "",
		Item:    item,
		visitor: visitor,
		owner:   f,
	})
}

func (f *fileFinder) navigate(ctx Context) {
	if ctx.Depth >= 0 {
		ctx.visitor.Visit(ctx)
	}

	if ctx.Item.IsDir() {
		relPath := ""

		if ctx.Depth >= 0 {
			relPath = ctx.RelPath()
		}

		absPath := path.Join(f.basePath, relPath)
		files, err := ioutil.ReadDir(absPath)

		if nil != err {
			log.Printf("Could not read directory: %v", err)
			return
		}

		for _, child := range files {
			f.navigate(Context{
				Depth:   ctx.Depth + 1,
				Path:    relPath,
				Item:    child,
				visitor: ctx.visitor,
				owner:   f,
			})
		}
	}
}

// RelPath returns the relative path of the given item for the backup index.
func (ctx *Context) RelPath() string {
	return path.Join(ctx.Path, ctx.Item.Name())
}

// AbsPath returns the absolute path of the given item.
func (ctx *Context) AbsPath() string {
	return path.Join(ctx.owner.basePath, ctx.RelPath())
}

// Timestamp provides the last modified timestamp of the file item in the context.
func (ctx *Context) Timestamp() int64 {
	return ctx.Item.ModTime().Unix()
}

// Entry creates a new entry for an index from this context.
func (ctx *Context) Entry() domain.Entry {
	return domain.Entry{
		RelPath: ctx.RelPath(),
		EntryMetadata: domain.EntryMetadata{
			Timestamp: ctx.Timestamp(),
		},
	}
}

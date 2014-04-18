// Test rclone by doing real transactions to a storage provider to and
// from the local disk

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path"
	"strings"
	"time"

	"github.com/ncw/rclone/fs"
	"github.com/ogier/pflag"

	// Active file systems
	_ "github.com/ncw/rclone/drive"
	_ "github.com/ncw/rclone/local"
	_ "github.com/ncw/rclone/s3"
	_ "github.com/ncw/rclone/swift"
)

// Globals
var (
	localName, remoteName string
)

// Represents an item for checking
type Item struct {
	Path    string
	Md5sum  string
	ModTime time.Time
	Size    int64
}

// Represents all items for checking
type Items struct {
	byName map[string]*Item
	items  []Item
}

// Make an Items
func NewItems(items []Item) *Items {
	is := &Items{
		byName: make(map[string]*Item),
		items:  items,
	}
	// Fill up byName
	for i := range items {
		is.byName[items[i].Path] = &items[i]
	}
	return is
}

// Check off an item
func (is *Items) Find(obj fs.Object) {
	i, ok := is.byName[obj.Remote()]
	if !ok {
		log.Fatalf("Unexpected file %q", obj.Remote())
	}
	delete(is.byName, obj.Remote())
	// Check attributes
	Md5sum, err := obj.Md5sum()
	if err != nil {
		log.Fatalf("Failed to read md5sum for %q: %v", obj.Remote(), err)
	}
	if i.Md5sum != Md5sum {
		log.Fatalf("%s: Md5sum incorrect - expecting %q got %q", obj.Remote(), i.Md5sum, Md5sum)
	}
	if i.Size != obj.Size() {
		log.Fatalf("%s: Size incorrect - expecting %d got %d", obj.Remote(), i.Size, obj.Size())
	}
	// check the mod time to the given precision
	modTime := obj.ModTime()
	dt := modTime.Sub(i.ModTime)
	if dt >= fs.Config.ModifyWindow || dt <= -fs.Config.ModifyWindow {
		log.Fatalf("%s: Modification time difference too big |%s| > %s (%s vs %s)", obj.Remote(), dt, fs.Config.ModifyWindow, modTime, i.ModTime)
	}

}

// Check all done
func (is *Items) Done() {
	if len(is.byName) != 0 {
		for name := range is.byName {
			log.Printf("Not found %q", name)
		}
		log.Fatalf("%d objects not found", len(is.byName))
	}
}

// Checks the fs to see if it has the expected contents
func CheckListing(f fs.Fs, items []Item) {
	is := NewItems(items)
	for obj := range f.List() {
		is.Find(obj)
	}
	is.Done()
}

// Parse a time string or explode
func Time(timeString string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, timeString)
	if err != nil {
		log.Fatalf("Failed to parse time %q: %v", timeString, err)
	}
	return t
}

// Write a file
func WriteFile(filePath, content string, t time.Time) {
	// FIXME make directories?
	filePath = path.Join(localName, filePath)
	err := ioutil.WriteFile(filePath, []byte(content), 0600)
	if err != nil {
		log.Fatalf("Failed to write file %q: %v", filePath, err)
	}
	err = os.Chtimes(filePath, t, t)
	if err != nil {
		log.Fatalf("Failed to chtimes file %q: %v", filePath, err)
	}
}

// Create a random string
func RandomString(n int) string {
	source := "abcdefghijklmnopqrstuvwxyz0123456789"
	out := make([]byte, n)
	for i := range out {
		out[i] = source[rand.Intn(len(source))]
	}
	return string(out)
}

func TestMkdir(flocal, fremote fs.Fs) {
	err := fs.Mkdir(fremote)
	if err != nil {
		log.Fatalf("Mkdir failed: %v", err)
	}
	items := []Item{}
	CheckListing(flocal, items)
	CheckListing(fremote, items)
}

var t1 = Time("2001-02-03T04:05:06.999999999Z")
var t2 = Time("2011-12-25T12:59:59.123456789Z")

func TestCopy(flocal, fremote fs.Fs) {
	WriteFile("empty", "", t1)

	err := fs.Sync(fremote, flocal, false)
	if err != nil {
		log.Fatalf("Copy failed: %v", err)
	}

	items := []Item{
		{Path: "empty", Size: 0, ModTime: t1, Md5sum: "d41d8cd98f00b204e9800998ecf8427e"},
	}
	CheckListing(flocal, items)
	CheckListing(fremote, items)
}

func TestSync(flocal, fremote fs.Fs) {
	log.Printf("Sync after changing file modtime only")
	err := os.Chtimes(localName+"/empty", t2, t2)
	if err != nil {
		log.Fatalf("Chtimes failed: %v", err)
	}
	err = fs.Sync(fremote, flocal, true)
	if err != nil {
		log.Fatalf("Sync failed: %v", err)
	}
	items := []Item{
		{Path: "empty", Size: 0, ModTime: t2, Md5sum: "d41d8cd98f00b204e9800998ecf8427e"},
	}
	CheckListing(flocal, items)
	CheckListing(fremote, items)

	// ------------------------------------------------------------

	log.Printf("Sync after adding a file")
	WriteFile("potato", "------------------------------------------------------------", t1)
	err = fs.Sync(fremote, flocal, true)
	if err != nil {
		log.Fatalf("Sync failed: %v", err)
	}
	items = []Item{
		{Path: "empty", Size: 0, ModTime: t2, Md5sum: "d41d8cd98f00b204e9800998ecf8427e"},
		{Path: "potato", Size: 60, ModTime: t1, Md5sum: "d6548b156ea68a4e003e786df99eee76"},
	}
	CheckListing(flocal, items)
	CheckListing(fremote, items)

	// ------------------------------------------------------------

	log.Printf("Sync after changing a file's size only")
	WriteFile("potato", "smaller but same date", t1)
	err = fs.Sync(fremote, flocal, true)
	if err != nil {
		log.Fatalf("Sync failed: %v", err)
	}
	items = []Item{
		{Path: "empty", Size: 0, ModTime: t2, Md5sum: "d41d8cd98f00b204e9800998ecf8427e"},
		{Path: "potato", Size: 21, ModTime: t1, Md5sum: "100defcf18c42a1e0dc42a789b107cd2"},
	}
	CheckListing(flocal, items)
	CheckListing(fremote, items)

	// ------------------------------------------------------------

	log.Printf("Sync after removing a file")
	err = os.Remove(localName + "/potato")
	if err != nil {
		log.Fatalf("Remove failed: %v", err)
	}
	err = fs.Sync(fremote, flocal, true)
	if err != nil {
		log.Fatalf("Sync failed: %v", err)
	}

	items = []Item{
		{Path: "empty", Size: 0, ModTime: t2, Md5sum: "d41d8cd98f00b204e9800998ecf8427e"},
	}
	CheckListing(flocal, items)
	CheckListing(fremote, items)

}

func TestLs(flocal, fremote fs.Fs) {
	// Underlying List has been tested above, so we just make sure it runs
	err := fs.List(fremote)
	if err != nil {
		log.Fatalf("List failed: %v", err)
	}
}

func TestLsd(flocal, fremote fs.Fs) {
}

func TestCheck(flocal, fremote fs.Fs) {
}

func TestPurge(flocal, fremote fs.Fs) {
	err := fs.Purge(fremote)
	if err != nil {
		log.Fatalf("Purge failed: %v", err)
	}
}

func TestRmdir(flocal, fremote fs.Fs) {
	err := fs.Rmdir(fremote)
	if err != nil {
		log.Fatalf("Rmdir failed: %v", err)
	}
}

func syntaxError() {
	fmt.Fprintf(os.Stderr, `Test rclone with a remote to find bugs in either.

Syntax: [options] remote:

Need a remote: as argument.  This will create a random container or
directory under it and perform tests on it, deleting it at the end.

Options:

`)
	pflag.PrintDefaults()
}

// Clean the temporary directory
func cleanTempDir() {
	log.Printf("Cleaning temporary directory: %q", localName)
	err := os.RemoveAll(localName)
	if err != nil {
		log.Printf("Failed to remove %q: %v", localName, err)
	}
}

func main() {
	pflag.Usage = syntaxError
	pflag.Parse()
	fs.LoadConfig()
	rand.Seed(time.Now().UnixNano())
	args := pflag.Args()

	if len(args) != 1 {
		syntaxError()
		os.Exit(1)
	}

	remoteName = args[0]
	if !strings.HasSuffix(remoteName, ":") {
		remoteName += "/"
	}
	remoteName += RandomString(32)
	log.Printf("Testing with remote %q", remoteName)
	var err error
	localName, err = ioutil.TempDir("", "rclone")
	if err != nil {
		log.Fatalf("Failed to create temp dir: %v", err)
	}
	log.Printf("Testing with local %q", localName)

	fremote, err := fs.NewFs(remoteName)
	if err != nil {
		log.Fatalf("Failed to make %q: %v", remoteName, err)
	}
	flocal, err := fs.NewFs(localName)
	if err != nil {
		log.Fatalf("Failed to make %q: %v", remoteName, err)
	}

	fs.CalculateModifyWindow(fremote, flocal)

	TestMkdir(flocal, fremote)
	TestCopy(flocal, fremote)
	TestSync(flocal, fremote)
	TestLs(flocal, fremote)
	TestLsd(flocal, fremote)
	TestCheck(flocal, fremote)
	TestPurge(flocal, fremote)
	//TestRmdir(flocal, fremote)

	cleanTempDir()
	log.Printf("Tests OK")
}

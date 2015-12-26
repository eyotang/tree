package tree

import (
	"os"
	"syscall"
	"testing"
	"time"
)

// Mock file/FileInfo
type file struct {
	name    string
	size    int64
	files   []*file
	lastMod time.Time
	stat_t  interface{}
}

func (f *file) Name() string { return f.name }
func (f *file) Size() int64  { return f.size }
func (f *file) Mode() (o os.FileMode) {
	if f.stat_t != nil {
		stat := (f.stat_t).(*syscall.Stat_t)
		o = os.FileMode(stat.Mode)
	}
	return
}
func (f *file) ModTime() time.Time { return f.lastMod }
func (f *file) IsDir() bool        { return nil != f.files }
func (f *file) Sys() interface{} {
	if f.stat_t == nil {
		return new(syscall.Stat_t)
	}
	return f.stat_t
}

// Mock filesystem
type MockFs struct {
	files map[string]*file
}

func NewFs() *MockFs {
	return &MockFs{make(map[string]*file)}
}

func (fs *MockFs) clean() *MockFs {
	fs.files = make(map[string]*file)
	return fs
}

func (fs *MockFs) addFile(path string, file *file) *MockFs {
	fs.files[path] = file
	if file.IsDir() {
		for _, f := range file.files {
			fs.addFile(path+"/"+f.name, f)
		}
	}
	return fs
}

func (fs *MockFs) Stat(path string) (os.FileInfo, error) {
	return fs.files[path], nil
}
func (fs *MockFs) ReadDir(path string) ([]string, error) {
	var names []string
	for _, file := range fs.files[path].files {
		names = append(names, file.Name())
	}
	return names, nil
}

// Mock output file
type Out struct {
	str string
}

func (o *Out) equal(s string) bool {
	return o.str == s
}

func (o *Out) Write(p []byte) (int, error) {
	o.str += string(p)
	return len(p), nil
}

func (o *Out) clear() {
	o.str = ""
}

// FileSystem and Stdout mocks
var (
	fs  = NewFs()
	out = new(Out)
)

type treeTest struct {
	name     string
	opts     *Options
	expected string
}

var listTests = []treeTest{
	{"basic", &Options{Fs: fs, OutFile: out}, `root
├── a
├── b
└── c
    ├── d
    └── e
`}, {"all", &Options{Fs: fs, OutFile: out, All: true}, `root
├── a
├── b
└── c
    ├── d
    ├── e
    └── .f
`}, {"dirs", &Options{Fs: fs, OutFile: out, DirsOnly: true}, `root
└── c
`}, {"fullPath", &Options{Fs: fs, OutFile: out, FullPath: true}, `root
├── root/a
├── root/b
└── root/c
    ├── root/c/d
    └── root/c/e
`}, {"deepLevel", &Options{Fs: fs, OutFile: out, DeepLevel: 1}, `root
├── a
├── b
└── c
`}, {"pattern", &Options{Fs: fs, OutFile: out, Pattern: "(a|e)"}, `root
├── a
└── c
    └── e
`}, {"ipattern", &Options{Fs: fs, OutFile: out, IPattern: "(a|e)"}, `root
├── b
└── c
    └── d
`}, {"ignore-case", &Options{Fs: fs, OutFile: out, Pattern: "(A)", IgnoreCase: true}, `root
├── a
└── c
`}}

// Tests
func TestSimple(t *testing.T) {
	root := &file{
		"root",
		200,
		[]*file{
			&file{"a", 50, nil, time.Now(), nil},
			&file{"b", 50, nil, time.Now(), nil},
			&file{
				"c",
				100,
				[]*file{
					&file{"d", 50, nil, time.Now(), nil},
					&file{"e", 50, nil, time.Now(), nil},
					&file{".f", 0, nil, time.Now(), nil},
				},
				time.Now(),
				nil},
		},
		time.Now(),
		nil,
	}
	fs.clean().addFile(root.name, root)
	for _, test := range listTests {
		inf := New(root.name)
		inf.Visit(test.opts)
		inf.Print("", test.opts)
		if !out.equal(test.expected) {
			t.Errorf("%s:\ngot:\n%+v\nexpected:\n%+v", test.name, out.str, test.expected)
		}
		out.clear()
	}
}

var sortTests = []treeTest{
	{"name-sort", &Options{Fs: fs, OutFile: out, NameSort: true}, `root
├── a
├── b
└── c
    └── d
`}, {"dirs-first sort", &Options{Fs: fs, OutFile: out, DirSort: true}, `root
├── c
│   └── d
├── b
└── a
`}, {"reverse sort", &Options{Fs: fs, OutFile: out, ReverSort: true, DirSort: true}, `root
├── b
├── a
└── c
    └── d
`}, {"no-sort", &Options{Fs: fs, OutFile: out, NoSort: true, DirSort: true}, `root
├── b
├── c
│   └── d
└── a
`}, {"size-sort", &Options{Fs: fs, OutFile: out, SizeSort: true}, `root
├── a
├── c
│   └── d
└── b
`}, {"last-mod-sort", &Options{Fs: fs, OutFile: out, ModSort: true}, `root
├── a
├── b
└── c
    └── d
`}, {"c-time-sort", &Options{Fs: fs, OutFile: out, CTimeSort: true}, `root
├── b
├── c
│   └── d
└── a
`}}

func TestSort(t *testing.T) {

	tFmt := "2006-Jan-02"
	aTime, _ := time.Parse(tFmt, "2015-Aug-01")
	bTime, _ := time.Parse(tFmt, "2015-Sep-01")
	cTime, _ := time.Parse(tFmt, "2015-Oct-01")
	root := &file{
		"root",
		200,
		[]*file{
			&file{"b", 11, nil, bTime, nil},
			&file{"c", 10, []*file{
				&file{"d", 10, nil, cTime, nil},
			}, cTime, nil},
			&file{"a", 9, nil, aTime, nil},
		},
		time.Now(),
		nil,
	}
	fs.clean().addFile(root.name, root)
	for _, test := range sortTests {
		inf := New(root.name)
		inf.Visit(test.opts)
		inf.Print("", test.opts)
		if !out.equal(test.expected) {
			t.Errorf("%s:\ngot:\n%+v\nexpected:\n%+v", test.name, out.str, test.expected)
		}
		out.clear()
	}
}

var graphicTests = []treeTest{
	{"no-indent", &Options{Fs: fs, OutFile: out, NoIndent: true}, `root
a
b
c
`}, {"quotes", &Options{Fs: fs, OutFile: out, Quotes: true}, `"root"
├── "a"
├── "b"
└── "c"
`}, {"byte-size", &Options{Fs: fs, OutFile: out, ByteSize: true}, `root
├── [       1500]  a
├── [       9999]  b
└── [       1000]  c
`}, {"unit-size", &Options{Fs: fs, OutFile: out, UnitSize: true}, `root
├── [1.5K]  a
├── [9.8K]  b
└── [1000]  c
`}, {"show-gid", &Options{Fs: fs, OutFile: out, ShowGid: true}, `root
├── [1   ]  a
├── [2   ]  b
└── [1   ]  c
`}, {"mode", &Options{Fs: fs, OutFile: out, FileMode: true}, `root
├── [-rw-r--r--]  a
├── [-rwxr-xr-x]  b
└── [-rw-rw-rw-]  c
`}, {"lastMod", &Options{Fs: fs, OutFile: out, LastMod: true}, `root
├── [Feb 11 00:00]  a
├── [Jan 28 00:00]  b
└── [Jul 12 00:00]  c
`}}

func TestGraphics(t *testing.T) {
	tFmt := "2006-Jan-02"
	aTime, _ := time.Parse(tFmt, "2015-Feb-11")
	bTime, _ := time.Parse(tFmt, "2006-Jan-28")
	cTime, _ := time.Parse(tFmt, "2015-Jul-12")
	root := &file{
		"root",
		11499,
		[]*file{
			&file{"a", 1500, nil, aTime, &syscall.Stat_t{Gid: 1, Mode: 0644}},
			&file{"b", 9999, nil, bTime, &syscall.Stat_t{Gid: 2, Mode: 0755}},
			&file{"c", 1000, nil, cTime, &syscall.Stat_t{Gid: 1, Mode: 0666}},
		},
		time.Now(),
		&syscall.Stat_t{Gid: 1},
	}
	fs.clean().addFile(root.name, root)
	for _, test := range graphicTests {
		inf := New(root.name)
		inf.Visit(test.opts)
		inf.Print("", test.opts)
		if !out.equal(test.expected) {
			t.Errorf("%s:\ngot:\n%+v\nexpected:\n%+v", test.name, out.str, test.expected)
		}
		out.clear()
	}
}
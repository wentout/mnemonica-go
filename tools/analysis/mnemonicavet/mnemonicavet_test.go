// Command mnemonicavet tests: analysistest against the want-comment
// fixture, plus the singlechecker main wrapper.
package main

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
)

// TestMain links the runtime package into the analysistest GOPATH: the
// fixture imports github.com/wentout/mnemonica-go/mnemonica, and
// analysistest resolves imports from testdata/src only, so the link
// mirrors the import path's directory shape. The link is removed after
// the run.
func TestMain(m *testing.M) {
	link := filepath.Join("testdata", "src", "github.com", "wentout", "mnemonica-go", "mnemonica")
	target, err := filepath.Abs(filepath.Join("..", "..", "..", "mnemonica"))
	if err != nil {
		panic(err)
	}
	if mkdirErr := os.MkdirAll(filepath.Dir(link), 0o755); mkdirErr != nil {
		panic(mkdirErr)
	}
	_ = os.Remove(link)
	if linkErr := os.Symlink(target, link); linkErr != nil {
		panic(linkErr)
	}
	code := m.Run()
	_ = os.Remove(link)
	os.Exit(code)
}

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer, "lineage")
}

// TestMainWrapper covers the singlechecker delegation without running it
// (singlechecker.Main would exit the test binary).
func TestMainWrapper(t *testing.T) {
	var got *analysis.Analyzer
	original := singlecheckerMain
	singlecheckerMain = func(analyzer *analysis.Analyzer) {
		got = analyzer
	}
	t.Cleanup(func() {
		singlecheckerMain = original
	})
	main()
	if got != Analyzer {
		t.Error("main did not delegate the Analyzer to the singlechecker")
	}
}

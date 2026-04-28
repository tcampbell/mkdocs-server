package build

import (
	"bytes"
	"flag"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// update regenerates golden files in testdata/<fixture>/golden/ for every
// fixture run. After updating, inspect with `git diff testdata/` and commit
// when intentional. See testdata/README.md.
var update = flag.Bool("update", false, "update golden files")

func TestGolden(t *testing.T) {
	fixtures, err := fs.ReadDir(os.DirFS("testdata"), ".")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	var names []string
	for _, e := range fixtures {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		t.Fatal("no fixtures found in testdata/")
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			runFixture(t, name)
		})
	}
}

func runFixture(t *testing.T, name string) {
	t.Helper()
	srcFixture := filepath.Join("testdata", name)
	goldenDir := filepath.Join(srcFixture, "golden")

	// Copy the fixture (minus the golden tree) into a temp dir so Build's
	// output does not pollute testdata/.
	workDir := t.TempDir()
	if err := copyTree(srcFixture, workDir, "golden"); err != nil {
		t.Fatalf("copy fixture: %v", err)
	}

	configPath := filepath.Join(workDir, "mkdocs.yml")
	if err := Build(configPath); err != nil {
		t.Fatalf("Build: %v", err)
	}

	siteDir := filepath.Join(workDir, "site")
	got := snapshotSite(t, siteDir)

	if *update {
		writeGolden(t, goldenDir, got)
		return
	}

	want := readGolden(t, goldenDir)
	compareSnapshots(t, want, got)
}

// snapshot is the deterministic representation of a built site that the
// golden tree captures: file contents for HTML and search JSON, and a
// manifest of asset filenames (contents not compared — too much churn for
// minified upstream blobs).
type snapshot struct {
	files    map[string][]byte // relative path → contents (HTML, JSON, copied non-md)
	manifest []string          // sorted list of _assets/** filenames
}

func snapshotSite(t *testing.T, siteDir string) snapshot {
	t.Helper()
	snap := snapshot{files: map[string][]byte{}}
	err := filepath.WalkDir(siteDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(siteDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, "_assets/") {
			snap.manifest = append(snap.manifest, rel)
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		snap.files[rel] = data
		return nil
	})
	if err != nil {
		t.Fatalf("walk site: %v", err)
	}
	sort.Strings(snap.manifest)
	return snap
}

func readGolden(t *testing.T, goldenDir string) snapshot {
	t.Helper()
	snap := snapshot{files: map[string][]byte{}}

	manifestPath := filepath.Join(goldenDir, "_assets.txt")
	if data, err := os.ReadFile(manifestPath); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			if line != "" {
				snap.manifest = append(snap.manifest, line)
			}
		}
	} else if !os.IsNotExist(err) {
		t.Fatalf("read manifest: %v", err)
	}

	err := filepath.WalkDir(goldenDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(goldenDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "_assets.txt" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		snap.files[rel] = data
		return nil
	})
	if err != nil {
		t.Fatalf("walk golden: %v", err)
	}
	return snap
}

func writeGolden(t *testing.T, goldenDir string, snap snapshot) {
	t.Helper()
	if err := os.RemoveAll(goldenDir); err != nil {
		t.Fatalf("clean golden: %v", err)
	}
	for rel, data := range snap.files {
		dst := filepath.Join(goldenDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			t.Fatalf("mkdir golden: %v", err)
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}
	manifest := strings.Join(snap.manifest, "\n") + "\n"
	if err := os.MkdirAll(goldenDir, 0o755); err != nil {
		t.Fatalf("mkdir golden: %v", err)
	}
	if err := os.WriteFile(filepath.Join(goldenDir, "_assets.txt"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	t.Logf("updated %s (%d files, %d assets)", goldenDir, len(snap.files), len(snap.manifest))
}

func compareSnapshots(t *testing.T, want, got snapshot) {
	t.Helper()

	// Asset manifest comparison.
	if !equalSlices(want.manifest, got.manifest) {
		extra, missing := diffSlices(want.manifest, got.manifest)
		t.Errorf("asset manifest mismatch:\n  unexpected: %v\n  missing:    %v\n  run with -update to regenerate", extra, missing)
	}

	// File set comparison.
	for rel := range want.files {
		if _, ok := got.files[rel]; !ok {
			t.Errorf("missing file: %s (run with -update if intentional)", rel)
		}
	}
	for rel, gotData := range got.files {
		wantData, ok := want.files[rel]
		if !ok {
			t.Errorf("unexpected file: %s (run with -update if intentional)", rel)
			continue
		}
		if !bytes.Equal(wantData, gotData) {
			t.Errorf("contents differ: %s\n%s\nrun with -update to regenerate", rel, firstDiff(wantData, gotData))
		}
	}
}

// firstDiff returns a short rendering of the first byte where want and got
// diverge. Enough to tell a reviewer what changed without dumping the whole
// file (which they will see in `git diff` after running -update).
func firstDiff(want, got []byte) string {
	n := len(want)
	if len(got) < n {
		n = len(got)
	}
	for i := 0; i < n; i++ {
		if want[i] != got[i] {
			start := i - 30
			if start < 0 {
				start = 0
			}
			endW := i + 60
			if endW > len(want) {
				endW = len(want)
			}
			endG := i + 60
			if endG > len(got) {
				endG = len(got)
			}
			return "  first diff at byte " + itoa(i) +
				"\n  want: …" + escape(want[start:endW]) + "…" +
				"\n  got:  …" + escape(got[start:endG]) + "…"
		}
	}
	if len(want) != len(got) {
		return "  length differs: want=" + itoa(len(want)) + " got=" + itoa(len(got))
	}
	return ""
}

func escape(b []byte) string {
	s := string(b)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// diffSlices returns (in_b_not_in_a, in_a_not_in_b). Both inputs sorted.
func diffSlices(a, b []string) (extra, missing []string) {
	aset := map[string]bool{}
	for _, x := range a {
		aset[x] = true
	}
	bset := map[string]bool{}
	for _, x := range b {
		bset[x] = true
	}
	for _, x := range b {
		if !aset[x] {
			extra = append(extra, x)
		}
	}
	for _, x := range a {
		if !bset[x] {
			missing = append(missing, x)
		}
	}
	return
}

// copyTree copies src into dst, skipping any top-level entry whose name
// matches skipDir (used to omit the golden tree from the work copy).
func copyTree(src, dst, skipDir string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(dst, 0o755)
		}
		if rel == skipDir {
			return fs.SkipDir
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	})
}

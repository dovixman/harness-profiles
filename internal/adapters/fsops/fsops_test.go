package fsops

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInspectReplaceAndRemoveSymlink(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "target")
	link := filepath.Join(base, "config", "current")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}

	info, err := InspectLink(link)
	if err != nil || info.Exists {
		t.Fatalf("InspectLink(missing) = %+v, %v", info, err)
	}
	if err := ReplaceSymlink(link, target); err != nil {
		t.Fatalf("ReplaceSymlink() = %v", err)
	}
	info, err = InspectLink(link)
	if err != nil || !info.Exists || !info.IsSymlink || info.Target != target {
		t.Fatalf("InspectLink(link) = %+v, %v", info, err)
	}
	if err := RemoveSymlink(link); err != nil {
		t.Fatalf("RemoveSymlink() = %v", err)
	}
	info, err = InspectLink(link)
	if err != nil || info.Exists {
		t.Fatalf("InspectLink(removed) = %+v, %v", info, err)
	}
}

func TestReplaceAndRemoveSymlink_RejectRealPath(t *testing.T) {
	base := t.TempDir()
	path := filepath.Join(base, "real")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := ReplaceSymlink(path, base); err == nil {
		t.Fatal("ReplaceSymlink(real path) = nil, want error")
	}
	if err := RemoveSymlink(path); err == nil {
		t.Fatal("RemoveSymlink(real path) = nil, want error")
	}
}

func TestCopyDir_PreservesAndMaterializesSymlinks(t *testing.T) {
	base := t.TempDir()
	src := filepath.Join(base, "src")
	preserved := filepath.Join(base, "preserved")
	materialized := filepath.Join(base, "materialized")
	if err := os.MkdirAll(filepath.Join(src, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(src, "nested", "real.txt"), "real")
	if err := os.Symlink(filepath.Join("nested", "real.txt"), filepath.Join(src, "link.txt")); err != nil {
		t.Fatal(err)
	}

	if err := CopyDir(src, preserved, false); err != nil {
		t.Fatalf("CopyDir(preserve) = %v", err)
	}
	info, err := os.Lstat(filepath.Join(preserved, "link.txt"))
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("preserved link info = %v, %v", info, err)
	}

	if err := CopyDir(src, materialized, true); err != nil {
		t.Fatalf("CopyDir(materialize) = %v", err)
	}
	info, err = os.Lstat(filepath.Join(materialized, "link.txt"))
	if err != nil || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("materialized link info = %v, %v", info, err)
	}
}

func TestCopyArtifact_CopiesRegularFilesAndDirectories(t *testing.T) {
	base := t.TempDir()
	dirSrc := filepath.Join(base, "dir")
	dirDst := filepath.Join(base, "dir-copy")
	fileSrc := filepath.Join(base, "file.txt")
	fileDst := filepath.Join(base, "file-copy.txt")
	if err := os.MkdirAll(dirSrc, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(dirSrc, "nested.txt"), "nested")
	writeTestFile(t, fileSrc, "content")

	if err := CopyArtifact(dirSrc, dirDst, false); err != nil {
		t.Fatalf("CopyArtifact(dir) = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dirDst, "nested.txt")); err != nil {
		t.Fatalf("copied dir file stat = %v", err)
	}

	if err := CopyArtifact(fileSrc, fileDst, false); err != nil {
		t.Fatalf("CopyArtifact(file) = %v", err)
	}
	data, err := os.ReadFile(fileDst)
	if err != nil {
		t.Fatalf("copied file read = %v", err)
	}
	if string(data) != "content" {
		t.Fatalf("copied file = %q, want content", string(data))
	}
}

func TestCopyArtifact_FollowsSymlinkSourcesForRegularFiles(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "target.txt")
	link := filepath.Join(base, "link.txt")
	if err := os.WriteFile(target, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target.txt", link); err != nil {
		t.Fatal(err)
	}

	if err := CopyArtifact(link, filepath.Join(base, "out.txt"), false); err != nil {
		t.Fatalf("CopyArtifact(symlink) = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(base, "out.txt"))
	if err != nil {
		t.Fatalf("copied symlink target read = %v", err)
	}
	if string(data) != "payload" {
		t.Fatalf("copied symlink target = %q, want payload", string(data))
	}
}

func TestMoveDir_FallsBackToCopyAndDeleteWhenRenameFails(t *testing.T) {
	base := t.TempDir()
	src := filepath.Join(base, "missing-parent", "src")
	dst := filepath.Join(base, "dst")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(src, "file.txt"), "data")
	if err := MoveDir(src, dst); err != nil {
		t.Fatalf("MoveDir() = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "file.txt")); err != nil {
		t.Fatalf("moved file stat = %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("source stat = %v, want not exist", err)
	}
}

func TestMoveArtifact_MovesRegularFiles(t *testing.T) {
	base := t.TempDir()
	src := filepath.Join(base, "source.txt")
	dst := filepath.Join(base, "dst.txt")
	writeTestFile(t, src, "hello")
	if err := MoveArtifact(src, dst); err != nil {
		t.Fatalf("MoveArtifact(file) = %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("source stat = %v, want not exist", err)
	}
	if _, err := os.Stat(dst); err != nil {
		t.Fatalf("destination stat = %v", err)
	}
}

func TestDeletePath_IgnoresMissingPath(t *testing.T) {
	if err := DeletePath(filepath.Join(t.TempDir(), "missing")); err != nil {
		t.Fatalf("DeletePath(missing) = %v", err)
	}
}

func TestOSMethodsDelegateToFilesystemOperations(t *testing.T) {
	base := t.TempDir()
	osfs := OS{}
	root := filepath.Join(base, "root")
	if err := osfs.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("MkdirAll() = %v", err)
	}
	writeTestFile(t, filepath.Join(root, "file.txt"), "data")
	if entries, err := osfs.ReadDir(root); err != nil || len(entries) != 1 {
		t.Fatalf("ReadDir() = %d entries, %v", len(entries), err)
	}
	if info, err := osfs.Stat(root); err != nil || !info.IsDir() {
		t.Fatalf("Stat() = %v, %v", info, err)
	}
	if _, err := osfs.Lstat(filepath.Join(root, "file.txt")); err != nil {
		t.Fatalf("Lstat() = %v", err)
	}
	moved := filepath.Join(base, "moved")
	if err := osfs.Rename(root, moved); err != nil {
		t.Fatalf("Rename() = %v", err)
	}
	link := filepath.Join(base, "link")
	if err := osfs.ReplaceSymlink(link, moved); err != nil {
		t.Fatalf("ReplaceSymlink() = %v", err)
	}
	if info, err := osfs.InspectLink(link); err != nil || !info.IsSymlink {
		t.Fatalf("InspectLink() = %+v, %v", info, err)
	}
	copyDst := filepath.Join(base, "copy")
	if err := osfs.CopyDir(moved, copyDst, false); err != nil {
		t.Fatalf("CopyDir() = %v", err)
	}
	if err := osfs.CopyArtifact(moved, filepath.Join(copyDst, "copy-artifact"), false); err != nil {
		t.Fatalf("CopyArtifact() = %v", err)
	}
	moveDst := filepath.Join(base, "move-copy")
	if err := osfs.MoveDir(copyDst, moveDst); err != nil {
		t.Fatalf("MoveDir() = %v", err)
	}
	if err := osfs.MoveArtifact(filepath.Join(moved, "file.txt"), filepath.Join(moveDst, "moved-file.txt")); err != nil {
		t.Fatalf("MoveArtifact(file) = %v", err)
	}
	if err := osfs.RemoveSymlink(link); err != nil {
		t.Fatalf("RemoveSymlink() = %v", err)
	}
	if err := osfs.DeletePath(moveDst); err != nil {
		t.Fatalf("DeletePath() = %v", err)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

package fsops

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/dovixman/harness-profiles/internal/app"
)

type OS struct{}

func (OS) InspectLink(path string) (app.LinkInfo, error) {
	info, err := InspectLink(path)
	return app.LinkInfo{Exists: info.Exists, IsSymlink: info.IsSymlink, Target: info.Target}, err
}

func (OS) ReadDir(path string) ([]os.DirEntry, error) { return os.ReadDir(path) }

func (OS) Stat(path string) (os.FileInfo, error) { return os.Stat(path) }

func (OS) Lstat(path string) (os.FileInfo, error) { return os.Lstat(path) }

func (OS) MkdirAll(path string, mode os.FileMode) error { return os.MkdirAll(path, mode) }

func (OS) Rename(oldPath, newPath string) error { return os.Rename(oldPath, newPath) }

func (OS) ReplaceSymlink(path, target string) error { return ReplaceSymlink(path, target) }

func (OS) RemoveSymlink(path string) error { return RemoveSymlink(path) }

func (OS) CopyDir(src, dst string, materializeSymlinks bool) error {
	return CopyDir(src, dst, materializeSymlinks)
}

func (OS) CopyArtifact(src, dst string, materializeSymlinks bool) error {
	return CopyArtifact(src, dst, materializeSymlinks)
}

func (OS) MoveDir(src, dst string) error { return MoveDir(src, dst) }

func (OS) MoveArtifact(src, dst string) error { return MoveArtifact(src, dst) }

func (OS) DeletePath(path string) error { return DeletePath(path) }

func (OS) WriteFile(path string, contents []byte) error { return os.WriteFile(path, contents, 0o644) }

type LinkInfo struct {
	Exists    bool
	IsSymlink bool
	Target    string
}

func InspectLink(path string) (LinkInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return LinkInfo{}, nil
		}
		return LinkInfo{}, err
	}

	result := LinkInfo{Exists: true, IsSymlink: info.Mode()&os.ModeSymlink != 0}
	if result.IsSymlink {
		target, err := os.Readlink(path)
		if err != nil {
			return LinkInfo{}, err
		}
		result.Target = target
	}
	return result, nil
}

func ReplaceSymlink(path, target string) error {
	info, err := InspectLink(path)
	if err != nil {
		return err
	}
	if info.Exists && !info.IsSymlink {
		return fmt.Errorf("%s exists and is not a symlink", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	_ = os.Remove(tmp)
	if err := os.Symlink(target, tmp); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func RemoveSymlink(path string) error {
	info, err := InspectLink(path)
	if err != nil {
		return err
	}
	if !info.Exists {
		return nil
	}
	if !info.IsSymlink {
		return fmt.Errorf("%s exists and is not a symlink", path)
	}
	return os.Remove(path)
}

func CopyDir(src, dst string, materializeSymlinks bool) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", src)
	}
	if _, err := os.Lstat(dst); err == nil {
		return fmt.Errorf("%s already exists", dst)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return filepath.WalkDir(src, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if rel == "." {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		return copyDirEntry(path, target, entry, materializeSymlinks)
	})
}

func copyDirEntry(path, target string, entry os.DirEntry, materializeSymlinks bool) error {
	entryInfo, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if entryInfo.Mode()&os.ModeSymlink != 0 {
		return copySymlink(path, target, materializeSymlinks)
	}
	if entry.IsDir() {
		return os.MkdirAll(target, entryInfo.Mode().Perm())
	}
	return copyFile(path, target, entryInfo.Mode().Perm())
}

func copySymlink(path, target string, materialize bool) error {
	linkTarget, err := os.Readlink(path)
	if err != nil {
		return err
	}
	if !materialize {
		return os.Symlink(linkTarget, target)
	}
	realPath := linkTarget
	if !filepath.IsAbs(realPath) {
		realPath = filepath.Join(filepath.Dir(path), realPath)
	}
	realInfo, err := os.Stat(realPath)
	if err != nil {
		return err
	}
	if realInfo.IsDir() {
		return CopyDir(realPath, target, true)
	}
	return copyFile(realPath, target, realInfo.Mode().Perm())
}

func CopyArtifact(src, dst string, materializeSymlinks bool) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return CopyDir(src, dst, materializeSymlinks)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s is neither a directory nor regular file", src)
	}
	return copyFile(src, dst, info.Mode().Perm())
}

func MoveDir(src, dst string) error {
	if _, err := os.Lstat(dst); err == nil {
		return fmt.Errorf("%s already exists", dst)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if err := CopyDir(src, dst, false); err != nil {
		return err
	}
	return os.RemoveAll(src)
}

func MoveArtifact(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return MoveDir(src, dst)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s is neither a directory nor regular file", src)
	}
	if _, err := os.Lstat(dst); err == nil {
		return fmt.Errorf("%s already exists", dst)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if err := copyFile(src, dst, info.Mode().Perm()); err != nil {
		return err
	}
	return os.Remove(src)
}

func DeletePath(path string) error {
	if _, err := os.Lstat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return os.RemoveAll(path)
}

func copyFile(src, dst string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

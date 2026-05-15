package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

var fixedTime = time.Unix(0, 0).UTC()

const (
	FormatZip   = "zip"
	FormatTarGz = "tar.gz"
)

type Item struct {
	Source string
	Name   string
	Mode   os.FileMode
}

func Create(path string, format string, items []Item) error {
	switch format {
	case FormatZip:
		return createZip(path, items)
	case FormatTarGz:
		return createTarGz(path, items)
	default:
		return fmt.Errorf("unsupported archive format: %s", format)
	}
}

func createZip(path string, items []Item) (err error) {
	items = sortedItems(items)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer func() {
		if closeErr := zw.Close(); err == nil {
			err = closeErr
		}
	}()

	for _, item := range items {
		if err := writeZipItem(zw, item); err != nil {
			return err
		}
	}

	return nil
}

func createTarGz(path string, items []Item) (err error) {
	items = sortedItems(items)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewWriterLevel(f, gzip.BestCompression)
	if err != nil {
		return err
	}
	gz.Header.ModTime = fixedTime
	gz.Header.OS = 255

	tw := tar.NewWriter(gz)
	defer func() {
		if closeErr := tw.Close(); err == nil {
			err = closeErr
		}
		if closeErr := gz.Close(); err == nil {
			err = closeErr
		}
	}()

	for _, item := range items {
		if err := writeTarItem(tw, item); err != nil {
			return err
		}
	}

	return nil
}

func openItem(path string) (*os.File, os.FileInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, nil, err
	}
	return f, info, nil
}

func writeZipItem(zw *zip.Writer, item Item) (err error) {
	src, info, err := openItem(item.Source)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := src.Close(); err == nil {
			err = closeErr
		}
	}()

	hdr, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	hdr.Name = item.Name
	hdr.Method = zip.Deflate
	hdr.SetMode(item.Mode)
	hdr.SetModTime(fixedTime)

	w, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	if _, err := io.Copy(w, src); err != nil {
		return err
	}
	return nil
}

func writeTarItem(tw *tar.Writer, item Item) (err error) {
	src, info, err := openItem(item.Source)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := src.Close(); err == nil {
			err = closeErr
		}
	}()

	hdr := &tar.Header{
		Name:     item.Name,
		Mode:     int64(item.Mode),
		Size:     info.Size(),
		ModTime:  fixedTime,
		Typeflag: tar.TypeReg,
		Uid:      0,
		Gid:      0,
		Uname:    "",
		Gname:    "",
		Format:   tar.FormatUSTAR,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	if _, err := io.Copy(tw, src); err != nil {
		return err
	}
	return nil
}

func sortedItems(items []Item) []Item {
	out := append([]Item(nil), items...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func WriteAtomic(path string, format string, items []Item) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	if err := os.Remove(tmpPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := Create(tmpPath, format, items); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

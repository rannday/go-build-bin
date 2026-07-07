package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rannday/go-build-bin/internal/atomicfile"
)

type Format string

const (
	FormatZip   Format = "zip"
	FormatTarGz Format = "tar.gz"
)

func (f Format) String() string {
	return string(f)
}

type Item struct {
	Name string
	Path string
}

var fixedArchiveTime = time.Unix(0, 0).UTC()

func WriteAtomic(path string, format Format, items []Item) error {
	return atomicfile.Write(path, func(tmpPath string) error {
		switch format {
		case FormatZip:
			return writeZip(tmpPath, items)
		case FormatTarGz:
			return writeTarGz(tmpPath, items)
		default:
			return fmt.Errorf("unsupported archive format: %s", format)
		}
	})
}

func writeZip(path string, items []Item) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	zw := zip.NewWriter(file)
	for _, item := range items {
		if err := writeZipItem(zw, item); err != nil {
			_ = zw.Close()
			return err
		}
	}

	return zw.Close()
}

func writeZipItem(zw *zip.Writer, item Item) error {
	info, file, err := openItem(item)
	if err != nil {
		return err
	}
	defer file.Close()

	hdr, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	hdr.Name = archiveName(item)
	hdr.Method = zip.Deflate
	hdr.Modified = fixedArchiveTime

	w, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, file)
	return err
}

func writeTarGz(path string, items []Item) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	gz := gzip.NewWriter(file)
	tw := tar.NewWriter(gz)

	for _, item := range items {
		if err := writeTarItem(tw, item); err != nil {
			_ = tw.Close()
			_ = gz.Close()
			return err
		}
	}

	if err := tw.Close(); err != nil {
		_ = gz.Close()
		return err
	}
	return gz.Close()
}

func writeTarItem(tw *tar.Writer, item Item) error {
	info, file, err := openItem(item)
	if err != nil {
		return err
	}
	defer file.Close()

	hdr, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	hdr.Name = archiveName(item)
	hdr.ModTime = fixedArchiveTime
	hdr.AccessTime = hdr.ModTime
	hdr.ChangeTime = hdr.ModTime
	hdr.Uid = 0
	hdr.Gid = 0
	hdr.Uname = ""
	hdr.Gname = ""

	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}

	_, err = io.Copy(tw, file)
	return err
}

func openItem(item Item) (os.FileInfo, *os.File, error) {
	file, err := os.Open(item.Path)
	if err != nil {
		return nil, nil, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, nil, err
	}
	return info, file, nil
}

func archiveName(item Item) string {
	if item.Name != "" {
		return filepath.ToSlash(item.Name)
	}
	return filepath.Base(item.Path)
}

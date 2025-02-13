package mutation

import (
	"archive/tar"
	"bruce/loader"
	"bytes"
	"compress/gzip"
	"fmt"
	"github.com/rs/zerolog/log"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func useGzipReader(filename string, fileReader io.Reader) io.Reader {
	if strings.HasSuffix(filename, ".tgz") || strings.HasSuffix(filename, ".tar.gz") {
		gzr, err := gzip.NewReader(fileReader)
		if err != nil {
			log.Error().Err(err).Msg("the file is not a gzip file, cannot proceed.")
			return fileReader
		}
		return gzr
	}
	return fileReader
}

func sanitizePath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("path traversal attempt: %s", path)
	}
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("path traversal attempt: %s", path)
	}
	return cleanPath, nil
}

func ExtractTarball(src, dst string, force, stripRoot bool) error {
	// We just check dest currently as we will read from multiple source locations and they may fail by time we cleaned up so worthless to check upfront.
	if _, err := os.Stat(dst); err == nil {
		if !force {
			log.Info().Msgf("%s already exists cannot extract tarball to location", dst)
			return nil
		}
	}
	err := os.MkdirAll(dst, 0755)
	if err != nil {
		log.Error().Err(err).Msgf("cannot create directory at dst: %s", dst)
		return err
	}
	rsrc, _, err := loader.GetRemoteData(src, "")
	if err != nil {
		log.Error().Err(err).Msgf("cannot read tarball at src: %s", src)
		return err
	}
	rr := bytes.NewReader(rsrc)
	tr := tar.NewReader(useGzipReader(src, rr))
	for {
		header, err := tr.Next()
		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		case header == nil:
			continue
		}

		// the target location where the dir/file should be created
		sanitizedPath, err := sanitizePath(header.Name)
		if err != nil {
			log.Error().Err(err).Msg("path traversal attempt detected, skipping entry")
			continue
		}
		target := filepath.Join(dst, sanitizedPath)
		isTopLevel := false
		if stripRoot {
			elements := strings.Split(sanitizedPath, string(os.PathSeparator))
			if len(elements) > 1 {
				target = filepath.Join(dst, filepath.Join(elements[1:]...))
			} else {
				isTopLevel = true
			}
		}
		log.Debug().Msgf("extracting: %s", target)
		switch header.Typeflag {
		case tar.TypeDir:
			if !isTopLevel {
				if _, err := os.Stat(target); err != nil {
					if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
						return err
					}
				}
			}
		// create file with existing file mode
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			// save contents
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}
			err = f.Close()
			if err != nil {
				return err
			}
		}
	}
}

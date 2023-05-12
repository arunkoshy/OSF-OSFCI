package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"github.com/ulikunitz/xz"
	"io"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: extract <filename> [<targetDir>]")
		return
	}

	filename := os.Args[1]
	targetDir := "."
	if len(os.Args) > 2 {
		targetDir = os.Args[2]
	}

	err := extractCompressedFile(filename, targetDir)
	if err != nil {
		fmt.Println("Error:", err)
	}
}

func detectFormat(file *os.File) (string, error) {
	buffer := make([]byte, 261)

	_, err := file.Read(buffer)
	if err != nil {
		return "", err
	}

	// Check for gzip format
	if buffer[0] == 0x1f && buffer[1] == 0x8b {
		return "gzip", nil
	}

	// Check for tar format
	if buffer[257] == 'u' && buffer[258] == 's' && buffer[259] == 't' && buffer[260] == 'a' && buffer[261] == 'r' {
		return "tar", nil
	}

	// Check for zip format
	if buffer[0] == 'P' && buffer[1] == 'K' && buffer[2] == 0x03 && buffer[3] == 0x04 {
		return "zip", nil
	}

	// Check for xz format
	if buffer[0] == 0xfd && buffer[1] == '7' && buffer[2] == 'z' && buffer[3] == 'X' && buffer[4] == 'Z' && buffer[5] == 0x00 {
		return "xz", nil
	}

	return "", fmt.Errorf("unsupported file format")
}

func extractCompressedFile(filename, targetDir string) error {
	if targetDir == "" {
		targetDir = "."
	}

	// Check if target directory exists and create it if it doesn't
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		err := os.MkdirAll(targetDir, 0755)
		if err != nil {
			return err
		}
	}

	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	format, err := detectFormat(file)
	if err != nil {
		return err
	}

	file.Seek(0, 0)

	switch format {
	case "gzip":
		return extractTarGz(file, targetDir)
	case "tar":
		return extractTar(file, targetDir)
	case "zip":
		return extractZip(file, targetDir)
	case "xz":
		return extractTarXz(file, targetDir)
	default:
		return fmt.Errorf("unsupported file format")
	}
}

func extractTar(file *os.File, targetDir string) error {
	tarReader := tar.NewReader(file)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		targetPath := filepath.Join(targetDir, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			outFile, err := os.Create(targetPath)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return err
			}
			outFile.Close()
		}
	}
	return nil
}

func extractTarGz(file *os.File, targetDir string) error {
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		targetPath := filepath.Join(targetDir, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			outFile, err := os.Create(targetPath)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return err
			}
			outFile.Close()
		}
	}
	return nil
}

func extractZip(file *os.File, targetDir string) error {
	stat, err := file.Stat()
	if err != nil {
		return err
	}

	zipReader, err := zip.NewReader(file, stat.Size())
	if err != nil {
		return err
	}

	for _, zf := range zipReader.File {
		targetPath := filepath.Join(targetDir, zf.Name)

		if zf.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, zf.Mode()); err != nil {
				return err
			}
		} else {
			outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, zf.Mode())
			if err != nil {
				return err
			}

			rc, err := zf.Open()
			if err != nil {
				return err
			}

			if _, err := io.Copy(outFile, rc); err != nil {
				return err
			}

			rc.Close()
			outFile.Close()
		}
	}
	return nil
}

func extractTarXz(file *os.File, targetDir string) error {
	xzReader, err := xz.NewReader(file)
	if err != nil {
		return err
	}

	tarReader := tar.NewReader(xzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		targetPath := filepath.Join(targetDir, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			outFile, err := os.Create(targetPath)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}

	return nil
}

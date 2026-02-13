package gojinn

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
)

func (r *Gojinn) CreateGlobalSnapshot() (string, error) {
	r.logger.Info("Starting Global Snapshot Engine...")
	startTime := time.Now()

	snapshotDir := filepath.Join(r.DataDir, "snapshots")
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create snapshot dir: %w", err)
	}

	timestamp := startTime.Format("20060102_150405")
	snapshotFilename := fmt.Sprintf("gojinn_snapshot_%s.tar.gz", timestamp)
	snapshotPath := filepath.Join(snapshotDir, snapshotFilename)

	stageDir, err := os.MkdirTemp("", "gojinn_stage_*")
	if err != nil {
		return "", fmt.Errorf("failed to create staging dir: %w", err)
	}
	defer os.RemoveAll(stageDir)

	if r.db != nil {
		r.logger.Info("Snapshotting Database (VACUUM INTO)...")
		dbBackupPath := filepath.Join(stageDir, "replica.db")

		_, err := r.db.Exec(fmt.Sprintf("VACUUM INTO '%s'", dbBackupPath))
		if err != nil {
			r.logger.Error("Database snapshot failed", zap.Error(err))
			return "", fmt.Errorf("db vacuum into failed: %w", err)
		}
	}

	r.logger.Info("Snapshotting NATS JetStream & KV Store...")
	natsStorePath := filepath.Join(r.DataDir, "nats_store")
	natsStagePath := filepath.Join(stageDir, "nats_store")

	if err := copyDir(natsStorePath, natsStagePath); err != nil {
		return "", fmt.Errorf("failed to snapshot nats store: %w", err)
	}

	r.logger.Info("Compressing Snapshot Archive...")
	if err := createTarGz(stageDir, snapshotPath); err != nil {
		return "", fmt.Errorf("failed to compress snapshot: %w", err)
	}

	duration := time.Since(startTime)

	stat, _ := os.Stat(snapshotPath)
	sizeMb := float64(stat.Size()) / 1024.0 / 1024.0

	r.logger.Info("Global Snapshot Completed Successfully!",
		zap.String("file", snapshotPath),
		zap.Float64("size_mb", sizeMb),
		zap.Duration("duration", duration))

	return snapshotPath, nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(src, path)
		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		s, err := os.Open(path)
		if err != nil {
			return err
		}
		defer s.Close()

		d, err := os.Create(targetPath)
		if err != nil {
			return err
		}
		defer d.Close()

		_, err = io.Copy(d, s)
		return err
	})
}

func createTarGz(srcDir, destFile string) error {
	out, err := os.Create(destFile)
	if err != nil {
		return err
	}
	defer out.Close()

	gw := gzip.NewWriter(out)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	return filepath.Walk(srcDir, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !fi.Mode().IsRegular() {
			return nil
		}

		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(srcDir, file)
		header.Name = filepath.ToSlash(relPath)

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		f, err := os.Open(file)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(tw, f)
		return err
	})
}

func (r *Gojinn) RestoreGlobalSnapshot(archivePath string) error {
	r.logger.Warn("INITIATING GLOBAL SNAPSHOT RESTORE", zap.String("file", archivePath))

	stageDir, err := os.MkdirTemp("", "gojinn_restore_*")
	if err != nil {
		return fmt.Errorf("failed to create staging dir: %w", err)
	}
	defer os.RemoveAll(stageDir)

	r.logger.Info("Extracting archive...")
	if err := extractTarGz(archivePath, stageDir); err != nil {
		return fmt.Errorf("failed to extract snapshot: %w", err)
	}

	r.logger.Warn("Shutting down internal engines for disk swap...")
	_ = r.Cleanup()

	r.logger.Info("Swapping DataDir files...")

	natsTarget := filepath.Join(r.DataDir, "nats_store")
	natsStage := filepath.Join(stageDir, "nats_store")
	if _, err := os.Stat(natsStage); err == nil {
		r.logger.Info("Restoring NATS JetStream State...")
		_ = os.RemoveAll(natsTarget)
		_ = copyDir(natsStage, natsTarget)
	}

	dbStage := filepath.Join(stageDir, "replica.db")
	if _, err := os.Stat(dbStage); err == nil {
		r.logger.Info("Restoring Relational Database State...")

		dbTarget := filepath.Join(r.DataDir, "gojinn.db")
		if r.DBDSN != "" {
			dbTarget = r.DBDSN
		}
		_ = os.Remove(dbTarget)
		_ = copyFile(dbStage, dbTarget)
	}

	r.logger.Warn("Files successfully swapped! The server will now shut down to safely load the new state on the next boot.")
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func extractTarGz(srcFile, destDir string) error {
	f, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(destDir, hdr.Name)
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			outFile, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}
	return nil
}

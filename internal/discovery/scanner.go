package discovery

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"logpulse/internal/db"
)

var supportedExtensions = map[string]string{
	".log":   "log",
	".txt":   "text",
	".trace": "trace",
	".jsonl": "jsonl",
	".out":   "out",
	".evtx":  "evtx",
}

var ignoredDirs = map[string]bool{
	".git":         true,
	".svn":         true,
	"node_modules": true,
	"temp":         true,
	"winsxs":       true,
	"$recycle.bin": true,
}

type Scanner struct {
	store       *db.Store
	customRoots []string
}

func NewScanner(store *db.Store) *Scanner {
	return &Scanner{
		store: store,
	}
}

func (s *Scanner) SetCustomRoots(roots []string) {
	s.customRoots = roots
}

func InferAppName(filePath string) string {
	clean := filepath.Clean(filePath)
	dir := filepath.Dir(clean)
	parent := filepath.Base(dir)

	// If parent is generic (like "logs", "log", "current"), look one level up
	lowerParent := strings.ToLower(parent)
	if lowerParent == "logs" || lowerParent == "log" || lowerParent == "current" || lowerParent == "winevt" {
		grandParent := filepath.Base(filepath.Dir(dir))
		if grandParent != "." && grandParent != "/" && grandParent != "\\" {
			return grandParent
		}
	}

	if parent != "." && parent != "/" && parent != "\\" && parent != "" {
		return parent
	}

	// Fallback to filename without extension
	base := filepath.Base(clean)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

func (s *Scanner) ScanTier1() ([]db.DiscoveredLog, error) {
	var roots []string
	if len(s.customRoots) > 0 {
		roots = s.customRoots
	} else {
		roots = getTier1Paths()
	}

	var results []db.DiscoveredLog
	for _, root := range roots {
		logs, err := s.ScanDirectory(root)
		if err == nil {
			results = append(results, logs...)
		}
	}
	return results, nil
}

func (s *Scanner) ScanDirectory(dir string) ([]db.DiscoveredLog, error) {
	var discovered []db.DiscoveredLog

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip unreadable directories/files without breaking the walk
		}

		if d.IsDir() {
			baseLower := strings.ToLower(d.Name())
			if ignoredDirs[baseLower] {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(d.Name()))
		format, ok := supportedExtensions[ext]
		if !ok {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		entry := db.DiscoveredLog{
			FilePath:     path,
			AppName:      InferAppName(path),
			FileSize:     info.Size(),
			LastModified: info.ModTime(),
			Format:       format,
		}

		if s.store != nil {
			_ = s.store.SaveDiscoveredLog(entry)
		}

		discovered = append(discovered, entry)
		return nil
	})

	return discovered, err
}

func (s *Scanner) StartTier2Background(ctx context.Context, onFound func(db.DiscoveredLog)) {
	go func() {
		roots := s.customRoots
		if len(roots) == 0 {
			roots = getTier1Paths()
		}

		for _, root := range roots {
			_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
				select {
				case <-ctx.Done():
					return filepath.SkipAll
				default:
				}

				if err != nil {
					return nil
				}

				if d.IsDir() {
					// Sleep 5ms between folders to keep CPU and disk IO low
					time.Sleep(5 * time.Millisecond)
					baseLower := strings.ToLower(d.Name())
					if ignoredDirs[baseLower] {
						return filepath.SkipDir
					}
					return nil
				}

				ext := strings.ToLower(filepath.Ext(d.Name()))
				format, ok := supportedExtensions[ext]
				if !ok {
					return nil
				}

				info, err := d.Info()
				if err != nil {
					return nil
				}

				entry := db.DiscoveredLog{
					FilePath:     path,
					AppName:      InferAppName(path),
					FileSize:     info.Size(),
					LastModified: info.ModTime(),
					Format:       format,
				}

				if s.store != nil {
					_ = s.store.SaveDiscoveredLog(entry)
				}

				if onFound != nil {
					onFound(entry)
				}

				return nil
			})
		}
	}()
}

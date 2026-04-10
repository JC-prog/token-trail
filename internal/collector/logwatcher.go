package collector

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/jcprog/token-trail/internal/database"
)

type LogWatcher struct {
	db        *database.DB
	watchPath string
	ctx       context.Context
	cancel    context.CancelFunc
	watcher   *fsnotify.Watcher
	mu        sync.Mutex
	isRunning bool
}

// NewLogWatcher creates a new log watcher
func NewLogWatcher(db *database.DB) *LogWatcher {
	return &LogWatcher{
		db: db,
	}
}

// Start starts watching the Claude Code log directory
func (lw *LogWatcher) Start(appCtx context.Context, watchPath string) error {
	lw.mu.Lock()
	defer lw.mu.Unlock()

	if lw.isRunning {
		return fmt.Errorf("log watcher already running")
	}

	// Use provided path or detect default
	if watchPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		watchPath = filepath.Join(home, ".claude")
	}

	// Check if path exists
	if _, err := os.Stat(watchPath); err != nil {
		return fmt.Errorf("log path does not exist: %w", err)
	}

	// Create watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	lw.ctx, lw.cancel = context.WithCancel(appCtx)
	lw.watchPath = watchPath
	lw.watcher = watcher

	// Add watch
	if err := watcher.Add(watchPath); err != nil {
		watcher.Close()
		return fmt.Errorf("failed to add watch: %w", err)
	}

	// Start watching in background
	go lw.run()
	lw.isRunning = true
	return nil
}

// Stop stops watching the log directory
func (lw *LogWatcher) Stop() {
	lw.mu.Lock()
	defer lw.mu.Unlock()

	if !lw.isRunning {
		return
	}

	lw.cancel()
	if lw.watcher != nil {
		lw.watcher.Close()
	}
	lw.isRunning = false
}

// run watches for file changes
func (lw *LogWatcher) run() {
	defer lw.watcher.Close()

	for {
		select {
		case <-lw.ctx.Done():
			return
		case event, ok := <-lw.watcher.Events:
			if !ok {
				return
			}

			// Handle write events
			if event.Op&fsnotify.Write == fsnotify.Write {
				lw.handleFileChange(event.Name)
			}

		case err, ok := <-lw.watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("Log watcher error: %v\n", err)
		}
	}
}

// handleFileChange processes a changed log file
func (lw *LogWatcher) handleFileChange(filename string) {
	// For MVP, log watcher is stubbed
	// Real implementation would:
	// 1. Parse .jsonl files in ~/.claude/
	// 2. Extract usage events
	// 3. Deduplicate against API-polled data
	// 4. Insert into database
	_ = filename
}

// SetWatchPath updates the watch path
func (lw *LogWatcher) SetWatchPath(path string) error {
	lw.mu.Lock()
	defer lw.mu.Unlock()

	if lw.isRunning {
		// Remove old watch
		if lw.watcher != nil {
			lw.watcher.Remove(lw.watchPath)
			// Add new watch
			if err := lw.watcher.Add(path); err != nil {
				// Restore old watch
				lw.watcher.Add(lw.watchPath)
				return fmt.Errorf("failed to add new watch: %w", err)
			}
		}
	}

	lw.watchPath = path
	return nil
}

// GetDefaultWatchPath returns the default log watch path for the OS
func GetDefaultWatchPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".claude"), nil
}

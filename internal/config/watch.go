package config

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Watch monitors a config file and triggers onChange after poll changes or SIGHUP.
// Poll interval is fixed to 2 seconds.
func Watch(path string, onChange func(*Config, error)) (func(), error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	done := make(chan struct{})
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP)

	lastMod := info.ModTime()
	lastSize := info.Size()
	ticker := time.NewTicker(2 * time.Second)

	reload := func() {
		cfg, loadErr := Load(path)
		if onChange != nil {
			onChange(cfg, loadErr)
		}
	}

	go func() {
		defer ticker.Stop()
		defer signal.Stop(sigCh)
		for {
			select {
			case <-done:
				return
			case <-sigCh:
				reload()
			case <-ticker.C:
				current, statErr := os.Stat(path)
				if statErr != nil {
					if onChange != nil {
						onChange(nil, statErr)
					}
					continue
				}
				if current.ModTime().After(lastMod) || current.Size() != lastSize {
					lastMod = current.ModTime()
					lastSize = current.Size()
					reload()
				}
			}
		}
	}()

	var once sync.Once
	stop := func() {
		once.Do(func() {
			close(done)
		})
	}
	return stop, nil
}

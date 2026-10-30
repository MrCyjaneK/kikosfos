package main

import (
	"os"
	"path/filepath"
)

// kikoDirs is the Silica app's config and data, org wtf.kiko / app kiko.
// The car process is independent of that app. Root (the AOA daemon) reads
// defaultuser's home; anyone else uses their own.
func kikoDirs() (config, data string) {
	home := "/home/defaultuser"
	if os.Getuid() != 0 {
		if h, err := os.UserHomeDir(); err == nil && h != "" {
			home = h
		}
	}
	return filepath.Join(home, ".config", "wtf.kiko", "kiko"),
		filepath.Join(home, ".local", "share", "wtf.kiko", "kiko")
}

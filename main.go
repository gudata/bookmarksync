package main

import (
	"bufio"
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"gopkg.in/ini.v1"
)

const Version = "0.4.0"

func main() {
	var syncFrom string
	var showVersion bool
	var showHelp bool

	flag.StringVar(&syncFrom, "sync-from", "", "CLI mode: sync from a particular backend (gtk, kde, qt)")
	flag.StringVar(&syncFrom, "f", "", "CLI mode: sync from a particular backend (gtk, kde, qt) (shorthand)")
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.BoolVar(&showHelp, "help", false, "Show help information")
	flag.Parse()

	if showVersion {
		fmt.Printf("BookmarkSync %s\n", Version)
		return
	}

	if showHelp || syncFrom == "" {
		fmt.Println("BookmarkSync - A utility to sync bookmarks between GTK+, KDE, and Qt file dialogs")
		fmt.Printf("Version: %s\n\n", Version)
		fmt.Println("Usage:")
		fmt.Println("  bookmarksync-go [OPTIONS]")
		fmt.Println("\nOptions:")
		fmt.Println("  -f, --sync-from BACKEND   Sync from a particular backend (gtk, kde, qt)")
		fmt.Println("  --version                 Show version information")
		fmt.Println("  --help                    Show this help message")
		return
	}

	backend := strings.ToLower(syncFrom)
	if backend != "gtk" && backend != "kde" && backend != "qt" {
		log.Fatalf("Unknown backend: %s", backend)
	}

	fmt.Printf("Running sync from %s backend\n", backend)

	sync := NewBookmarkSync()
	if err := sync.SyncFrom(backend); err != nil {
		log.Fatalf("Sync failed: %v", err)
	}
}

// Place represents a bookmark entry
type Place struct {
	Label  string
	Target string
}

// BookmarkSyncBackend defines the interface for bookmark backends
type BookmarkSyncBackend interface {
	GetPlaces() ([]Place, error)
	Replace(places []Place) error
	Name() string
}

// BookmarkSync manages syncing between backends
type BookmarkSync struct {
	backends map[string]BookmarkSyncBackend
}

// NewBookmarkSync creates a new BookmarkSync instance
func NewBookmarkSync() *BookmarkSync {
	return &BookmarkSync{
		backends: map[string]BookmarkSyncBackend{
			"gtk": &GTKBackend{},
			"kde": &KDEBackend{},
			"qt":  &QtBackend{},
		},
	}
}

// SyncFrom syncs bookmarks from the specified backend to all others
func (bs *BookmarkSync) SyncFrom(backendName string) error {
	sourceBackend, exists := bs.backends[backendName]
	if !exists {
		return fmt.Errorf("unknown backend: %s", backendName)
	}

	places, err := sourceBackend.GetPlaces()
	if err != nil {
		return fmt.Errorf("failed to get places from %s: %v", backendName, err)
	}

	for name, backend := range bs.backends {
		if name != backendName {
			if err := backend.Replace(places); err != nil {
				log.Printf("Warning: failed to sync to %s: %v", name, err)
			}
		}
	}

	return nil
}

// GTKBackend implements BookmarkSyncBackend for GTK bookmarks
type GTKBackend struct{}

func (g *GTKBackend) Name() string {
	return "gtk"
}

func (g *GTKBackend) GetPlaces() ([]Place, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	bookmarksPath := filepath.Join(homeDir, ".config", "gtk-3.0", "bookmarks")
	file, err := os.Open(bookmarksPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Place{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var places []Place
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		target := parts[0]
		label := ""
		if len(parts) > 1 {
			label = parts[1]
		} else {
			// Extract label from URL path
			if u, err := url.Parse(target); err == nil {
				label = filepath.Base(u.Path)
				if decoded, err := url.QueryUnescape(label); err == nil {
					label = decoded
				}
			}
		}
		places = append(places, Place{Label: label, Target: target})
	}

	return places, scanner.Err()
}

func (g *GTKBackend) Replace(places []Place) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(homeDir, ".config", "gtk-3.0")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	bookmarksPath := filepath.Join(configDir, "bookmarks")
	file, err := os.Create(bookmarksPath)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, place := range places {
		if place.Label != "" {
			fmt.Fprintf(file, "%s %s\n", place.Target, place.Label)
		} else {
			fmt.Fprintf(file, "%s\n", place.Target)
		}
	}

	return nil
}

// KDEBackend implements BookmarkSyncBackend for KDE bookmarks
type KDEBackend struct{}

func (k *KDEBackend) Name() string {
	return "kde"
}

type XBEL struct {
	XMLName   xml.Name   `xml:"xbel"`
	Bookmarks []Bookmark `xml:"bookmark"`
}

type Bookmark struct {
	Href  string `xml:"href,attr"`
	Title string `xml:"title"`
	Info  Info   `xml:"info"`
}

type Info struct {
	Metadata []Metadata `xml:"metadata"`
}

type Metadata struct {
	Owner        string        `xml:"owner,attr"`
	IsSystemItem *IsSystemItem `xml:"isSystemItem"`
}

type IsSystemItem struct{}

func (k *KDEBackend) GetPlaces() ([]Place, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	xbelPath := filepath.Join(homeDir, ".local", "share", "user-places.xbel")
	file, err := os.Open(xbelPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Place{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var xbel XBEL
	if err := xml.NewDecoder(file).Decode(&xbel); err != nil {
		return nil, err
	}

	var places []Place
	for _, bookmark := range xbel.Bookmarks {
		// Skip system items
		isSystem := false
		for _, metadata := range bookmark.Info.Metadata {
			if metadata.IsSystemItem != nil {
				isSystem = true
				break
			}
		}
		if !isSystem {
			places = append(places, Place{
				Label:  bookmark.Title,
				Target: bookmark.Href,
			})
		}
	}

	return places, nil
}

func (k *KDEBackend) Replace(places []Place) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// First, read existing file to preserve system items
	xbelPath := filepath.Join(homeDir, ".local", "share", "user-places.xbel")
	var existingXBEL XBEL
	
	if file, err := os.Open(xbelPath); err == nil {
		xml.NewDecoder(file).Decode(&existingXBEL)
		file.Close()
	}

	// Create directory if it doesn't exist
	shareDir := filepath.Join(homeDir, ".local", "share")
	if err := os.MkdirAll(shareDir, 0755); err != nil {
		return err
	}

	// Keep system items, replace user items
	var newBookmarks []Bookmark
	for _, bookmark := range existingXBEL.Bookmarks {
		isSystem := false
		for _, metadata := range bookmark.Info.Metadata {
			if metadata.IsSystemItem != nil {
				isSystem = true
				break
			}
		}
		if isSystem {
			newBookmarks = append(newBookmarks, bookmark)
		}
	}

	// Add new user places
	for _, place := range places {
		newBookmarks = append(newBookmarks, Bookmark{
			Href:  place.Target,
			Title: place.Label,
			Info: Info{
				Metadata: []Metadata{{
					Owner: "http://www.kde.org",
				}},
			},
		})
	}

	xbel := XBEL{
		Bookmarks: newBookmarks,
	}

	file, err := os.Create(xbelPath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := xml.NewEncoder(file)
	encoder.Indent("", "  ")
	file.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	file.WriteString(`<!DOCTYPE xbel PUBLIC "+//IDN python.org//DTD XML Bookmark Exchange Language 1.0//EN//XML" "http://www.python.org/topics/xml/dtds/xbel-1.0.dtd">` + "\n")
	return encoder.Encode(&xbel)
}

// QtBackend implements BookmarkSyncBackend for Qt bookmarks
type QtBackend struct{}

func (q *QtBackend) Name() string {
	return "qt"
}

func (q *QtBackend) GetPlaces() ([]Place, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	qtConfigPath := filepath.Join(homeDir, ".config", "QtProject.conf")
	cfg, err := ini.Load(qtConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Place{}, nil
		}
		return nil, err
	}

	fileDialogSection := cfg.Section("FileDialog")
	shortcuts := fileDialogSection.Key("shortcuts").String()
	if shortcuts == "" {
		return []Place{}, nil
	}

	var places []Place
	for _, shortcut := range strings.Split(shortcuts, ", ") {
		shortcut = strings.TrimSpace(shortcut)
		if shortcut != "" {
			// Qt doesn't support custom labels, use basename
			label := filepath.Base(shortcut)
			places = append(places, Place{
				Label:  label,
				Target: "file://" + shortcut,
			})
		}
	}

	return places, nil
}

func (q *QtBackend) Replace(places []Place) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	qtConfigPath := filepath.Join(homeDir, ".config", "QtProject.conf")
	
	// Load existing config or create new one
	var cfg *ini.File
	if _, err := os.Stat(qtConfigPath); os.IsNotExist(err) {
		cfg = ini.Empty()
	} else {
		cfg, err = ini.Load(qtConfigPath)
		if err != nil {
			return err
		}
	}

	// Filter places to only include local file:// URLs
	var shortcuts []string
	for _, place := range places {
		if strings.HasPrefix(place.Target, "file://") {
			// Remove file:// prefix and URL decode
			path := strings.TrimPrefix(place.Target, "file://")
			if decoded, err := url.QueryUnescape(path); err == nil {
				shortcuts = append(shortcuts, decoded)
			} else {
				shortcuts = append(shortcuts, path)
			}
		}
	}

	fileDialogSection := cfg.Section("FileDialog")
	fileDialogSection.Key("shortcuts").SetValue(strings.Join(shortcuts, ", "))

	// Create config directory if it doesn't exist
	configDir := filepath.Join(homeDir, ".config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	return cfg.SaveTo(qtConfigPath)
}
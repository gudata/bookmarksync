#!/usr/bin/env python3
"""Simple test for Go implementation"""
import os
import shutil
import subprocess
import tempfile

def test_go_implementation():
    # Create temporary directory
    with tempfile.TemporaryDirectory() as tmpdir:
        print(f"Testing in {tmpdir}")
        
        # Test 1: Sync from GTK
        print("\n=== Test 1: Sync from GTK ===")
        testdir = os.path.join(tmpdir, "from-gtk")
        os.makedirs(testdir)
        
        # Set up GTK bookmarks
        gtk_dir = os.path.join(testdir, ".config", "gtk-3.0")
        os.makedirs(gtk_dir)
        shutil.copy("gtk-bookmarks.example", os.path.join(gtk_dir, "bookmarks"))
        
        # Run sync
        env = {"HOME": testdir}
        result = subprocess.run(["../bookmarksync-go", "-f", "gtk"], 
                              env=env, capture_output=True, text=True)
        print(f"Exit code: {result.returncode}")
        print(f"Output: {result.stdout}")
        if result.stderr:
            print(f"Error: {result.stderr}")
        
        # Check files were created
        kde_file = os.path.join(testdir, ".local", "share", "user-places.xbel")
        qt_file = os.path.join(testdir, ".config", "QtProject.conf")
        
        print(f"KDE file exists: {os.path.exists(kde_file)}")
        print(f"Qt file exists: {os.path.exists(qt_file)}")
        
        if os.path.exists(kde_file):
            with open(kde_file) as f:
                print("KDE file contents:")
                print(f.read()[:200] + "...")
        
        # Test 2: Sync from KDE
        print("\n=== Test 2: Sync from KDE ===")
        testdir2 = os.path.join(tmpdir, "from-kde")
        os.makedirs(testdir2)
        
        # Set up KDE bookmarks
        kde_dir = os.path.join(testdir2, ".local", "share")
        os.makedirs(kde_dir)
        shutil.copy("user-places.xbel.example", os.path.join(kde_dir, "user-places.xbel"))
        
        # Run sync
        env = {"HOME": testdir2}
        result = subprocess.run(["../bookmarksync-go", "-f", "kde"], 
                              env=env, capture_output=True, text=True)
        print(f"Exit code: {result.returncode}")
        print(f"Output: {result.stdout}")
        if result.stderr:
            print(f"Error: {result.stderr}")
        
        # Check files were created
        gtk_file = os.path.join(testdir2, ".config", "gtk-3.0", "bookmarks")
        qt_file = os.path.join(testdir2, ".config", "QtProject.conf")
        
        print(f"GTK file exists: {os.path.exists(gtk_file)}")
        print(f"Qt file exists: {os.path.exists(qt_file)}")
        
        print("\n=== All tests completed ===")

if __name__ == "__main__":
    os.chdir(os.path.dirname(os.path.abspath(__file__)))
    test_go_implementation()
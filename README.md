# BookmarkSync

This is a Go (Golang) port of the CLI (command-line interface) component of [BookmarkSync](https://github.com/jlu5/bookmarksync/), implemented with the help of AI.

I specifically needed and tested the -f kde flag, and it worked as expected for me.


**BookmarkSync** is a simple program that manages the bookmarks (pinned folders) in GTK+, KDE, and Qt's native file pickers.

## Do I need this?

On GTK+ based desktop environments like GNOME or Xfce, you can make Qt use the GTK+ file picker by setting `QT_QPA_PLATFORMTHEME=gtk3` or `QT_QPA_PLATFORMTHEME=gtk2`. This is usually the easiest option on those environments.

Alternatively: Chrome, Firefox ([with `about:config` override](https://wiki.archlinux.org/title/Firefox#XDG_Desktop_Portal_integration)), and most sandboxed apps support [XDG Desktop Portal](https://wiki.archlinux.org/title/XDG_Desktop_Portal) which will automatically load the desktop environment's native file picker.

## CLI mode

As of v0.3.0 there is support for running sync from the command line: `$ bookmarksync --sync-from {gtk,kde,qt}`.

## Under the hood

- **GTK+** stores bookmarks in a simple plain text format at `~/.config/gtk-3.0/bookmarks`, which BookmarkSync manipulates as a plain text file.
- **KDE** stores bookmarks in XML form at `~/.local/share/user-places.xbel`. BookmarkSync uses [KFilePlacesModel](https://api.kde.org/frameworks/kio/html/classKFilePlacesModel.html) from KIO to edit these natively.
- **Qt** stores bookmarks in the Qt config file (INI format) at `~/.config/QtProject.conf`. BookmarkSync uses [QFileDialog](https://doc.qt.io/qt-5/qfiledialog.html#setSidebarUrls) methods (and hidden file dialog instances) to read and write to these.

### Known limitations

- Only KDE supports custom icons for places; syncing from others will erase all custom icons.
- Only GTK+ and KDE support remote locations like `sftp://` or `smb://` in bookmarks: syncing *from* Qt will remove all remote places from the list.
- Editing bookmarks from another program while BookmarkSync is running may cause things to go out of sync. This mainly affects the Qt backend, as the KDE and GTK+ backends tend to refresh faster.

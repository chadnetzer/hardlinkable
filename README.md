## `hardlinkable` - Find and optionally link identical files

`hardlinkable` is a tool to scan directories and report files that could be hardlinked together because they have identical content, and (by default) other matching criteria such as modification time, permissions and ownership.  It can optionally perform the linking as well, saving storage space (but by default, it only reports information).

This program is faster and with more accurate reporting of results than other variants that I have tried.  It works by gathering full inode information before deciding what action (if any) to take.  Full information allows it to produce exact reporting of what will happen before any modifications occur.  It also uses content information from previous comparisons to drastically reduce search times.

---
## Help
```
$ hardlinkable --help
A tool to scan directories and report on the space that could be saved
by hardlinking identical files.  It can also perform the linking.

Usage:
  hardlinkable [OPTIONS] dir1 [dir2...] [files...]

Flags:
  -v, --verbose           Increase verbosity level (up to 3 times)
      --no-progress       Disable progress output while processing
      --json              Output results as JSON
      --enable-linking    Perform the actual linking (implies --quiescence)
  -f, --same-name         Filenames need to be identical
  -t, --ignore-time       File modification times need not match
  -p, --ignore-perm       File permission (mode) need not match
  -o, --ignore-owner      File uid/gid need not match
  -x, --ignore-xattr      Xattrs need not match
  -c, --content-only      Only file contents have to match (ie. -potx)
  -s, --min-size N        Minimum file size (default 1)
  -S, --max-size N        Maximum file size
  -i, --include RE        Regex(es) used to include files (overrides excludes)
  -e, --exclude RE        Regex(es) used to exclude files
  -E, --exclude-dir RE    Regex(es) used to exclude dirs
  -d, --debug             Increase debugging level
      --ignore-walkerr    Continue on file/dir read errs
      --ignore-linkerr    Continue when linking fails
      --quiescence        Abort if filesystem is being modified
      --disable-newest    Disable using newest link mtime/uid/gid
      --search-thresh N   Ino search length before enabling digests (default 1)
  -h, --help              help for hardlinkable
      --version           version for hardlinkable
```

The include/exclude options can be given multiple times to support multiple regex matches.

`--debug` outputs additional information about program state in the final stats and the progress information.

`--ignore-walkerr` allows the program to skip over unreadable files and directories, and continue with the information gathering.

`--ignore-linkerr` allows the program to skip any links that cannot be made due to permission problems or other errors, and continue with the processing.  It is only applicable when linking is enabled, and should be used with caution.

`--quiescence` checks that the files haven't changed between the initial scan and the attempt to link (such as filesizes or timestamps changing), etc.  This would suggest they are being modified, and the program stops when this is detected.  Specifying `--quiescence` during a normal scan, where linking is not enabled, will perform these checks anyway at a small performance cost.

`--disable-newest` will turn off the default behavior of attempting to set the src inode to the most recent modification time of the linked inodes, and also change the uid/gid to those of the more recent inode.  This behavior can be useful for backup programs, so that they see inodes as being newer, and will back them up.  Only applicable when linking is enabled.

`--search-thresh` can be set to (-1) to disable the use of digests, which may save a small amount of memory (at the cost of possibly many more comparisons done).  Otherwise this controls the length that inode hashes must grow to before enabling the use of digests.  Safe to ignore, this option will not affect results, only possibly the time required to complete a run.

---
## Example output
```
$ hardlinkable download_dirs
Hard linking statistics
-----------------------
Directories               : 3408
Files                     : 89177
Hardlinkable this run     : 2462
Removable inodes          : 2462
Currently linked bytes    : 23480519   (22.393 MiB)
Additional saveable bytes : 245927685  (234.535 MiB)
Total saveable bytes      : 269408204  (256.928 MiB)
Total run time            : 4.691s
```

Additional verbosity levels will provide additional stats, a list of linkable files, and previously linked files:

```
$ hardlinkable -vvv download_dirs
Currently hardlinked files
--------------------------
from: download_dir/testfont/BlackIt/testfont.otf
  to: download_dir/testfont/BoldIt/testfont.otf
  to: download_dir/testfont/ExtraLightIt/testfont.otf
  to: download_dir/testfont/It/testfont.otf
Filesize: 4.146 KiB  Total saved: 12.438 KiB
...

Files that are hardlinkable
-----------------------
from: download_dir/bak1/some_image1.png
  to: download_dir/bak2/some_image1.png
...
from: download_dir/fonts1/some_font.otf
  to: download_dir/other_fonts1/some_font.otf

Hard linking statistics
-----------------------
Directories                 : 3408
Files                       : 89177
Hardlinkable this run       : 2462
Removable inodes            : 2462
Currently linked bytes      : 23480519   (22.393 MiB)
Additional saveable bytes   : 245927685  (234.535 MiB)
Total saveable bytes        : 269408204  (256.928 MiB)
Total run time              : 4.765s
Comparisons                 : 21479
Inodes                      : 80662
Existing links              : 8515
Total old + new links       : 10977
Total too small files       : 71
Total bytes compared        : 246099717  (234.699 MiB)
Total remaining inodes      : 78200
```

A more detailed breakdown of the various stats can be found in the [Results.md](Results.md).

---
## Methodology

This program is named `hardlinkable` to indicate that, by default, it does *not* perform any linking, and the user has to explicitly opt-in to having it perform the linking step.  This (to me) is a safer and more-sensible default than the alternatives; it's not unusual to want to run it a few times with different options to see what would result, before actually deciding whether to perform the linking.

The program first gathers all the information from the directory and file walk, and uses this information to execute a linking strategy which minimizes the number of moved links required to reach the final state.

Besides having more accurate statistics, this version can be significantly faster than other versions, due to opportunistically keeping track of simple file content hashes as the inode hash comparison lists grow.  It computes these content hashes at first only when comparing files (when the file data will be read anyway), to avoid unnecessary I/O.  Using this data and quick set operations, it can drastically reduce the amount of file comparisons attempted as the number of walked files grows.

---
## History

There are a number of programs that will perform hardlinking of identical files, and both Redhat and Debian/Ubuntu each include a `hardlink` program, with different implementation and capabilities.  The Redhat variant is based upon `hardlink.c` originally written by Jakub Jelinek, which later inspired John Villalovos to write his own version in Python, now known as `hardlinkpy` with multiple additional contributors (Antti Kaihola, Carl Henrik Lunde, et al.)  The Python version inspired Julian Andres Klode to do yet another re-implementation in C, which also added support for Xattrs.  There are numerous other variants floating around as well.

The previous versions that I've encountered do the hardlinking while walking the directory tree, before gathering complete information on all the inodes and pathnames.  This tends to lead to inaccurate statistics reported during a "dry run", and can also cause links to be needlessly moved from inode to inode multiple times during a run.  They also don't use "dry run" mode as the default, so you have to remember to enable "dry run" if you just want to play with different options, or find information on the amount of duplicate files that exist.

This version is written in Go and incorporates ideas from previous versions, as well as it's own innovations, to ensure exactly accurate results when in "dry run" mode and actual linking mode.  I expect and intend for it to be the fastest version, due to avoiding unnecessary I/O, minimizing extraneous searches and comparisons, and because it never moves a link more than once during a run.

---
## Contributing

Contributions are welcome, including bug reports, suggestions, and code patches/pull requests/etc.  I'm interested in hearing what you use `hardlinkable` for, and what could make it more useful to you.  If you've used other space-recovery hardlinking programs, I'm also interested to know if `hardlinkable` bests them in speed and report accuracy, or if you've found a regression in performance or capability.

## Build

```
go test -short ./...
go test ./...  # Could take a minute
go install ./...  # installs to GOPATH/bin

or

cd cmd/hardlinkable && go build  # builds in cmd/hardlinkable
```

## Install `hardlinkable` command
```
go get github.com/chadnetzer/hardlinkable/cmd/hardlinkable
```

## License

`hardlinkable` is released under the MIT license.

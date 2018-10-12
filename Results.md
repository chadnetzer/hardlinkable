## Results

Below is an example output from `hardlinkable`, with full debug output enabled (`-ddd`).  This run includes a directory with existing links from outside the walked directories, to illustrate how it can affect the displayed results.

---
```
Directories                 : 6793
Files                       : 178296
Hardlinkable this run       : 95374
```

These counts are straightforward, showing the number of directories and files found, and the number of links that would have been moved had `--enable-linking` been selected.

---
```
Removable inodes            : 79839
```

Space is only actually saved when inodes are removed, ie. when all the links are moved away from an inode to others and the nlink count drops to zero, allowing the filespace to be recovered.

---
```
Currently linked bytes      : 195270885     (186.225 MiB)
Additional saveable bytes   : 55361515670   (51.559 GiB)
Total saveable bytes        : 55556786555   (51.741 GiB)
```

These show the amount of bytes saved, counting both the existing links as "saved" space, and the additional bytes saved because inode nlink counts would drop to zero during a `hardlinkable --enable-linking` run.

---
```
Total run time              : 37m19.926s
Comparisons                 : 111096
Inodes                      : 169634
Inode total nlinks          : 267417        (Unwalked Nlinks: 89121)
Existing links              : 8662
Total old + new links       : 104036
Total too small files       : 142
```

Comparisons are the count of the times that the run found compatible inodes (ie. equal file size, and optionally time, permissions, etc.) and the bytes of the files were compared to try to determine if they were exactly equal.  There is a count of the total number of unique inodes encountered, and the total count of inode nlinks found (ie. the sum of all the nlinks for all the inodes).  The additional "`Unwalked Nlinks`" field here is explicit feedback that inodes were found with links that weren't included in the directory walk, which can restrict inode nlink counts from reaching zero and actually saving any space.

This also shows the number of existing links found during the directory/file scan (ie. the "walk").  This count doesn't include nlinks that are not found in the directory walk (another indication of the existence of filesystem paths that weren't included in the walk).

The program will report the count of files that were outside the range of file sizes to be considered.

---
```
Equal files w/ unequal time : 3835          (658.915 MiB)
Equal files w/ unequal mode : 728           (53.825 MiB)
Total equal file mismatches : 3869          (659.618 MiB)
```

This shows that, because we are ignoring file permissions (ie. mode), we were able to find linkable files that have non-equal modes.  The report of 'mismatches', including those reported when ignoring mtime, ownership, etc., can be useful in determining how to tune future runs, or when `--enable-linking` is selected.  Because files may have multiple mismatches, the total is also given which accounts for any overlap.

---
```
Total bytes compared        : 111248669194  (103.608 GiB)
Total remaining inodes      : 89795
Dir errors this run         : 1
```

The total amount of bytes compared, the total number of inodes remaining after those with nlink count zero are removed, and the count of errors reading directories or files (typically a permissions issue).


---
```
Total file hash hits        : 142564        misses: 35733  sum total: 178297
Total hash mismatches       : 42572         (+ total links: 146608)
Total hash searches         : 133901
Total hash list iterations  : 111246        (avg per search: 0.8)
Total equal comparisons     : 91329
Total digests computed      : 110237
Mem Alloc                   : 22.842 MiB
Mem Sys                     : 341.127 MiB
Num live objects            : 26206
```

These are all debugging stats, including 'hash' hits which indicate inodes that are found with compatible paramaters, based on the selected equality parameters (ie. equal file sizes, file times, file mode/permissions, ownership, xattrs, etc.)  A hash hit doesn't mean a file matches, but it typically means a file comparison will be performed to determine if the file contents are equal or not.  No linking can be performed without first doing a full file comparison, thus when there a lot of files that can be consolidated, the I/O required is the main limit on run speed.

When a hash match is found, the program has to search the list of files with equivalent hashes, comparing the new file to each candidate looking for a match.  Rather than always comparing the full list, 'digests' are used to remember the beginning of already compared files (ie. another hash of the first 4KiB or so), and can be used to quickly eliminate files with matching inode hashes, but definitely different file contents.  This can greatly reduce the total number of comparisons attempted, and greatly speed up the runs.  The `--search-thresh` option determines how long an inode hash list can grow to, before the program starts to use digests.  By increasing the `--search-thresh`, or disabling by setting it to `-1`, you can see how the count of comparisons grows quadratically.

Finally there are data on the amount of memory used during the run, including the current memory used (Mem Alloc), and the peak amount of memory requested from the operating system (Mem Sys).  The program makes an effort to minimize memory use, but still has to keep track of the path and inode information for every file discovered, and the number of existing and new links created, so when huge numbers of files are scanned, the memory usage will necessarily grow with the number of files.

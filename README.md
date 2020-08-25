# media-storage-cataloger
A command-line tool written in Go/Golang to help with cataloging photos and videos according to their probable taken dates.

# Dependencies
This tool needs [exiftool](https://exiftool.org/install.html) to be available in a directory from your "PATH" environment variable.

# Supported formats
**exiftool** supports a lot of [media file formats](https://exiftool.org/#supported), but we limited them to
**heic**, **jpeg**, **gif**, **png**, **3gp**, **m4v**, **mov**, **mp4**, **mpg**, **mpo** (our current needs, should be extended).

# Supported OS platforms
[exiftool](https://exiftool.org/) works on Linux, Windows and MacOS.
However, due to limits of using exiftool-wrapper named **go-exiftool**, this tool works only on OSes with '\n' line breaker (Linux).
Windows and MacOS will be available later after [this PR](https://github.com/barasher/go-exiftool/pull/7) are approved and
**media-storage-cataloger**'s dependencies are updated.

# Command "rename"
It renames media files according to their probable date taken: `IMG_2024.jpg` => `2019.03.08 19.30.30.dt (IMG_2024).jpg`.

The tool tries to read exif-tags in the following order: **CreateDate**, **DateTimeOriginal**, **CreationDate**, **MediaCreateDate**.
Gets source timezone from the found tag, the following exif-tags: **OffsetTimeOriginal**, **OffsetTimeDigitized**, **OffsetTime** or set it to UTC (when source timezone is unavailable).

If media has date in its name or has exif-tag **Comment** with content like `timestamp=1596285651480` (IPhone GIFs), the date taken is constructed from it.

After all, if date taken still not found, **media-storage-cataloger** uses file modify date as date taken (inaccurate way).

When a media file already has compliant filename format and probable date taken is too far from filename date, the tool skips the file.

## Options and flags
```
media-storage-cataloger rename <mediaDir> [flags]

Flags:
  -a, --action string               Action to do with media: "print" or "execute" renaming (default "print")
  -h, --help                        help for rename
  -d, --maxDatesDistance duration   Maximum time distance between date in old filename and date in new filename (default 26h0m0s)
  -z, --timezoneCustom string       Timezone for date in new filename (default "00:00")
  -s, --timezoneSource string       The source of timezone: "media" or "custom". Use parameter "timezoneCustom" to set the custom timezone (default "media")
```

## Usage
You can run the command with `--action print` option to see what **media-storage-cataloger** is going to do with your media files.
```
$ media-storage-cataloger rename --timezoneSource custom --timezoneCustom +00:00 --maxDatesDistance 5h00m --action print ~/Pictures/Photos
Loading configs...
Media Dir: /home/user/Pictures/Photos
Timezone Source: custom
Timezone Custom: +00:00
Max Dates Distance: 5h0m0s
Action: print

2019.05.09 09.43.59.dt (IMG_2597).jpg - ok filename
2020.01.01 00.00.01.dm (ae25edce-aab3-41ac-8916-8fe4bf19ce71).mp4 => 2020.07.22 17.21.51.dt (ae25edce-aab3-41ac-8916-8fe4bf19ce71).mp4 - 4889h21m50s - Warning: Old date and new date are too far, skipping
IMG_2024.heic => 2019.03.08 19.30.30.dt (IMG_2024).heic, printing

Summary
totalFilesFound: 4
totalMediaFound: 3
totalMediaOkFilenames: 1
totalMediaWarnings: 1
totalMediaActions: 1
```

Then rename it with `--action execute` option.

# Additional info
You can convert a lot media files on Windows with [IrfanView](https://www.irfanview.com/) (eg, ***.heic** => ***.jpg**).

You can copy exif-tags to converted files with [exiftool](https://exiftool.org/):
```
exiftool -tagsfromfile '%d%f.heic' -r -ext jpg -all:all -overwrite_original_in_place ~/Pictures/Photos
```

# Contributing
Feel free to create PRs :)

# Thanks
- Phil Harvey and the company, the authors of [exiftool](https://exiftool.org/), for the great command-line application
to read, write and edit meta information in a wide variety of files.
- Barasher, the author of [go-exiftool](https://github.com/barasher/go-exiftool), and contributors of the project
for the golang library that wraps ExifTool.
- Everything and everyone, what or who makes this world better :)

# Changelog
- v1.0.0
  - Implement a basic app and command "rename"
package commands

import (
	"fmt"
	"math"
	"os"
	filepathLib "path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	exifToolLib "github.com/barasher/go-exiftool"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var errDateTakenInExifNotFound = errors.New("date taken not found in exif")
var errDateTakenInOriginalFilenameNotFound = errors.New("date taken not found in original filename")

var formattedFilenameRep = regexp.MustCompile(`^(?P<year>\d{4})\.(?P<month>\d{2})\.(?P<day>\d{2}) ` +
	`(?P<hour>\d{2})\.(?P<min>\d{2})\.(?P<sec>\d{2}|xy)(?P<dateSource>\.[A-Za-z0-9]+)? ` +
	`\((?P<originalFilename>.*)\)$`)

var dateInFilenameRep = regexp.MustCompile(`(?P<year>\d{4}).?(?P<month>\d{2}).?(?P<day>\d{2}).?` +
	`(?P<hour>\d{2}).?(?P<min>\d{2}).?(?P<sec>\d{2})`)

var timezoneRep = regexp.MustCompile(`^(?P<timezoneSign>[+-])?(?P<timezoneHour>\d{2}):(?P<timezoneMin>\d{2})?$`)

var exifDateRep = regexp.MustCompile(`^(?P<year>\d{4}):(?P<month>\d{2}):(?P<day>\d{2}) ` +
	`(?P<hour>\d{2}):(?P<min>\d{2}):(?P<sec>\d{2})(\.(?P<msec>\d{2}))?` +
	`((?P<timezoneSign>\+|-)(?P<timezoneHour>\d{2}):(?P<timezoneMin>\d{2}))?$`)

var commentTimestampRep = regexp.MustCompile(`timestamp=(?P<timestamp>\d+)`)

var supportedExts = map[string]bool{
	"heic": true, // IPhone
	"jpeg": true, // IPhone
	"jpg":  true, // IPhone
	"gif":  true, // IPhone
	"png":  true,

	"3gp": true,
	"m4v": true,
	"mov": true, // IPhone
	"mp4": true, // IPhone
	"mpg": true,
	"mpo": true,
}

const (
	actionPrint   string = "print"
	actionExecute string = "execute"

	timezoneSourceMedia  string = "media"
	timezoneSourceCustom string = "custom"
)

var actions = map[string]bool{
	actionPrint:   true,
	actionExecute: true,
}

var timezoneSources = map[string]bool{
	timezoneSourceMedia:  true,
	timezoneSourceCustom: true,
}

func parseTimezone(timezoneString string) (loc *time.Location, err error) {
	timezoneSubmatch := timezoneRep.FindStringSubmatch(timezoneString)

	timezoneSign, err := strconv.ParseInt(timezoneSubmatch[1]+"1", 10, 32)
	if err != nil {
		err = errors.Wrap(err, "parse timezoneSign")
		return
	}
	timezoneHour, err := strconv.ParseInt(timezoneSubmatch[2], 10, 32)
	if err != nil {
		err = errors.Wrap(err, "parse timezoneHour")
		return
	}
	timezoneMin, err := strconv.ParseInt(timezoneSubmatch[3], 10, 32)
	if err != nil {
		err = errors.Wrap(err, "parse timezoneHour")
		return
	}

	loc = time.FixedZone(timezoneString, int(timezoneSign*timezoneHour*60*timezoneMin))
	return
}

func parseDate(yearString, monthString, dayString, hourString, minString, secString, msecString, timezoneString string) (date time.Time, err error) {
	year, err := strconv.ParseInt(yearString, 10, 32)
	if err != nil {
		err = errors.Wrap(err, "parse year")
		return
	}
	month, err := strconv.ParseInt(monthString, 10, 32)
	if err != nil {
		err = errors.Wrap(err, "parse month")
		return
	}
	day, err := strconv.ParseInt(dayString, 10, 32)
	if err != nil {
		err = errors.Wrap(err, "parse day")
		return
	}
	hour, err := strconv.ParseInt(hourString, 10, 32)
	if err != nil {
		err = errors.Wrap(err, "parse hour")
		return
	}
	min, err := strconv.ParseInt(minString, 10, 32)
	if err != nil {
		err = errors.Wrap(err, "parse min")
		return
	}
	sec, err := strconv.ParseInt(secString, 10, 32)
	if err != nil {
		err = errors.Wrap(err, "parse sec")
		return
	}

	var msec int64
	if len(msecString) > 0 {
		msec, err = strconv.ParseInt(msecString, 10, 32)
		if err != nil {
			err = errors.Wrap(err, "parse msec")
			return
		}
	}

	var timezone *time.Location
	if len(timezoneString) > 0 {
		timezone, err = parseTimezone(timezoneString)
	} else {
		timezone = time.UTC
	}

	if err != nil {
		err = errors.Wrap(err, "parseTimezone()")
		return
	}

	date = time.Date(int(year), time.Month(int(month)), int(day), int(hour), int(min), int(sec), int(msec*1000), timezone)
	return
}

func parseExifDate(dateString string, tags map[string]interface{}) (date time.Time, err error) {
	dateSubmatch := exifDateRep.FindStringSubmatch(dateString)

	yearString := dateSubmatch[1]
	monthString := dateSubmatch[2]
	dayString := dateSubmatch[3]
	hourString := dateSubmatch[4]
	minString := dateSubmatch[5]
	secString := dateSubmatch[6]
	msecString := dateSubmatch[8]

	timezoneString := dateSubmatch[8]
	if len(timezoneString) == 0 {
		offsetTags := []string{"OffsetTimeOriginal", "OffsetTimeDigitized", "OffsetTime"}
		timezoneString = findFirstTag(tags, offsetTags)
	}

	date, err = parseDate(yearString, monthString, dayString, hourString, minString, secString, msecString, timezoneString)
	return date, errors.Wrap(err, "parseDate()")
}

func parseDateFromFilename(filename string) (date time.Time, err error) {
	dateSubmatch := dateInFilenameRep.FindStringSubmatch(filename)

	if len(dateSubmatch) == 0 {
		err = errors.Wrap(errDateTakenInOriginalFilenameNotFound, "commands/rename->parseDateFromFilename(): FindStringSubmatch()")
		return
	}

	yearString := dateSubmatch[1]
	monthString := dateSubmatch[2]
	dayString := dateSubmatch[3]
	hourString := dateSubmatch[4]
	minString := dateSubmatch[5]
	secString := dateSubmatch[6]
	msecString := ""
	timezoneString := ""

	date, err = parseDate(yearString, monthString, dayString, hourString, minString, secString, msecString, timezoneString)
	return date, errors.Wrap(err, "parseDateFromFilename()")
}

func parseTimestamp(timestampString string) (date time.Time, err error) {
	timestampSubmatch := commentTimestampRep.FindStringSubmatch(timestampString)
	if len(timestampSubmatch) == 2 {
		var secs int64
		secs, err = strconv.ParseInt(timestampSubmatch[1], 10, 64)
		if err != nil {
			err = errors.Wrap(err, "ParseInt()")
			return
		}
		date = time.Unix(secs, 0)
	}
	return
}

func findFirstTag(fields map[string]interface{}, tags []string) (foundTagValue string) {
	for _, tag := range tags {
		if tagValue, tagFound := fields[tag]; tagFound {
			return tagValue.(string)
		}
	}
	return
}

func getNewFilename(dateTaken time.Time, dateSource string, originalFilename string) (newFilename string) {
	return fmt.Sprintf("%d.%02d.%02d %02d.%02d.%02d.%s (%s)",
		dateTaken.Year(), dateTaken.Month(), dateTaken.Day(),
		dateTaken.Hour(), dateTaken.Minute(), dateTaken.Second(), dateSource,
		originalFilename)
}

func parseFilename(filename string) (originalFilename string, date time.Time, err error) {
	originalFilenameSubmatch := formattedFilenameRep.FindStringSubmatch(filename)
	if len(originalFilenameSubmatch) == 0 {
		originalFilename = filename
		return
	}

	yearString := originalFilenameSubmatch[1]
	monthString := originalFilenameSubmatch[2]
	dayString := originalFilenameSubmatch[3]
	hourString := originalFilenameSubmatch[4]
	minString := originalFilenameSubmatch[5]
	secString := originalFilenameSubmatch[6]
	// TODO: remove
	if secString == "xy" {
		secString = "00"
	}
	msecString := ""
	timezoneString := ""
	originalFilename = originalFilenameSubmatch[8]

	date, err = parseDate(yearString, monthString, dayString, hourString, minString, secString, msecString, timezoneString)
	return
}

func processExif(path string, exifTool *exifToolLib.Exiftool) (dateTaken time.Time, err error) {
	fileInfos := exifTool.ExtractMetadata(path)
	if len(fileInfos) != 1 {
		err = errors.New("len(fileInfos) != 1")
		return
	}

	fileInfo := fileInfos[0]
	if fileInfo.Err != nil {
		err = errors.Wrap(fileInfo.Err, "commands/rename->processExif(): fileInfo.Err")
		return
	}

	accurateDateTags := []string{"CreateDate", "DateTimeOriginal", "CreationDate", "MediaCreateDate"}
	accurateDateValue := findFirstTag(fileInfo.Fields, accurateDateTags)

	if len(accurateDateValue) > 0 {
		dateTaken, err = parseExifDate(accurateDateValue, fileInfo.Fields)
		if err != nil {
			err = errors.Wrap(err, "parseExifDate()")
			return
		}

	} else if commentI, commentFound := fileInfo.Fields["Comment"]; commentFound {
		dateTaken, err = parseTimestamp(commentI.(string))
		if err != nil {
			err = errors.Wrap(err, "parseTimestamp()")
			return
		}

	} else {
		err = errDateTakenInExifNotFound
		return
	}

	return
}

func checkDatesTooFar(distance time.Duration, maxDatesDistanceMilliseconds int64) (tooFar bool) {
	change := math.Abs(float64(distance.Milliseconds()))
	if change > float64(maxDatesDistanceMilliseconds) {
		return true
	}
	return false
}

func runRename(cmd *cobra.Command, args []string) (err error) {
	mediaDir := args[0]

	timezoneSource, err := cmd.Flags().GetString("timezoneSource")
	if enabled, found := timezoneSources[timezoneSource]; !found || !enabled {
		return errors.New("unknown timezoneSource")
	}

	timezoneCustomString, err := cmd.Flags().GetString("timezoneCustom")
	if err != nil {
		return errors.Wrap(err, `commands/rename->runRename(): GetString("timezoneCustom")`)
	}
	timezoneCustom, err := parseTimezone(timezoneCustomString)
	if err != nil {
		return errors.Wrap(err, `commands/rename->runRename(): parseTimezone(timezoneCustomString)`)
	}

	maxDatesDistance, err := cmd.Flags().GetDuration("maxDatesDistance")
	if err != nil {
		return errors.Wrap(err, `commands/rename->runRename(): GetDuration("maxDatesDistance")`)
	}

	action, err := cmd.Flags().GetString("action")
	if enabled, found := actions[action]; !found || !enabled {
		return errors.New("unknown action")
	}

	fmt.Printf("Media Dir: %v\n", mediaDir)
	fmt.Printf("Timezone Source: %v\n", timezoneSource)
	fmt.Printf("Timezone Custom: %v\n", timezoneCustomString)
	fmt.Printf("Max Dates Distance: %v\n", maxDatesDistance)
	fmt.Printf("Action: %v\n\n", action)

	maxDatesDistanceInMilliseconds := maxDatesDistance.Milliseconds()

	exifTool, err := exifToolLib.NewExiftool(exifToolLib.Charset("filename=utf8"))
	if err != nil {
		return errors.Wrap(err, "NewExiftool()")
	}
	defer func() {
		closeErr := exifTool.Close()
		if closeErr != nil {
			err = errors.Wrap(closeErr, "exifTool.Close()")
		}
	}()

	totalFilesFound := 0
	totalMediaFound := 0
	totalMediaOkFilenames := 0
	totalMediaWarnings := 0
	totalMediaActions := 0

	err = filepathLib.Walk(mediaDir, func(curFilepath string, curFileInfo os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrap(err, "err")
		}

		if curFileInfo.IsDir() {
			return nil
		}

		totalFilesFound++
		dir, filenameWithExt := filepathLib.Split(curFilepath)
		extOriginal := filepathLib.Ext(filenameWithExt)
		filename := strings.TrimSuffix(filenameWithExt, extOriginal)
		ext := strings.ToLower(strings.TrimLeft(extOriginal, "."))

		originalFilename, oldDateTaken, err := parseFilename(filename)
		if err != nil {
			return errors.Wrap(err, "parseFilename()")
		}

		if enabled, found := supportedExts[ext]; !found || !enabled {
			return nil
		}

		totalMediaFound++
		dateSource := "dt"
		dateTaken, err := processExif(curFilepath, exifTool)
		if err != nil && err != errDateTakenInExifNotFound {
			return errors.Wrap(err, "processExif()")
		}

		if err == errDateTakenInExifNotFound {
			dateTaken, err = parseDateFromFilename(originalFilename)
			if err != nil && errors.Cause(err) != errDateTakenInOriginalFilenameNotFound {
				return errors.Wrap(err, "parseDateFromFilename()")
			}
			dateSource = "dn"
		}

		if err != nil {
			dateTaken = curFileInfo.ModTime()
			dateSource = "dmz"
			err = nil
		}

		if timezoneSource == timezoneSourceCustom {
			dateTaken = dateTaken.In(timezoneCustom)

			if dateSource == "dmz" {
				dateSource = "dm"
			}
		}

		newFilename := getNewFilename(dateTaken, dateSource, originalFilename)

		if filename == newFilename {
			fmt.Printf("%s - ok filename\n", filenameWithExt)
			totalMediaOkFilenames++
			return nil
		}

		newFilenameWithExt := newFilename + "." + ext
		newFilepath := fmt.Sprintf("%s%s", dir, newFilenameWithExt)

		fmt.Printf("%s => %s", filenameWithExt, newFilenameWithExt)

		var datesTooFar = false
		if oldDateTaken != (time.Time{}) {
			distance := dateTaken.Sub(oldDateTaken)
			fmt.Printf(" - %v", distance)
			datesTooFar = checkDatesTooFar(distance, maxDatesDistanceInMilliseconds)
		}

		if datesTooFar {
			fmt.Println(" - Warning: Old date and new date are too far, skipping")
			totalMediaWarnings++
			return nil
		}

		totalMediaActions++

		switch action {
		case actionPrint:
			fmt.Print(", printing")
			break
		case actionExecute:
			fmt.Print(", renaming")
			err = os.Rename(curFilepath, newFilepath)
			if err != nil {
				return errors.Wrap(err, "os.Rename()")
			}
			break
		}

		fmt.Println()

		return nil
	})

	fmt.Printf("\nSummary\n"+
		"totalFilesFound: %d\n"+
		"totalMediaFound: %d\n"+
		"totalMediaOkFilenames: %d\n"+
		"totalMediaWarnings: %d\n"+
		"totalMediaActions: %d\n",
		totalFilesFound,
		totalMediaFound,
		totalMediaOkFilenames,
		totalMediaWarnings,
		totalMediaActions)

	return errors.Wrap(err, "filepath.Walk()")
}

func GetRenameCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rename <mediaDir>",
		Short: "Rename media files according to their probable date taken",
		Args:  cobra.ExactArgs(1),
		RunE:  runRename,
	}

	cmd.Flags().StringP("timezoneSource", "s", timezoneSourceMedia, `The source of timezone: "media" or "custom". Use parameter "timezoneCustom" to set the custom timezone`)
	cmd.Flags().StringP("timezoneCustom", "z", "00:00", "Timezone for date in new filename")
	cmd.Flags().DurationP("maxDatesDistance", "d", 26*time.Hour, "Maximum time distance between date in old filename and date in new filename")
	cmd.Flags().StringP("action", "a", actionPrint, `Action to do with media: "print" or "execute" renaming`)
	//_ = cmd.MarkFlagRequired("maxDatesDistance")

	return cmd
}

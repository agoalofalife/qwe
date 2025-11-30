package display

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	cp "github.com/mainak55512/qwe/compressor"
	"github.com/mainak55512/qwe/qwerror"
	utl "github.com/mainak55512/qwe/qweutils"
	tr "github.com/mainak55512/qwe/tracker"
)

// later add to configs and user might set itself
var excludedDirs = map[string]struct{}{
	".git": {},
	".qwe": {},
}

const (
	FileName              = "_track_files.qwe"
	QweDir                = ".qwe"
	MaxCommitMessageLen   = 60
	CommitMessageTruncLen = 57
	CommitTimeLayout      = time.DateTime
	TrackFilePermissions  = 0o644
)

// todo check all underscore variables if exists - golang convention recheck

// Check if file with tracked path files is exists
// if File exists, read all records and displau the information immediately

// otherwise, recursively search among all files in the root qwe directory and match store hashing with already tracked files
// if it is matched, add new record to the file, if not skip it
// when all files are scanned, store this information in a file, and display all tracked files

// after adding new track files, add new record to the _tracker.qwe file

type TrackFiles map[string]*TrackFile

func (tf TrackFiles) Print() error {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "Path\tCommits count\tLast\tLast Message")

	for _, trackFile := range tf {
		commitTime, err := parseCommitTime(trackFile.LastCommitTime)

		if err != nil {
			return fmt.Errorf("parse commit time for %s: %w", trackFile.FilePath, err)
		}
		fmt.Fprintf(w, "%s\t%d\t%s\t%s\n",
			trackFile.FilePath,
			trackFile.CommitCount,
			relativeTime(commitTime),
			truncateMsg(trackFile.LastCommitMessage),
		)
	}
	return w.Flush()
}

func (tf TrackFiles) Save(filename string) error {
	bytes, err := json.MarshalIndent(tf, "", "  ")

	if err != nil {
		return fmt.Errorf("marshal tracked files: %w", err)
	}

	// atomic write
	filePath := filepath.Join(QweDir, filename)
	tmp := filePath + ".tmp"

	if err := os.WriteFile(tmp, bytes, TrackFilePermissions); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := os.Rename(tmp, filePath); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}

	if err = cp.CompressFile(filePath); err != nil {
		return fmt.Errorf("compress file: %w", err)
	}
	return nil
}

type TrackFile struct {
	FilePath          string `json:"filepath"`
	CommitCount       int    `json:"commit_count"`
	LastCommitMessage string `json:"last_commit_message"`
	LastCommitTime    string `json:"last_commit_time"`
}

func NewTrackFile(filepath string) *TrackFile {
	return &TrackFile{
		FilePath:          filepath,
		CommitCount:       0,
		LastCommitMessage: "-",
		LastCommitTime:    "-",
	}
}

func relativeTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}

	d := time.Since(t)

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func PrintTrackedFiles() error {
	if !utl.QweIsInWorkingDir() {
		return qwerror.RepoNotFound
	}

	trackFilePath := filepath.Join(QweDir, FileName)

	tf, err := loadOrScanTrackedFiles(trackFilePath)
	if err != nil {
		return err
	}

	return tf.Print()
}

func loadOrScanTrackedFiles(filePath string) (TrackFiles, error) {
	if utl.FileExists(filePath) {
		return LoadTrackedFilesFromFile(filePath)
	}
	workingDir, err := os.Getwd()

	if err != nil {
		return nil, err
	}

	trackedFiles, err := findTrackedFilesInCurrDir(workingDir)

	if err != nil {
		return nil, err
	}
	err = trackedFiles.Save(filePath)

	if err != nil {
		return nil, fmt.Errorf("save tracked files: %w", err)
	}

	return trackedFiles, nil
}

// scanning all directories and files to find appropriate files
func findTrackedFilesInCurrDir(dir string) (TrackFiles, error) {
	trackedFiles := make(TrackFiles)
	tracker, _, err := tr.GetTracker(0)

	if err != nil {
		return nil, err
	}

	fmt.Println("Scanning files in", dir, "...")

	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		// exclude qwe itself, git also
		if err != nil {
			if os.IsPermission(err) {
				fmt.Printf("Permission denied: %s\n", path)
				return nil
			}
			fmt.Printf("Error accessing path %q: %v\n", path, err)
			return err
		}

		if d.IsDir() {
			if isExcludedDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}

		// is file
		fileId := utl.Hasher(d.Name())

		if trackInfo, ok := tracker[fileId]; ok {

			if len(trackInfo.Versions) == 0 {
				trackedFiles[fileId] = NewTrackFile(path)
			} else {
				trackedFiles[fileId] = &TrackFile{
					FilePath:          path,
					CommitCount:       trackInfo.CommitsCount(),
					LastCommitMessage: trackInfo.LastVersion().CommitMessage,
					LastCommitTime:    trackInfo.LastVersion().TimeStamp,
				}
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return trackedFiles, nil
}

func LoadTrackedFilesFromFile(filename string) (TrackFiles, error) {
	trackedFiles := make(TrackFiles)

	if err := cp.DecompressFile(filename); err != nil {
		return nil, err
	}

	// If file doesn't exist, return empty map
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return trackedFiles, nil
	} else if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}

	bytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	if err := json.Unmarshal(bytes, &trackedFiles); err != nil {
		return nil, fmt.Errorf("unmarshal tracked files: %w", err)
	}
	if err := cp.CompressFile(filename); err != nil {
		return nil, err
	}
	return trackedFiles, nil
}

func truncateMsg(msg string) string {
	if len(msg) > MaxCommitMessageLen {
		return msg[:CommitMessageTruncLen] + "..."
	}
	return msg
}

func parseCommitTime(timeStr string) (time.Time, error) {
	// Handle the ":00" suffix issue properly
	if timeStr == "-" {
		return time.Time{}, nil
	}
	// I think we have to remain Datetime format not trimmed version in tracker from here commit/commit.go:135
	return time.Parse(CommitTimeLayout, timeStr+":00")
}
func isExcludedDir(name string) bool {
	_, ok := excludedDirs[name]
	return ok
}

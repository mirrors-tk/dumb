package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/rclone/rclone/fs"
)

type Repo struct {
	Name     string `json:"name"`
	IsMaster bool   `json:"is_master"`
	Upstream string `json:"upstream"`
	Status   Status `json:"status"`

	LastUpdateISO8601     string `json:"last_update" status:"success"`
	LastUpdateTimestamp   int64  `json:"last_update_ts"`
	LastStartedISO8601    string `json:"last_started" status:"syncing"`
	LastStartedTimestamp  int64  `json:"last_started_ts"`
	LastEndedISO8601      string `json:"last_ended" status:"success,failed"`
	LastEndedTimestamp    int64  `json:"last_ended_ts"`
	NextScheduleISO8601   string `json:"next_schedule" status:"pending"`
	NextScheduleTimestamp int64  `json:"next_schedule_ts"`
	SizeHumanReadable     string `json:"size"`
	SizeBytes             int64  `json:"size_bytes"`
}

type Status string

const (
	Pending Status = "pending"
	Syncing Status = "syncing"
	Success Status = "success"
	Failed  Status = "failed"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func run() (err error) {
	var status Status
	if len(os.Args) > 2 {
		status = Status(os.Args[1])
	}

	switch status {
	case Syncing, Success, Failed:
	default:
		return fmt.Errorf(
			"usage: dumb {syncing|success|failed} <repo> [<size> [<next schedule>]] < src.json > dest.json")
	}

	var (
		repoName = os.Args[2]
		repoSize int64
		schedule time.Duration
	)
	if len(os.Args) > 3 {
		if repoSize, err = strconv.ParseInt(os.Args[3], 10, 64); err != nil {
			return err
		}
	}
	if len(os.Args) > 4 {
		if schedule, err = time.ParseDuration(os.Args[4]); err != nil {
			return err
		}
	}

	var repoList []*Repo
	if err = json.NewDecoder(os.Stdin).Decode(&repoList); err != nil {
		return err
	}

	repoList = append(repoList, &Repo{Name: repoName, IsMaster: true})
	for i, repo := range repoList {
		if repo.Name != repoName {
			continue
		}

		repo.Status = status
		if repoSize != 0 {
			repo.SizeBytes = repoSize
			repo.SizeHumanReadable = fs.SizeSuffix(repoSize).ByteUnit()
		}

		now := time.Now()
		if schedule != 0 {
			setTime(repo, Pending, now.Add(schedule))
		}
		setTime(repo, status, now)

		if i+1 < len(repoList) {
			repoList = repoList[:len(repoList)-1]
		}

		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "\t")
		return encoder.Encode(repoList)
	}
	return nil
}

func setTime(repo *Repo, status Status, t time.Time) {
	r := reflect.TypeOf(*repo)
	for i := 0; i < r.NumField(); i++ {
		f := r.Field(i)
		if !strings.Contains(f.Tag.Get("status"), string(status)) {
			continue
		}
		reflect.ValueOf(repo).Elem().Field(i).SetString(t.Format(time.RFC3339))
		reflect.ValueOf(repo).Elem().Field(i + 1).SetInt(t.Unix())
	}
}

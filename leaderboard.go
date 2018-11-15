// Package leaderboard interacts with Advent of Code [0] private leaderboards. It can retrieve the
// JSON formatted leaderboards and convert them to usable slices of leaderboard Members.
//
// [0]: https://adventofcode.com/
//
// Written by: Michiel Appelman <michiel@appelman.se>
package leaderboard

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	resty "gopkg.in/resty.v1"
)

type LeaderboardSort int

const (
	NoSort = iota
	SortByLocalScore
	SortByGlobalScore
	SortByStars
)
const timeLayout = "2006-01-02T15:04:05-0700"

// JSONTime is a custom time struct for ISO8601 type in JSON
type JSONTime struct {
	time.Time
}

func (t *JSONTime) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), "\"")
	if s == "null" {
		t.Time = time.Time{}
		return
	}
	i, err := strconv.ParseInt(s, 10, 64)
	t.Time = time.Unix(i, 0)
	return
}

// Define the Leaderboard JSON structure
type Leaderboard struct {
	OwnerID string            `json:"owner_id"`
	Event   string            `json:"event"`
	Members map[string]Member `json:"members"`
}

type Member struct {
	ID          string                      `json:"id"`
	Name        string                      `json:"name"`
	Stars       int                         `json:"stars"`
	LocalScore  int                         `json:"local_score"`
	GlobalScore int                         `json:"global_score"`
	LastStarTS  JSONTime                    `json:"last_star_ts"`
	Days        map[string]map[string]Level `json:"completion_day_level"`
}

type Level struct {
	Timestamp JSONTime `json:"get_star_ts"`
}

type membersSortedByLocalScore []Member

func (m membersSortedByLocalScore) Len() int      { return len(m) }
func (m membersSortedByLocalScore) Swap(i, j int) { m[i], m[j] = m[j], m[i] }
func (m membersSortedByLocalScore) Less(i, j int) bool {
	if m[i].LocalScore < m[j].LocalScore {
		return true
	}
	if m[i].LocalScore > m[j].LocalScore {
		return false
	}
	return m[i].Stars < m[j].Stars
}

type membersSortedByGlobalScore []Member

func (m membersSortedByGlobalScore) Len() int      { return len(m) }
func (m membersSortedByGlobalScore) Swap(i, j int) { m[i], m[j] = m[j], m[i] }
func (m membersSortedByGlobalScore) Less(i, j int) bool {
	if m[i].GlobalScore < m[j].GlobalScore {
		return true
	}
	if m[i].GlobalScore > m[j].GlobalScore {
		return false
	}
	return m[i].LocalScore < m[j].LocalScore
}

type membersSortedByStars []Member

func (m membersSortedByStars) Len() int      { return len(m) }
func (m membersSortedByStars) Swap(i, j int) { m[i], m[j] = m[j], m[i] }
func (m membersSortedByStars) Less(i, j int) bool {
	if m[i].Stars < m[j].Stars {
		return true
	}
	if m[i].Stars > m[j].Stars {
		return false
	}
	return m[i].LocalScore < m[j].LocalScore
}

func JSONToNormalTime(jt JSONTime) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, jt.Format(time.RFC3339))
	if err != nil {
		return time.Now(), errors.New("could not convert JSON time")
	}
	return t, nil
}

// GetMembers returns a slice of private leaderboard Members sorted by a sorting function
// (SortByLocalScore, SortByGlobalScore or SortByStars) given the private leaderboard ID, a session
// cookie and the year of the Advent of Code challenge.
func GetMembers(lbID int, cookie string, year int, sorted LeaderboardSort) ([]Member, error) {
	resp, err := resty.R().
		SetHeader("Accept", "application/json").
		SetHeader("Cookie", fmt.Sprintf("session=%s", cookie)).
		SetResult(Leaderboard{}).
		Get(fmt.Sprintf("https://adventofcode.com/%d/leaderboard/private/view/%d.json", year, lbID))
	if err != nil {
		return nil, err
	}
	switch {
	case resp.StatusCode() == 500:
		return nil, errors.New("Advent of Code server error, wrong cookie perhaps?")
	case resp.StatusCode() != 200:
		return nil, fmt.Errorf("error connecting to Advent of Code, HTTP code %d", resp.StatusCode())
	}

	lb := resp.Result().(*Leaderboard)
	var members []Member

	for _, member := range lb.Members {
		members = append(members, member)
		if err != nil {
			return nil, err
		}
	}
	switch sorted {
	case SortByLocalScore:
		sort.Sort(sort.Reverse(membersSortedByLocalScore(members)))
	case SortByGlobalScore:
		sort.Sort(sort.Reverse(membersSortedByGlobalScore(members)))
	case SortByStars:
		sort.Sort(sort.Reverse(membersSortedByStars(members)))
	}
	return members, nil
}

// CountTotalStars counts the total number of stars from the given slice of Members.
func CountTotalStars(members []Member) int {
	stars := 0
	for _, m := range members {
		stars += m.Stars
	}
	return stars
}

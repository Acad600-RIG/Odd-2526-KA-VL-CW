package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kennethandrew67/go-backend/model"
)

// GetTeachingJob proxies the job list from Bluejack LAPI for a given assistant username.
func GetTeachingJob(c *gin.Context) {
	username := c.Query("username")
	mode := c.DefaultQuery("mode", "current")
	semesterID := getActiveSemester()

	if username == "" || semesterID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing username or failed to get active semester",
		})
		return
	}

	params := url.Values{}
	params.Add("username", username)
	params.Add("semesterId", semesterID)
	params.Add("mode", mode)

	req, err := http.NewRequest(http.MethodGet, "https://bluejack.binus.ac.id/lapi/api/Schedule/GetJobByAssistant"+"?"+params.Encode(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create HTTP request",
			"details": err.Error(),
		})
		return
	}

	token, err := GetBearerToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to obtain bearer token",
			"details": err.Error(),
		})
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to call external API",
			"details": err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.JSON(http.StatusBadGateway, gin.H{
			"error":  "External API returned non-200",
			"status": resp.Status,
			"body":   string(body),
		})
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to read API response",
			"details": err.Error(),
		})
		return
	}

	var jobs []model.Job
	if err := json.Unmarshal(body, &jobs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to parse API response",
			"details": err.Error(),
		})
		return
	}

	filtered := make([]model.Job, 0, len(jobs))
	for _, j := range jobs {
		if strings.EqualFold(j.JobType, "Teaching") {
			subject, classCode, room := parseDescription(j.Description)
			if subject != "" {
				j.Subject = subject
			}
			if classCode != "" {
				j.Class = classCode
			}
			if room != "" {
				j.Room = room
			}
			filtered = append(filtered, j)
		}
	}

	c.JSON(http.StatusOK, filtered)
}

// GetNextTeachingRoom returns the next applicable teaching job (by shift) for today along with the room.
// Rules:
// - If now is before the first shift start, pick shift 1.
// - If now falls within a shift window, pick the next shift (unless it is the last shift, then none).
// - If now is between shifts, pick the upcoming shift.
// - Only considers teaching jobs for today.
func GetNextTeachingRoom(c *gin.Context) {
	username := c.Query("username")
	mode := c.DefaultQuery("mode", "current")
	semesterID := getActiveSemester()

	if username == "" || semesterID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing username or failed to get active semester"})
		return
	}

	jobs, err := fetchTeachingJobs(username, semesterID, mode)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	desiredShift := nextShiftForNow(now)
	if desiredShift == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No upcoming shift available today"})
		return
	}

	for _, j := range jobs {
		jobTime, shift := parseStartToShift(j.StartDate)
		if shift == 0 {
			continue
		}
		if sameDay(jobTime, now) && shift == desiredShift {
			// Enrich subject/class/room from description if needed
			subject, classCode, room := parseDescription(j.Description)
			if subject != "" {
				j.Subject = subject
			}
			if classCode != "" {
				j.Class = classCode
			}
			if room != "" {
				j.Room = room
			}

			c.JSON(http.StatusOK, gin.H{
				"shift": desiredShift,
				"room":  j.Room,
				"job":   j,
			})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "No teaching job found for the upcoming shift today"})
}

// fetchTeachingJobs retrieves and filters teaching jobs for a user.
func fetchTeachingJobs(username, semesterID, mode string) ([]model.Job, error) {
	params := url.Values{}
	params.Add("username", username)
	params.Add("semesterId", semesterID)
	params.Add("mode", mode)

	req, err := http.NewRequest(http.MethodGet, "https://bluejack.binus.ac.id/lapi/api/Schedule/GetJobByAssistant"+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	token, err := GetBearerToken()
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("external API returned %s: %s", resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var jobs []model.Job
	if err := json.Unmarshal(body, &jobs); err != nil {
		return nil, err
	}

	filtered := make([]model.Job, 0, len(jobs))
	for _, j := range jobs {
		if strings.EqualFold(j.JobType, "Teaching") {
			filtered = append(filtered, j)
		}
	}
	return filtered, nil
}

// nextShiftForNow determines the desired shift number based on current time and fixed shift windows.
// Returns 0 if no further shifts today.
func nextShiftForNow(now time.Time) int {
	shifts := []struct {
		shift int
		start string
		end   string
	}{
		{1, "07:20", "09:00"},
		{2, "09:20", "11:00"},
		{3, "11:20", "13:00"},
		{4, "13:20", "15:00"},
		{5, "15:20", "17:00"},
		{6, "17:20", "19:00"},
	}

	loc := now.Location()

	for i, s := range shifts {
		start := mustTime(now, s.start, loc)
		end := mustTime(now, s.end, loc)

		if now.Before(start) {
			return s.shift
		}
		if !now.Before(start) && now.Before(end) {
			// Inside this shift, pick the next one if exists
			if i+1 < len(shifts) {
				return shifts[i+1].shift
			}
			return 0
		}
	}

	return 0
}

// parseStartToShift parses start date string and maps it to a shift number based on the fixed windows.
func parseStartToShift(startStr string) (time.Time, int) {
	loc := time.Local
	parsed, err := time.ParseInLocation("2006-01-02T15:04:05", startStr, loc)
	if err != nil {
		return time.Time{}, 0
	}

	shift := shiftFromTime(parsed)
	return parsed, shift
}

// shiftFromTime returns the shift number that matches the start time.
func shiftFromTime(t time.Time) int {
	shifts := map[string]int{
		"07:20": 1,
		"09:20": 2,
		"11:20": 3,
		"13:20": 4,
		"15:20": 5,
		"17:20": 6,
	}

	key := t.Format("15:04")
	return shifts[key]
}

// sameDay checks if two times fall on the same calendar day.
func sameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.YearDay() == b.YearDay()
}

// mustTime builds a time.Time on the given date with hh:mm (seconds=0).
func mustTime(ref time.Time, hhmm string, loc *time.Location) time.Time {
	parts := strings.Split(hhmm, ":")
	if len(parts) != 2 {
		return ref
	}
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	return time.Date(ref.Year(), ref.Month(), ref.Day(), h, m, 0, 0, loc)
}

// parseDescription extracts subject, class, and room from a description string, ignoring a trailing numeric token.
// Example: "MATH6183001-Scientific Computing BA09  622 1" -> subject "MATH6183001-Scientific Computing", class "BA09", room "622".
func parseDescription(desc string) (subject, classCode, room string) {
	parts := strings.Fields(desc)
	if len(parts) == 0 {
		return "", "", ""
	}

	// Drop the last token if present (often an ordering/index number we don't need).
	if len(parts) > 0 {
		parts = parts[:len(parts)-1]
	}

	if len(parts) >= 2 {
		room = parts[len(parts)-1]
		classCode = parts[len(parts)-2]
		if len(parts) > 2 {
			subject = strings.Join(parts[:len(parts)-2], " ")
		}
	} else if len(parts) == 1 {
		subject = parts[0]
	}

	return subject, classCode, room
}
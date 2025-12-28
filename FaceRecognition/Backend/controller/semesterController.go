package controller

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/kennethandrew67/go-backend/model"
)

func getActiveSemester() string {
	baseURL := "https://bluejack.binus.ac.id/lapi/api/Semester/Active"
	resp, err := http.Get(baseURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	var result model.SemesterResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return ""
	}

	return result.SemesterId
}
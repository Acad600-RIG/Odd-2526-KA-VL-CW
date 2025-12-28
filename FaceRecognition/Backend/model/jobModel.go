package model

// Job mirrors fields returned by the Bluejack LAPI job endpoint.
// Some fields may be empty depending on the job type.
type Job struct {
	Description              string `json:"Description"`
	StartDate                string `json:"StartDate"`
	EndDate                  string `json:"EndDate"`
	Status                   string `json:"Status"`
	JobType                  string `json:"JobType"`
	ClassTransactionDetailID string `json:"ClassTransactionDetailId"`

	Campus      string `json:"Campus"`
	Class       string `json:"Class"`
	Day         int    `json:"Day"`
	Realization string `json:"Realization"`
	Room        string `json:"Room"`
	Shift       string `json:"Shift"`
	Subject     string `json:"Subject"`
}
package models

type ProgramInfo struct {
	Name         string `json:"name"`
	Classrooms   int    `json:"classrooms"`
	Laboratories int    `json:"laboratories"`
}

type AllocateRequest struct {
	Semester string `json:"semester"`
	Faculty  string `json:"faculty"`

	Programs []ProgramInfo `json:"programs"`
}

type ProgramAllocation struct {
	Name         string   `json:"name"`
	Classrooms   []string `json:"classrooms"`
	Laboratories []string `json:"laboratories"`
	Adapted      []string `json:"adapted"`
}

type AllocateResponse struct {
	Semester string `json:"semester"`
	Faculty  string `json:"faculty"`

	Programs []ProgramAllocation `json:"programs"`
}

type ConfirmRequest struct {
	Semester string `json:"semester"`
	Faculty  string `json:"faculty"`
	Accept   bool   `json:"accept"`
}

type ConfirmResponse struct {
	Semester string `json:"semester"`
	Faculty  string `json:"faculty"`
}

package folders

import (
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	DB *pgxpool.Pool
}

type requestError struct {
	status  int
	message string
}

func (e *requestError) Error() string {
	return e.message
}

type optionalString struct {
	Present bool
	Value   *string
}

func (o *optionalString) UnmarshalJSON(data []byte) error {
	o.Present = true
	if string(data) == "null" {
		o.Value = nil
		return nil
	}
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	o.Value = &value
	return nil
}

type createPayload struct {
	Name     string  `json:"name"`
	ParentID *string `json:"parentId"`
	TeamID   *string `json:"teamId"`
}

type updatePayload struct {
	Name     optionalString `json:"name"`
	ParentID optionalString `json:"parentId"`
}

type folderResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	ParentID  *string   `json:"parentId"`
	SortOrder int       `json:"sortOrder"`
	TeamID    *string   `json:"teamId,omitempty"`
	TeamName  *string   `json:"teamName,omitempty"`
	Scope     string    `json:"scope"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type listResponse struct {
	Personal []folderResponse `json:"personal"`
	Team     []folderResponse `json:"team"`
}

type accessResult struct {
	Folder   folderResponse
	TeamRole *string
}

func normalizeListResponse(resp listResponse) listResponse {
	if resp.Personal == nil {
		resp.Personal = []folderResponse{}
	}
	if resp.Team == nil {
		resp.Team = []folderResponse{}
	}
	return resp
}

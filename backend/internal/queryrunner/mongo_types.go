package queryrunner

import "go.mongodb.org/mongo-driver/v2/mongo"

type mongoQuerySpec struct {
	Database   string           `json:"database,omitempty"`
	Collection string           `json:"collection,omitempty"`
	Operation  string           `json:"operation"`
	Filter     map[string]any   `json:"filter,omitempty"`
	Projection map[string]any   `json:"projection,omitempty"`
	Sort       map[string]any   `json:"sort,omitempty"`
	Limit      int64            `json:"limit,omitempty"`
	Skip       int64            `json:"skip,omitempty"`
	Pipeline   []map[string]any `json:"pipeline,omitempty"`
	Document   map[string]any   `json:"document,omitempty"`
	Documents  []map[string]any `json:"documents,omitempty"`
	Update     map[string]any   `json:"update,omitempty"`
	Command    map[string]any   `json:"command,omitempty"`
	Field      string           `json:"field,omitempty"`
}

type mongoTargetConn struct {
	client   *mongo.Client
	database *mongo.Database
}

var mongoOperationAliases = map[string]string{
	"find":                   "find",
	"aggregate":              "aggregate",
	"count":                  "count",
	"countdocument":          "count",
	"countdocuments":         "count",
	"estimateddocumentcount": "count",
	"distinct":               "distinct",
	"insertone":              "insertone",
	"insertmany":             "insertmany",
	"updateone":              "updateone",
	"updatemany":             "updatemany",
	"deleteone":              "deleteone",
	"deletemany":             "deletemany",
	"runcmd":                 "runcommand",
	"runcommand":             "runcommand",
}

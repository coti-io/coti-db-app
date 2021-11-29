package dto

type SyncResponse struct {
	NodeTipIndex              int64  `json: "nodeTipIndex"`
	SyncIterationNodeTipIndex int64  `json: "syncIterationNodeTipIndex"`
	LastMonitoredIndex        int64  `json: "lastMonitoredIndex"`
	SyncPercentage            int64  `json: "syncPercentage"`
	Description               string `json: "description"`
}

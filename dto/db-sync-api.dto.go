package dto

type SyncResponse struct {
	NodeMaxIndex                      int64   `json:"nodeMaxIndex"`
	NodeLastIndex                     int64   `json:"nodeLastIndex"`
	BackupNodeLastIndex               int64   `json:"backupNodeLastIndex"`
	SyncIterationLastTransactionIndex int64   `json:"SyncIterationLastTransactionIndex"`
	LastMonitoredTransactionIndex     int64   `json:"lastMonitoredTransactionIndex"`
	SyncPercentage                    float64 `json:"syncPercentage"`
	IsNodeSynced                      bool    `json:"isNodeSynced"`
}

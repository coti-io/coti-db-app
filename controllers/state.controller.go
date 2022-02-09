package controllers

import (
	"math"
	"net/http"
	"strconv"

	"github.com/coti-io/coti-db-app/dto"
	"github.com/coti-io/coti-db-app/entities"
	service "github.com/coti-io/coti-db-app/services"

	dbprovider "github.com/coti-io/coti-db-app/db-provider"

	"github.com/gin-gonic/gin"
)

// GetSyncState Get all books
func GetSyncState(c *gin.Context) {
	transactionService := service.NewTransactionService()
	// check both nodes for last index
	syncHistory := transactionService.GetSyncHistory()

	nodeLastIndex := int64(math.Max(float64(syncHistory.LastIndexMainNode), float64(syncHistory.LastIndexBackupNode)))
	syncIterationLastTransactionIndex := transactionService.GetLastIteration()
	var appState entities.AppState
	dbprovider.DB.Where("name = ?", entities.LastMonitoredTransactionIndex).First(&appState)
	var lastMonitoredIndex int64
	lastMonitoredIndexInt, err := strconv.Atoi(appState.Value)
	if err != nil {
		panic(err)
	}
	lastMonitoredIndex = int64(lastMonitoredIndexInt)
	syncPercentage := (float64(lastMonitoredIndex) / float64(syncIterationLastTransactionIndex)) * 100

	c.JSON(http.StatusOK, gin.H{"data": dto.SyncResponse{NodeMaxIndex: nodeLastIndex, NodeLastIndex: syncHistory.LastIndexMainNode, BackupNodeLastIndex: syncHistory.LastIndexBackupNode, SyncIterationLastTransactionIndex: syncIterationLastTransactionIndex, LastMonitoredTransactionIndex: lastMonitoredIndex, SyncPercentage: syncPercentage, IsNodeSynced: syncHistory.IsSynced}})
}

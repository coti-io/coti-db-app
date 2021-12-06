package controllers

import (
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
	nodeTip := transactionService.GetTip().LastIndex
	syncIterationNodeIndexTip := transactionService.GetLastIterationTip()
	var appState entities.AppState
	dbprovider.DB.Where("name = ?", entities.LastMonitoredTransactionIndex).First(&appState)
	var lastMonitoredIndex int64
	lastMonitoredIndexInt, err := strconv.Atoi(appState.Value)
	if err != nil {
		panic(err)
	}
	lastMonitoredIndex = int64(lastMonitoredIndexInt)
	description := ``
	syncPercentage := (lastMonitoredIndex / syncIterationNodeIndexTip) * 100

	c.JSON(http.StatusOK, gin.H{"data": dto.SyncResponse{nodeTip, syncIterationNodeIndexTip, lastMonitoredIndex, syncPercentage, description}})
}

package controller

import (
	"net/http"
	"strconv"

	perfmetrics "github.com/QuantumNous/new-api/pkg/perf_metrics"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

func parsePerfMetricsRange(c *gin.Context) (startTs int64, endTs int64, ok bool) {
	startRaw := c.Query("start_timestamp")
	endRaw := c.Query("end_timestamp")
	if startRaw == "" || endRaw == "" {
		return 0, 0, false
	}
	startTs, startErr := strconv.ParseInt(startRaw, 10, 64)
	endTs, endErr := strconv.ParseInt(endRaw, 10, 64)
	if startErr != nil || endErr != nil || startTs <= 0 || endTs <= 0 {
		return 0, 0, false
	}
	if endTs < startTs {
		startTs, endTs = endTs, startTs
	}
	return startTs, endTs, true
}

func parsePerfMetricsHours(c *gin.Context) int {
	hours := 24
	if rawHours := c.Query("hours"); rawHours != "" {
		if parsed, err := strconv.Atoi(rawHours); err == nil {
			hours = parsed
		}
	}
	return hours
}

func GetPerfMetricsSummary(c *gin.Context) {
	activeGroups := append(lo.Keys(ratio_setting.GetGroupRatioCopy()), "auto")
	var result perfmetrics.SummaryAllResult
	var err error
	if startTs, endTs, ok := parsePerfMetricsRange(c); ok {
		result, err = perfmetrics.QuerySummaryAllRange(startTs, endTs, activeGroups)
	} else {
		result, err = perfmetrics.QuerySummaryAll(parsePerfMetricsHours(c), activeGroups)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

func GetPerfMetrics(c *gin.Context) {
	modelName := c.Query("model")
	if modelName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "model is required",
		})
		return
	}

	startTs, endTs, hasRange := parsePerfMetricsRange(c)

	params := perfmetrics.QueryParams{
		Model: modelName,
		Group: c.Query("group"),
		Hours: parsePerfMetricsHours(c),
	}
	if hasRange {
		params.StartTs = startTs
		params.EndTs = endTs
	}

	result, err := perfmetrics.Query(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	result.Groups = filterActiveGroups(result.Groups)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

func filterActiveGroups(groups []perfmetrics.GroupResult) []perfmetrics.GroupResult {
	activeRatios := ratio_setting.GetGroupRatioCopy()
	return lo.Filter(groups, func(g perfmetrics.GroupResult, _ int) bool {
		_, ok := activeRatios[g.Group]
		return ok || g.Group == "auto"
	})
}

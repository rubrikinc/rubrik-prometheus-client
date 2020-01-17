package main

import (
	"log"
	"strconv"
	"github.com/rubrikinc/rubrik-sdk-for-go/rubrikcdm"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// SQL DB storage stats
	rubrikMssqlDbCapacityLocalUsed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rubrik_mssql_db_capacity_local_used_bytes",
			Help: "Local storage consumption for SQL DB snapshots.",
		},
		[]string{
			"clusterName",
			"objectName",
			"objectID",
			"location",
		},
	)
	rubrikMssqlDbCapacityArchiveUsed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rubrik_mssql_db_capacity_archive_used_bytes",
			Help: "Archive storage consumption for SQL DB snapshots.",
		},
		[]string{
			"clusterName",
			"objectName",
			"objectID",
			"location",
		},
	)
)

func init() {
	// SQL DB storage stats
	prometheus.MustRegister(rubrikMssqlDbCapacityLocalUsed)
	prometheus.MustRegister(rubrikMssqlDbCapacityArchiveUsed)
}

// GetMssqlCapacityStats ...
func GetMssqlCapacityStats(rubrik *rubrikcdm.Credentials, clusterName string) {
	reportData,err := rubrik.Get("internal","/report?report_template=ObjectProtectionSummary&report_type=Canned") // get our object protection summary report
	if err != nil {
		log.Fatal(err)
	}
	reports := reportData.(map[string]interface{})["data"].([]interface{})
	reportID := reports[0].(map[string]interface{})["id"]
	body := map[string]interface{}{
		"limit": 100,
		"requestFilters": map[string]interface{}{
			"objectType": "Mssql",
		},
	}
	for {
		hasMore := true
		tableData,err := rubrik.Post("internal","/report/"+reportID.(string)+"/table",body) // get our first page of data for the report
		if err != nil {
			log.Fatal(err)
		}
		dataGrid := tableData.(map[string]interface{})["dataGrid"].([]interface{})
		hasMore = tableData.(map[string]interface{})["hasMore"].(bool)
		cursor := tableData.(map[string]interface{})["cursor"]
		columns := tableData.(map[string]interface{})["columns"].([]interface{})
		for _, v := range dataGrid {
			thisObjectID, thisObjectName, thisLocation := "null","null","null"
			thisLocalStorage, thisArchiveStorage := 0.0,0.0
			for i := 0; i < len(columns); i++ {
				switch columns[i] {
				case "ObjectId":
					thisObjectID = v.([]interface{})[i].(string)
				case "ObjectName":
					thisObjectName = v.([]interface{})[i].(string)
				case "Location":
					thisLocation = v.([]interface{})[i].(string)
				case "LocalStorage":
					thisLocalStorage, _ = strconv.ParseFloat(v.([]interface{})[i].(string),64)
				case "ArchiveStorage":
					thisArchiveStorage, _ = strconv.ParseFloat(v.([]interface{})[i].(string),64)
				}
			}
			rubrikMssqlDbCapacityLocalUsed.WithLabelValues(
				clusterName,
				thisObjectName,
				thisObjectID,
				thisLocation).Set(thisLocalStorage)
			rubrikMssqlDbCapacityArchiveUsed.WithLabelValues(
				clusterName,
				thisObjectName,
				thisObjectID,
				thisLocation).Set(thisArchiveStorage)
		}
		if !hasMore {
			break
		} else {
			body = map[string]interface{}{
				"limit": 1000,
				"cursor": cursor,
				"requestFilters": map[string]interface{}{
					"objectType": "Mssql",
				},
			}
		}
	}
}
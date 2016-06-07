package api

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"net/http"
	"path/filepath"
	"fmt"
	"os"
	"net/http/httputil"
	"io"
)

// Route handlers
// /api/v1/system/health, get a units status, used by 3dt puller
func unitsHealthStatus(w http.ResponseWriter, r *http.Request, config *Config) {
	if err := json.NewEncoder(w).Encode(unitsHealthReport.GetHealthReport()); err != nil {
		log.Error("Failed to encode responses to json")
	}
}

// /api/v1/system/health/units, get an array of all units collected from all hosts in a cluster
func getAllUnitsHandler(w http.ResponseWriter, r *http.Request) {
	if err := json.NewEncoder(w).Encode(globalMonitoringResponse.GetAllUnits()); err != nil {
		log.Error("Failed to encode responses to json")
	}
}

// /api/v1/system/health/units/:unit_id:
func getUnitByIDHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	unitResponse, err := globalMonitoringResponse.GetUnit(vars["unitid"])
	if err != nil {
		log.Error(err)
		json.NewEncoder(w).Encode(err)
		return
	}
	if err := json.NewEncoder(w).Encode(unitResponse); err != nil {
		log.Error("Failed to encode responses to json")
	}
}

// /api/v1/system/health/units/:unit_id:/nodes
func getNodesByUnitIDHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodesForUnitResponse, err := globalMonitoringResponse.GetNodesForUnit(vars["unitid"])
	if err != nil {
		log.Error(err)
		json.NewEncoder(w).Encode(err)
		return
	}
	if err := json.NewEncoder(w).Encode(nodesForUnitResponse); err != nil {
		log.Error("Failed to encode responses to json")
	}
}

// /api/v1/system/health/units/:unit_id:/nodes/:node_id:
func getNodeByUnitIDNodeIDHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodePerUnit, err := globalMonitoringResponse.GetSpecificNodeForUnit(vars["unitid"], vars["nodeid"])

	if err != nil {
		log.Error(err)
		json.NewEncoder(w).Encode(err)
		return
	}
	if err := json.NewEncoder(w).Encode(nodePerUnit); err != nil {
		log.Error("Failed to encode responses to json")
	}
}

// list the entire tree
func reportHandler(w http.ResponseWriter, r *http.Request) {
	if err := json.NewEncoder(w).Encode(globalMonitoringResponse); err != nil {
		log.Error("Failed to encode responses to json")
	}
}

// /api/v1/system/health/nodes
func getNodesHandler(w http.ResponseWriter, r *http.Request) {
	if err := json.NewEncoder(w).Encode(globalMonitoringResponse.GetNodes()); err != nil {
		log.Error("Failed to encode responses to json")
	}
}

// /api/v1/system/health/nodes/:node_id:
func getNodeByIDHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodes, err := globalMonitoringResponse.GetNodeByID(vars["nodeid"])
	if err != nil {
		log.Error(err)
		json.NewEncoder(w).Encode(err)
		return
	}

	if err := json.NewEncoder(w).Encode(nodes); err != nil {
		log.Error("Failed to encode responses to json")
	}
}

// /api/v1/system/health/nodes/:node_id:/units
func getNodeUnitsByNodeIDHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	units, err := globalMonitoringResponse.GetNodeUnitsID(vars["nodeid"])
	if err != nil {
		log.Error(err)
		json.NewEncoder(w).Encode(err)
		return
	}

	if err := json.NewEncoder(w).Encode(units); err != nil {
		log.Error("Failed to encode responses to json")
	}
}

func getNodeUnitByNodeIDUnitIDHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	unit, err := globalMonitoringResponse.GetNodeUnitByNodeIDUnitID(vars["nodeid"], vars["unitid"])
	if err != nil {
		log.Error(err)
		json.NewEncoder(w).Encode(err)
		return
	}
	if err := json.NewEncoder(w).Encode(unit); err != nil {
		log.Error("Failed to encode responses to json")
	}
}

// A helper function to send a response.
func writeResponse(w http.ResponseWriter, response snapshotReportResponse) {
	w.WriteHeader(response.ResponseCode)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error(err)
	}
}

// snapshot handlers
// A handler responsible for removing snapshots. First it will try to find a snapshot locally, if failed
// it will send a broadcast request to all cluster master members and check if snapshot it available.
// If snapshot was found on a remote host the local node will send a POST request to remove the snapshot.
func deleteSnapshotHandler(w http.ResponseWriter, r *http.Request, dt Dt) {
	vars := mux.Vars(r)
	response, err := dt.DtSnapshotJob.delete(vars["file"],dt.Cfg, dt.DtDCOSTools)
	if err != nil {
		log.Error(err)
	}
	writeResponse(w, response)
}

// A handler function return a snapshot job status of type `snapshotReportStatus`
func statusSnapshotReporthandler(w http.ResponseWriter, r *http.Request, dt Dt) {
	if err := json.NewEncoder(w).Encode(dt.DtSnapshotJob.getStatus(dt.Cfg)); err != nil {
		log.Error("Failed to encode responses to json")
	}
}

// A handler function returns a map of master node ip address as a key and snapshotReportStatus as a value.
func statusAllSnapshotReporthandler(w http.ResponseWriter, r *http.Request, dt Dt) {
	status, err := dt.DtSnapshotJob.getStatusAll(dt.Cfg, dt.DtDCOSTools)
	if err != nil {
		response, _ := prepareResponseWithErr(http.StatusServiceUnavailable, err)
		writeResponse(w, response)
		return
	}
	if err := json.NewEncoder(w).Encode(status); err != nil {
		log.Error("Failed to encode responses to json")
	}
}

// A handler function cancels a job running on a local node first. If a job is running on a remote node
// it will try to send a POST request to cancel it.
func cancelSnapshotReportHandler(w http.ResponseWriter, r *http.Request, dt Dt) {
	response, err := dt.DtSnapshotJob.cancel(dt.Cfg, dt.DtDCOSTools)
	if err != nil {
		log.Error(err)
	}
	writeResponse(w, response)
}

// A handler function returns a map of master ip as a key and a list of snapshots as a value.
func listAvailableGLobalSnapshotFilesHandler(w http.ResponseWriter, r *http.Request, dt Dt) {
	allSnapshots, err := listAllSnapshots(dt.Cfg, dt.DtDCOSTools)
	if err != nil {
		response, _ := prepareResponseWithErr(http.StatusServiceUnavailable, err)
		writeResponse(w, response)
		return
	}
	if err := json.NewEncoder(w).Encode(allSnapshots); err != nil {
		log.Error("Failed to encode responses to json")
	}
}

// A handler function returns a list of URLs to download snapshots
func listAvailableLocalSnapshotFilesHandler(w http.ResponseWriter, r *http.Request, dt Dt) {
	matches, err := dt.DtSnapshotJob.findLocalSnapshot(dt.Cfg)
	if err != nil {
		response, _ := prepareResponseWithErr(http.StatusServiceUnavailable, err)
		writeResponse(w, response)
		return
	}

	var snapshots []string
	for _, file := range matches {
		baseFile := filepath.Base(file)
		snapshots = append(snapshots, fmt.Sprintf("%s/report/snapshot/serve/%s", BaseRoute, baseFile))
	}
	if err := json.NewEncoder(w).Encode(snapshots); err != nil {
		log.Error("Failed to encode responses to json")
	}
}

// A handler function serves a static local file. If a file not available locally and
// listAvailableGLobalSnapshotFilesHandler returns that a file resides on a different node, it will create a reverse
// proxy to download the file.
func downloadSnapshotHandler(w http.ResponseWriter, r *http.Request, dt Dt) {
	vars := mux.Vars(r)
	serveFile := dt.Cfg.FlagSnapshotDir + "/" + vars["file"]
	_, err := os.Stat(serveFile)
	if err == nil {
		w.Header().Add("Content-disposition", fmt.Sprintf("attachment; filename=%s", vars["file"]))
		http.ServeFile(w, r, serveFile)
		return
	}
	// do a reverse proxy
	node, location, ok, err := dt.DtSnapshotJob.isSnapshotAvailable(vars["file"], dt.Cfg, dt.DtDCOSTools)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if ok {
		director := func(req *http.Request) {
			req = r
			req.URL.Scheme = "http"
			req.URL.Host = fmt.Sprintf("%s:%d", node, dt.Cfg.FlagPort)
			req.URL.Path = location
		}
		proxy := &httputil.ReverseProxy{Director: director}
		proxy.ServeHTTP(w, r)
		return
	}
	http.NotFound(w, r)
}

// A handler function to start a snapshot job.
func createSnapshotHandler(w http.ResponseWriter, r *http.Request, dt Dt) {
	var req snapshotCreateRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		response, _ := prepareResponseWithErr(http.StatusBadRequest, err)
		writeResponse(w, response)
		return
	}
	response, err := dt.DtSnapshotJob.run(req, dt.Cfg, dt.DtDCOSTools)
	if err != nil {
		log.Error(err)
	}
	writeResponse(w, response)
}

// A handler function to to get a list of available logs on a node.
func logsListHandler(w http.ResponseWriter, r *http.Request, dt Dt) {
	endspoints, err := dt.DtSnapshotJob.getLogsEndpoints(dt.Cfg, dt.DtDCOSTools)
	if err != nil {
		response, _ := prepareResponseWithErr(http.StatusServiceUnavailable, err)
		writeResponse(w, response)
		return
	}
	if err := json.NewEncoder(w).Encode(endspoints); err != nil {
		log.Error("Failed to encode responses to json")
	}
}

// return a log for past N hours for a specific systemd unit
func getUnitLogHandler(w http.ResponseWriter, r *http.Request, dt Dt) {
	vars := mux.Vars(r)
	unitLogOut, err := dt.DtSnapshotJob.dispatchLogs(vars["provider"], vars["entity"], dt.Cfg, dt.DtDCOSTools)
	if err != nil {
		response, _ := prepareResponseWithErr(http.StatusServiceUnavailable, err)
		writeResponse(w, response)
		return
	}
	log.Infof("Start read %s", vars["entity"])
	io.Copy(w, unitLogOut)
	log.Infof("Done read %s", vars["entity"])
	unitLogOut.Close()
}

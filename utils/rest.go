package helpers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-client-go/utils"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/prom2json"

	"github.com/jfrog/jfrog-client-go/utils/log"
)

type detailArtifact struct {
	Status                        string `json:"status"`
	Step                          string `json:"step"`
	Reason                        string `json:"reason"`
	IsImpactPathsRecoveryRequired bool   `json:"is_impact_paths_recovery_required"`
}

//TraceData trace data struct
type TraceData struct {
	File string
	Line int
	Fn   string
}

//Data struct
type Data struct {
	Name   string    `json:"name"`
	Help   string    `json:"help"`
	Type   string    `json:"type"`
	Metric []Metrics `json:"metrics"`
}

type FileInfo struct {
	Size     string          `json:"size"`
	MimeType string          `json:"mimeType"`
	Children []FileInfoChild `json:"children"`
}

type FileInfoChild struct {
	Uri string `json:"uri"`
}

//Metrics struct
type Metrics struct {
	TimestampMs string       `json:"timestamp_ms"`
	Value       string       `json:"value"`
	Labels      LabelsStruct `json:"labels,omitempty"`
}

//LabelsStruct struct
type LabelsStruct struct {
	Start     string `json:"start"`
	End       string `json:"end"`
	Status    string `json:"status"`
	Type      string `json:"type"`
	Max       string `json:"max"`
	Pool      string `json:"pool"`
	QueueName string `json:"queue_name"`
}

type SupportedTypes struct {
	SupportedPackageTypes []SupportedPackageType `json:"supportedPackageTypes"`
}

type SupportedPackageType struct {
	Type      string       `json:"type"`
	Extension []Extensions `json:"extensions"`
}

type Extensions struct {
	Extension string `json:"extension"`
	IsFile    bool   `json:"is_file"`
}

type FileList struct {
	Files []Files `json:"files"`
}

type Files struct {
	Uri    string `json:"uri"`
	Sha256 string `json:"sha2"`
}

func GetSupportedTypesJSON() (SupportedTypes, error) {
	var supportTypesFile SupportedTypes
	credsFile, err := os.Open(utils.GetUserHomeDir() + "/.jfrog/supported_types.json")
	if err != nil {
		return supportTypesFile, errors.New("Invalid supported_types.json file:" + err.Error())
	}
	defer credsFile.Close()
	scanner, _ := ioutil.ReadAll(credsFile)
	err = json.Unmarshal(scanner, &supportTypesFile)
	if err != nil {
		return supportTypesFile, err
	}
	return supportTypesFile, nil
}

//GetConfig get config from cli
func GetConfig() (*config.ServerDetails, error) {
	//TODO handle custom server id input
	serversIds, serverIDDefault, _ := GetServersIdAndDefault()
	if len(serversIds) == 0 {
		return nil, errorutils.CheckError(errors.New("no JFrog servers configured. Use the 'jfrog rt c' command to set the Artifactory server details"))
	}

	//TODO handle if user is not admin

	//fmt.Print(serversIds, serverIdDefault)
	config, err := config.GetSpecificConfig(serverIDDefault, true, false)
	if err != nil {
		//TODO print some error and exit
	}

	ping, respCode, _ := GetRestAPI("GET", true, config.Url+"xray/api/v1/system/ping", config, "", nil, 1)
	if respCode != 200 {
		return nil, errors.New("Xray is not up:" + string(ping))
	}

	return config, nil
}

func GetMetricsDataRaw(config *config.ServerDetails) ([]byte, error) {
	metrics, respCode, _ := GetRestAPI("GET", true, config.Url+"xray/api/v1/metrics", config, "", nil, 1)
	if respCode != 200 {
		return nil, errors.New("Received " + strconv.Itoa(respCode) + " HTTP code while getting metrics")
	}
	log.Debug("Received ", respCode, " while getting metrics")
	return metrics, nil
}

func match(s string) string {
	i := strings.Index(s, "pool=\"")
	if i >= 0 {
		j := strings.Index(s, "\"}")
		if j >= 0 {
			return s[i+6 : j]
		}
	}
	return ""
}

func GetMetricsDataJSON(config *config.ServerDetails, prettyPrint bool) ([]byte, error) {
	metrics, err := GetMetricsDataRaw(config)
	if err != nil {
		return nil, err
	}
	mfChan := make(chan *dto.MetricFamily, 1024)

	// Missing input means we are reading from an URL. stupid hack because Artifactory is missing a newline return
	file := string(metrics) + "\n"

	go func() {
		if err := prom2json.ParseReader(strings.NewReader(file), mfChan); err != nil {
			//issue with reading metrics if there is an unescaped new line char
			log.Warn("error reading metrics:", err)

			return
		}
	}()

	//TODO: Hella inefficient?
	//fmt.Println("before", time.Now())
	result := []*prom2json.Family{}
	for mf := range mfChan {
		result = append(result, prom2json.NewFamily(mf))
	}

	var jsonText []byte
	//pretty print
	if prettyPrint {
		jsonText, err := json.MarshalIndent(result, "", "    ")
		if err != nil {
			return nil, errors.New(err.Error() + " at " + string(Trace().Fn) + " on line " + string(strconv.Itoa(Trace().Line)))
		}
		fmt.Println(string(jsonText))
		return jsonText, nil
	}
	jsonText, err = json.Marshal(result)
	if err != nil {
		return nil, errors.New(err.Error() + " at " + string(Trace().Fn) + " on line " + string(strconv.Itoa(Trace().Line)))
	}
	//fmt.Println("after", time.Now())
	return jsonText, nil
}

//StringToInt64 self explanatory
func StringToInt64(data string) (int64, error) {
	convert, err := strconv.ParseInt(data, 10, 64)
	if err != nil {
		log.Warn(data, " is not of type integer:", err)
		return 0, err
	}
	return convert, nil
}

func ByteCountDecimal(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "kMGTPE"[exp])
}

func GetMetricsData(config *config.ServerDetails, counter int, prettyPrint bool, interval int) ([]Data, string, int, error) {
	//log.Info("hello")
	//TODO check if token vs password apikey
	jsonText, err := GetMetricsDataJSON(config, prettyPrint)
	if err != nil {
		//no need to show error fn here
		return nil, "", 0, err
	}

	var metricsData []Data
	err = json.Unmarshal(jsonText, &metricsData)
	if err != nil {
		return nil, "", 0, errors.New(err.Error() + " at " + string(Trace().Fn) + " on line " + string(strconv.Itoa(Trace().Line)))
	}

	currentTime := time.Now()

	if len(metricsData) == 0 {
		counter = counter + 1*interval
		currentTime = currentTime.Add(time.Second * -1 * time.Duration(counter))
	} else {
		counter = 0
	}
	return metricsData, currentTime.Format("2006.01.02 15:04:05"), counter, nil
}

func GetServersIdAndDefault() ([]string, string, error) {
	allConfigs, err := config.GetAllServersConfigs()
	if err != nil {
		return nil, "", errors.New(err.Error() + " at " + string(Trace().Fn) + " on line " + string(strconv.Itoa(Trace().Line)))
	}
	var defaultVal string
	var serversId []string
	for _, v := range allConfigs {
		if v.IsDefault {
			defaultVal = v.ServerId
		}
		serversId = append(serversId, v.ServerId)
	}
	return serversId, defaultVal, nil
}

//Check logger for errors
func Check(e error, panicCheck bool, logs string, trace TraceData) {
	if e != nil && panicCheck {
		log.Error(logs, " failed with error:", e, " ", trace.Fn, " on line:", trace.Line)
		panic(e)
	}
	if e != nil && !panicCheck {
		log.Warn(logs, " failed with error:", e, " ", trace.Fn, " on line:", trace.Line)
	}
}

//Trace get function data
func Trace() TraceData {
	var trace TraceData
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		log.Warn("Failed to get function data")
		return trace
	}

	fn := runtime.FuncForPC(pc)
	trace.File = file
	trace.Line = line
	trace.Fn = fn.Name()
	return trace
}

//
func GetStatus(repo, pkgtype, uri, sha256, scanType string, config *config.ServerDetails) (string, bool) {

	//there are odd ball cases where there is no Sha256 returned e.g. yum_3.2-25-2_all.deb that need to be considered
	if sha256 == "" && scanType == "artifact" {
		return "No sha256 in filelist", false
	}
	var body string
	switch scanType {
	case "artifact":
		body = "{\"repository_pkg_type\":" + "\"" + pkgtype + "\"," +
			"\"path\":" + "\"" + repo + uri + "\"," +
			"\"sha256\":" + "\"" + sha256 + "\"" +
			"}"
	case "build":
		//re-use repo = build name, uri = build number
		body = "{\"name\":" + "\"" + repo + "\"," +
			"\"version\":" + "\"" + uri + "\"" +
			"}"
	case "releaseBundle":
	default:
		return scanType + " not supported", false
	}

	headers := map[string]string{"Content-type": "application/json"}
	resp, respCode, _ := GetRestAPI("POST", true, config.XrayUrl+"api/v1/scan/status/"+scanType, config, body, headers, 0)
	if respCode != 200 {
		log.Debug("Error getting details:", string(resp), body, headers, config.User)
		return "Failed getting details", false
	}

	var detail detailArtifact
	err := json.Unmarshal(resp, &detail)
	if err != nil {
		fmt.Println("Error unmarshalling details:", err)
	}
	//statuses
	//"failed"/"not supported"/"in progress"/"not scanned"/"scanned"
	if detail.Status == "scanned" {
		return detail.Status, true
	} else {
		return detail.Status, false
	}
}

type IndexedRepo struct {
	Name    string `json:"name"`
	PkgType string `json:"pkgType"`
	Type    string `json:"type"`
}

//Test if remote repository exists and is a remote
func CheckTypeAndRepoParams(config *config.ServerDetails) ([]IndexedRepo, error) {
	repoCheckData, repoStatusCode, _ := GetRestAPI("GET", true, config.ArtifactoryUrl+"api/xrayRepo/getIndex", config, "", nil, 1)
	var result []IndexedRepo
	if repoStatusCode != 200 {
		return result, errors.New("Repo list does not exist.")
	}

	json.Unmarshal(repoCheckData, &result)
	return result, nil
}

//GetRestAPI GET rest APIs response with error handling
func GetRestAPI(method string, auth bool, urlInput string, config *config.ServerDetails, providedfilepath string, header map[string]string, retry int) ([]byte, int, http.Header) {
	if retry > 5 {
		log.Warn("Exceeded retry limit, cancelling further attempts")
		return nil, 0, nil
	}
	body := new(bytes.Buffer)
	if method == "POST" && providedfilepath != "" {
		body = bytes.NewBuffer([]byte(providedfilepath))
	}
	//PUT upload file
	if method == "PUT" && providedfilepath != "" {
		//req.Header.Set()
		file, err := os.Open(providedfilepath)
		Check(err, false, "open", Trace())
		defer file.Close()

		writer := multipart.NewWriter(body)

		part, err := writer.CreateFormFile("file", filepath.Base(providedfilepath))
		Check(err, false, "create", Trace())
		io.Copy(part, file)
		err = writer.Close()
		Check(err, false, "writer close", Trace())
	}

	client := http.Client{}
	req, err := http.NewRequest(method, urlInput, body)
	if auth {
		if config.Password != "" {
			req.SetBasicAuth(config.User, config.Password)
		} else if config.AccessToken != "" {
			req.Header.Set("Authorization", "Bearer"+config.AccessToken)
		}
	}
	for x, y := range header {
		log.Debug("Recieved extra header:", x+":"+y)
		req.Header.Set(x, y)
	}

	if err != nil {
		log.Warn("The HTTP request failed with error", err)
	} else {

		resp, err := client.Do(req)
		Check(err, false, "The HTTP response", Trace())

		if err != nil {
			return nil, 0, nil
		}
		// need to account for 403s with xray, or other 403s, 429? 204 is bad too (no content for docker)
		switch resp.StatusCode {
		case 200:
			log.Debug("Received ", resp.StatusCode, " OK on ", method, " request for ", urlInput, " continuing")
		case 201:
			if method == "PUT" {
				log.Debug("Received ", resp.StatusCode, " ", method, " request for ", urlInput, " continuing")
			}
		case 403:
			log.Error("Received ", resp.StatusCode, " Forbidden on ", method, " request for ", urlInput, " continuing")
			// should we try retry here? probably not
		case 404:
			log.Debug("Received ", resp.StatusCode, " Not Found on ", method, " request for ", urlInput, " continuing")
		case 429:
			log.Error("Received ", resp.StatusCode, " Too Many Requests on ", method, " request for ", urlInput, ", sleeping then retrying, attempt ", retry)
			time.Sleep(10 * time.Second)
			GetRestAPI(method, auth, urlInput, config, providedfilepath, header, retry+1)
		case 204:
			if method == "GET" {
				log.Error("Received ", resp.StatusCode, " No Content on ", method, " request for ", urlInput, ", sleeping then retrying")
				time.Sleep(10 * time.Second)
				GetRestAPI(method, auth, urlInput, config, providedfilepath, header, retry+1)
			} else {
				log.Debug("Received ", resp.StatusCode, " OK on ", method, " request for ", urlInput, " continuing")
			}
		case 500:
			log.Error("Received ", resp.StatusCode, " Internal Server error on ", method, " request for ", urlInput, " failing out")
			return nil, 0, nil
		default:
			log.Warn("Received ", resp.StatusCode, " on ", method, " request for ", urlInput, " continuing")
		}
		//Mostly for HEAD requests
		statusCode := resp.StatusCode
		headers := resp.Header

		if providedfilepath != "" && method == "GET" {
			// Create the file
			out, err := os.Create(providedfilepath)
			Check(err, false, "File create:"+providedfilepath, Trace())
			defer out.Close()

			//done := make(chan int64)
			//go helpers.PrintDownloadPercent(done, filepath, int64(resp.ContentLength))
			_, err = io.Copy(out, resp.Body)
			Check(err, false, "The file copy:"+providedfilepath, Trace())
		} else {
			//maybe skip the download or retry if error here, like EOF
			data, err := ioutil.ReadAll(resp.Body)
			Check(err, false, "Data read:"+urlInput, Trace())
			if err != nil {
				log.Warn("Data Read on ", urlInput, " failed with:", err, ", sleeping then retrying, attempt:", retry)
				time.Sleep(10 * time.Second)

				GetRestAPI(method, auth, urlInput, config, providedfilepath, header, retry+1)
			}

			return data, statusCode, headers
		}
	}
	return nil, 0, nil
}

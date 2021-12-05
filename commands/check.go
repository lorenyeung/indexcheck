package commands

import (
	"container/list"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	helpers "github.com/lorenyeung/indexcheck/utils"
	"github.com/prometheus/common/log"
)

//repo, pkgType, types, creds, repoType, flags, fileListStruct.Files[i], notIndexCount, totalCount
type queueDetails struct {
	Repo          string
	PkgType       string
	Types         helpers.SupportedTypes
	RepoType      string
	FileListData  helpers.Files
	NotIndexCount int
	TotalCount    int
}

type IndexedRepo struct {
	Name    string `json:"name"`
	PkgType string `json:"pkgType"`
	Type    string `json:"type"`
}

func GetCheckCommand() components.Command {
	return components.Command{
		Name:        "check",
		Description: "Get indexes.",
		Aliases:     []string{"c"},
		Arguments:   getCheckArguments(),
		Flags:       getCheckFlags(),
		EnvVars:     getCheckEnvVar(),
		Action: func(c *components.Context) error {
			return CheckCmd(c)
		},
	}
}

func getCheckArguments() []components.Argument {
	return []components.Argument{
		{
			Name:        "all",
			Description: "list metrics.",
		},
		{
			Name:        "list",
			Description: "list metrics.",
		},
		{
			Name:        "repo",
			Description: "list metrics.",
		},
		{
			Name:        "path",
			Description: "list metrics.",
		},
	}
}

func getCheckFlags() []components.Flag {
	return []components.Flag{
		components.BoolFlag{
			Name:         "indexed",
			Description:  "Output straight from Xray",
			DefaultValue: false,
		},
		components.BoolFlag{
			Name:         "min",
			Description:  "Get minimum JSON from Xray (no whitespace)",
			DefaultValue: false,
		},
	}
}

func getCheckEnvVar() []components.EnvVar {
	return []components.EnvVar{}
}

type CheckConfiguration struct {
	addressee string
	raw       bool
	repeat    int
	prefix    string
	min       bool
}

func CheckCmd(c *components.Context) error {
	timeStart := time.Now()
	config, err := helpers.GetConfig()
	if err != nil {
		return errors.New(err.Error() + " at " + string(helpers.Trace().Fn) + " on line " + string(strconv.Itoa(helpers.Trace().Line)))
	}
	if len(c.Arguments) == 0 {
		fmt.Println("TBD")
		return nil
	}
	supportedTypes := helpers.GetSupportedTypesJSON()
	indexList := CheckTypeAndRepoParams(config)

	//convert to map for speedier look up
	indexedMap := make(map[string]IndexedRepo)
	for i := 0; i < len(indexList); i += 1 {
		indexedMap[indexList[i].Name] = indexList[i]
	}

	// probably not the right way to do it
	if len(c.Arguments) > 1 && len(c.Arguments) < 4 {
		var err error
		switch arg := c.Arguments[0]; arg {
		case "all":
			fmt.Println("This may take a while")
			return nil
		case "list":
			return nil
		case "repo":
			//check repo, and get type
			err = validateCheck(c.Arguments[1], "", indexedMap, supportedTypes, config)
		case "path":
			if len(c.Arguments) == 2 {
				return errors.New("missing path")
			}
			path := c.Arguments[2]
			if !strings.HasPrefix(path, "/") {
				path = "/" + path
			}
			//weird behaviour with docker
			err = validateCheck(c.Arguments[1], path, indexedMap, supportedTypes, config)
		default:
			return errors.New(err.Error() + " at " + string(helpers.Trace().Fn) + " on line " + string(strconv.Itoa(helpers.Trace().Line)))
		}
		if err != nil {
			return err
		}
		endTime := time.Now()
		totalTime := endTime.Sub(timeStart)
		fmt.Println("Execution took:", totalTime)
		return nil
	}
	//return if wrong num arg
	return errors.New("Wrong number of arguments. Expected: 0,1 or 2, " + "Received: " + strconv.Itoa(len(c.Arguments)))

}

func validateCheck(repoName, path string, indexedMap map[string]IndexedRepo, supportedTypes helpers.SupportedTypes, config *config.ServerDetails) error {
	//check repo, and get type
	repo := indexedMap[repoName]
	if repo.Name == "" {
		return errors.New("repository " + repoName + " does not exist or is not marked for indexing")
	}
	fmt.Println("checking:" + repoName + " at path:" + path)
	//path validation may be needed
	indexRepo(repo.Name, repo.PkgType, supportedTypes, repo.Type, config, path)
	return nil
}

func indexRepo(repo string, pkgType string, types helpers.SupportedTypes, repoType string, config *config.ServerDetails, folder string) {
	var extensions []helpers.Extensions
	pkgType = strings.ToLower(pkgType)
	log.Debug("type:", repoType, " pkgType:", pkgType, " repo:", repo)
	for i := range types.SupportedPackageTypes {
		if types.SupportedPackageTypes[i].Type == pkgType {
			log.Debug("found package type:", types.SupportedPackageTypes[i].Type)
			extensions = types.SupportedPackageTypes[i].Extension
		}
	}

	var repoMap = make(map[string]bool)
	for y := range extensions {
		repoMap[extensions[y].Extension] = true
		log.Debug("Extension added to list:", extensions[y].Extension)
	}
	var fileListData []byte
	var respCode int
	if repoType == "remote" {
		repo = repo + "-cache"
	}
	//TODO need workaround if using token, use content reader for larger amounts of data, or only allow path
	fileListData, respCode, _ = helpers.GetRestAPI("GET", true, config.ArtifactoryUrl+"api/storage/"+repo+folder+"?list&deep=1", config, "", nil, 0)
	if respCode != 200 {
		log.Fatalf("File list received unexpected response code:", respCode, " :", string(fileListData))
	}
	log.Debug("File list received:", string(fileListData))

	var UnindexableMap = make(map[string]int)
	var fileListStruct helpers.FileList
	json.Unmarshal(fileListData, &fileListStruct)
	var notIndexCount, totalCount, notIndexableCount, noExtCount int
	indexAnalysis := list.New()
	for i := range fileListStruct.Files {
		fileListStruct.Files[i].Uri = folder + fileListStruct.Files[i].Uri
		for j := range extensions {
			//maybe here?
			log.Debug("File found:", fileListStruct.Files[i].Uri, " matching against:", extensions[j].Extension)
			if strings.Contains(fileListStruct.Files[i].Uri, extensions[j].Extension) {
				var indexAnalysisTest bool = true
				if indexAnalysisTest {
					var queueDetails queueDetails
					queueDetails.Repo = repo
					queueDetails.PkgType = pkgType
					queueDetails.Types = types
					queueDetails.RepoType = repoType
					queueDetails.FileListData = fileListStruct.Files[i]
					queueDetails.NotIndexCount = notIndexCount
					queueDetails.TotalCount = totalCount
					indexAnalysis.PushBack(queueDetails)
				} else {
					log.Info("File being sent to indexing:", fileListStruct.Files[i].Uri)
					//send to indexing
					m := map[string]string{
						"Content-Type": "application/json",
					}
					body := "{\"artifacts\": [{\"repository\":\"" + repo + "\",\"path\":\"" + fileListStruct.Files[i].Uri + "\"}]}"

					resp, respCode, _ := helpers.GetRestAPI("POST", true, config.XrayUrl+"api/v1/forceReindex", config, body, m, 0)
					if respCode != 200 {
						notIndexCount++
						log.Warn("Unexpected Xray response:HTTP", respCode, " ", string(resp))
					} else {
						log.Info("Xray response:", string(resp))
					}
					totalCount++
				}
				break
			} else if j+1 == len(extensions) {
				//failed the last match
				//if flags.LogUnindexableVar {
				//fmt.Println("not indexable:", fileListStruct.Files[i].Uri)
				//}
				filePath := strings.Split(fileListStruct.Files[i].Uri, "/")
				fileName := filePath[len(filePath)-1]
				fileExt := strings.Split(fileName, ".")
				notIndexableCount++
				log.Debug("name, name array, uri:", fileName, fileExt, " ", fileListStruct.Files[i].Uri)
				if len(fileExt)-1 > 0 {
					//dont add files without file ext
					UnindexableMap["."+fileExt[len(fileExt)-1]]++
				} else {
					noExtCount++
				}

			}
		}
	}

	numJobs := indexAnalysis.Len()
	jobs := make(chan int, numJobs)
	results := make(chan int, numJobs)

	//worker pool TODO add user flag here instead of 5
	for w := 1; w <= 4; w++ {
		go worker(w, jobs, results, indexAnalysis, config)
	}
	for j := 1; j <= numJobs; j++ {
		jobs <- j
	}
	close(jobs)
	var x int
	for a := 1; a <= numJobs; a++ {
		x = <-results
		if x == 0 {
			notIndexCount++
		}
		totalCount++
	}

	log.Info("Total indexed count:", totalCount-notIndexCount, "/", totalCount, " Total not indexable:", notIndexableCount, " Files with no extension:", noExtCount)
	log.Info("Unindexable file types count:", UnindexableMap)
}

func worker(id int, jobs <-chan int, results chan<- int, queue *list.List, config *config.ServerDetails) {
	for queue.Len() > 0 {
		e := queue.Front().Value
		queue.Remove(queue.Front())
		log.Debug("worker ", id, " working on ", e)
		notIndexCount, totalCount := Details(e.(queueDetails), config)
		log.Debug("not index:", notIndexCount, " total:", totalCount)
		results <- totalCount
	}
}

//Test if remote repository exists and is a remote
func CheckTypeAndRepoParams(config *config.ServerDetails) []IndexedRepo {
	repoCheckData, repoStatusCode, _ := helpers.GetRestAPI("GET", true, config.ArtifactoryUrl+"api/xrayRepo/getIndex", config, "", nil, 1)
	if repoStatusCode != 200 {
		log.Fatalf("Repo list does not exist.")
	}
	var result []IndexedRepo
	json.Unmarshal(repoCheckData, &result)
	return result
}

func Details(q queueDetails, config *config.ServerDetails) (int, int) {
	//send to details
	var printAll bool
	//TODO flag
	selectorflag := "all"
	switch selectorflag {
	case "unindexed":
	case "all":
		printAll = true
	default:
		log.Fatalf("Please provide one of the following: unindexed all")
	}
	//status, proc := internal.GetDetails(q.Repo, q.PkgType, q.FileListData.Uri, config)
	status, proc := helpers.GetStatusArtifact(q.Repo, q.PkgType, q.FileListData.Uri, q.FileListData.Sha256, config)
	if !proc {
		q.NotIndexCount++
		printStatus(status, q.Repo, q.PkgType, q.FileListData.Uri, config)
	} else {
		q.TotalCount++
		if printAll {
			printStatus(status, q.Repo, q.PkgType, q.FileListData.Uri, config)
		}
	}
	//log.Info("not index:", q.NotIndexCount, " total:", q.TotalCount)
	return q.NotIndexCount, q.TotalCount
}

func printStatus(status string, repo string, pkgType string, uri string, config *config.ServerDetails) error {
	var fileDetails []byte
	var fileInfo helpers.FileInfo
	var size string
	if pkgType == "docker" || strings.HasSuffix(uri, "manifest.json") {
		uri = strings.TrimSuffix(uri, "/manifest.json")
		folderDetails, _, _ := helpers.GetRestAPI("GET", true, config.ArtifactoryUrl+"api/storage/"+repo+uri, config, "", nil, 0)
		json.Unmarshal(folderDetails, &fileInfo)
		var size64 int64
		for i := range fileInfo.Children {
			path := fileInfo.Children[i].Uri
			var fileInfoDocker helpers.FileInfo
			fileDetailsDocker, _, _ := helpers.GetRestAPI("GET", true, config.ArtifactoryUrl+"api/storage/"+repo+uri+path, config, "", nil, 0)
			json.Unmarshal(fileDetailsDocker, &fileInfoDocker)
			size64 = size64 + helpers.StringToInt64(fileInfoDocker.Size)
		}
		//hardcode mimetype for now
		fileInfo.MimeType = "application/json"
		size = helpers.ByteCountDecimal(size64)
	} else {
		fileDetails, _, _ = helpers.GetRestAPI("GET", true, config.ArtifactoryUrl+"api/storage/"+repo+uri, config, "", nil, 0)
		json.Unmarshal(fileDetails, &fileInfo)
		size = helpers.ByteCountDecimal(helpers.StringToInt64(fileInfo.Size))
	}
	status = fmt.Sprintf("%-19v", status)
	size = fmt.Sprintf("%-10v", size)
	//not really helpful for docker
	fmt.Println(status, "\t", size, "\t", fmt.Sprintf("%-16v", strings.TrimPrefix(fileInfo.MimeType, "application/")), " ", repo+uri)
	return nil
}

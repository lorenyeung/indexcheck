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
	"github.com/jfrog/jfrog-client-go/utils/log"
	"github.com/lorenyeung/indexcheck/internal"
	helpers "github.com/lorenyeung/indexcheck/utils"
)

//repo, pkgType, types, creds, repoType, flags, fileListStruct.Files[i], notIndexCount, totalCount
type queueDetails struct {
	Repo          string
	PkgType       string
	Types         helpers.SupportedTypes
	RepoType      string
	ScanType      string
	FileListData  helpers.Files
	NotIndexCount int
	TotalCount    int
}

type IndexedRepo struct {
	Name    string `json:"name"`
	PkgType string `json:"pkgType"`
	Type    string `json:"type"`
}

type buildList struct {
	Data []buildListData `json:"buildsNumbers"`
}
type buildListData struct {
	Uri string `json:"uri"`
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
			Name:        "repo-all",
			Description: "verify all repositories.",
		},
		{
			Name:        "build-list",
			Description: "verify comma delimited list of builds",
		},
		{
			Name:        "build-single",
			Description: "verify a speciic build",
		},
		{
			Name:        "repo-list",
			Description: "verify comma delimited list of repositories.",
		},
		{
			Name:        "repo-single",
			Description: "verify a single repository.",
		},
		{
			Name:        "repo-path",
			Description: "verify a speciic path within a repository.",
		},
	}
}

func getCheckFlags() []components.Flag {
	return []components.Flag{
		components.StringFlag{
			Name:         "worker",
			Description:  "Worker count for getting scan details",
			DefaultValue: "5",
		},
		components.BoolFlag{
			Name:         "showall",
			Description:  "Show all results, scanned or not",
			DefaultValue: false,
		},
		components.BoolFlag{
			Name:         "experimental",
			Description:  "experimental scan details (artifacts only)",
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
		return errors.New("Please provide appropiate arguments")
	}

	indexedMap := make(map[string]IndexedRepo)
	var supportedTypes helpers.SupportedTypes
	if strings.HasPrefix(c.Arguments[0], "repo-") {
		supportedTypes, err = helpers.GetSupportedTypesJSON()
		if err != nil {
			return err
		}
		indexList := CheckTypeAndRepoParams(config)
		//convert to map for speedier look up
		for i := 0; i < len(indexList); i += 1 {
			indexedMap[indexList[i].Name] = indexList[i]
		}
	}

	// probably not the right way to do it
	if len(c.Arguments) > 0 && len(c.Arguments) < 4 {
		var err error
		switch arg := c.Arguments[0]; arg {
		case "repo-all":
			fmt.Println("This may take a while")
			for i := range indexedMap {
				log.Debug("sending " + indexedMap[i].Name + " for validation")
				err = validateCheck(indexedMap[i].Name, "", indexedMap, supportedTypes, config, c)
				if err != nil {
					break
				}
			}
			return nil
		case "repo-list":
			repos := strings.Split(c.Arguments[1], ",")
			for repo := range repos {
				err = validateCheck(repos[repo], "", indexedMap, supportedTypes, config, c)
				if err != nil {
					break
				}
			}
		case "repo-single":
			//check repo, and get type
			err = validateCheck(c.Arguments[1], "", indexedMap, supportedTypes, config, c)
		case "repo-path":
			if len(c.Arguments) == 2 {
				return errors.New("missing path")
			}
			path := c.Arguments[2]
			if !strings.HasPrefix(path, "/") {
				path = "/" + path //api requires leading forward slash
			}
			err = validateCheck(c.Arguments[1], path, indexedMap, supportedTypes, config, c)
		case "build-single":
			if len(c.Arguments) == 1 {
				return errors.New("missing build name")
			}
			err = indexBuild(c.Arguments[1], config, c)
		case "build-list":
			builds := strings.Split(c.Arguments[1], ",")
			for build := range builds {
				err = indexBuild(builds[build], config, c)
				if err != nil {
					break
				}
			}
		default:
			return errors.New("non existent argument:" + arg)
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
	return errors.New("Wrong number of arguments. Expected: 1-3, " + "Received: " + strconv.Itoa(len(c.Arguments)))

}

func validateCheck(repoName, path string, indexedMap map[string]IndexedRepo, supportedTypes helpers.SupportedTypes, config *config.ServerDetails, c *components.Context) error {
	//check repo, and get type
	repo := indexedMap[repoName]
	if repo.Name == "" {
		return errors.New("repository " + repoName + " does not exist or is not marked for indexing")
	}
	fmt.Println("checking:" + repoName + " at path:" + path)
	indexRepo(repo.Name, repo.PkgType, supportedTypes, repo.Type, config, path, c)
	return nil
}

func indexBuild(buildName string, config *config.ServerDetails, c *components.Context) error {
	var buildListStruct buildList
	var notIndexCount, totalCount int
	buildListData, respCode, _ := helpers.GetRestAPI("GET", true, config.ArtifactoryUrl+"api/build/"+buildName, config, "", nil, 0)
	if respCode != 200 {
		return errors.New("Build list received unexpected response code:" + strconv.Itoa(respCode) + " :" + string(buildListData))
	}
	json.Unmarshal(buildListData, &buildListStruct)
	if len(buildListStruct.Data) == 0 {
		return errors.New("No build versions found for:" + buildName)
	}
	buildAnalysis := list.New()
	for i := range buildListStruct.Data {
		var queueDetails queueDetails
		queueDetails.Repo = buildName
		var fileData helpers.Files
		fileData.Uri = strings.TrimPrefix(buildListStruct.Data[i].Uri, "/")
		queueDetails.FileListData = fileData
		queueDetails.NotIndexCount = notIndexCount
		queueDetails.TotalCount = totalCount
		queueDetails.ScanType = "build"
		buildAnalysis.PushBack(queueDetails)
	}

	totalCount, notIndexCount = workerPool(buildAnalysis, config, c, totalCount, notIndexCount)
	fmt.Println("Total "+buildName+" scanned count:", totalCount-notIndexCount, "/", totalCount)

	return nil
}

func indexRepo(repo string, pkgType string, types helpers.SupportedTypes, repoType string, config *config.ServerDetails, folder string, c *components.Context) {
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
		if !strings.HasSuffix(repo, "-cache") {
			repo = repo + "-cache"
		}
	}
	//use content reader for larger amounts of data, or only allow path
	fileListData, respCode, _ = helpers.GetRestAPI("GET", true, config.ArtifactoryUrl+"api/storage/"+repo+folder+"?list&deep=1", config, "", nil, 0)
	if respCode != 200 {
		log.Error("File list received unexpected response code:", respCode, " :", string(fileListData))
		return
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
			log.Debug("File found:", fileListStruct.Files[i].Uri, " matching against:", extensions[j].Extension)
			if strings.Contains(fileListStruct.Files[i].Uri, extensions[j].Extension) {
				var queueDetails queueDetails
				queueDetails.Repo = repo
				queueDetails.PkgType = pkgType
				queueDetails.Types = types
				queueDetails.RepoType = repoType
				queueDetails.FileListData = fileListStruct.Files[i]
				log.Debug(fileListStruct.Files[i].Uri + " Sha256:" + fileListStruct.Files[i].Sha256)
				queueDetails.NotIndexCount = notIndexCount
				queueDetails.ScanType = "artifact"
				queueDetails.TotalCount = totalCount
				indexAnalysis.PushBack(queueDetails)
				break
			} else if j+1 == len(extensions) {
				//failed the last match
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
	totalCount, notIndexCount = workerPool(indexAnalysis, config, c, totalCount, notIndexCount)
	fmt.Println("Total "+repo+" indexed count:", totalCount-notIndexCount, "/", totalCount, " Total not indexable:", notIndexableCount, " Files with no extension:", noExtCount)
	fmt.Println("Unindexable file types count:", UnindexableMap)
}

func workerPool(indexAnalysis *list.List, config *config.ServerDetails, c *components.Context, totalCount, notIndexCount int) (int, int) {
	numJobs := indexAnalysis.Len()
	jobs := make(chan int, numJobs)
	results := make(chan int, numJobs)

	workers, err := strconv.Atoi(c.GetStringFlagValue("worker"))
	if err != nil {
		fmt.Println("error setting workers, using default of 5:", err)
		workers = 5
	}

	for w := 1; w <= workers; w++ {
		go worker(w, jobs, results, indexAnalysis, config, c)
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
	return totalCount, notIndexCount
}

func worker(id int, jobs <-chan int, results chan<- int, queue *list.List, config *config.ServerDetails, c *components.Context) {
	for queue.Len() > 0 {
		e := queue.Front().Value
		queue.Remove(queue.Front())
		log.Debug("worker ", id, " working on ", e)
		notIndexCount, totalCount := Details(e.(queueDetails), config, c)
		log.Debug("not index:", notIndexCount, " total:", totalCount)
		results <- totalCount
	}
}

//Test if remote repository exists and is a remote
func CheckTypeAndRepoParams(config *config.ServerDetails) []IndexedRepo {
	repoCheckData, repoStatusCode, _ := helpers.GetRestAPI("GET", true, config.ArtifactoryUrl+"api/xrayRepo/getIndex", config, "", nil, 1)
	var result []IndexedRepo
	if repoStatusCode != 200 {
		log.Error("Repo list does not exist.")
		return result
	}

	json.Unmarshal(repoCheckData, &result)
	return result
}

func Details(q queueDetails, config *config.ServerDetails, c *components.Context) (int, int) {
	//send to details
	var status string
	var proc bool
	if c.GetBoolFlagValue("experimental") && q.ScanType == "artifact" {
		status, proc = internal.GetDetails(q.Repo, q.PkgType, q.FileListData.Uri, config)
	} else {
		status, proc = helpers.GetStatus(q.Repo, q.PkgType, q.FileListData.Uri, q.FileListData.Sha256, q.ScanType, config)
	}
	if !proc {
		q.NotIndexCount++
		printStatus(status, q.Repo, q.PkgType, q.FileListData.Uri, config)
	} else {
		q.TotalCount++
		if c.GetBoolFlagValue("showall") {
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
	var respCode int
	if pkgType == "docker" {
		uri = strings.TrimSuffix(uri, "/manifest.json")
		folderDetails, _, _ := helpers.GetRestAPI("GET", true, config.ArtifactoryUrl+"api/storage/"+repo+uri, config, "", nil, 0)
		json.Unmarshal(folderDetails, &fileInfo)
		var size64 int64
		for i := range fileInfo.Children {
			path := fileInfo.Children[i].Uri
			var fileInfoDocker helpers.FileInfo
			fileDetailsDocker, _, _ := helpers.GetRestAPI("GET", true, config.ArtifactoryUrl+"api/storage/"+repo+uri+path, config, "", nil, 0)
			json.Unmarshal(fileDetailsDocker, &fileInfoDocker)
			sizeConv, err := helpers.StringToInt64(fileInfoDocker.Size)
			if err != nil {
				log.Warn(err)
				size64 = 0
			} else {
				size64 = size64 + sizeConv
			}
		}
		//hardcode mimetype for now
		fileInfo.MimeType = "application/json"
		size = helpers.ByteCountDecimal(size64)
	} else {
		fileDetails, respCode, _ = helpers.GetRestAPI("GET", true, config.ArtifactoryUrl+"api/storage/"+repo+uri, config, "", nil, 0)
		if respCode == 200 {
			json.Unmarshal(fileDetails, &fileInfo)
			sizeConv, err := helpers.StringToInt64(fileInfo.Size)
			if err != nil {
				log.Warn(err)
				sizeConv = 0
			}
			size = helpers.ByteCountDecimal(sizeConv)
		}
	}
	status = fmt.Sprintf("%-19v", status)
	size = fmt.Sprintf("%-10v", size)
	//not really helpful for docker
	fmt.Println(status, "\t", size, "\t", fmt.Sprintf("%-25v", strings.TrimPrefix(fileInfo.MimeType, "application/")), " ", repo+":"+uri)
	return nil
}

package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	helpers "github.com/lorenyeung/indexcheck/utils"
)

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
			Name:         "raw",
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

	_, err := helpers.GetConfig()
	if err != nil {
		return errors.New(err.Error() + " at " + string(helpers.Trace().Fn) + " on line " + string(strconv.Itoa(helpers.Trace().Line)))
	}
	if len(c.Arguments) == 0 {
		return nil
	}
	// probably not the right way to do it
	if len(c.Arguments) == 1 || len(c.Arguments) == 2 {
		var err error
		switch arg := c.Arguments[0]; arg {
		case "all":
			fmt.Println("This may take a while")
			return nil
		case "list":
			return nil
		case "repo":
			return nil
		case "path":
			//printStatus()
			return nil
		default:
			err = errors.New("Unrecognized argument:" + arg)
		}
		return errors.New(err.Error() + " at " + string(helpers.Trace().Fn) + " on line " + string(strconv.Itoa(helpers.Trace().Line)))
	}
	return errors.New("Wrong number of arguments. Expected: 0,1 or 2, " + "Received: " + strconv.Itoa(len(c.Arguments)))

}

func printStatus(status string, repo string, pkgType string, uri string) error {
	config, err := helpers.GetConfig()
	if err != nil {
		return errors.New(err.Error() + " at " + string(helpers.Trace().Fn) + " on line " + string(strconv.Itoa(helpers.Trace().Line)))
	}

	var fileDetails []byte
	var fileInfo helpers.FileInfo
	var size string
	if pkgType == "docker" {
		uri = strings.TrimSuffix(uri, "/manifest.json")
		folderDetails, _, _ := helpers.GetRestAPI("GET", true, config.ArtifactoryUrl+"/api/storage/"+repo+uri, config.User, config.Password, "", nil, 0)
		json.Unmarshal(folderDetails, &fileInfo)
		var size64 int64
		for i := range fileInfo.Children {
			path := fileInfo.Children[i].Uri
			var fileInfoDocker helpers.FileInfo
			fileDetailsDocker, _, _ := helpers.GetRestAPI("GET", true, config.ArtifactoryUrl+"/api/storage/"+repo+uri+path, config.User, config.Password, "", nil, 0)
			json.Unmarshal(fileDetailsDocker, &fileInfoDocker)
			size64 = size64 + helpers.StringToInt64(fileInfoDocker.Size)
		}
		//hardcode mimetype for now
		fileInfo.MimeType = "application/json"
		size = helpers.ByteCountDecimal(size64)
	} else {
		fileDetails, _, _ = helpers.GetRestAPI("GET", true, config.ArtifactoryUrl+"/api/storage/"+repo+uri, config.User, config.Password, "", nil, 0)
		json.Unmarshal(fileDetails, &fileInfo)
		size = helpers.ByteCountDecimal(helpers.StringToInt64(fileInfo.Size))
	}
	status = fmt.Sprintf("%-19v", status)
	//not really helpful for docker
	log.Info(status, "\t", size, "\t", fmt.Sprintf("%-16v", strings.TrimPrefix(fileInfo.MimeType, "application/")), " ", repo+uri)
}

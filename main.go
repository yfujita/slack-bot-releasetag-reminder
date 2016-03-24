package main

import (
	"os"
	"os/exec"
	"fmt"
	"strings"
	"strconv"
	"flag"
	"github.com/yfujita/slackutil"
	"path"
	"io/ioutil"
	goyaml "gopkg.in/yaml.v1"
)

const (
	TEMP_DIR = "releasetag-reminder-temp"
)

type GitRepository struct {
	GitRepositoryName string
	GitRepositoryUrl  string
	SlackUrl          string
	SlackChannel      string
	SlackBotName      string
	SlackBotIcon      string
}

func main() {
	var confPath string
	flag.StringVar(&confPath, "conf", "blank", "config file path")
	flag.Parse()
	confPath = path.Clean(confPath)
	if confPath == "blank" {
		panic("Invalid conf parameter")
	}
	gitRepositories := loadConfig(confPath)

	err := os.RemoveAll(TEMP_DIR)
	if err != nil {
		panic(err.Error())
	}
	os.Mkdir(TEMP_DIR, 0777)
	os.Chdir(TEMP_DIR)

	for _, gitRepository := range gitRepositories {
		executeCmd("git", "clone", gitRepository.GitRepositoryUrl)
		os.Chdir(gitRepository.GitRepositoryName)
		commitTimestamp := getLastCommitTimestamp()
		tagTimestamp := getLastTagTimestamp()
		fmt.Println(gitRepository.GitRepositoryName + "commit: " + strconv.Itoa(int(commitTimestamp)) + " tag: " + strconv.Itoa(int(tagTimestamp)))
		if commitTimestamp > tagTimestamp {
			fmt.Println(gitRepository.GitRepositoryName + " にリリースタグをつけてください o(>_<)o")
			slackMessage(gitRepository.SlackUrl, gitRepository.SlackChannel, gitRepository.SlackBotName, gitRepository.SlackBotIcon, gitRepository.GitRepositoryName + " にリリースタグをつけてください o(>_<)o", "")
		} else {
			fmt.Println(gitRepository.GitRepositoryName + " のリリースタグは大丈夫。")
		}
		os.Chdir("..")
		os.RemoveAll(gitRepository.GitRepositoryName)
	}

	os.Chdir("..")
	os.RemoveAll(TEMP_DIR)
}

func slackMessage(url, channel, botName, botIcon, title, msg string) {
	bot := slackutil.NewBot(url, channel, botName, botIcon)
	bot.Message(title, msg)
}

func getLastCommitTimestamp() int64 {
	out := executeCmd("git", "log")
	tags := strings.Split(out, "\n")
	if len(tags) == 0 {
		return 0
	}

	lastCommit := strings.Replace(tags[0], "commit ", "", -1)
	return getLastTimestamp(lastCommit)
}

func getLastTagTimestamp() int64 {
	out := executeCmd("git", "tag")
	tags := strings.Split(out, "\n")
	if len(tags) == 0 {
		return 0
	}

	var lastTag string
	for _, tag := range tags {
		if len(tag) > 5 {
			lastTag = tag
		}
	}
	if lastTag == "" {
		return 0
	}

	return getLastTimestamp(lastTag)
}

func getLastTimestamp(updateName string) int64 {
	out := executeCmd("git", "show", updateName, "--date=raw")
	lines := strings.Split(out, "\n")

	time := 0
	for _, line := range lines {
		if strings.Index(line, "Date") == 0 {
			raw := strings.Replace(line, "Date:", "", -1)
			raw = strings.Replace(raw, "+0900", "", -1)
			raw = strings.Replace(raw, " ", "", -1)

			var err error
			time, err = strconv.Atoi(raw)
			if err != nil {
				panic(err.Error())
			}
			break;
		}
	}
	return int64(time)
}

func executeCmd(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	out, err := cmd.Output()
	if err != nil {
		panic(err.Error())
	}
	return string(out)
}

func loadConfig(path string) []GitRepository {
	fmt.Println("Loading... " + path)
	yaml, err := ioutil.ReadFile(path)

	m := make(map[interface{}]interface{})
	err = goyaml.Unmarshal(yaml, &m)
	if err != nil {
		panic(err.Error())
	}
	fmt.Print("Configures: ")
	fmt.Println(m)


	slackDefaultUrl := m["slack-default-url"].(string)
	slackDefaultChannel := m["slack-default-channel"].(string)
	slackDefaultBotName := m["slack-default-botname"].(string)
	slackDefaultBotIcon := m["slack-default-boticon"].(string)

	repositories := m["git-repositories"].([]interface{})
	gitRepositories := make([]GitRepository, len(repositories))
	for i, repository := range repositories {
		m := repository.(map[interface{}]interface{})
		gitRepositories[i].GitRepositoryName = m["git-reponame"].(string)
		gitRepositories[i].GitRepositoryUrl = m["git-url"].(string)
		if m["slack-url"] == nil {
			gitRepositories[i].SlackUrl = slackDefaultUrl
		} else {
			gitRepositories[i].SlackUrl = m["slack-url"].(string)
		}
		if m["slack-channel"] == nil {
			gitRepositories[i].SlackChannel = slackDefaultChannel
		} else {
			gitRepositories[i].SlackChannel = m["slack-channel"].(string)
		}
		if m["slack-botname"] == nil {
			gitRepositories[i].SlackBotName = slackDefaultBotName
		} else {
			gitRepositories[i].SlackBotName = m["slack-botname"].(string)
		}
		if m["slack-boticon"] == nil {
			gitRepositories[i].SlackBotIcon = slackDefaultBotIcon
		} else {
			gitRepositories[i].SlackBotIcon = m["slack-boticon"].(string)
		}
	}

	return gitRepositories
}
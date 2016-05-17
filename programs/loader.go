/*
 Copyright 2016 Padduck, LLC

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

 	http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package programs

import (
	"encoding/json"
	"github.com/pufferpanel/pufferd/programs/types"
	"io/ioutil"
	"strings"
	"path/filepath"
	"strconv"
	"github.com/pufferpanel/pufferd/logging"
	"fmt"
	"os"
)

const (
	serverFolder = "servers";
)

var (
	programs []Program = make([]Program, 5);
)

func LoadFromFolder() {
	os.Mkdir(serverFolder, os.ModeDir);
	var programFiles, err = ioutil.ReadDir(serverFolder);
	if (err != nil) {
		logging.Critical("Error reading from server data folder", err);
	}
	var data []byte;
	var program Program;
	for _, element := range programFiles {
		data, err = ioutil.ReadFile(joinPath(serverFolder, element.Name()));
		if (err != nil) {
			logging.Error(fmt.Sprintf("Error loading server details (%s)", element.Name()), err);
			continue;
		}
		program, err = LoadProgramFromData(data);
		if (err != nil) {
			logging.Error(fmt.Sprintf("Error loading server details (%s)", element.Name()), err);
			continue;
		}
		programs = append(programs, program);
	}
}

func GetProgram(id string) (program Program, err error) {
	program = getFromCache(id);
	if (program == nil) {
		program, err = LoadProgram(id);
	}
	return;
}

func LoadProgram(id string) (program Program, err error) {
	var data []byte;
	data, err = ioutil.ReadFile(joinPath(serverFolder, id + ".json"));
	program, err = LoadProgramFromData(data);
	return;
}

func LoadProgramFromData(source []byte) (program Program, err error) {
	var data map[string]interface{};
	err = json.Unmarshal(source, &data);
	if (err != nil) {
		return;
	}
	var pufferdData = GetMapOrNull(data, "pufferd");
	var t = GetStringOrNull(pufferdData, "type");
	var install = GetInstallSection(GetMapOrNull(pufferdData, "install"));
	switch(t) {
	case "java":
		var runBlock types.JavaRun;
		if (pufferdData["run"] == nil) {
			runBlock = types.JavaRun{};
		} else {
			var runSection = GetMapOrNull(pufferdData, "run");
			var stop = GetStringOrNull(runSection, "stop");
			var pre = GetStringArrayOrNull(runSection, "pre");
			var post = GetStringArrayOrNull(runSection, "post");
			var arguments = GetStringOrNull(runSection, "arguments");

			runBlock = types.JavaRun{Stop: stop, Pre: pre, Post: post, Arguments: arguments};
		}
		program = &types.Java{InstallData: install, RunData: runBlock};
	}
	return;
}

func GetInstallSection(data map[string]interface{}) types.JavaInstall {
	var install = types.JavaInstall{};
	install.Files = GetStringArrayOrNull(data, "files");
	install.Pre = GetStringArrayOrNull(data, "pre");
	install.Post = GetStringArrayOrNull(data, "post");
	return install;
}

func GetStringOrNull(data map[string]interface{}, key string) string {
	if (data == nil) {
		return "";
	}
	var section = data[key];
	if (section == nil) {
		return "";
	} else {
		return section.(string);
	}
}

func GetMapOrNull(data map[string]interface{}, key string) map[string]interface{} {
	if (data == nil) {
		return (map[string]interface{})(nil);
	}
	var section = data[key];
	if (section == nil) {
		return (map[string]interface{})(nil);
	} else {
		return section.(map[string]interface{});
	}
}

func GetStringArrayOrNull(data map[string]interface{}, key string) []string {
	if (data == nil) {
		return ([]string)(nil);
	}
	var section = data[key];
	if (section == nil) {
		return ([]string)(nil);
	} else {
		var sec = section.([]interface{});
		var newArr = make([]string, len(sec));
		for i := 0; i < len(sec); i++ {
			newArr[i] = sec[i].(string);
		}
		return newArr;
	}
}

func joinPath(paths ...string) string {
	return strings.Join(paths, strconv.QuoteRune(filepath.Separator));
}

func getFromCache(id string) (Program) {
	for _, element := range programs {
		if (element.Id() == id) {
			return element;
		}
	}
	return nil;
}
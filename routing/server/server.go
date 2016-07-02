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

package server

import (
	"github.com/gin-gonic/gin"
	"github.com/pufferpanel/pufferd/httphandlers"
	"github.com/pufferpanel/pufferd/logging"
	"github.com/pufferpanel/pufferd/permissions"
	"github.com/pufferpanel/pufferd/programs"
	"github.com/pufferpanel/pufferd/utils"
	"io"
	"os"
	"encoding/json"
	"errors"
)

func RegisterRoutes(e *gin.Engine) {
	l2 := e.Group("/server", httphandlers.AdminServerAccessHandler, httphandlers.HasServerAccessHandler)
	{
		l2.PUT("/:id", CreateServer)
		l2.DELETE("/:id", DeleteServer)
	}

	l1 := e.Group("/server", httphandlers.UserServerAccessHandler)
	{
		l1.GET("/:id/start", StartServer)
		l1.GET("/:id/stop", StopServer)
		l1.POST("/:id/install", InstallServer)
		l1.GET("/:id/file/*filename", GetFile)
		l1.PUT("/:id/file/*filename", PutFile)
	}
}

func StartServer(c *gin.Context) {
	valid, existing := handleInitialCallServer(c, "server.start", false)

	if !valid {
		return
	}

	existing.Start()
}

func StopServer(c *gin.Context) {
	valid, existing := handleInitialCallServer(c, "server.stop", false)

	if !valid {
		return
	}

	existing.Stop()
}

func CreateServer(c *gin.Context) {
	serverId := c.Param("id")
	privKey := c.Query("privkey")
	serverType := c.Query("type")
	data := make(map[string]interface{}, 0)
	//var postedData = json.Marshal(c.Request.Body)
	err := json.NewDecoder(c.Request.Body).Decode(&data)

	if err != nil {
		logging.Error("Error decoding JSON body", err)
		c.AbortWithError(400, err)
		return
	}

	user, okay := data["user"].(string)

	if user == "" {
		logging.Error("No user provided")
		c.AbortWithError(400, errors.New("No user provided with request"))
		return
	}

	if !okay {
		c.AbortWithError(400, errors.New("No user provided with request in string format"))
		return
	}

	delete(data, "user")

	if !permissions.GetGlobal().HasPermission(privKey, "server.create") {
		c.AbortWithStatus(403)
		return
	}

	existing := programs.GetFromCache(serverId)

	if existing != nil {
		c.AbortWithStatus(409)
		return
	}

	programs.Create(serverId, serverType, user, data)
}

func DeleteServer(c *gin.Context) {
	valid, existing := handleInitialCallServer(c, "server.delete", true)

	if !valid {
		return
	}

	programs.Delete(existing.Id())
}

func InstallServer(c *gin.Context) {
	valid, existing := handleInitialCallServer(c, "server.install", false)

	if !valid {
		return
	}

	existing.Install()
}

func GetFile(c *gin.Context) {

	valid, server := handleInitialCallServer(c, "server.file.get", false)

	if !valid {
		return
	}

	targetPath := c.Param("filename")
	if targetPath == "" {
		c.Status(404)
		return
	}

	c.File(utils.JoinPath(server.GetEnvironment().GetRootDirectory(), targetPath))
}

func PutFile(c *gin.Context) {
	valid, server := handleInitialCallServer(c, "server.file.put", false)

	if !valid {
		return
	}

	targetPath := c.Param("filename")

	if targetPath == "" {
		c.Status(404)
		return
	}

	file, err := os.Create(utils.JoinPath(server.GetEnvironment().GetRootDirectory(), targetPath))

	if err != nil {
		logging.Error("Error writing file", err)
		return
	}

	_, err = io.Copy(file, c.Request.Body)

	if err != nil {
		logging.Error("Error writing file", err)
	}
}

func handleInitialCallServer(c *gin.Context, perm string, requireGlobal bool) (valid bool, program programs.Program) {
	serverId := c.Param("id")
	privKey := c.Query("privkey")

	if requireGlobal && !permissions.GetGlobal().HasPermission(privKey, perm) {
		c.AbortWithStatus(403)
		valid = false
		return
	}

	program, _ = programs.Get(serverId)

	if program == nil {
		c.AbortWithStatus(404)
		valid = false
		return
	}

	if !requireGlobal && !program.GetPermissionManager().HasPermission(privKey, perm) {
		c.AbortWithStatus(403)
		valid = false
		return
	}

	valid = true
	return
}
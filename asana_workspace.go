package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func getWorkspaceList() Workspace {

	/*
		curl --request GET \
		--url https://app.asana.com/api/1.0/workspaces \
		--header 'accept: application/json' \
		--header 'authorization: Bearer ?'
	*/

	req, err := http.NewRequest("GET", asana_workspace_endpoint, nil)
	if err != nil {
		// handle err
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("authorization", "Bearer "+asana_access_token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// handle err
	}

	var workspace Workspace

	if resp.StatusCode == http.StatusOK {

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		var data map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &data); err != nil {
			panic(err)
		}

		strs := data["data"].([]interface{})
		data_block := strs[0].(map[string]interface{})

		workspace.gid = data_block["gid"].(string)
		workspace.name = data_block["name"].(string)
		workspace.resource_type = data_block["resource_type"].(string)
	} else {
		log.Fatalln("Error: Unable to get Workspace data : ", resp.StatusCode)
	}

	return workspace
}

func storeWorkspaceData(workspace Workspace) bool {

	// curl --request GET \
	// --url https://app.asana.com/api/1.0/workspaces/95125793591890 \
	// --header 'accept: application/json' \
	// --header 'authorization: Bearer ?'

	req, err := http.NewRequest("GET", asana_workspace_endpoint+"/"+workspace.gid, nil)
	if err != nil {
		return false
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("authorization", "Bearer "+asana_access_token)

	resp, err := http.DefaultClient.Do(req)
	directory := asana_workspace_directory + "workspace"

	if err != nil {
		return false
	}

	if resp.StatusCode == http.StatusOK {

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		var data map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &data); err != nil {
			panic(err)
		}

		workspace_info_file := directory + "/" + workspace.name + "/" + workspace.name + ".json"

		os.WriteFile(workspace_info_file, bodyBytes, 0755)

		fmt.Println(string(bodyBytes))

	}

	return false
}

func prepareWorkspace(workspace Workspace) bool {

	directory := asana_workspace_directory + "workspace"
	if prepareDirectory(directory) {
		fmt.Println("Root directory ready: ", directory)
	} else {
		fmt.Println("Workspace directory not found: ", directory)
	}

	if len(workspace.name) == 0 {
		fmt.Println("Workspace name missing")
		return false
	}

	workspace_directory := directory + "/" + workspace.name
	asana_workspace_projects_directory = workspace_directory + "/projects"

	if prepareDirectory(workspace_directory) {
		fmt.Println("Workspace directory ready: ", workspace_directory)

		storeWorkspaceData(workspace)
	}

	// fmt.Println(workspace.name)

	return true
}

package main

import (
	"fmt"
	"log"
	"runtime"
)

// Environment variables
var asana_access_token string = ""
var asana_workspace_directory = ""
var path_slash_replace string = "_"
var asana_task_limit int = 100
var asana_timeout_limit int = 0

// shared variables
var asana_workspace_projects_directory = ""

// Asana API endpoints
var asana_workspace_endpoint string = "https://app.asana.com/api/1.0/workspaces"
var asana_projects_endpoint string = "https://app.asana.com/api/1.0/projects"
var asana_tasks_endpoint string = "https://app.asana.com/api/1.0/tasks"
var asana_attachment_endpoint string = "https://app.asana.com/api/1.0/attachments"

type Workspace struct {
	gid           string
	name          string
	resource_type string
}

type Project struct {
	gid           string
	name          string
	resource_type string
}

type AssetData struct {
	Id        string
	Url       string
	Filename  string
	Permalink string
}

type Task struct {
	Gid             string `json:"Gid"`
	Name            string `json:"Name"`
	ResourceType    string `json:"ResourceType"`
	ResourceSubtype string `json:"ResourceSubtype"`
}

func main() {

	fmt.Println("Kanc Asana GO Exporter")
	log.Printf("Used CPUs / Max CPUs: %d/%d", runtime.GOMAXPROCS(0), runtime.NumCPU())

	// Get relevant ENV data.
	getEnvValues()
	workspace := getWorkspaceList()
	project_list := getAllProjects()

	// Prepare local containers.
	fmt.Println("------------------PREPARE------------------")
	prepareWorkspace(workspace)
	prepareProjects(project_list)

	fmt.Println("------------------TASKS------------------")

	// asana api accepts only project ID or workscpace ID as top level filter.
	// Itterate all projects.
	collectProjectListTask(project_list)

	fmt.Println("------------------Extract Tasks------------------")

	// Import tasks from task json.
	importProjectListTask(project_list)

	// fmt.Println("------------------Build MD files------------------")

	fmt.Println("-END-")
}

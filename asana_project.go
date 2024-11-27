package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"
)

var directory_task string

func getAllProjects() []Project {

	fmt.Println("Getting all projects")
	/*
		curl --request GET \
		     --url https://app.asana.com/api/1.0/projects \
		     --header 'accept: application/json' \
		     --header 'authorization: Bearer ?'
	*/

	req, err := http.NewRequest("GET", asana_projects_endpoint, nil)
	if err != nil {
		// handle err
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("authorization", "Bearer "+asana_access_token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// handle err
	}

	var project_list []Project

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

		for _, project := range strs {

			data_block := project.(map[string]interface{})
			var project_block Project

			project_block.gid = data_block["gid"].(string)
			project_block.name = data_block["name"].(string)
			project_block.resource_type = data_block["resource_type"].(string)

			project_list = append(project_list, project_block)
		}
	} else {
		log.Fatalln("Error: Unable to get Workspace data : ", resp.StatusCode)
	}

	return project_list

}

func storeProjectData(project Project, directory string) bool {

	// curl --request GET \
	// --url https://app.asana.com/api/1.0/projects/1200622970945159 \
	// --header 'accept: application/json' \
	// --header 'authorization: Bearer ?'

	log.Println("storeProjectData:start", project.name)

	project_name := strings.Replace(project.name, "/", path_slash_replace, -1)
	project_info_file := directory + "/" + project_name + ".json"

	if _, err := os.Stat(project_info_file); err == nil {
		// Here check last modified flag.
		// fmt.Println("file exists: ", project_info_file)
		return true
	} else {
		fmt.Println("file NOT: ", project_info_file)
	}

	req, err := http.NewRequest("GET", asana_projects_endpoint+"/"+project.gid+"", nil)
	if err != nil {
		log.Println("storeProjectData:error=1", project.name)
		return false
	}
	req.Header.Add("accept", "application/json")
	req.Header.Add("authorization", "Bearer "+asana_access_token)

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		log.Println("storeProjectData:error=2", project.name)
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

		os.WriteFile(project_info_file, bodyBytes, 0755)

		log.Println("storeProjectData:ok", project.name)

		return true
	} else {

		log.Println("storeProjectData:error=3 [", resp.StatusCode, "]", project.name)

		// handle rate limit - recall after 100ms.
		if resp.StatusCode == 429 {
			time.Sleep(time.Millisecond * 100)
			return storeProjectData(project, directory)
		}
	}

	log.Println("storeProjectData:error=4", project.name)
	return false
}

func prepareSingleProject(project Project, wg *sync.WaitGroup) bool {

	defer wg.Done()

	// Root project dir.
	directory := asana_workspace_projects_directory

	// fmt.Println("START: Prepare project [", thread_id, "]: ", project.name)

	if len(project.name) == 0 {
		fmt.Println("Project name missing, must stop.")
		return false
	}

	project_dir := strings.Replace(project.name, "/", path_slash_replace, -1)
	project_directory := directory + "/" + project_dir

	if prepareDirectory(project_directory) {
		// fmt.Println("Project directory ready: ", project_directory)
		//
		// Here check if project has changes since last sync time.
		//
		storeProjectData(project, project_directory)
		// fmt.Println("END: ", project.name)
	} else {
		// fmt.Println("Project directory not created: ", project_directory)
	}

	return true
}

func prepareProjects(project []Project) bool {

	// projects dir path is constructed in prepareWorkspace method.
	directory := asana_workspace_projects_directory

	if prepareDirectory(directory) {
		fmt.Println("Root projects directory ready: ", directory)
	} else {
		fmt.Println("Projects directory not found: ", directory)
	}

	//
	// populationSize := len(project)
	// population := make([]Individual, 0, populationSize)

	// we create a buffered channel so writing to it won't block while we wait for the waitgroup to finish
	// ch := make(chan Individual, populationSize)

	// Preppare worker wait group.
	wg := sync.WaitGroup{}

	for _, project_element := range project {

		wg.Add(1)
		thread_id := syscall.Gettid()
		start := time.Now()
		//status := prepareSingleProject(project_element)
		// status := prepareSingleProject(project_element, &wg)
		log.Println("************************************* START: project: ", project_element.name)

		go prepareSingleProject(project_element, &wg)

		elapsed := time.Since(start)
		log.Printf("-------------------------------> done in %s", elapsed, " | ", thread_id)

		// if status {
		// 	log.Println("Project data stored: ", project_element.name)
		// 	wg.Done()
		// }

	}
	wg.Wait()

	return true
}

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"sync"
)

type User struct {
	gid   string
	name  string
	rtype string
}

type TaskData struct {
	gid            string
	title          string
	description    string
	is_completed   bool
	assignee       User
	followers      []User
	time_created   string
	time_completed string
	permalink_url  string
	workspace_gid  string
	projects       []Project
}

type StoryData struct {
	gid          string
	time_created string
	created_by   User
	rtype        string
	class        string
	text         string
	order        int
}

func getFollowers(data interface{}) []User {

	var followers []User

	for _, follower := range data.([]interface{}) {

		follower_data := follower.(map[string]interface{})

		var user User
		user.gid = follower_data["gid"].(string)
		user.name = follower_data["name"].(string)
		user.rtype = follower_data["resource_type"].(string)

		followers = append(followers, user)
	}

	return followers
}

func getProjects(data interface{}) []Project {

	fmt.Println(data)
	var projects []Project

	for _, heap := range data.([]interface{}) {

		mapped_data := heap.(map[string]interface{})

		var current Project
		current.gid = mapped_data["gid"].(string)
		current.name = mapped_data["name"].(string)
		current.resource_type = mapped_data["resource_type"].(string)

		projects = append(projects, current)
	}

	return projects
}

func readJsonTaskFile(path string, wg *sync.WaitGroup) TaskData {

	defer wg.Done()

	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal("Cant open file (#01): ", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		panic(err)
	}

	strs := data["data"].(interface{})

	data_block := strs.(map[string]interface{})

	var task TaskData

	task.gid = data_block["gid"].(string)
	task.is_completed = data_block["completed"].(bool)
	task.title = data_block["name"].(string)
	task.description = data_block["notes"].(string)

	if data_block["assignee"] != nil {
		assignee := data_block["assignee"].(map[string]interface{})
		task.assignee.gid = assignee["gid"].(string)
		task.assignee.name = assignee["name"].(string)
		task.assignee.rtype = assignee["resource_type"].(string)
	}

	task.followers = getFollowers(data_block["followers"].([]interface{}))
	task.projects = getProjects(data_block["projects"].([]interface{}))

	task.time_created = data_block["created_at"].(string)
	if data_block["completed_at"] != nil {
		task.time_completed = data_block["completed_at"].(string)
	}
	task.permalink_url = data_block["permalink_url"].(string)

	return task
}

func readJsonStoryFile(path string, wg *sync.WaitGroup) []StoryData {

	defer wg.Done()

	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal("Cant open file (#02): ", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		panic(err)
	}

	strs := data["data"].([]interface{})

	// data_block := strs.(map[string]interface{})
	// fmt.Println(strs)
	var story_order int = 1
	var story_list []StoryData

	for _, heap := range strs {

		var story StoryData

		current := heap.(map[string]interface{})

		story.gid = current["gid"].(string)
		if current["text"] != nil {
			story.text = current["text"].(string)
		}
		if current["created_at"] != nil {
			story.time_created = current["created_at"].(string)
		}
		if current["created_by"] != nil {
			createdby := current["created_by"].(map[string]interface{})
			story.created_by.gid = createdby["gid"].(string)
			story.created_by.name = createdby["name"].(string)
			story.created_by.rtype = createdby["resource_type"].(string)
		}

		story.rtype = current["resource_type"].(string)
		story.class = current["type"].(string)
		story.order = story_order

		story_list = append(story_list, story)

		story_order++
	}

	return story_list
}

func readTaskStoryJson(task Task, wg *sync.WaitGroup) bool {

	asset_id_list = nil
	var path_task string = directory_project_task + "/" + task.Gid + "/" + task.Gid + "-task" + ".json"
	var path_story string = directory_project_task + "/" + task.Gid + "/" + task.Gid + "-story" + ".json"
	var path_asset string = directory_project_task + "/" + task.Gid + "/assets"
	var path_asset_file string = path_asset + "/asset_list.json"

	if checkFileExists(path_asset_file) {
		fmt.Println("SKIP : Task-Asset json exists: ", path_asset_file)
		wg.Done()
		return true
	}

	log.Println("readTaskStoryJson->Read: ", path_task)

	wg1 := sync.WaitGroup{}
	wg1.Add(2)

	var task_data TaskData
	var story_data []StoryData

	go func() {
		task_data = readJsonTaskFile(path_task, &wg1)
	}()
	go func() {
		story_data = readJsonStoryFile(path_story, &wg1)
	}()
	wg1.Wait()

	// First extract all task assets.
	var asset_list []string
	asset_list = extractTaskAssetList(task_data)
	asset_list = extractStoryAssetList(story_data)

	if len(asset_list) > 0 {

		// download asset from asana.
		if prepareDirectory(path_asset) {
			//
		} else {
			log.Println("Can't get prepare dir: ", path_asset)
			panic("Can't get prepare dir: " + path_asset)
			return false
		}

		fmt.Println("Asset list: ", asset_list)
		download := getAssetFiles(asset_list, path_asset)
		fmt.Println(download)
	} else {
		fmt.Println("No assets found.")
	}

	log.Println("readTaskStoryJson->END: ", path_task)

	wg.Done()

	return false
}

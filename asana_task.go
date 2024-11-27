package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// var task_list_page_offset string = ""
var project_task_list []Task
var directory_project_task string = ""
var directory_project_task_gid string = ""

// var project_task_count []int
var project_task_count map[int]int
var project_task_list2 map[int][]Task
var task_list_page_offset map[int]string

var counter_ratelimit_fail int = 0

func getTaskListPartial(project_id string, offset string) []Task {

	/*
		curl --request GET \
		     --url 'https://app.asana.com/api/1.0/tasks?limit=50&project=95134536856946' \
		     --header 'accept: application/json' \
		     --header 'authorization: ?'
	*/

	// offset
	/*
		"uri": "https://app.asana.com/api/1.0/tasks
		?limit=50
		&project=95134536856946
		&offset=eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJib3JkZXJfcmFuayI6IltcIkZLNUtNSzkwR0wyQ0RHXCIsMTE0Nzc0NjUxMDk0MTc1MSxcIjlaN0U0VE44MDhETFwiLDEyMDc4ODcwNjkzMzE5NTldIiwiaWF0IjoxNzI1NjE1NzcxLCJleHAiOjE3MjU2MTY2NzF9.33a69_KW2yC-f5u5QebB7WV_j4_0Ve1UO8ROGZec4ws
	*/

	var url string = asana_tasks_endpoint + "?limit=" + strconv.Itoa(asana_task_limit) + "&project=" + project_id
	if len(offset) > 0 {
		url += "&offset=" + offset
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		// handle err
	}
	req.Header.Add("accept", "application/json")
	req.Header.Add("authorization", "Bearer "+asana_access_token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// handle err
	}

	pid, _ := strconv.Atoi(project_id)

	var task_list []Task
	if len(project_task_list2[pid]) > 0 {
		task_list = project_task_list2[pid]
	}

	if resp.StatusCode == http.StatusOK {

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		resp.Body.Close()

		var data map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &data); err != nil {
			panic(err)
		}

		strs := data["data"].([]interface{})

		// next_page is null when no paging offset.
		if data["next_page"] == nil {
			task_list_page_offset[pid] = ""
		} else {

			page := data["next_page"].(map[string]interface{})

			if len(page["offset"].(string)) > 0 {
				task_list_page_offset[pid] = page["offset"].(string)
			} else {
				task_list_page_offset[pid] = ""
			}
		}

		for _, project := range strs {

			data_block := project.(map[string]interface{})

			var task Task
			task.Gid = data_block["gid"].(string)
			task.Name = data_block["name"].(string)
			task.ResourceType = data_block["resource_type"].(string)
			task.ResourceSubtype = data_block["resource_subtype"].(string)
			task_list = append(task_list, task)
			/*
				task2 := &Task{
					gid:              "a",
					name:             "b",
					resource_type:    "c",
					resource_subtype: "d",
				}

				// var task_list []Task
				jsondata, err := json.Marshal(&task2)
				if err != nil {
					panic(err)
				}

				fmt.Println(task2)
				fmt.Println(string(jsondata))
			*/
			// return task_list
		}

		if counter_ratelimit_fail > 0 {
			counter_ratelimit_fail--
		}

	} else {
		log.Println("getTaskListPartial:error=3 [", resp.StatusCode, "]", project_id)

		// handle rate limit - recall after 100ms.
		if resp.StatusCode == 429 {
			counter_ratelimit_fail++
			waitms := rateLimitWait(0)

			time.Sleep(time.Millisecond * time.Duration(waitms))
			return getTaskListPartial(project_id, offset)
		}

		log.Fatalln("Error: Unable to get Project Task data : ", resp.StatusCode)
	}

	return task_list
}

func storeTaskData(workspace Workspace) bool {

	return false
}

func prepareTask(workspace Workspace) bool {

	return false
}

/**
 * Collect all tasks by project recursive with page offset.
 */
func collectAllTaskByProject(project Project) []Task {

	/*
	       {
	         "gid": "914946431770930",
	         "name": "instalacija - ZDUR Vps Srce - zdur.dkd.hr",
	         "resource_type": "task",
	         "resource_subtype": "default_task"
	       }
	     ],

	     "next_page": null
	   }
	*/
	project_id, _ := strconv.Atoi(project.gid)
	project_task_list2[project_id] = getTaskListPartial(project.gid, task_list_page_offset[project_id])
	if len(task_list_page_offset[project_id]) > 0 {
		collectAllTaskByProject(project)
	}

	return project_task_list2[project_id]
}

func prepareTaskContainer() bool {

	return false
}

func getTaskDetailsJsonParallel(task Task, wg *sync.WaitGroup) bool {

	// defer wg.Done()

	/*
			curl --request GET \
		     --url https://app.asana.com/api/1.0/tasks/1208152460471103 \
		     --header 'accept: application/json' \
		     --header 'authorization: Bearer 2/95394412573480/1207102741941759:81f49d06365229bec77b42dcd0a398f9'
	*/

	project_info_file := directory_project_task + "/" + task.Gid + "/" + task.Gid + "-task" + ".json"
	var url string = asana_tasks_endpoint + "/" + task.Gid

	if checkFileExists(project_info_file) {
		log.Println("SKIP : Task json exists: ", project_info_file)
		wg.Done()

		return true
	}

	resource := getResource(url, true)

	if resource.status == true {

		log.Println("getTaskStoryJsonParallel->Write: ", project_info_file)
		if err := os.WriteFile(project_info_file, resource.body, 0755); err != nil {
			log.Fatal(err)
			panic("Error: Cant write project info file : " + project_info_file)

			return false
		}
		log.Println("Task stored: ", task.Name)

		counter_ratelimit_fail = 0

		wg.Done()

		return true

	} else {
		// retry
		if resource.code == 429 {
			// here i will rely on global asana timeout limit.
			log.Println("429 Error: Unable to get Project Task-Details data - reloading (", task.Name, ") (", strconv.Itoa(asana_timeout_limit), ") <-------------------")
			return getTaskDetailsJsonParallel(task, wg)

		} else if resource.code == 403 {
			counter_ratelimit_fail++
			waitms := rateLimitWait(3000)

			log.Println("403 Error: Unable to get Project Task-Details data - reloading (", task.Name, ") (", strconv.Itoa(waitms), " / ", counter_ratelimit_fail, ") <-------------------")
			time.Sleep(time.Millisecond * time.Duration(waitms))

			return getTaskDetailsJsonParallel(task, wg)
		} else {
			log.Println("-------------------ERROR-task:")
			fmt.Println("file: ", project_info_file)
			fmt.Println("task gid: ", task.Gid)
			fmt.Println("task name: ", task.Name)

			panic("Error (unhandled): Unable to get Project Task-Details data (2.1): " + strconv.Itoa(resource.code))
			//fail, error
		}
	}

	wg.Done()

	return false
}

func getTaskDetailsJson(task Task) bool {
	/*
			curl --request GET \
		     --url https://app.asana.com/api/1.0/tasks/1208152460471103 \
		     --header 'accept: application/json' \
		     --header 'authorization: Bearer 2/95394412573480/1207102741941759:81f49d06365229bec77b42dcd0a398f9'
	*/

	var url string = asana_tasks_endpoint + "/" + task.Gid
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false
	}
	req.Header.Add("accept", "application/json")
	req.Header.Add("authorization", "Bearer "+asana_access_token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}

	if resp.StatusCode == http.StatusOK {

		project_info_file := directory_project_task_gid + "/" + task.Gid + "-task" + ".json"
		log.Println("Store task details file: ", project_info_file)

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		//var data map[string]interface{}
		//if err := json.Unmarshal(bodyBytes, &data); err != nil {
		//	panic(err)
		//}

		os.WriteFile(project_info_file, bodyBytes, 0755)

		return true
	}

	return false
}

func getTaskStoryJsonParallel(task Task, wg *sync.WaitGroup) bool {

	project_info_file := directory_project_task + "/" + task.Gid + "/" + task.Gid + "-story" + ".json"
	var url string = asana_tasks_endpoint + "/" + task.Gid + "/stories"

	if checkFileExists(project_info_file) {
		fmt.Println("SKIP : Task-Story json exists: ", project_info_file)

		wg.Done()

		return true
	}

	resource := getResource(url, true)

	if resource.status == true {

		log.Println("getTaskStoryJsonParallel->Write: ", project_info_file)
		if err := os.WriteFile(project_info_file, resource.body, 0755); err != nil {
			log.Fatal(err)
			return false
		}
		log.Println("Task-Story stored: ", task.Name)

		counter_ratelimit_fail = 0

		wg.Done()

		return true

	} else {
		// retry
		if resource.code == 429 {
			// here i will rely on global asana timeout limit.
			log.Println("429 Error: Unable to get Project Task-Story data - reloading (", task.Name, ") (", strconv.Itoa(asana_timeout_limit), ") <-------------------")
			return getTaskStoryJsonParallel(task, wg)
		} else if resource.code == 403 {

			counter_ratelimit_fail++
			waitms := rateLimitWait(2000)

			log.Println("403 Error: Unable to get Project Task-Story data - reloading (", task.Name, ") (", strconv.Itoa(waitms), " / ", counter_ratelimit_fail, ") <-------------------")
			time.Sleep(time.Millisecond * time.Duration(waitms))

			return getTaskStoryJsonParallel(task, wg)
		} else {
			panic("Error (unhandled): Unable to get Project Task data (2.3): " + strconv.Itoa(resource.code))
			//fail, error
		}
	}

	/*
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return false
		}
		req.Header.Add("accept", "application/json")
		req.Header.Add("authorization", "Bearer "+asana_access_token)

		http.DefaultClient.Timeout = time.Millisecond * 10000

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Println("Story GET error: ", err)
			log.Println(resp)
			return false
		}

		if resp.StatusCode == http.StatusOK {

			// log.Println("Store task story file: ", project_info_file)

			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Fatal(err)
			}
			log.Println("getTaskStoryJsonParallel->Write: ", project_info_file)
			if err := os.WriteFile(project_info_file, bodyBytes, 0755); err != nil {
				log.Fatal(err)
				return false
			}
			resp.Body.Close()
			log.Println("Task-Story stored: ", task.Name)

			if counter_ratelimit_fail > 0 {
				counter_ratelimit_fail--
			}

			wg.Done()

			return true
		} else {

			if resp.StatusCode == 429 {

				counter_ratelimit_fail++
				waitms := rateLimitWait(0)

				log.Println("429 Error: Unable to get Project Task-Story data - reloading (", strconv.Itoa(waitms), ") <-------------------")
				time.Sleep(time.Millisecond * time.Duration(waitms))

				return getTaskStoryJsonParallel(task, wg)
			}
			panic("Error (unhandled): Unable to get Project Task data (2.4): " + strconv.Itoa(resp.StatusCode))
		}
	*/
	return false
}

func getTaskStoryJson(task Task) bool {
	/*
		curl --request GET \
		     --url https://app.asana.com/api/1.0/tasks/1208152460471103/stories \
		     --header 'accept: application/json' \
		     --header 'authorization: Bearer 2/95394412573480/1207102741941759:81f49d06365229bec77b42dcd0a398f9'
	*/

	var url string = asana_tasks_endpoint + "/" + task.Gid + "/stories"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false
	}
	req.Header.Add("accept", "application/json")
	req.Header.Add("authorization", "Bearer "+asana_access_token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}

	if resp.StatusCode == http.StatusOK {

		project_info_file := directory_project_task_gid + "/" + task.Gid + "-story" + ".json"
		log.Println("Store task story file: ", project_info_file)

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		os.WriteFile(project_info_file, bodyBytes, 0755)

		return true
	}

	return false
}

func taskGenerate(task_list []Task) bool {

	// filename: 1208152460471103-

	// Preppare worker wait group.
	wg := sync.WaitGroup{}

	for key, task := range task_list {
		wg.Add(3)

		// task ID : task.gid
		fmt.Println("Task | ", key, " | [", task.Gid, "]", task.Name)

		directory_project_task_gid = directory_project_task + "/" + task.Gid

		if prepareDirectory(directory_project_task_gid) {
			getTaskDetailsJson(task)
			getTaskStoryJson(task)
			// get task stories
		} else {
			log.Println("Can't get task details: ", task.Name)
		}

		//return true
	}
	wg.Wait()

	return true
}

func taskGenerateParallel(task_list []Task) bool {

	// filename: 1208152460471103-
	// Preppare worker wait group.

	// First, prepare all directories.
	//wg := sync.WaitGroup{}
	log.Println("Prepare all task directories: ", directory_project_task)
	for _, task := range task_list {
		// wg.Add(1)
		directory_project_task_gid = directory_project_task + "/" + task.Gid
		// go prepareDirectoryParallel(directory_project_task_gid, &wg)
		prepareDirectory(directory_project_task_gid)
	}
	// wg.Wait()

	// Second, collect all task json files.
	wg1 := sync.WaitGroup{}
	log.Println("Collect all task json files.")
	for _, task := range task_list {

		log.Println("Task | [", task.Gid, "] ", task.Name)
		var path_asset string = directory_project_task + "/" + task.Gid + "/assets"
		asset_path := path_asset + "/asset_list.json"

		// asset file is indicator that all task/story files are collected.
		if checkFileExists(asset_path) {
			fmt.Println("Got assets, skipping..... ", asset_path)
			continue
		}

		wg1.Add(1)

		// reset rate limit fail counter.
		counter_ratelimit_fail = 0

		go getTaskDetailsJsonParallel(task, &wg1)
		// go getTaskStoryJsonParallel(task, &wg1)

	}
	wg1.Wait()

	// Second, collect all task-story json files.
	wg3 := sync.WaitGroup{}
	log.Println("Collect all task-story json files.")
	for _, task := range task_list {

		log.Println("Task | [", task.Gid, "] ", task.Name)
		var path_asset string = directory_project_task + "/" + task.Gid + "/assets"
		asset_path := path_asset + "/asset_list.json"

		// asset file is indicator that all task/story files are collected.
		if checkFileExists(asset_path) {
			fmt.Println("Got assets, skipping..... ", asset_path)
			continue
		}

		wg3.Add(1)

		// reset rate limit fail counter.
		counter_ratelimit_fail = 0

		go getTaskStoryJsonParallel(task, &wg3)

	}
	wg3.Wait()

	// reset rate limit fail counter.
	counter_ratelimit_fail = 0

	// itterate all task/story detect assets list.
	log.Println("Collect all assets.")
	wg2 := sync.WaitGroup{}
	for _, task := range task_list {
		log.Println("Assets: | [", task.Gid, "] ", task.Name)
		wg2.Add(1)
		go readTaskStoryJson(task, &wg2)
	}
	wg2.Wait()

	log.Println("Task store task loop end.")

	//
	// Third all assets.
	//
	//
	// this for loop is here temporaray, to test the task asset get.
	// method should be converted to go routine after testing.
	// should be called last.
	//
	/*
		wg = sync.WaitGroup{}
		log.Println("Collect all task asset files.")
		for _, task := range task_list {
			wg.Add(1)

			go readTaskStoryJson(task, &wg)
		}
		wg.Wait()
	*/
	return true
}

func collectAllTasksByProject(project_element Project, wg *sync.WaitGroup) bool {

	defer wg.Done()

	directory := asana_workspace_projects_directory + "/"

	// reset task list page offset
	// task_list_page_offset = ""
	project_task_list = nil

	project_id, _ := strconv.Atoi(project_element.gid)

	project_dir := strings.Replace(project_element.name, "/", path_slash_replace, -1)
	project_directory := directory + "/" + project_dir
	project_task_file := project_directory + "/" + project_dir + "-task-list.json"

	fmt.Println("Project task file: ", project_task_file)

	_, err := os.Stat(project_task_file)
	if os.IsNotExist(err) {
		log.Println("Collect all tasks for project: ", project_element.name)

		task_list := collectAllTaskByProject(project_element)
		// fmt.Println(task_list)
		log.Println("Task list collected.")

		jsondata, _ := json.Marshal(task_list)
		log.Println("Json prepared.")

		task_count := len(task_list)
		log.Println("Task count:", task_count)

		project_task_count[project_id] = task_count
		log.Println("Task count stored:", task_count)

		if err := os.WriteFile(project_task_file, jsondata, os.ModePerm); err != nil {
			log.Fatal(err)
			return false
		} else {
			log.Println("Task list for project \"", project_element.name, "\" written [", len(task_list), "]")
		}

		delete(project_task_count, project_id)
		delete(project_task_list2, project_id)
		delete(task_list_page_offset, project_id)

	} else {
		log.Println("Project task exists: [", project_id, "] ", project_element.name)
	}

	return true
}

func collectProjectListTask(project []Project) bool {

	if len(project_task_count) == 0 {
		project_task_count = make(map[int]int)
	}

	if len(project_task_count) == 0 {
		project_task_list2 = make(map[int][]Task)
	}

	if len(task_list_page_offset) == 0 {
		task_list_page_offset = make(map[int]string)
	}

	wg := sync.WaitGroup{}
	for _, project_element := range project {
		wg.Add(1)
		go collectAllTasksByProject(project_element, &wg)
	}
	wg.Wait()

	return true
}

func getTaskListFromFile(file string) []Task {
	jsonFile, err := os.Open(file)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Reading file: ", file)
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var task_list []Task
	json.Unmarshal(byteValue, &task_list)

	return task_list
}

func importProjectListTask(project []Project) bool {

	directory := asana_workspace_projects_directory + "/"
	//var task_list []Task

	for _, project_element := range project {

		// reset task list page offset
		project_task_list = nil

		project_dir := strings.Replace(project_element.name, "/", path_slash_replace, -1)
		project_directory := directory + "" + project_dir
		project_task_file := project_directory + "/" + project_dir + "-task-list.json"
		directory_project_task = project_directory + "/tasks"

		_, err := os.Stat(project_task_file)
		if os.IsNotExist(err) {
			log.Println("Project has no task list! : ", project_element.name)
		} else {
			log.Println("Get task list for project : ", project_element.name)

			task_list := getTaskListFromFile(project_task_file)

			fmt.Println("Got ", len(task_list), " tasks.")

			status := taskGenerateParallel(task_list)
			log.Println("All Tasks collected.", status)
		}
	}

	return true
}

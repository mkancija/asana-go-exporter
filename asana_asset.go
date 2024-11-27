package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var asset_id_list []string

func detectAssetId(text string) string {

	var asset_id string = ""

	ss := strings.Split(text, "\n")
	if len(ss) >= 1 {
		asset_id = ss[0]
	}

	return asset_id
}

func detectAssetLink(text string) bool {

	asset_string := "asset_id="

	if strings.Contains(text, asset_string) {
		ss := strings.Split(text, asset_string)
		var asset_id string = ""
		if len(ss) > 0 {
			// asset id detected, try to collect id.
			asset_id = detectAssetId(ss[1])
			if len(asset_id) > 0 {
				asset_id_list = append(asset_id_list, asset_id)
			} else {
				return false
			}

			firstSubstr := strings.Index(text, asset_string)
			substring := text[(firstSubstr + len(asset_string)):]

			// recursive call
			detectAssetLink(substring)
		}

		return false
	}

	return false
}

func extractTaskAssetList(task TaskData) []string {

	// task json data contains asset id links in text array-key.
	if len(task.description) > 0 {
		detectAssetLink(task.description)
	}

	// check asset list.
	if len(asset_id_list) > 0 {
		return asset_id_list
	} else {
		return asset_id_list
	}

}

func extractStoryAssetItem(story StoryData) []string {

	// task json data contains asset id links in text array-key.
	if len(story.text) > 0 {
		detectAssetLink(story.text)
	}

	// check asset list.
	if len(asset_id_list) > 0 {
		return asset_id_list
	} else {
		return asset_id_list
	}

}

func downloadAssetJson(asset_id string, path string) bool {
	/*
		curl --request GET \
		--url https://app.asana.com/api/1.0/attachments/1208042032410728 \
		--header 'accept: application/json' \
		--header 'authorization: Bearer 2/95394412573480/1207102741941759:81f49d06365229bec77b42dcd0a398f9'
	*/

	var url string = asana_attachment_endpoint + "/" + asset_id
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

		log.Println("Store task attachemnt file: ", path)

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		os.WriteFile(path, bodyBytes, 0755)

		return true
	}

	return false
}

func downloadAssetBytes(asset_id string) []byte {
	/*
		curl --request GET \
		--url https://app.asana.com/api/1.0/attachments/1208042032410728 \
		--header 'accept: application/json' \
		--header 'authorization: Bearer 2/95394412573480/1207102741941759:81f49d06365229bec77b42dcd0a398f9'
	*/
	var bodyBytes []byte
	var url string = asana_attachment_endpoint + "/" + asset_id

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return bodyBytes
	}
	req.Header.Add("accept", "application/json")
	req.Header.Add("authorization", "Bearer "+asana_access_token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return bodyBytes
	}

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		// os.WriteFile(path, bodyBytes, 0755)
		return bodyBytes
	}

	return bodyBytes
}

func parseDownloadFile(path string) string {

	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal("Cant open file: ", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		panic(err)
	}

	strs := data["data"].(interface{})

	data_block := strs.(map[string]interface{})

	return data_block["download_url"].(string)
}

func parseAssetBytes(content []byte) AssetData {

	var data map[string]interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		panic(err)
	}

	strs := data["data"].(interface{})
	data_block := strs.(map[string]interface{})

	var asset AssetData

	asset.Id = data_block["gid"].(string)
	if data_block["download_url"] != nil {
		asset.Url = data_block["download_url"].(string)
	}
	asset.Filename = data_block["name"].(string)
	asset.Permalink = data_block["permanent_url"].(string)

	return asset
}

func downloadAssetFile(asset AssetData, path_asset string) bool {

	// get extension from asset
	ext := filepath.Ext(asset.Filename)

	// Create the file
	filepath := path_asset + ext
	out, err := os.Create(filepath)
	if err != nil {
		return false
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(asset.Url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		fmt.Errorf("bad status: %s", resp.Status)
		return false
	}

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return false
	}

	return true
}

func getAssetFiles(asset_list []string, path_asset string) bool {

	path := path_asset + "/asset_list.json"
	var asset_data_all []AssetData

	if _, err := os.Stat(path); err == nil {
		// asset list file exists, skip.
		return true
	} else {
		fmt.Println("No asset file (#02): ", path)
	}

	for _, asset_id := range asset_list {

		// dlstatus := downloadAssetJson(asset_id, path)
		dlbytes := downloadAssetBytes(asset_id)

		if binary.Size(dlbytes) > 0 {

			// Prepare current asset file root path.
			asset_path := path_asset + "/" + string(asset_id)

			// Get asana asset info.
			asset_data := parseAssetBytes(dlbytes)

			// Dowload file via temporary asana amazon aws url.
			dlstatus := downloadAssetFile(asset_data, asset_path)

			log.Println("Download: ", asset_data.Filename, " : ", dlstatus)

			// Collect all asset data info.
			asset_data_all = append(asset_data_all, asset_data)
		}
	}

	jsondata, _ := json.Marshal(asset_data_all)
	if err := os.WriteFile(path, jsondata, os.ModePerm); err != nil {
		log.Fatal(err)
		return false
	}

	return true
}

func extractStoryAssetList(story []StoryData) []string {

	// Detect all story assets
	for _, story_data := range story {
		extractStoryAssetItem(story_data)
	}

	return asset_id_list
}

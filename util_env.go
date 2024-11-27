package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

func getEnvValues() {

	file, error := os.Open("asanaexporter.env")
	if error != nil {
		log.Fatalln(error)
	}

	//	defer closing the file until the function exits
	defer file.Close()

	// read the file line by line using scanner
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		var line string = scanner.Text()
		// fmt.Println(line)

		env_var := strings.Split(line, "=")

		if len(env_var) == 2 {

			if env_var[0] == "ASANA_ACCESS_TOKEN" {
				asana_access_token = env_var[1]
			}
			if env_var[0] == "BACKUP_DESTINATION" {
				asana_workspace_directory = env_var[1]
			}
			if env_var[0] == "TASK_PROCESS_LIMIT" {
				asana_task_limit, _ = strconv.Atoi(env_var[1])
			}

			// os.Setenv(env_var[0], env_var[1])
			fmt.Println("Setting env var: ", env_var[0], " | ", env_var[1])
		} else {
			fmt.Println("Undeclared env var: \"", line, "\"")
		}
	}

}

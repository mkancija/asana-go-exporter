package main

import (
	"log"
	"os"
	"sync"
)

func prepareDirectory(directory string) bool {

	_, err := os.Stat(directory)

	if os.IsNotExist(err) {
		// log.Println("Directory does not exist")

		err := os.MkdirAll(directory, 0750)

		if err == nil {
			log.Println("Directory created: ", directory)
			return true
		}

		// log.Println("Directory could not be created: ", directory)
		return false
	}

	return true
}

func prepareDirectoryParallel(directory string, wg *sync.WaitGroup) bool {

	_, err := os.Stat(directory)

	if os.IsNotExist(err) {
		err := os.MkdirAll(directory, 0750)
		if err == nil {
			return true
		}
		log.Println("Directory could not be created: ", directory)
		return false
	}

	wg.Done()

	return true
}

func checkFileExists(file string) bool {

	if _, err := os.Stat(file); err == nil {
		return true
	} else {
		return false
	}
}

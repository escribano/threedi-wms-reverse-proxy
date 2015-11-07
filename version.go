package main

import "io/ioutil"

func getVersion() (string, error) {
	versionArray, err := ioutil.ReadFile("version.txt")
	if err != nil {
		return "", err
	} else {
		return string(versionArray[:]), nil
	}
}

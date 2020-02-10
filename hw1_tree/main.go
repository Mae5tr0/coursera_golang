package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"bytes"
	"strings"	
)

func main() {
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	out := new(bytes.Buffer)
	err := dirTree(out, path, printFiles)
	out.WriteTo(os.Stdout)
	if err != nil {
		panic(err.Error())
	}
}

func dirTree(output *bytes.Buffer, path string, printFiles bool) error {	
	var dirTree strings.Builder
	printDirErr := printDir("", path, &dirTree, printFiles)
	if printDirErr != nil {
		return fmt.Errorf("error in printing dir: %v", printDirErr.Error())
	}

	output.WriteString(dirTree.String())

	return nil
}

func printDir(prefix string, path string, result *strings.Builder, printFiles bool) error {
	list, err := ioutil.ReadDir(path)
	if err != nil {		
		return fmt.Errorf(err.Error())
	}

	if !printFiles {
		list = filterFiles(list)
	}

	for pos, item := range list {
		isLast := pos == len(list) - 1
		itemPrefix := "├───"
		nextLevelPrefix := "│	"
		if (isLast) {
			itemPrefix = "└───"
			nextLevelPrefix = "	"
		}

		if (item.IsDir()) {
			result.WriteString(fmt.Sprintf("%s%s%s\n", prefix, itemPrefix, item.Name()))
			printDir(prefix + nextLevelPrefix, path + "/" + item.Name(), result, printFiles)
		} else {
			printFile(prefix + itemPrefix, item, result)
		}
	}

	return nil
}

func printFile(prefix string, file os.FileInfo, result *strings.Builder) {
	result.WriteString(fmt.Sprintf("%s%s (%s)\n", prefix, file.Name(), formatSize(file.Size())))		
}

func formatSize(size int64) string {
	if (size > 0) {
		return fmt.Sprintf("%db", size)
	} 
		
	return "empty"
}

func filterFiles(items []os.FileInfo) (result []os.FileInfo) {
	for _, item := range items {
			if item.IsDir() {
					result = append(result, item)					
			}			
	}
	return
}
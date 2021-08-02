package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"
)

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		fmt.Println("usage go run main.go . [-f]")
        return
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}
func dirTree(out io.Writer, currDir string, printFiles bool) error {
	printDirTree("", out, currDir, printFiles)
	return nil
}

func printDirTree(prefix string, out io.Writer, currDir string, printFiles bool) {
	f, _ := os.Open(currDir)
	//	if err != nil {
	//		fmt.Println("Could not open %s: %s", currDir, err.Error())
	//	}
	fileName := f.Name()
	f.Close()
	files, _ := ioutil.ReadDir(fileName)
	//if err != nil {
	//	fmt.Println("Could not read dir names in %s: %s", currDir, err.Error())
	//}
	filesMap := make(map[string]os.FileInfo)
	var arrName []string
	for _, file := range files {
		if file.IsDir() || printFiles {
			arrName = append(arrName, file.Name())
			filesMap[file.Name()] = file
		}
	}
	sort.Strings(arrName)
	var sortedFiles []os.FileInfo
	for _, name := range arrName {
		sortedFiles = append(sortedFiles, filesMap[name])
	}
	files = sortedFiles

	length := len(files)

	for i, file := range files {
		if file.IsDir() {
			var nextPrefix string
			if length > i+1 {
				fmt.Fprintf(out, prefix+"├───"+"%s\n", file.Name())
				nextPrefix = prefix + "│\t"
			} else {
				fmt.Fprintf(out, prefix+"└───"+"%s\n", file.Name())
				nextPrefix = prefix + "\t"
			}
			nextDir := currDir + "/" + file.Name()
			printDirTree(nextPrefix, out, nextDir, printFiles)
		} else if printFiles {
			if file.Size() > 0 {
				if length > i+1 {
					fmt.Fprintf(out, prefix+"├───%s (%vb)\n", file.Name(), file.Size())
				} else {
					fmt.Fprintf(out, prefix+"└───%s (%vb)\n", file.Name(), file.Size())
				}
			} else {
				if length > i+1 {
					fmt.Fprintf(out, prefix+"├───%s (empty)\n", file.Name())
				} else {
					fmt.Fprintf(out, prefix+"└───%s (empty)\n", file.Name())
				}
			}
		}
	}
}

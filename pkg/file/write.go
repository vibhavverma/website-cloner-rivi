package file

import (
	"log"
	"os"
	"path/filepath"
)

// CreateProject initializes the project directory and returns the path to the project
// TODO make function more modular to obtain different html files
func CreateProject(projectName string) string {
	// current workin directory
	path := currentDirectory()

	// define project path
	projectPath := filepath.Join(path, projectName)

	// create base directory
	err := os.MkdirAll(projectPath, 0755)
	check(err)

	// create asset directories
	for _, dir := range []string{"css", "js", "imgs", "fonts", "assets", "media", "models", "textures", "shaders"} {
		err := os.MkdirAll(filepath.Join(projectPath, dir), 0755)
		check(err)
	}

	// main index file
	err = os.WriteFile(filepath.Join(projectPath, "index.html"), nil, 0644)
	check(err)
	// project path
	return projectPath
}

// currentDirectory get the current working directory
func currentDirectory() string {
	path, err := os.Getwd()
	check(err)
	return path
}

func check(err error) {
	if err != nil {
		log.Println(err)
	}
}

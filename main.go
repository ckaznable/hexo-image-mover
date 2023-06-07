package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

func main() {
	if len(os.Args) == 1 {
		panic("hexo project path is required")
	}

	root := os.Args[1]
	paths, err := getMDPath(path.Join(root, "source/_posts"))
	if err != nil {
		panic(err)
	}

	totalTasks := len(paths)
	concurrentTasks := 10

	var wg sync.WaitGroup
	taskCh := make(chan string)

	wg.Add(totalTasks)

	for i := 0; i < concurrentTasks; i++ {
		go worker(taskCh, &wg)
	}

	for _, path := range paths {
		taskCh <- path
	}

	wg.Wait()
	fmt.Println("All tasks completed")
}

func getMDPath(root string) ([]string, error) {
	var paths []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(path) == ".md" {
			paths = append(paths, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return paths, nil
}

func worker(taskCh <-chan string, wg *sync.WaitGroup) {
	for task := range taskCh {
		process(task)
		wg.Done()
	}
}

func process(path string) {
	filename := filepath.Base(path)
	dirPath := filepath.Join(filepath.Dir(path), strings.Replace(filename, filepath.Ext(path), "", -1))

	content, err := getFileContent(path)
	if err != nil {
		println(err.Error())
		return
	}

	urls := getImageLink(content)
	if len(urls) == 0 {
		println(filename, " no image found")
		return
	}

	os.Mkdir(dirPath, os.ModePerm)
	for _, url := range urls {
		target := filepath.Join(dirPath, filepath.Base(url))
		err := downloadImage(url, target)
		if err == nil {
			content = strings.ReplaceAll(content, fmt.Sprintf("](%s)", url), fmt.Sprintf("](%s)", filepath.Base(url)))
		}
	}

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Printf("can't not open file: %s\n", err.Error())
		return
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		fmt.Printf("can't not write file: %s\n", err.Error())
		return
	}

	println(filename, " done")
}

func getFileContent(filepath string) (string, error) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return "", err
	}

	fileContent := string(content)
	return fileContent, nil
}

func getImageLink(str string) []string {
	pattern := `\[[^]]+\]\(([^)]+)\)`
	reg := regexp.MustCompile(pattern)
	matches := reg.FindAllStringSubmatch(str, -1)

	var url []string
	for _, match := range matches {
		ext, err := getURLExtension(match[1])
		if err == nil {
			switch ext {
			case ".jpg", ".jpeg", ".png":
				url = append(url, match[1])
			}
		}
	}

	return url
}

func getURLExtension(url string) (string, error) {
	ext := filepath.Ext(url)
	if ext == "" {
		return "", fmt.Errorf("can't not get file ext")
	}

	return ext, nil
}

func downloadImage(fileURL string, outputPath string) error {
	response, err := http.Get(fileURL)
	if err != nil {
		return fmt.Errorf("can't not download file")
	}
	defer response.Body.Close()

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("can't not create file")
	}
	defer outputFile.Close()

	_, err = io.Copy(outputFile, response.Body)
	if err != nil {
		return fmt.Errorf("can't not download file")
	}

	return nil
}

package transformer

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	indexerNum = 10

	mdExtension = ".md"
	aliasKey    = "aliases: "
)

// return a map of string key=alias -> string value=filename
func buildAliasIndex(rootPath string) sync.Map {
	m := sync.Map{}
	fileCh := make(chan string, indexerNum)

	var wg sync.WaitGroup
	for i := 0; i < indexerNum; i++ {
		wg.Add(1)

		go scanFile(&wg, fileCh, &m)
	}

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if filepath.Ext(path) == mdExtension {
			fileCh <- path
		}

		return nil
	})

	if err != nil {
		fmt.Printf("Error walking file tree: %v\n", err)
	}

	close(fileCh)
	wg.Wait()

	return m
}

func scanFile(wg *sync.WaitGroup, fileCh chan string, m *sync.Map) {
	defer wg.Done()

	for path := range fileCh {
		file, err := os.Open(path)
		if err != nil {
			fmt.Printf("Error opening file: %v\n", err)
			return
		}

		reader := bufio.NewReader(file)
		counter := 0
		for {
			line, err := reader.ReadString('\n')
			if err == io.EOF || counter >= 3 { // only read first 3 lines
				break
			}
			if err != nil {
				fmt.Printf("Error reading file: %v\n", err)
				break
			}

			if strings.HasPrefix(line, aliasKey) {
				aliasID := strings.ReplaceAll(line, aliasKey, "") // TODO assumed one alias as of now
				filename := strings.ReplaceAll(filepath.Base(path), mdExtension, "")

				m.Store(strings.TrimSpace(aliasID), filename)
				break
			}
		}

		file.Close()
	}
}
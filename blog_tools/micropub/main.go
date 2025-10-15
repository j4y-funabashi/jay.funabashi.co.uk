package main

import (
	"encoding/json"
	"flag"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/j4y-funabashi/jay.funabashi.co.uk/blog_tools/micropub/pkg/microformats"
	"github.com/j4y-funabashi/jay.funabashi.co.uk/blog_tools/micropub/pkg/micropub"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	outputDirectory := flag.String("output", ".", "output directory")
	flag.Parse()

	logger = logger.With(
		slog.Group("flags",
			slog.String("outputDir", *outputDirectory),
		),
	)

	err := os.MkdirAll(*outputDirectory, 0755)
	if err != nil {
		logger.Error("failed to make output directory %v", "error", err)
	}

	outputDir, err := os.Stat(*outputDirectory)
	if os.IsNotExist(err) {
		logger.Error("output directory does not exists")
	}
	if outputDir.IsDir() != true {
		logger.Error("output directory is not a directory")
	}

	logger.Info("importing posts from micropub")

	postList, err := micropub.List()
	if err != nil {
		logger.Error("failed to list posts %v", "error", err)
	}
	logger.Info("listed micropub posts", "postCount", len(postList))

	for _, postFile := range postList {
		logger.Info("importing micropub post", "postFile", postFile)
		postData, err := micropub.Download(postFile)
		if err != nil {
			logger.Error("failed to download post %v", "error", err)
			continue
		}
		hugoPost, err := microformats.Parse(postData)
		if err != nil {
			logger.Error("failed to parse hugo post %v", "error", err)
			continue
		}
		hugoPostJson, err := json.Marshal(hugoPost)

		// create dir
		postOutputDir := filepath.Join(*outputDirectory, hugoPost.Params.Year, hugoPost.Params.Month, hugoPost.Params.Day, hugoPost.Params.Uid)
		err = os.MkdirAll(postOutputDir, 0755)
		if err != nil {
			logger.Error("failed to create hugo post directory %v", "error", err)
			break
		}

		// create _index files
		err = os.WriteFile(filepath.Join(*outputDirectory, hugoPost.Params.Year, "_index.md"), []byte{}, 0755)
		if err != nil {
			logger.Error("failed to save _index file %v", "error", err)
			break
		}
		err = os.WriteFile(filepath.Join(*outputDirectory, hugoPost.Params.Year, hugoPost.Params.Month, "_index.md"), []byte{}, 0755)
		if err != nil {
			logger.Error("failed to save _index file %v", "error", err)
			break
		}
		err = os.WriteFile(filepath.Join(*outputDirectory, hugoPost.Params.Year, hugoPost.Params.Month, hugoPost.Params.Day, "_index.md"), []byte{}, 0755)
		if err != nil {
			logger.Error("failed to save _index file %v", "error", err)
			break
		}

		// save post
		postOutputFilename := filepath.Join(postOutputDir, "index.md")
		logger.Info("saving post", "post", string(hugoPostJson))
		err = os.WriteFile(postOutputFilename, hugoPostJson, 0755)
		if err != nil {
			logger.Error("failed to save file %v", "error", err)
			continue
		}
		logger.Info("done")
	}

}

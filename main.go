package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"github.com/joho/godotenv"
	"golang.org/x/image/webp"
	"image/png"
	"io"
	"net/http"
)

var mu sync.Mutex
var visited = map[string]bool{}
var imageCounter = 0

func downloadAndConvertImage(imageURL, folder string) error {
	resp, err := http.Get(imageURL)
	if err != nil {
		return fmt.Errorf("could not download image: %w", err)
	}
	defer resp.Body.Close()

	err = os.MkdirAll(folder, os.ModePerm)
	if err != nil {
		return fmt.Errorf("could not create folder: %w", err)
	}

	imageExt := filepath.Ext(imageURL)
	imageCounter++
	imageFileName := fmt.Sprintf("image-%d%s", imageCounter, imageExt)

	if imageExt == ".webp" {
		img, err := webp.Decode(resp.Body)
		if err != nil {
			return fmt.Errorf("could not decode webp image: %w", err)
		}

		outPath := filepath.Join(folder, imageFileName+".png")
		outFile, err := os.Create(outPath)
		if err != nil {
			return fmt.Errorf("could not create output file: %w", err)
		}
		defer outFile.Close()

		err = png.Encode(outFile, img)
		if err != nil {
			return fmt.Errorf("could not encode image to png: %w", err)
		}
		fmt.Println("Converted and saved image as PNG:", outPath)
		return nil
	}

	outPath := filepath.Join(folder, imageFileName)
	outFile, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("could not create output file: %w", err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return fmt.Errorf("could not save image: %w", err)
	}

	fmt.Println("Downloaded image:", outPath)
	return nil
}

func crawlPage(baseURL, folder string) {
	c := colly.NewCollector(
		colly.CacheDir("./colly_cache"),
	)

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Request.AbsoluteURL(e.Attr("href"))
		if strings.HasPrefix(link, baseURL) {
			mu.Lock()
			if !visited[link] {
				visited[link] = true
				mu.Unlock()
				e.Request.Visit(link)
			} else {
				mu.Unlock()
			}
		}
	})

	c.OnHTML("img[src]", func(e *colly.HTMLElement) {
		imgURL := e.Request.AbsoluteURL(e.Attr("src"))
		go func() {
			err := downloadAndConvertImage(imgURL, folder)
			if err != nil {
				fmt.Println("Error downloading image:", err)
			}
		}()
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting:", r.URL.String())
	})

	c.Visit(baseURL)
}

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	baseURL := os.Getenv("WEBSITE_URL")
	folder := os.Getenv("DOWNLOAD_DIR")

	crawlPage(baseURL, folder)
}

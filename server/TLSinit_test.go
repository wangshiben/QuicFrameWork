package server

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateTLS(t *testing.T) {
	//CreateTLS()
	//CreateESDATLS()
	var fitelters []int
	testA := []int{}
	fmt.Println(fitelters)
	fmt.Println(testA)
	fitelters = append(fitelters, 1)
	fmt.Println(fitelters)
	//config, err := generateTLSConfig()
	//fmt.Println(config)
	//if err != nil {
	//	fmt.Println(err)
	//}
	//TempStart()
}
func UnzipFile(r *http.Request) {
	// 确保请求方法是POST
	if r.Method != http.MethodPost {
		return
	}

	// 解压目标目录
	destDir := "extracted_files"

	// 创建一个临时文件来存储上传的ZIP内容
	tempFile, err := os.CreateTemp("", "upload-*.zip")
	if err != nil {
		log.Println("Error creating temp file:", err)
		return
	}
	defer tempFile.Close()
	defer os.Remove(tempFile.Name()) // 清理临时文件
	// 将请求体的内容写入临时文件
	_, err = io.Copy(tempFile, r.Body)
	if err != nil {
		log.Println("Error copying request body to temp file:", err)
		return
	}

	// 重新打开临时文件用于读取
	_, err = tempFile.Seek(0, io.SeekStart)
	if err != nil {
		log.Println("Error seeking in temp file:", err)
		return
	}

	// 使用zip.Reader解压临时文件
	reader, err := zip.OpenReader(tempFile.Name())
	if err != nil {
		log.Println("Error opening zip reader:", err)
		return
	}
	defer reader.Close()

	// 解压
	for _, file := range reader.File {
		path := filepath.Join(destDir, file.Name)
		if file.FileInfo().IsDir() {
			err = os.MkdirAll(path, file.Mode())
			if err != nil {
				log.Println("Error creating directory:", err)
				continue
			}
		} else {
			srcFile, err := file.Open()
			if err != nil {
				log.Println("Error opening file in archive:", err)
				continue
			}
			defer srcFile.Close()

			outFile, err := os.Create(path)
			if err != nil {
				log.Println("Error creating file:", err)
				continue
			}
			defer outFile.Close()

			_, err = io.Copy(outFile, srcFile)
			if err != nil {
				log.Println("Error copying file content:", err)
				continue
			}

			err = os.Chmod(path, file.Mode())
			if err != nil {
				log.Println("Error setting file permissions:", err)
			}
		}
		fmt.Printf("Extracted %s\n", file.Name)
	}

}

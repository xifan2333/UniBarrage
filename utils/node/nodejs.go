package node

import (
	log "UniBarrage/utils/trace"
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"github.com/goccy/go-json"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type NodeRelease struct {
	Version  string      `json:"version"`
	Date     string      `json:"date"`
	Files    []string    `json:"files"`
	NPM      string      `json:"npm"`
	LTS      interface{} `json:"lts"`
	Security bool        `json:"security"`
}

// EnsureNodeInstalled 检查并安装 Node.js 的函数，并返回可用的 node 可执行文件路径
func EnsureNodeInstalled(destinationDir string) string {
	nodePath := findNodePath(destinationDir)
	if nodePath != "" {
		//fmt.Println("Node.js is already installed.")
		//log.Print("INFO", "Node.js is already installed.")
		return nodePath
	}

	nodePath = ""

	log.Print("INFO", "缺少必要的运行时依赖")
	version, link := getLatestNodeDownloadLink()
	if version != "" && link != "" {
		log.Printf("INFO", "获取最新版本的运行时: %s\n", version)
		//log.Printf("INFO","Download Link: %s\n", link)

		downloadPath := "nodejs-latest" + getFileExtension(link) // 根据链接扩展名确定下载文件名

		err := downloadFile(link, downloadPath)
		if err != nil {
			log.Printf("ERROR", "Failed to download file: %v", err)
		}

		err = extractFile(downloadPath, destinationDir)
		if err != nil {
			log.Printf("ERROR", "Failed to extract archive: %v", err)
		}

		nodePath = findNodePath(destinationDir)
		//log.Printf("INFO", "Node.js extracted to %s\n", destinationDir)
		return nodePath
	} else {
		log.Print("INFO", "找不到与系统适配的运行时下载链接")
	}

	return ""
}

// 查找 Node.js 的可执行文件路径
func findNodePath(path string) string {
	// 检查全局安装的 Node.js
	cmd := exec.Command("node", "-v")
	err := cmd.Run()
	if err == nil {
		nodePath, _ := exec.LookPath("node")
		return nodePath
	}

	// 在指定路径中模糊搜索 node 可执行文件
	var foundNodePath string
	err = filepath.WalkDir(path, func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && (d.Name() == "node" || d.Name() == "node.exe") {
			foundNodePath = filePath
			return filepath.SkipDir // 停止搜索
		}
		return nil
	})
	if err == nil && foundNodePath != "" {
		return foundNodePath
	}

	return ""
}

// 获取最新 Node.js 版本的下载链接
func getLatestNodeDownloadLink() (string, string) {
	url := "https://nodejs.org/download/release/index.json"

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("ERROR", "Failed to send request: %v", err)
		return "", ""
	}
	defer resp.Body.Close()

	var releases []NodeRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		log.Printf("ERROR", "Failed to parse JSON: %v", err)
		return "", ""
	}

	latestRelease := releases[0]
	platform := runtime.GOOS
	arch := runtime.GOARCH

	fileKey := getFileKey(platform, arch, latestRelease.Files)
	downloadLink := generateDownloadLink(latestRelease.Version, fileKey)

	return latestRelease.Version, downloadLink
}

// 获取下载链接的文件标识符
func getFileKey(platform, arch string, availableFiles []string) string {
	switch platform {
	case "linux":
		if arch == "amd64" && contains(availableFiles, "linux-x64") {
			return "linux-x64.tar.gz"
		} else if arch == "arm64" && contains(availableFiles, "linux-arm64") {
			return "linux-arm64.tar.gz"
		} else if strings.HasPrefix(arch, "arm") && contains(availableFiles, "linux-armv7l") {
			return "linux-armv7l.tar.gz"
		}
	case "darwin":
		if arch == "amd64" {
			return "osx-x64.tar.gz"
		} else if arch == "arm64" {
			return "osx-arm64.tar.gz"
		}
	case "windows":
		if arch == "amd64" {
			return "win-x64.zip"
		} else if arch == "arm64" {
			return "win-arm64.zip"
		}
	}
	return ""
}

// 生成下载链接
func generateDownloadLink(version, fileKey string) string {
	if fileKey != "" {
		baseURL := "https://nodejs.org/dist"
		return fmt.Sprintf("%s/%s/node-%s-%s", baseURL, version, version, fileKey)
	}
	return ""
}

// 下载文件
func downloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// 解压文件，根据扩展名自动判断解压方式，并在成功后删除压缩文件
func extractFile(filepath, destination string) error {
	var err error
	if strings.HasSuffix(filepath, ".zip") {
		err = extractZip(filepath, destination)
	} else if strings.HasSuffix(filepath, ".tar.gz") || strings.HasSuffix(filepath, ".tgz") {
		err = extractTarGz(filepath, destination)
	} else {
		return fmt.Errorf("unsupported file format: %s", filepath)
	}

	if err != nil {
		return err
	}

	// 检查解压后的内容是否需要移动
	err = moveContentsIfSingleSubdirectory(destination)
	if err != nil {
		return err
	}

	// 删除压缩文件
	err = os.Remove(filepath)
	if err != nil {
		return fmt.Errorf("failed to remove the archive file: %v", err)
	}

	//log.Printf("INFO", "Archive file %s removed successfully.\n", filepath)
	return nil
}

// 解压 ZIP 文件
func extractZip(zipPath, destination string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, file := range r.File {
		filePath := filepath.Join(destination, file.Name)
		if file.FileInfo().IsDir() {
			os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		defer outFile.Close()

		rc, err := file.Open()
		if err != nil {
			return err
		}

		_, err = io.Copy(outFile, rc)
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// 解压 TAR.GZ 文件
func extractTarGz(tarGzPath, destination string) error {
	file, err := os.Open(tarGzPath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	return extractTar(gzr, destination)
}

// 解压 TAR 文件
func extractTar(reader io.Reader, destination string) error {
	tr := tar.NewReader(reader)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // 结束
		}
		if err != nil {
			return err
		}

		targetPath := filepath.Join(destination, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.ModePerm); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), os.ModePerm); err != nil {
				return err
			}
			outFile, err := os.Create(targetPath)
			if err != nil {
				return err
			}
			defer outFile.Close()

			if _, err := io.Copy(outFile, tr); err != nil {
				return err
			}
		default:
			// 忽略其他类型
		}
	}
	return nil
}

// 检查并移动内容，如果解压后的目录只有一个子目录，将其内容移动到外部，并删除子目录
func moveContentsIfSingleSubdirectory(destination string) error {
	entries, err := os.ReadDir(destination)
	if err != nil {
		return err
	}

	if len(entries) == 1 && entries[0].IsDir() {
		subDir := filepath.Join(destination, entries[0].Name())
		subEntries, err := os.ReadDir(subDir)
		if err != nil {
			return err
		}

		for _, entry := range subEntries {
			oldPath := filepath.Join(subDir, entry.Name())
			newPath := filepath.Join(destination, entry.Name())

			err := os.Rename(oldPath, newPath)
			if err != nil {
				return err
			}
		}

		// 删除子目录
		err = os.Remove(subDir)
		if err != nil {
			return err
		}
	}

	return nil
}

// 获取文件扩展名
func getFileExtension(url string) string {
	if strings.HasSuffix(url, ".zip") {
		return ".zip"
	} else if strings.HasSuffix(url, ".tar.gz") || strings.HasSuffix(url, ".tgz") {
		return ".tar.gz"
	}
	return ""
}

// 检查切片中是否包含指定元素
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

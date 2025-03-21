package main

import (
        "bufio"
        "encoding/json"
        "fmt"
        "io"
        "net/http"
        "os"
        "path/filepath"
        "regexp"
        "strings"
        "time" // 新增：用于获取当前时间
        "sort" // 新增：用于排序当前时间
)

const baseDir = "./files" // 存储文件的基本目录
var requiredAccessCodes []string

// Response 包装响应消息
type Response struct {
        Message string `json:"message"`
}

// FileInfo 用于存储文件名和文件大小
type FileInfo struct {
        Name string `json:"name"`
        Size int64  `json:"size"`
}

// 读取配置文件
func loadConfig() {
        file, err := os.Open("env.conf")
        if err != nil {
                fmt.Println("Error opening config file:", err)
                return
        }
        defer file.Close()

        scanner := bufio.NewScanner(file)
        for scanner.Scan() {
                line := scanner.Text()
                if strings.HasPrefix(line, "accessCodes=") {
                        accessCodeStr := strings.TrimPrefix(line, "accessCodes=")
                        requiredAccessCodes = strings.Split(accessCodeStr, ",")
                }
        }

        if err := scanner.Err(); err != nil {
                fmt.Println("Error reading config file:", err)
        }
}

// 检查授权
func checkAccess(r *http.Request) (string, string, bool) {
        accessCode := r.Header.Get("accessCode")
        collisionString := r.Header.Get("collisionString")

        // 检查 accessCode 是否在列表中
        validAccessCode := false
        for _, code := range requiredAccessCodes {
                if accessCode == code {
                        validAccessCode = true
                        break
                }
        }
        if !validAccessCode || collisionString == "" {
                return "", "", false
        }
        return accessCode, collisionString, true
}

// 获取文件的根目录路径
func getDirectoryPath(collisionString, accessCode string) string {
        return filepath.Join(baseDir, collisionString, accessCode)
}

// cleanPathMiddleware 是一个中间件函数，用于清理请求路径中的多余斜杠
func cleanPathMiddleware(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                // 使用正则表达式去除路径中的多余斜杠
                cleanPath := cleanURLPath(r.URL.Path)
                // 更新请求的路径
                r.URL.Path = cleanPath
                // 将请求传递给下一个处理程序
                next.ServeHTTP(w, r)
        })
}

// cleanURLPath 使用正则表达式清理路径中的多余斜杠
func cleanURLPath(path string) string {
        // 替换多个斜杠为一个斜杠
        return regexp.MustCompile(`/+`).ReplaceAllString(path, "/")
}

// 设置CORS头和记录日志，同时记录 accessCode 和请求时间
func setCORSAndLog(w http.ResponseWriter, r *http.Request, task string) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, accessCode, collisionString")
        w.Header().Set("Access-Control-Max-Age", "86400") // 缓存时间，1天

        // 获取当前时间
        currentTime := time.Now().Format("2006-01-02 15:04:05")

        // 记录请求的IP、任务、accessCode和时间
        ip := r.RemoteAddr
        accessCode := r.Header.Get("accessCode")

        if xForwardedFor := r.Header.Get("X-Forwarded-For"); xForwardedFor != "" {
                parts := strings.Split(xForwardedFor, ",")
                if len(parts) > 0 {
                        ip = strings.TrimSpace(parts[0])
                }
        } else if xRealIP := r.Header.Get("X-Real-IP"); xRealIP != "" {
                ip = xRealIP
        }

        if task != "Preflight" && task != "Options" {
                fmt.Printf("[%s] Request from IP: %s, Task: %s, AccessCode: %s\n", currentTime, ip, task, accessCode)
        }
}

// 处理OPTIONS请求
func handlePreflight(w http.ResponseWriter, r *http.Request) {
        setCORSAndLog(w, r, "Preflight")
        if r.Method == http.MethodOptions {
                w.Header().Set("Access-Control-Allow-Origin", "*")
                w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
                w.Header().Set("Access-Control-Allow-Headers", "Content-Type, accessCode, collisionString")
                w.Header().Set("Access-Control-Max-Age", "86400") // 缓存1天

                w.WriteHeader(http.StatusOK)
                return
        }
}

// jsonError 用于返回JSON格式的错误消息
func jsonError(w http.ResponseWriter, message string, statusCode int) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(statusCode)
        json.NewEncoder(w).Encode(Response{Message: message})
}

// 上传文件处理
func handleBackup(w http.ResponseWriter, r *http.Request) {
        setCORSAndLog(w, r, "Backup")
        if r.Method == http.MethodOptions {
                w.WriteHeader(http.StatusOK)
                return
        }
        if r.Method != http.MethodPost {
                jsonError(w, "只允许使用 POST 方法", http.StatusMethodNotAllowed)
                return
        }

        accessCode, collisionString, valid := checkAccess(r)
        if !valid {
                jsonError(w, "无效的 accessCode 或 collisionString", http.StatusForbidden)
                return
        }

        // 创建目录路径
        dirPath := getDirectoryPath(collisionString, accessCode)
        err := os.MkdirAll(dirPath, os.ModePerm)
        if err != nil {
                jsonError(w, "无法创建目录", http.StatusInternalServerError)
                return
        }

        // 读取文件
        file, header, err := r.FormFile("file")
        if err != nil {
                jsonError(w, "读取文件失败", http.StatusBadRequest)
                return
        }
        defer file.Close()

        // 保存文件到指定路径
        filePath := filepath.Join(dirPath, header.Filename)

        dst, err := os.Create(filePath)
        if err != nil {
                jsonError(w, "保存文件失败", http.StatusInternalServerError)
                return
        }
        defer dst.Close()

        _, err = io.Copy(dst, file)
        if err != nil {
                jsonError(w, "复制文件失败", http.StatusInternalServerError)
                return
        }

        json.NewEncoder(w).Encode(Response{Message: "文件上传成功！"})
}

// 下载文件处理
func handleImport(w http.ResponseWriter, r *http.Request) {
        setCORSAndLog(w, r, "Import")

        if r.Method == http.MethodOptions {
                w.WriteHeader(http.StatusOK)
                return
        }
        if r.Method != http.MethodGet {
                jsonError(w, "只允许使用 GET 方法", http.StatusMethodNotAllowed)
                return
        }

        accessCode, collisionString, valid := checkAccess(r)
        if !valid {
                jsonError(w, "无效的 accessCode 或 collisionString", http.StatusForbidden)
                return
        }

        // 获取文件路径
        fileName := r.URL.Query().Get("filename")
        if fileName == "" {
                jsonError(w, "缺少 filename 参数", http.StatusBadRequest)
                return
        }

        filePath := filepath.Join(getDirectoryPath(collisionString, accessCode), fileName)
        if _, err := os.Stat(filePath); os.IsNotExist(err) {
                jsonError(w, "文件不存在", http.StatusNotFound)
                return
        }
        http.ServeFile(w, r, filePath)
}

// 获取文件列表
func handleGetList(w http.ResponseWriter, r *http.Request) {
    setCORSAndLog(w, r, "GetList")

    if r.Method == http.MethodOptions {
        w.WriteHeader(http.StatusOK)
        return
    }
    if r.Method != http.MethodGet {
        jsonError(w, "只允许使用 GET 方法", http.StatusMethodNotAllowed)
        return
    }

    accessCode, collisionString, valid := checkAccess(r)
    if !valid {
        jsonError(w, "无效的 accessCode 或 collisionString", http.StatusForbidden)
        return
    }

    // 获取目录下的文件列表
    dirPath := getDirectoryPath(collisionString, accessCode)
    entries, err := os.ReadDir(dirPath)
    if err != nil {
        jsonError(w, "读取目录失败", http.StatusInternalServerError)
        return
    }

    // 创建包含修改时间的切片
    type fileWithModTime struct {
        info FileInfo
        modTime time.Time
    }
    
    var files []fileWithModTime
    
    for _, entry := range entries {
        if entry.IsDir() {
            continue
        }
        info, err := entry.Info()
        if err != nil {
            fmt.Printf("获取文件信息错误 %s: %v\n", entry.Name(), err)
            continue
        }
        files = append(files, fileWithModTime{
            info:    FileInfo{Name: entry.Name(), Size: info.Size()},
            modTime: info.ModTime(),
        })
    }

    // 按修改时间降序排序（新文件在前）
    sort.Slice(files, func(i, j int) bool {
        return files[i].modTime.After(files[j].modTime)
    })

    // 提取排序后的文件信息
    fileList := make([]FileInfo, len(files))
    for i, f := range files {
        fileList[i] = f.info
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(fileList)
}

// 重命名文件处理
func handleRename(w http.ResponseWriter, r *http.Request) {
        setCORSAndLog(w, r, "Rename")

        if r.Method == http.MethodOptions {
                w.WriteHeader(http.StatusOK)
                return
        }
        if r.Method != http.MethodPost {
                jsonError(w, "只允许使用 POST 方法", http.StatusMethodNotAllowed)
                return
        }

        accessCode, collisionString, valid := checkAccess(r)
        if !valid {
                jsonError(w, "无效的 accessCode 或 collisionString", http.StatusForbidden)
                return
        }

        // 解析请求体中的 JSON
        var data struct {
                OldName string `json:"oldName"`
                NewName string `json:"newName"`
        }
        if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
                jsonError(w, "无效的 JSON 输入", http.StatusBadRequest)
                return
        }

        // 获取旧文件路径
        oldFilePath := filepath.Join(getDirectoryPath(collisionString, accessCode), data.OldName)
        newFilePath := filepath.Join(getDirectoryPath(collisionString, accessCode), data.NewName)

        // 检查旧文件是否存在
        if _, err := os.Stat(oldFilePath); os.IsNotExist(err) {
                jsonError(w, "旧文件不存在", http.StatusNotFound)
                return
        }

        // 检查新文件是否已存在
        if _, err := os.Stat(newFilePath); err == nil {
                jsonError(w, "新文件名已存在", http.StatusConflict)
                return
        }

        // 重命名文件
        err := os.Rename(oldFilePath, newFilePath)
        if err != nil {
                jsonError(w, "重命名文件失败", http.StatusInternalServerError)
                return
        }

        json.NewEncoder(w).Encode(Response{Message: "文件重命名成功！"})
}

// 删除文件处理
func handleDelete(w http.ResponseWriter, r *http.Request) {
        setCORSAndLog(w, r, "Delete")

        if r.Method == http.MethodOptions {
                w.WriteHeader(http.StatusOK)
                return
        }
        if r.Method != http.MethodDelete {
                jsonError(w, "只允许使用 DELETE 方法", http.StatusMethodNotAllowed)
                return
        }

        accessCode, collisionString, valid := checkAccess(r)
        if !valid {
                jsonError(w, "无效的 accessCode 或 collisionString", http.StatusForbidden)
                return
        }

        // 获取文件名
        fileName := strings.TrimPrefix(r.URL.Path, "/api/delete/")
        if fileName == "" {
                jsonError(w, "缺少文件名", http.StatusBadRequest)
                return
        }

        // 删除文件
        filePath := filepath.Join(getDirectoryPath(collisionString, accessCode), fileName)
        if _, err := os.Stat(filePath); os.IsNotExist(err) {
                jsonError(w, "文件不存在", http.StatusNotFound)
                return
        }

        err := os.Remove(filePath)
        if err != nil {
                jsonError(w, "删除文件失败", http.StatusInternalServerError)
                return
        }

        json.NewEncoder(w).Encode(Response{Message: "文件成功从云端删除，该操作不可撤回！"})
}

// 删除所有文件处理
func handleDeleteAll(w http.ResponseWriter, r *http.Request) {
        setCORSAndLog(w, r, "DeleteAll")

        if r.Method == http.MethodOptions {
                w.WriteHeader(http.StatusOK)
                return
        }
        if r.Method != http.MethodDelete {
                jsonError(w, "只允许使用 DELETE 方法", http.StatusMethodNotAllowed)
                return
        }

        accessCode, collisionString, valid := checkAccess(r)
        if !valid {
                jsonError(w, "无效的 accessCode 或 collisionString", http.StatusForbidden)
                return
        }

        // 获取目录路径
        dirPath := getDirectoryPath(collisionString, accessCode)

        // 确保目录存在
        if _, err := os.Stat(dirPath); os.IsNotExist(err) {
                jsonError(w, "目录不存在", http.StatusNotFound)
                return
        }

        // 删除目录下的所有文件
        err := os.RemoveAll(dirPath)
        if err != nil {
                jsonError(w, "清空文件失败", http.StatusInternalServerError)
                return
        }

        // 重新创建目录
        err = os.MkdirAll(dirPath, os.ModePerm)
        if err != nil {
                jsonError(w, "无法重新创建目录", http.StatusInternalServerError)
                return
        }

        json.NewEncoder(w).Encode(Response{Message: "所有文件已成功清空！"})
}

func main() {
        loadConfig() // 加载配置文件

        // 使用 http.NewServeMux 来注册路由
        mux := http.NewServeMux()

        // 处理不同的 API 路由
        mux.HandleFunc("/", handlePreflight)           // 处理所有的OPTIONS请求
        mux.HandleFunc("/api/backup", handleBackup)
        mux.HandleFunc("/api/import", handleImport)
        mux.HandleFunc("/api/getlist", handleGetList)
        mux.HandleFunc("/api/rename", handleRename)
        mux.HandleFunc("/api/delete/", handleDelete)    // 使用通配符匹配文件名
        mux.HandleFunc("/api/deleteALL", handleDeleteAll)

        fmt.Println("Server running on http://localhost:3456")
        http.ListenAndServe(":3456", cleanPathMiddleware(mux))
}

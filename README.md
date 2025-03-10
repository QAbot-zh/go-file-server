# 超简易文件服务器

## 项目概述

本项目是一个基于 Go 语言开发的超简易文件服务器，旨在提供轻量级的文件服务功能，**简化备份和文件管理操作**，特别是为 [ChatGPT-Next-Web](https://github.com/QAbot-zh/ChatGPT-Next-Web) 提供高效的云备份解决方案。

---

## 项目背景

[ChatGPT-Next-Web](https://github.com/QAbot-zh/ChatGPT-Next-Web) 项目提供了强大的对话生成平台，但其“导入”和“导出”备份方式较为繁琐，限制了高频备份和文件管理的效率。本文件服务器通过提供简单的 **云备份接口**，极大优化了文件管理流程，助力 ChatGPT-Next-Web 用户更方便地保存和管理数据。

---

## 功能特点

- **轻量级**：采用 Go 语言开发，简单高效。
- **云备份支持**：提供便捷的文件上传、备份和管理功能。
- **适配 ChatGPT-Next-Web**：针对其备份与导入需求，提供高度集成的服务。
- **多功能路由支持**：涵盖备份、获取列表、删除等核心操作。
- **用户聊天记录隔离**：默认使用原项目的 CODE 进行用户聊天记录的隔离，确保不同用户的数据安全和独立。

---

## 配合 ChatGPT-Next-Web 使用的具体说明

本项目通过以下路由提供完整的云备份管理功能：

### 路由功能列表

| 路由                        | HTTP 方法 | 功能描述                              |
|-----------------------------|-----------|---------------------------------------|
| `/api/backup`               | `POST`    | 上传并备份文件                        |
| `/api/getlist`              | `GET`     | 获取已备份文件列表                    |
| `/api/import`               | `GET`     | 导入备份文件                          |
| `/api/delete/{filename}`    | `DELETE`  | 删除指定备份文件                      |
| `/api/deleteALL`            | `DELETE`  | 删除所有备份文件                      |
| `/api/rename`               | `POST`    | 重命名备份文件                        |

### 用户聊天记录隔离

本项目默认使用原项目的 CODE 进行用户聊天记录的隔离。每个用户的聊天记录将被存储在独立的文件夹中，文件夹名称与用户的 CODE 相对应。这样可以确保不同用户的数据安全和独立。

---

## 部署与使用说明

### Go 文件服务器部署与使用完整教程  

---

#### **简介**  
本教程指导如何部署一个基于 Go 编写的简易文件服务器，支持文件上传、下载、删除和目录查看功能。服务默认监听 `3456` 端口。
---

#### **1. 准备 Go 环境**  
**1.1 安装 Go**  
- **Ubuntu/Debian**  
  ```bash
  sudo apt update
  sudo apt install golang
  ```  
- **其他系统**  
  参考 [Go 官方安装指南](https://golang.org/doc/install)。  

**1.2 验证安装**  
```bash
go version
# 确保版本 ≥1.18（若版本过低，需手动安装新版）
```

---

#### **2. 获取项目代码**  
**2.1 克隆仓库**  
```bash
git clone https://github.com/QAbot-zh/go-file-server.git
cd go-file-server
```  

**2.2 （备用）直接下载代码**  
如果未安装 Git，可从仓库主页下载 ZIP 文件：  
https://github.com/QAbot-zh/go-file-server  

---

#### **3. 编译与运行**  
**3.1 编译项目**  
在项目根目录执行：  
```bash
go build  main.go
```  
生成可执行文件 `main`。  

**3.2 启动服务**  
- **前台运行（调试用）**  
  ```bash
  ./main
  ```  
- **后台运行（生产环境）**  
  ```bash
  nohup ./main > output.log 2>&1 &
  ```  
  日志将输出到 `output.log`。  

---

#### **4. 验证服务状态**  
**4.1 检查进程**  
```bash
ps aux | grep main
# 应看到类似 `/path/to/main` 的进程
```  

**4.2 测试端口监听**  
```bash
netstat -tuln | grep 3456
# 输出应有 `LISTEN` 状态
```  

   服务器启动后监听默认端口，提供路由服务。

![image](https://github.com/user-attachments/assets/469144be-fb8b-4216-9da1-57c1a84864ad)

---

## 贡献

欢迎对本项目提出建议或贡献代码。您可以通过 GitHub 提交 issue 或 pull request，与我们一起优化项目。

---

## 许可证

本项目采用 MIT 许可证，允许用户自由使用、修改和分发。详情请参阅 LICENSE 文件。

---

感谢您对本项目的支持！如有问题，欢迎与我们联系。

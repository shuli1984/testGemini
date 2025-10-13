# Go-CMS-i18n

[English](README.md) | [中文](README.zh.md) | [日本語](README.ja.md)

一个轻量级的、基于文件的内容管理系统（CMS），内置国际化（i18n）支持，使用 Go 语言构建。该项目完全由 Gemini 生成。

## 项目概述

Go-CMS-i18n 是一个简单的内容管理系统，旨在易于使用和支持多语言内容。它提供一个管理后台，用于管理页面、设置和翻译。其前端设计简洁、现代化且易于定制。

## 功能特性

* **页面管理:** 创建、编辑和删除页面。
* **管理后台:** 一个安全的管理区域，用于管理网站内容和设置。
* **国际化 (i18n):** 支持多语言内容，使用基于 JSON 的翻译文件。
* **可配置的首页轮播:** 直接从管理后台动态选择要在首页轮播中展示的页面。
* **自动翻译:** 集成了 Google Cloud Translation、DeepL 和一个模拟翻译器，可轻松在语言之间翻译内容。
* **维护模式:** 可轻松为所有非管理员用户将网站置于维护模式。
* **基于文件:** 使用本地 SQLite 数据库文件进行数据持久化。

## 先决条件

* **Go:** 1.18 或更高版本。从 [golang.org](https://golang.org/dl/) 下载。
* **Git:** (可选) 用于克隆代码仓库。

## 快速入门

### 1. 克隆仓库

```bash
git clone <repository_url>
cd <project_directory>
```

### 2. 设置环境变量

本项目使用 `.env` 文件来存储敏感密钥和本地配置。

1. **创建 `.env` 文件:** 在项目根目录下，创建一个名为 `.env` 的文件。
2. **添加变量:** 使用以下内容填充该文件。为 `SESSION_KEY` 和 `CSRF_KEY` 生成强随机值。

    ```bash
    SESSION_KEY=your_long_random_session_key_here_at_least_32_bytes
    CSRF_KEY=your_long_random_csrf_key_here_at_least_32_bytes
    ADMIN_USERNAME=admin
    ADMIN_PASSWORD=password
    TRANSLATOR_API_KEY="your_google_cloud_api_key_here"
    ```

    * **注意:** `.env` 文件已在 `.gitignore` 中列出，绝不应提交到版本控制中。

### 3. 安装依赖

下载所需的 Go 模块。

```bash
go mod tidy
```

### 4. 运行应用程序 (开发模式)

此命令将启动用于开发的 Web 服务器。

```bash
go run ./cmd/web
```

应用程序将在 `config.yml` 中指定的地址上监听 (默认为 `:8080`)。

### 5. 构建应用程序 (生产模式)

要构建生产环境的可执行文件，请运行构建脚本。

**Windows:**

```bash
build.bat
```

**Linux/macOS:**

```bash
go build -o bin/main cmd/web/main.go
```

这将在 `bin/` 目录下创建一个可执行文件。

## 配置

可以通过 `config.yml` 和环境变量（环境变量优先）来配置应用程序行为。

### 常规配置

**`config.yml`:**

```yaml
server:
  addr: ":8080"
  timezone: "UTC"
i18n:
  default_language: "en"
```

### 首页轮播

可以从管理后台配置首页轮播：

1. 登录到 `/admin`。
2. 导航到 **Settings** (设置)。
3. 在 **Homepage Carousel** (首页轮播) 部分，您可以将页面从 “Available Pages” (可用页面) 拖放到 “Selected Pages” (已选页面) 中，以在首页上展示它们。

### 翻译服务

在 `config.yml` 中或通过环境变量配置翻译服务。

**`config.yml`:**

```yaml
translator:
  type: "mock" # 可选 "google", "deepl", 或 "mock"
  api_key: "YOUR_API_KEY_HERE"
```

**环境变量:**

```bash
TRANSLATOR_TYPE="google"
TRANSLATOR_API_KEY="YOUR_API_KEY_HERE"
```

* **`mock`:** 用于开发的默认翻译器。不进行外部 API 调用。
* **`google`:** 使用 Google Cloud Translation API。需要有效的 API 密钥。
* **`deepl`:** 使用 DeepL API。需要有效的 API 密钥。

### 维护模式

在管理后台的 **Settings** (设置) 页面启用或禁用网站维护模式。启用后，只有登录的管理员才能浏览网站。

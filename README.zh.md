# 项目设置与运行说明

[English](README.md) | [中文](README.zh.md) | [日本語](README.ja.md)

本项目完全由 Gemini 生成。

本项目是一个 Go 应用程序。请按照以下说明来设置和运行该应用程序。

## 先决条件

- **Go:** Go 1.18 或更高版本。您可以从 [golang.org](https://golang.org/dl/) 下载。
- **Git:** (可选，用于克隆仓库) 如果您需要从版本控制系统克隆项目，请确保已安装 Git。

## 开始

1. **克隆仓库 (如果适用):**

    ```bash
    git clone <repository_url>
    cd <project_directory>
    ```

2. **导航到项目目录:**

    ```bash
    cd /path/to/your/project
    ```

3. **下载依赖:**

    ```bash
    go mod tidy
    ```

4. **运行应用程序:**

    ```bash
    go run ./cmd/web
    ```

    应用程序将启动并监听 `config.yml` 中指定的地址 (默认为 `:8080`)。

## 环境变量 (.env)

本项目使用 `.env` 文件来管理本地环境变量，特别是像会话（session）和 CSRF 令牌以及默认管理员凭据等敏感密钥。这可以防止在代码库中直接硬编码敏感信息，并确保在开发过程中应用程序重启时行为一致。

**设置:**

1. **创建 `.env` 文件:** 在项目的根目录中，创建一个名为 `.env` 的文件。
2. **添加变量:** 使用以下变量填充 `.env` 文件。您应该为 `SESSION_KEY` 和 `CSRF_KEY` 生成强大的随机值。

    ```bash
    SESSION_KEY=your_long_random_session_key_here_at_least_32_bytes
    CSRF_KEY=your_long_random_csrf_key_here_at_least_32_bytes
    ADMIN_USERNAME=admin
    ADMIN_PASSWORD=password
    TRANSLATOR_API_KEY="your_google_cloud_api_key_here"
    ```

    - **生成随机密钥:** 您可以使用命令行工具生成随机字符串。例如：
        - **Linux/macOS:** `head /dev/urandom | tr -dc A-Za-z0-9 | head -c 64 ; echo ''`
        - **Windows (PowerShell):** `[System.Convert]::ToBase64String((New-Object Byte[] 32 | Get-Random -Count 32))` (用于一个 32 字节的 base64 编码字符串)
        - 或使用在线随机字符串生成器。

3. **重要提示:** `.env` 文件已在 `.gitignore` 中列出，**绝不**应提交到您的版本控制系统中。

应用程序将在启动时自动加载这些变量。如果找不到 `.env` 文件或未设置变量，应用程序将回退到临时生成的密钥 (对于 `SESSION_KEY`, `CSRF_KEY`) 或默认值 (对于 `ADMIN_USERNAME`, `ADMIN_PASSWORD`)，这可能导致重启时出现会话/cookie 问题。

## 翻译功能

该应用程序包含一个使用谷歌云翻译 API、DeepL API 或用于开发的模拟翻译器来翻译页面内容的功能。可以在 `config.yml` 文件中配置所需的翻译服务。

## 用法

1. 在管理面板中导航到您希望翻译的页面 (`管理 -> 页面 -> 编辑`)。
2. 要翻译单个字段 (如标题、描述或主要内容)，请单击该字段旁边的“翻译”按钮。如果该字段已包含内容，将出现一个确认对话框以防止意外覆盖。
3. 要一次性翻译页面上的所有文本字段，请单击表单底部的“全部翻译”按钮。在继续操作之前，也会出现一个确认对话框。

翻译过程将始终使用**配置的默认语言** (见下文“默认语言配置”) 作为源语言，并翻译到当前选择的编辑语言。这使您可以从主要源重新翻译现有内容。

## 配置

## 默认语言配置

应用程序使用默认语言作为翻译的主要来源和后备。这可以在 `config.yml` 中配置，或通过环境变量覆盖。

**`config.yml`:**

```yaml
i18n:
  default_language: "en" # 设置您期望的默认语言 (例如, "en", "zh", "ja")
```

**环境变量:**

您可以通过在 `.env` 文件 (或系统环境) 中定义 `I18N_DEFAULT_LANGUAGE` 环境变量来覆盖 `config.yml` 的设置。这对于本地开发和测试特别有用。

```bash
I18N_DEFAULT_LANGUAGE="zh" # 示例: 覆盖 config.yml 以使用中文作为默认语言
```

---

翻译服务在 `config.yml` 文件的 `translator` 部分或通过环境变量进行配置。您可以在 `google`、`deepl` 和 `mock` 翻译器之间进行选择。

**`config.yml`:**

```yaml
translator:
  type: "mock" # 可以是 "google", "deepl", 或 "mock"
  api_key: "YOUR_API_KEY_HERE" # Google 和 DeepL 需要
```

**环境变量:**

```bash
TRANSLATOR_TYPE="mock" # 可以是 "google", "deepl", 或 "mock"
TRANSLATOR_API_KEY="YOUR_API_KEY_HERE"
```

## 模拟翻译器

这是默认的翻译器，用于开发。它不调用任何外部 API。它只是将字符串 `"Translated from [source] to [target]: "` 前置到原始文本。

要使用模拟翻译器，请在 `config.yml` 中将 `type` 设置为 `"mock"`。

## 谷歌云翻译 API

要使用谷歌云翻译 API，您必须提供一个 API 密钥。

1. **获取 API 密钥:** 从您的谷歌云平台项目获取一个 API 密钥，并确保已启用“Cloud Translation API”。
2. **设置配置:**
    - 在 `config.yml` 中，将 `translator.type` 设置为 `"google"`。
    - 将 `translator.api_key` 设置为您的谷歌云 API 密钥，或设置 `TRANSLATOR_API_KEY` 环境变量。

## DeepL API

要使用 DeepL API，您必须提供一个 API 密钥。

1. **获取 API 密钥:** 从您的 DeepL 帐户获取一个 API 密钥。
2. **设置配置:**
    - 在 `config.yml` 中，将 `translator.type` 设置为 `"deepl"`。
    - 将 `translator.api_key` 设置为您的 DeepL API 密钥，或设置 `TRANSLATOR_API_KEY` 环境变量。

## 维护模式

您可以在管理后台的“设置”页面中启用或禁用站点维护模式。

- **启用维护模式**:
  1. 登录到管理后台 (`/admin`)。
  2. 导航到“设置”页面 (`/admin/settings`)。
  3. 勾选“启用维护模式”复选框并保存设置。

- **重要提示**:
  - 启用维护模式后，所有未登录的访问者将看到维护页面。
  - **已登录的管理员不会看到维护页面**，可以正常访问和使用网站。
  - 如需测试维护页面，请确保您已**登出**管理后台，或使用另一个浏览器访问您的站点。

# Go-CMS-i18n

[English](README.md) | [中文](README.zh.md) | [日本語](README.ja.md)

Goで構築された、国際化（i18n）をサポートする軽量なファイルベースのCMSです。このプロジェクトは完全にGeminiによって生成されました。

## プロジェクト概要

Go-CMS-i18nは、使いやすさと多言語コンテンツを重視して設計されたシンプルなコンテンツ管理システムです。ページ、設定、翻訳を管理するための管理パネルを備えています。フロントエンドは、クリーンでモダン、そして簡単にカスタマイズできるように設計されています。

## 機能

* **ページ管理:** ページの作成、編集、削除。
* **管理パネル:** サイトのコンテンツと設定を管理するための安全な管理エリア。
* **国際化 (i18n):** JSONベースの翻訳ファイルを使用して、コンテンツの多言語をサポート。
* **設定可能なホームページカルーセル:** 管理パネルから直接、ホームページカルーセルに表示するページを動的に選択。
* **自動翻訳:** Google Cloud Translation、DeepL、およびモックトランスレータと統合し、言語間でコンテンツを簡単に翻訳。
* **メンテナンスモード:** 管理者以外のすべてのユーザーに対して、サイトを簡単にメンテナンスモードに設定。
* **ファイルベース:** データの永続化にローカルのSQLiteデータベースファイルを使用。

## 前提条件

* **Go:** バージョン1.18以上。[golang.org](https://golang.org/dl/)からダウンロードしてください。
* **Git:** (オプション) リポジトリのクローン用。

## はじめに

### 1. リポジトリをクローンする

```bash
git clone <repository_url>
cd <project_directory>
```

### 2. 環境変数を設定する

このプロジェクトでは、機密キーやローカル設定に`.env`ファイルを使用します。

1. **`.env`ファイルを作成する:** プロジェクトのルートに`.env`という名前のファイルを作成します。
2. **変数を追加する:** 以下の内容でファイルを入力します。`SESSION_KEY`と`CSRF_KEY`には、強力でランダムな値を生成してください。

    ```bash
    SESSION_KEY=your_long_random_session_key_here_at_least_32_bytes
    CSRF_KEY=your_long_random_csrf_key_here_at_least_32_bytes
    ADMIN_USERNAME=admin
    ADMIN_PASSWORD=password
    TRANSLATOR_API_KEY="your_google_cloud_api_key_here"
    ```

    * **注意:** `.env`ファイルは`.gitignore`に記載されており、バージョン管理には**決して**コミットしないでください。

### 3. 依存関係をインストールする

必要なGoモジュールをダウンロードします。

```bash
go mod tidy
```

### 4. アプリケーションを実行する (開発)

このコマンドは、開発用にWebサーバーを起動します。

```bash
go run ./cmd/web
```

アプリケーションは`config.yml`で指定されたアドレス（デフォルト：`:8080`）でリッスンします。

### 5. アプリケーションをビルドする (本番)

本番用の実行可能ファイルをビルドするには、ビルドスクリプトを実行します。

**Windows:**

```bash
build.bat
```

**Linux/macOS:**

```bash
go build -o bin/main cmd/web/main.go
```

これにより、`bin/`ディレクトリに実行可能ファイルが作成されます。

## 設定

アプリケーションの動作は、`config.yml`および環境変数（環境変数が優先されます）を介して設定できます。

### 一般設定

**`config.yml`:**

```yaml
server:
  addr: ":8080"
  timezone: "UTC"
i18n:
  default_language: "en"
```

### ホームページカルーセル

ホームページカルーセルは、管理パネルから設定できます。

1. `/admin`にログインします。
2. **設定**に移動します。
3. **ホームページカルーセル**セクションで、「利用可能なページ」から「選択されたページ」にページをドラッグアンドドロップして、ホームページに表示させることができます。

### 翻訳サービス

`config.yml`または環境変数を介して翻訳サービスを設定します。

**`config.yml`:**

```yaml
translator:
  type: "mock" # "google"、"deepl"、または "mock"
  api_key: "YOUR_API_KEY_HERE"
```

**環境変数:**

```bash
TRANSLATOR_TYPE="google"
TRANSLATOR_API_KEY="YOUR_API_KEY_HERE"
```

* **`mock`:** 開発用のデフォルトのトランスレータ。外部API呼び出しはありません。
* **`google`:** Google Cloud Translation APIを使用します。有効なAPIキーが必要です。
* **`deepl`:** DeepL APIを使用します。有効なAPIキーが必要です。

### メンテナンスモード

管理パネルの**設定**ページで、サイトのメンテナンスモードを有効または無効にします。有効にすると、ログインしている管理者のみがサイトを閲覧できます。

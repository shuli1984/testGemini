# プロジェクトのセットアップと実行手順

[English](README.md) | [中文](README.zh.md) | [日本語](README.ja.md)

このプロジェクトは完全にGeminiによって生成されました。

このプロジェクトはGoアプリケーションです。アプリケーションをセットアップして実行するには、以下の手順に従ってください。

## 前提条件

- **Go:** Go 1.18以上。 [golang.org](https://golang.org/dl/) からダウンロードできます。
- **Git:** (オプション、リポジトリのクローン用) バージョン管理システムからプロジェクトをクローンする必要がある場合は、Gitがインストールされていることを確認してください。

## はじめに

1. **リポジトリをクローンする (該当する場合):**

    ```bash
    git clone <repository_url>
    cd <project_directory>
    ```

2. **プロジェクトディレクトリに移動する:**

    ```bash
    cd /path/to/your/project
    ```

3. **依存関係をダウンロードする:**

    ```bash
    go mod tidy
    ```

4. **アプリケーションを実行する:**

    ```bash
    go run ./cmd/web
    ```

    アプリケーションが起動し、`config.yml`で指定されたアドレス（デフォルト：`:8080`）でリッスンします。

## 環境変数 (.env)

このプロジェクトでは、特にセッションキーやCSRFトークン、デフォルトの管理者認証情報などの機密キーについて、ローカルの環境変数を管理するために`.env`ファイルを使用します。これにより、機密情報をコードベースに直接ハードコーディングすることを防ぎ、開発中のアプリケーション再起動時の一貫した動作を保証します。

**セットアップ:**

1. **`.env`ファイルを作成する:** プロジェクトのルートディレクトリに`.env`という名前のファイルを作成します。
2. **変数を追加する:** `.env`ファイルに以下の変数を入力します。`SESSION_KEY`と`CSRF_KEY`には、強力でランダムな値を生成する必要があります。

    ```bash
    SESSION_KEY=your_long_random_session_key_here_at_least_32_bytes
    CSRF_KEY=your_long_random_csrf_key_here_at_least_32_bytes
    ADMIN_USERNAME=admin
    ADMIN_PASSWORD=password
    TRANSLATOR_API_KEY="your_google_cloud_api_key_here"
    ```

    - **ランダムキーの生成:** コマンドラインツールを使用してランダムな文字列を生成できます。例：
        - **Linux/macOS:** `head /dev/urandom | tr -dc A-Za-z0-9 | head -c 64 ; echo ''`
        - **Windows (PowerShell):** `[System.Convert]::ToBase64String((New-Object Byte[] 32 | Get-Random -Count 32))` (32バイトのbase64エンコード文字列の場合)
        - または、オンラインのランダム文字列ジェネレータを使用します。

3. **重要事項:** `.env`ファイルは`.gitignore`に記載されており、バージョン管理システムに**決して**コミットしないでください。

アプリケーションは起動時にこれらの変数を自動的に読み込みます。`.env`ファイルが見つからない場合や変数が設定されていない場合、アプリケーションは一時的に生成されたキー（`SESSION_KEY`、`CSRF_KEY`用）またはデフォルト値（`ADMIN_USERNAME`、`ADMIN_PASSWORD`用）にフォールバックし、再起動時にセッション/Cookieの問題が発生する可能性があります。

## 翻訳機能

このアプリケーションには、Google Cloud Translation API、DeepL API、または開発用のモックトランスレータを使用してページコンテンツを翻訳する機能が含まれています。希望する翻訳サービスは`config.yml`ファイルで設定できます。

## 使用方法

1. 管理パネルで翻訳したいページに移動します（`管理 -> ページ -> 編集`）。
2. 単一のフィールド（タイトル、説明、メインコンテンツなど）を翻訳するには、そのフィールドの横にある「翻訳」ボタンをクリックします。フィールドに既にコンテンツが含まれている場合は、誤って上書きするのを防ぐために確認ダイアログが表示されます。
3. ページ上のすべてのテキストフィールドを一度に翻訳するには、フォームの下部にある「すべて翻訳」ボタンをクリックします。処理を進める前に確認ダイアログも表示されます。

翻訳プロセスでは、常に**設定されたデフォルト言語**（下記の「デフォルト言語設定」を参照）をソースとして使用し、現在選択されている編集言語に翻訳します。これにより、プライマリソースから既存のコンテンツを再翻訳できます。

## 設定

## デフォルト言語設定

アプリケーションは、翻訳の主要なソースおよびフォールバックとしてデフォルト言語を使用します。これは`config.yml`で設定するか、環境変数で上書きできます。

**`config.yml`:**

```yaml
i18n:
  default_language: "en" # 希望のデフォルト言語を設定します（例：「en」、「zh」、「ja」）
```

**環境変数:**

`.env`ファイル（またはシステム環境）で`I18N_DEFAULT_LANGUAGE`環境変数を定義することで、`config.yml`の設定を上書きできます。これは、ローカルでの開発やテストに特に便利です。

```bash
I18N_DEFAULT_LANGUAGE="zh" # 例：config.ymlを上書きして中国語をデフォルトとして使用する
```

---

翻訳サービスは、`config.yml`ファイルの`translator`セクション、または環境変数を介して設定されます。`google`、`deepl`、`mock`の翻訳者から選択できます。

**`config.yml`:**

```yaml
translator:
  type: "mock" # 「google」、「deepl」、「mock」のいずれか
  api_key: "YOUR_API_KEY_HERE" # GoogleとDeepLに必要
```

**環境変数:**

```bash
TRANSLATOR_TYPE="mock" # 「google」、「deepl」、「mock」のいずれか
TRANSLATOR_API_KEY="YOUR_API_KEY_HERE"
```

## モックトランスレータ

これはデフォルトのトランスレータであり、開発に使用されます。外部APIを呼び出すことはありません。元のテキストに`"Translated from [source] to [target]: "`という文字列を単に追加するだけです。

モックトランスレータを使用するには、`config.yml`の`type`を`"mock"`に設定します。

## Google Cloud Translation API

Google Cloud Translation APIを使用するには、APIキーを提供する必要があります。

1. **APIキーを取得する:** Google Cloud PlatformプロジェクトからAPIキーを取得し、「Cloud Translation API」が有効になっていることを確認します。
2. **設定を行う:**
    - `config.yml`で、`translator.type`を`"google"`に設定します。
    - `translator.api_key`をGoogle Cloud APIキーに設定するか、`TRANSLATOR_API_KEY`環境変数を設定します。

## DeepL API

DeepL APIを使用するには、APIキーを提供する必要があります。

1. **APIキーを取得する:** DeepLアカウントからAPIキーを取得します。
2. **設定を行う:**
    - `config.yml`で、`translator.type`を`"deepl"`に設定します。
    - `translator.api_key`をDeepL APIキーに設定するか、`TRANSLATOR_API_KEY`環境変数を設定します。

## メンテナンスモード

サイトのメンテナンスモードは、管理パネルの「設定」ページで有効または無効にできます。

- **メンテナンスモードを有効にするには**:
  1. 管理パネルにログインします (`/admin`)。
  2. 「設定」ページに移動します (`/admin/settings`)。
  3. 「メンテナンスモードを有効にする」チェックボックスをオンにして、設定を保存します。

- **重要な注意点**:
  - メンテナンスモードが有効な場合、ログインしていないすべての訪問者にはメンテナンスページが表示されます。
  - **ログインしている管理者はメンテナンスページを表示せず**、通常通りサイトを閲覧できます。
  - メンテナンスページをテストするには、管理パネルから**ログアウト**していることを確認するか、別のブラウザを使用してください。

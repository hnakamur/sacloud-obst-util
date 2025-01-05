# sacloud-obst-util

さくらのクラウドのオブジェクトストレージ用のユーティリティコマンドラインツールです。

現在はバケットのストレージ使用量とオブジェクト数を出力する機能のみ提供しています。

## インストール方法

[Go](https://go.dev/)をインストールした環境では、以下のコマンドでインストールできます。

```
go install github.com/hnakamur/sacloud-obst-util@latest
```

## 使い方

`sacloud-obst-util --help` を実行すると使い方のヘルプが出力されます。

## スタティックリンクでのビルド手順

```
go build -tags netgo,osusergo
```

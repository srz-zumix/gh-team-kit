# instrctions

指示が曖昧な場合は編集せずに曖昧な箇所を指摘してください。

ソースコード中のコメントは英語で記載

## cmd

* オプションは基本的に変数で受け取ります
* RunE で処理を実装します
* Args で引数の検証をします
* gh/gh.go のラッパー関数を呼び出し、cmd package では github package を import しなくても良い設計にします

## gh package

* gh/github-client.go では API 呼び出しのエラーはフォーマットせずそのまま返します
* gh/gh.go には github-client.go の関数のラッパーを記述します
  * owner/repo などの string は使わず repository.Repository 型を引数に取ります

## README.md

* 各コマンドの Usage を記載します
* markdown の書式警告は修正してください
* README に記載するコマンドの説明は下記コメントブロックの内容のように記述してください

<!--
### コマンドの機能

```sh
コマンド列
```

コマンドの Long の説明
-->

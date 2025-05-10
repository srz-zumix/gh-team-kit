# instrctions

指示が曖昧な場合は編集せずに曖昧な箇所を指摘してください。

ソースコード中のコメントは英語で記載

## cmd

* 新しいコマンドを作成する場合は他のコマンドの実装を参照し、書き方など踏襲してください
* オプションは基本的に変数で受け取ります
* RunE で処理を実装します
* Args で引数の検証をします
* gh/*.go のラッパー関数を呼び出し、cmd package では github package を import しなくても良い設計にします
* エラーの場合はどういう操作をしようとしてエラーになったかメッセージに含めるようにしてください
* cmd/root.go は変更してはいけません
* cmd/**/*.go のサブコマンドは cobra.Command を return する関数を定義し、その中でコマンド実装してください
  * コマンドの登録は親コマンドの .go ファイルで行います
  * 関数名は New<コマンド名>Cmd としてください。例えば list コマンドであれば NewListCmd 、add コマンドであれば NewAddCmd となります
* owner もしくは repo オプションはオプショナルです。文字列が空の場合の処理は不要です

## gh package

* gh/client/*.go では API 呼び出しのエラーはフォーマットせずそのまま返します
* gh/member.go, gh/organizaion.go, gh/repo.go, gh/team.go, gh/user.go には github/client/*.go の関数のラッパーを記述します
  * owner/repo などの string は使わず repository.Repository 型を引数に取ります
  * ラッパー関数は ctx context.Context を第一、 g *GitHubClient を第二引数に取ります

## README.md

* 各コマンドの Usage を記載します
* markdown の書式警告は修正してください
* completion のヘルプは書かない
* サブコマンドを持っているサブコマンドの説明は不要です
* コマンドヘルプの項目は項目のタイトルでソートして記述します
* README に記載するコマンドの説明は下記コメントブロックの内容のように記述してください

<!--
### コマンドの機能

```sh
コマンド列
```

コマンドの Long の説明
-->

## コーディング規約

* fmt.Errorf: error strings should not end with punctuation or newlines (ST1005) go-staticcheck
* ローカルパッケージは github.com/srz-zumix/gh-team-kit/<path/to/dir> で import

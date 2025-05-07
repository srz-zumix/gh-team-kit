# instrctions

指示が曖昧な場合は編集せずに曖昧な箇所を指摘してください。

## cmd

* オプションは基本的に変数で受け取ります
* RunE で処理を実装します
* Args で引数の検証をします
* gh/gh.go のラッパー関数を呼び出し、cmd package では github package を import しなくても良い設計にします

## gh package

* gh/github.go では API 呼び出しのエラーはフォーマットせずそのまま返します
* gh/gh.go には github.go の関数のラッパーを記述します
  * owner/repo などの string は使わず repository.Repository 型を引数に取ります

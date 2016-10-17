sequenceDiagram
    participant Main処理
    participant 設定パラメータ取得
    participant Log設定
    participant メタデータ取得
    participant Agent登録
    participant ハートビート
    participant Log送信
    participant SQSポーリング
    participant Action実行
    participant Action結果送信

    Main処理->>設定パラメータ取得: GetConfig関数呼び出し
    設定パラメータ取得->>設定パラメータ取得: 設定情報(APIKey,EndPointなど)の取得
    Main処理->>Log設定: SetupLogger関数呼び出し    
    Log設定->>Log設定: ログ出力を設定
    Main処理->>メタデータ取得: GetHostMetaData関数呼び出し
    メタデータ取得->>メタデータ取得: Agentホストのメタデータ(ホスト名,IPアドレスなど)の取得
  loop 成功するまで
    Main処理->>Agent登録: RegisterAgent関数呼び出し
    Agent登録->>Agent登録: メタデータと共にServerにAgent登録
  end
  loop 定期的に実行
    Main処理->>ハートビート: Beat関数呼び出し
    ハートビート->>ハートビート: Serverとハートビート
    Main処理->>Log送信: UpdateLogs関数呼び出し
    Log送信->>Log送信: Serverにログを送信
    Main処理->>Agent登録: RegisterAgent関数呼び出し
    Agent登録->>Agent登録: ServerにAgent情報を送信
  end
    Main処理->>SQSポーリング: RunLoop関数呼び出し
  loop 定期的に実行
    SQSポーリング->>SQSポーリング: アラートメッセージがあるかキューを確認
  end
  opt アラートメッセージあり
    Main処理->>Action実行: ExecuteAction関数呼び出し
    Action実行->>Action実行: アラートに対応したRunbookの実行
  end
  opt Actionを実行した
    Main処理->>Action結果送信: SendActionOutput関数呼び出し
    Action結果送信->>Action結果送信: Runbook実行結果をServerに送信
  end

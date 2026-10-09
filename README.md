# ingest-poc

OpenAlex APIから研究成果データをページ単位で取得し、PostgreSQL・DuckDB・SQLiteへ転送するPoC。

AirbyteのSource／Destination／Connection、状態管理、再開処理の考え方を参考にしながら、障害からの復旧とマルチテナント環境での実行制御を自分で設計・検証する。

> [!NOTE]
> このREADMEは合意した要件と検証計画を記録したものです。実装状況・テスト結果・性能の実測値はまだ記録していません。「対応する」「保証する」は本PoCの実装・検証目標を表します。

## 目的

優先順位は次のとおり。

1. **正確性と復旧**：途中で失敗しても、再開により対象レコードを欠落・重複なく転送できることを検証する。
2. **リソースと性能**：処理時間・メモリ使用量・各処理の待ち時間を計測し、ボトルネックと改善効果を説明できるようにする。
3. **転送先の違い**：PostgreSQL、DuckDB、SQLiteで共通化できる責務と、DBごとの実装が必要な責務を整理する。

サービスとしての設計も考慮し、Workspaceをテナント境界として、設定・実行履歴・再開位置・転送先データを分離する。異なるWorkspaceのジョブは並行して処理する。


## スコープ

| 項目 | 今回の対象 |
| --- | --- |
| Source | OpenAlex APIのみ。取得対象は`works` |
| Destination | PostgreSQLを先に実装し、DuckDB・SQLiteを追加 |
| 転送先の選択 | 1つのConnection／Jobにつき1つのDestination |
| 取得方式 | フィルタで範囲を限定した初回取得と、同じJobの中断位置からの再開 |
| 起動 | 利用者による手動実行 |
| キュー | 管理用PostgreSQLに保存する共有の永続キュー |
| 並行実行 | システム全体とWorkspaceごとに上限を設定 |
| テナント | Workspace。認証・所属確認・操作対象の所有関係をサーバーで検証 |
| 実行基盤 | 最初はWorkerを1プロセスとし、複数Jobを並行実行 |
| UI | 接続設定、ジョブ開始、状態・履歴の確認、失敗・中断後の再開 |

今回は、定期実行、更新日時に基づく差分同期、削除の反映、別の実APIへの対応、汎用プラグイン機構、課金・招待・複雑な権限管理、自動フェイルオーバーを含めない。

OpenAlex以外のSourceを今すぐ実装する予定はない。Sourceのインターフェースを設け、固定データと障害を返すテスト用Sourceに差し替えられる構成にする。

## 基本構成

APIプロセスは認証・認可、設定管理、ジョブの永続化を担当する。Workerプロセスは管理DBから実行可能なJobを取り出し、Runnerで転送する。ブラウザを閉じても処理は継続する。

管理DBには、Workspace・設定・Job・Attempt・Checkpointなどを保存する。転送先DBにはOpenAlexから取得したレコードを保存する。管理用PostgreSQLと、転送先としてのPostgreSQLは役割が異なる。

| 用語 | 意味 |
| --- | --- |
| Source | データ取得の実装。初期実装はOpenAlex |
| Destination | データ書き込みの実装 |
| Connection | Source／Destinationの設定と取得条件の組み合わせ |
| Stream | 同種のレコードの集合。今回は`works` |
| Page | APIの1回の成功応答に含まれるレコードのまとまり |
| Batch | Destinationへ一度に渡す書き込み単位。初期構成ではPageと1対1 |
| Job | 利用者が開始する論理的な同期処理 |
| Attempt | Jobを実行する1回の試行。手動再開で追加される |
| Checkpoint | 書き込みが確認できた位置と、取得完了の状態 |

PageをStreamとは呼ばない。Pageごとの管理テーブルや、取得範囲を分割するSliceは今回設けない。

## 正確性と再開の方針

1. 最後に確定したCheckpointからPageを取得する。
2. そのPageのレコードをDestinationへ書き込む。
3. 書き込みの確定を確認してからCheckpointを保存する。
4. 次のPageへ進む。最後はCheckpointとJob／Attemptの成功状態を管理DBの同じトランザクションで確定する。

転送先への書き込みと管理DBへのCheckpoint保存を、単一のトランザクションにはできない。書き込み後・Checkpoint保存前に停止した場合は、同じPageを再取得・再書き込みする。

そのため、取得・配送は**少なくとも1回（at-least-once）**を前提とし、OpenAlex Work IDをキーとするupsertで、同じレコードの再処理による重複を防ぐ。全DBをまたぐ「厳密に1回だけの実行」は主張しない。

固定データを用いたテストでは、期待するID集合とJSONの内容を比較する。件数だけでは正確性を判定しない。実際のOpenAlex APIは取得中にデータが変化し得るため、APIからの取得を特定時点の完全なスナップショットとしては扱わない。

## キューとマルチテナント

- 待機Jobは管理DBに永続化し、実行枠が空くとWorkerが取り出す。
- システム全体とWorkspaceごとの上限を満たす、古い待機Jobから選ぶ。
- Workspace Aが上限に達していても、実行可能なWorkspace BのJobを取り出せるようにする。
- 初期の候補値は**全体2件・Workspaceごと1件**。数値は仮値であり、計測と検証で調整する。
- Jobの割り当ては短い管理DBトランザクションで直列化する。割り当て後のデータ転送は並行する。
- 失敗・中断したJobは自動で全面再実行せず、利用者の手動再開を待つ。待機Jobの自動取り出しとは区別する。

各リソースに`workspace_id`を持たせるだけでなく、API・関連付け・Worker・物理的な出力先の各段階でテナント境界を検証する。並行数の制限だけでは、CPU・メモリ・API利用枠の完全な性能分離までは実現しない。

## 保存形式

取得したWorkを、OpenAlex IDとJSONとして保存する。最初は列への展開や自動スキーマ推論を行わない。

| 転送先 | JSONの保存方式 |
| --- | --- |
| PostgreSQL | JSONB |
| DuckDB | JSON型 |
| SQLite | TEXTとJSON妥当性の検証 |

一意性の範囲はWorkspace・Connection・Record ID。物理的なDB／スキーマ／テーブル／ファイルの割り当ては詳細を決める必要がある。再度のJobでは同じIDを更新するが、今回取得しなかった既存レコードは削除しない。

## 実装の順序

1. Source・Runner・PostgreSQL Destinationで、固定データとOpenAlexのPageを転送する。
2. 管理DB、Job／Attempt／Checkpoint、永続キュー、最小API・Workerを実装する。Workspaceの所有関係は最初から持たせる。
3. リトライ、手動再開、正常終了処理、強制停止からの復旧を検証する。
4. 認証・認可、Workspace間の分離、並行数制御を一貫して実装・検証する。
5. 設定・開始・進捗・履歴・再開のUIを実装する。
6. DuckDB・SQLiteに、同じ書き込み・復旧の契約を実装する。
7. 性能を計測し、原因を特定してから改善する。

認証・認可が完成するまでは、マルチテナントサービスとして公開しない。

## 検証と成果物

主要な検証対象は、正常転送、書き込み直後の停止、Checkpoint保存失敗、書き込み結果が不明な通信障害、Worker再起動、テナント越境操作、並行数上限、重複リクエストである。

性能は処理時間、records/s、bytes/s、Workerの最大メモリ使用量、API取得・書き込み・Checkpoint保存の時間、リトライ・再処理量で評価する。固定データによる再現可能な試験と、実APIへの結合試験を分ける。

実装後に、実行手順、テスト結果、計測条件と実測値、改善前後の比較、既知の制約を追記する。現時点で起動コマンドや性能値を仮の実績として掲載しない。

## Airbyteとの関係

AirbyteはSource／Destination／Connection、状態管理、再開処理の参考にする。本PoCはOpenAlexの`works`に用途を絞り、次の判断と検証を自分で行う。

- 転送先とCheckpointが別DBでも、再実行に耐えられる契約を設計する。
- Workspaceごとの分離と共有キューの実行制御を実装する。
- 3種類の転送先で、同じ障害シナリオを検証する。
- 計測結果から、性能上の問題と改善効果を説明する。

アピールの根拠は設計理由・実装・再現可能な検証結果とする。機能数や性能でAirbyteを上回るとは、比較検証なしに主張しない。

## 詳細設計と参考資料

- [詳細設計・決定記録・未決定事項](docs/design.md)
- [Airbyte Documentation](https://docs.airbyte.com/platform)
- [Airbyte Core Concepts](https://docs.airbyte.com/platform/using-airbyte/core-concepts)
- [Airbyte Resumability](https://docs.airbyte.com/platform/understanding-airbyte/resumability)
- [OpenAlex Paging](https://help.openalex.org/api/paging/)
- [技術面接についての参考記事](https://note.com/jammaru/n/ne00f69c38fec)

# 完成形（第11章）

3つのコンテキストを1つの module に置く予約システムです。

## 確かめ方

```bash
go test ./...
go test -tags=integration ./...
golangci-lint run ./...
```

`-tags=integration` のテストは Docker で PostgreSQL を起動します。

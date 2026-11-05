## 改了什么



## 为什么



## 怎么验证的

- [ ] `go test ./...` 通过（改了跨 goroutine 的状态时还跑了 `go test -race ./...`）
- [ ] `go vet ./...` 和 `go tool mygo vet .` 没有报错
- [ ] 改了界面：用 `go tool mygo dev` 实际点过，浅色和深色外观都看过
- [ ] 用户能感知的改动：在 `CHANGELOG.md` 的「未发布」下加了一行
- [ ] 界面有明显变化：更新了 `docs/screenshots/`

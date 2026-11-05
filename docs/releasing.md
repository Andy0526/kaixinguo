# 发版流程

发版只能在维护者的 Mac 上做：需要 Developer ID 证书和苹果公证的凭据。

## 一次性准备

1. 钥匙串里有「Developer ID Application」证书。查看：

   ```sh
   security find-identity -v -p codesigning
   ```

2. 把公证凭据存进钥匙串（会要求 Apple ID、App 专用密码和 Team ID）：

   ```sh
   xcrun notarytool store-credentials notary
   ```

   脚本默认用名为 `notary` 的配置，可以用 `NOTARY_PROFILE` 换成别的名字。

公证放在 `scripts/release.sh` 里，而不是 `mygo.json`，这样没有证书的人从源码构建也不受影响。

## 每次发版

以发布 `0.2.0` 为例。

1. **确认 main 是绿的**：CI 通过，本地 `go test ./...` 通过，`go tool mygo dev` 实际用一遍新功能。

2. **改版本号**：`mygo.json` 的 `version` 改成 `0.2.0`。

3. **整理更新日志**：把 `CHANGELOG.md` 里「未发布」下的内容移到新的 `## [0.2.0] - <日期>` 下面，更新文末的链接。

4. **提交并打标签**：

   ```sh
   git commit -am "发布 0.2.0"
   git tag v0.2.0
   git push origin main v0.2.0
   ```

5. **打包、签名、公证**（通常几分钟）：

   ```sh
   MACOS_SIGNING_IDENTITY="Developer ID Application: 名字 (TEAMID)" scripts/release.sh
   ```

   脚本会构建通用版（Apple 芯片 + Intel），签名，提交公证并等待结果，把票据附到 `.dmg` 和 `.app` 上，生成 `.sha256`，最后用 Gatekeeper 验一遍。产物在 `build/darwin-universal/`：

   - `kai-xin-guo-0.2.0.dmg`
   - `kai-xin-guo-0.2.0.dmg.sha256`

   公证失败时脚本会打印提交 ID，用 `xcrun notarytool log <ID> --keychain-profile notary` 看原因。已经构建好、只想重新公证的话，加 `SKIP_BUILD=1`。

6. **发布到 GitHub**：说明照抄 `CHANGELOG.md` 里这个版本的内容，再加上安装说明。

   ```sh
   gh release create v0.2.0 \
     build/darwin-universal/kai-xin-guo-0.2.0.dmg \
     build/darwin-universal/kai-xin-guo-0.2.0.dmg.sha256 \
     --title "开心果 0.2.0" --notes-file <说明文件>
   ```

7. **验收**：从 Releases 页面下载 `.dmg`，在另一台 Mac（或删掉本地的应用后）打开，确认双击就能运行，没有「无法验证开发者」的提示。

## 版本号

遵循[语义化版本](https://semver.org/lang/zh-CN/)。1.0 之前：

- 新功能、可见的行为变化：次版本号加一（0.1.x → 0.2.0）
- 只有修复：修订号加一（0.2.0 → 0.2.1）

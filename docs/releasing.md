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

3. 准备好**更新签名私钥**。已安装的开心果只接受用这把私钥签名的更新，它对应 `mygo.json` 里 `updates.publicKey` 的公钥：

   - 默认位置是 `~/Library/Application Support/mygo/update-keys/kaixinguo/mygo-update.key`，放在别处就用 `MYGO_UPDATE_KEY_FILE` 指过去。
   - **务必备份到密码管理器**。丢了它，已安装的应用就再也收不到更新，只能让用户手动重装；泄露了，别人就能给用户推送代码。
   - 不要提交进仓库，也不要在终端里 `cat` 出来。换一台 Mac 发版时，把这个文件复制过去（权限保持 `600`）。
   - 万一真的丢了：用 `go tool mygo keygen -force -o <目录>` 生成新的一对，把新公钥写进 `mygo.json` 发版，并在发版说明里请用户手动下载这一版。

公证放在 `scripts/release.sh` 里，而不是 `mygo.json`，这样没有证书的人从源码构建也不受影响。没有私钥时 `mygo build` 只会提示「Not signing an update」，照常打包。

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

   脚本会构建通用版（Apple 芯片 + Intel），签名，用更新私钥签名更新包，提交公证并等待结果，把票据附到 `.dmg` 和 `.app` 上，生成 `.sha256`，最后用 Gatekeeper 验一遍，并列出要上传的文件。产物在 `build/darwin-universal/`：

   - `kai-xin-guo-0.2.0.dmg`、`kai-xin-guo-0.2.0.dmg.sha256`：给新用户下载安装
   - `kai-xin-guo-0.2.0-darwin-universal.tar.gz`：给已安装的应用自动更新
   - `kai-xin-guo-<旧版本>-to-0.2.0-darwin-universal.delta`：相对前几个版本的增量更新，通常只有完整包的一小部分；第一次带更新的版本没有
   - `update-darwin-universal.json`：更新清单，已安装的应用读它判断有没有新版本，更新说明取自 `CHANGELOG.md`（这一版没有对应的一节时构建会失败）

   公证失败时脚本会打印提交 ID，用 `xcrun notarytool log <ID> --keychain-profile notary` 看原因。已经构建好、只想重新公证的话，加 `SKIP_BUILD=1`。

6. **发布到 GitHub**：说明照抄 `CHANGELOG.md` 里这个版本的内容，再加上安装说明。先建成草稿，把脚本列在 `build/darwin-universal/upload.txt` 里的文件全部传上去（安装包、更新包、增量包和清单）：

   ```sh
   gh release create v0.2.0 --draft $(cat build/darwin-universal/upload.txt) \
     --title "开心果 0.2.0" --notes-file <说明文件>
   ```

   在网页上确认文件齐了，再发布：

   ```sh
   gh release edit v0.2.0 --draft=false
   ```

   发布后它就是「最新版本」，已安装的应用会读到 `releases/latest/download/update-darwin-universal.json`，在启动 10 秒后或下一次每日检查时提醒用户。文件名不能改：清单里写死了更新包的地址。

7. **验收**：从 Releases 页面下载 `.dmg`，在另一台 Mac（或删掉本地的应用后）打开，确认双击就能运行，没有「无法验证开发者」的提示。再打开一个旧版本，点「开心果 › 检查更新…」，确认能看到新版本、装完重启后版本号对了。

## 版本号

遵循[语义化版本](https://semver.org/lang/zh-CN/)。1.0 之前：

- 新功能、可见的行为变化：次版本号加一（0.1.x → 0.2.0）
- 只有修复：修订号加一（0.2.0 → 0.2.1）

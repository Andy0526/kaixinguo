# 开心果

一个极简的红果短剧桌面客户端：浏览推荐、排行榜、新剧和热搜，搜索短剧，选集播放，记住看到哪里。

基于 [openhongguo](https://github.com/beilunyang/openhongguo) SDK 和 [MyGo](https://github.com/egoist/mygo) 的原生 UI：界面由 Go 直接在 GPU 上绘制，没有网页、没有 JavaScript 运行时，整个 App 约 12 MB，窗口即开即用。

## 下载

从 [Releases](https://github.com/kadaliao/kaixinguo/releases) 下载 `kai-xin-guo-<版本>.dmg`，打开后把「开心果」拖进「应用程序」。一个安装包同时支持 Apple 芯片和 Intel，需要 macOS 12 或更高版本。

应用用 Developer ID 签名，并通过了苹果公证，双击即可打开。

## 从源码构建

需要 Go 1.27.3+，只在 macOS 上测试过。

```sh
go tool mygo dev     # 开发：改代码自动重启
go tool mygo build   # 打包：build/darwin-arm64/开心果.app 和 .dmg
go test ./...        # 测试，不需要网络和窗口
```

发版用 `scripts/release.sh`：打包通用版、用 Developer ID 签名、提交公证并附上公证票据。用法写在脚本开头。

排查网络问题时，用 `KX_DEBUG=1` 从终端启动 `开心果.app/Contents/MacOS/开心果`，会打印每个请求。

快捷键：⌘F 搜索，⌘[ 返回；播放时空格暂停，← → 快退快进 5 秒，↑ ↓ 换集，Esc 退出播放。

## 实现要点

- **启动快**：主窗口是原生 UI。上次的推荐列表和封面缓存在本地，打开就显示，新内容在后台刷新。只有开始播放时才创建 WebView。
- **播放**：剧集是 CENC（AES-CTR）加密的 HEVC MP4。`internal/cenc` 解析 `moov`，按需原地解密任意字节区间，并把 `encv`、`sinf` 等盒子改名为同长度的明文盒子。`proxy.go` 在 `127.0.0.1` 上边下载边解密，交给 WebView 里的 `<video>` 播放，可以随意拖动进度，不依赖 ffmpeg。
- **封面**：CDN 只给 HEIC。`internal/imageio` 通过 purego 调用 macOS 的 ImageIO 解码（不用 cgo），缩小后以 JPEG 缓存在磁盘。
- **观看记录**：保存在 `~/Library/Application Support/开心果/history.json`，侧栏显示最近观看，详情页可以继续播放。

暂不支持分段（DASH/HLS）的长视频，这类内容会提示不支持。

## 许可证

[MIT](LICENSE)

## 免责声明

本项目仅供学习和研究，请遵守 openhongguo 的[免责声明](https://github.com/beilunyang/openhongguo#免责声明)。内容版权归原权利人所有。

# 更新日志

这里记录每个版本用户能感知到的变化。格式参考 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，版本号遵循[语义化版本](https://semver.org/lang/zh-CN/)。

## [未发布]

## [0.2.0] - 2026-11-05

### 新增

- 收藏：剧集页新增「收藏」按钮，列表里的卡片右键也能收藏或取消收藏。侧栏新增「我的 › 收藏」，列出收藏的剧；已收藏的剧在列表里标有星号。收藏保存在 `~/Library/Application Support/开心果/favorites.json`。

### 修复

- 侧栏里亮着的那一项点了没反应：从「推荐」打开一部剧后再点「推荐」，或者在搜索结果里点「热搜」，现在会回到那一页。
- 退出时等待观看记录和收藏写入磁盘，刚操作完就退出也不会丢失。

## [0.1.0] - 2026-11-05

第一个版本。

### 新增

- 推荐、排行榜（按类型和榜单）、新剧上线（前后两周的日历）、热搜
- 搜索短剧，同一部剧的各季按顺序列出
- 详情页选季、选集，记住看到哪里，可以继续播放
- 播放时自动连播下一集，一季看完接下一季
- 跟随系统的浅色和深色外观
- 用 Developer ID 签名并经过苹果公证的通用安装包，支持 Apple 芯片和 Intel

### 已知限制

- 暂不支持分段（DASH/HLS）的长视频
- 第一次启动要注册匿名设备，网络慢时列表可能要等十几秒才出来；之后启动会先显示缓存

[未发布]: https://github.com/kadaliao/kaixinguo/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/kadaliao/kaixinguo/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/kadaliao/kaixinguo/releases/tag/v0.1.0

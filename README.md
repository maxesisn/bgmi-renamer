# bgmi-renamer
Automatic anime file renamer for your qBittorrent client with the power of LLM.

## 前置需求
1. `qBitTorrent`
2. `BGmi(可选)` 需保证对应客户端不依赖文件名工作

## 使用
1. 在Release页下载对应架构的`bgmi-renamer`
2. 放入与`qBitTorrent`相同的运行环境中
3. 在`qBitTorrent`的`选项`界面，选择`下载`页签，勾选`torrent 完成时运行外部程序`，在右侧文本框中输入
```
/xxx/bgmi-renamer <qBitTorrent网页地址> <qBitTorrent用户名> <qBitTorrent密码> <OpenAI SecretKey> "%L" "%F" "%I"
```
4. 等待新种子文件完成下载，查看效果

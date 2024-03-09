# bgmi-renamer
Automatic anime file renamer for your qBittorrent client with the power of LLM.

## 前置需求
1. `qBitTorrent`
2. `BGmi(可选)` 需保证对应客户端不依赖文件名工作

## 使用
1. 在Release页下载对应架构的`bgmi-renamer`
2. 放入与`qBitTorrent`相同的运行环境中
3. 在程序同目录下，新建`bgmi-renamer.conf`文件，写入：
    ```
    qbit-url=https://xxx.xx:114514
    qbit-username=admin
    qbit-password=password
    openai-token=sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
    openai-url=https://api.openai.com/v1
    ```
3. 在`qBitTorrent`的`选项`界面，选择`下载`页签，勾选`torrent 完成时运行外部程序`，在右侧文本框中输入
```
/xxx/bgmi-renamer -category "%L" -path "%F" -hash "%I"
```
4. 等待新种子文件完成下载，查看效果

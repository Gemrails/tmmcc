# 功能说明
完成从网卡抓取网络数据包，并根据不同协议进行解析。将结果通过zmq发送到server。
# 进度
v1 :完成对http协议的解码。暂时未输出到zmq.显示消息日志

# 整体架构
![](./test/jiagou.jpeg)
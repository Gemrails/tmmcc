# 多协议性能分析数据搜集插件
支持协议：
* http/1.1
* mysql
* http/2.0
* redis
* postgresql


## 统计项说明

### mysql
* sql执行数量（累计值）
* sql执行平均时间（瞬时值）
* sql执行最慢的10个sql（消息系统）
* sql执行最多的10个sql（消息系统）

### http/1.1
* 分方法请求数量(累计值)
* 异常请求数量(5xx,4xx)(累计值)
* 平均相应时间(瞬时值)
* 响应时间最长的10个地址（消息系统）
* 请求次数做多的10个地址（消息系统）
* 异常最多的10个地址 (消息系统)
* 独立来源IP数量 (累计瞬时值)--（如果是在负载均衡后面，来源IP从协议头中获取）



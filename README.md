# 雾霾探测系统

一个基于 `Go + SQLite + 原生 H5` 的手机端雾霾探测系统原型。

## 已实现

- 浏览器定位或手动切换城市
- 服务端保存客户端城市信息
- 实时天气查询
- 实时空气质量查询
- 24 小时温度/湿度折线图
- SQLite 缓存与快照保存

## 运行前准备

和风天气现在要求使用你自己的 `API Host`。只有 `KEY` 不够，请在控制台里找到类似下面这样的地址：

```text
abc1234xyz.def.qweatherapi.com
```

然后在项目根目录创建 `.env`：

```text
PORT=8080
QWEATHER_API_KEY=你的KEY
QWEATHER_BASE_URL=https://你的APIHost
SQLITE_PATH=./data/weather.db
CACHE_MINUTES=10
```

## 启动

```bash
go run ./cmd/server
```

启动后访问：

```text
http://localhost:8080
```

## 接口

- `GET /api/health`
- `GET /api/dashboard?client_id=xxx`
- `POST /api/location`
- `POST /api/city/select`
